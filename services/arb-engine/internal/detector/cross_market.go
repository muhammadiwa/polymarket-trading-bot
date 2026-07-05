package detector

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type CrossMarketDetector struct {
	registry        ports.RelationshipRepository
	nearResolution  *NearResolutionDetector
	minProfitThreshold decimal.Decimal
	scoreThreshold     decimal.Decimal
	logger             *zap.Logger
}

func NewCrossMarketDetector(
	registry ports.RelationshipRepository,
	nearResolution *NearResolutionDetector,
	minProfitThreshold string,
	scoreThreshold string,
	logger *zap.Logger,
) *CrossMarketDetector {
	th, err := decimal.NewFromString(minProfitThreshold)
	if err != nil {
		th = decimal.NewFromFloat(0.01)
		logger.Warn("invalid min_profit_threshold, using default 0.01", zap.String("value", minProfitThreshold), zap.Error(err))
	}
	st, err := decimal.NewFromString(scoreThreshold)
	if err != nil {
		st = decimal.NewFromFloat(0.01)
		logger.Warn("invalid cross_market_score_threshold, using default 0.01", zap.String("value", scoreThreshold), zap.Error(err))
	}
	return &CrossMarketDetector{
		registry:           registry,
		nearResolution:     nearResolution,
		minProfitThreshold: th,
		scoreThreshold:     st,
		logger:             logger,
	}
}

// Detect finds cross-market arbitrage opportunities for the given market event.
// It returns a slice of opportunities found between the event's market and related markets.
func (d *CrossMarketDetector) Detect(ctx context.Context, event ports.MarketPriceUpdated, prices map[string]ports.MarketPriceUpdated) []*ports.Opportunity {
	start := time.Now()

	related, err := d.registry.GetRelatedMarkets(ctx, event.MarketID)
	if err != nil {
		d.logger.Error("failed to get related markets", zap.String("market_id", event.MarketID), zap.Error(err))
		return nil
	}

	if len(related) == 0 {
		return nil
	}

	var opportunities []*ports.Opportunity

	for _, rel := range related {
		relatedPrice, ok := prices[rel.MarketBID]
		if !ok {
			continue
		}

		opp := d.evaluatePair(event, relatedPrice, rel)
		if opp != nil {
			opportunities = append(opportunities, opp)
		}
	}

	latencyMs := time.Since(start).Milliseconds()
	if latencyMs > 100 {
		d.logger.Warn("cross-market detection latency exceeded 100ms",
			zap.String("market_id", event.MarketID),
			zap.Int64("latency_ms", latencyMs),
		)
	}

	return opportunities
}

func (d *CrossMarketDetector) evaluatePair(eventA, eventB ports.MarketPriceUpdated, rel ports.MarketRelationship) *ports.Opportunity {
	one := decimal.RequireFromString("1.00")

	var spread decimal.Decimal
	var marketAID, marketBID string

	switch rel.RelationshipType {
	case "same_event":
		sum := eventA.YESPrice.Add(eventB.NOPrice)
		spread = one.Sub(sum)
		marketAID = eventA.MarketID
		marketBID = eventB.MarketID

	case "date_variant":
		spread = eventB.YESPrice.Sub(eventA.YESPrice)
		if spread.IsNegative() {
			return nil
		}
		marketAID = eventA.MarketID
		marketBID = eventB.MarketID

	case "correlated_outcome":
		diff := eventA.YESPrice.Sub(eventB.YESPrice).Abs()
		confidence := rel.Confidence
		if confidence < 0 {
			confidence = 0
		}
		if confidence > 1 {
			confidence = 1
		}
		spread = diff.Mul(decimal.NewFromFloat(confidence))
		marketAID = eventA.MarketID
		marketBID = eventB.MarketID

	default:
		return nil
	}

	// #2: Return opportunity with filter_reason instead of nil for below-threshold
	if spread.LessThanOrEqual(d.minProfitThreshold) {
		return &ports.Opportunity{
			ID:               uuid.New().String(),
			MarketID:         marketAID,
			YESPrice:         eventA.YESPrice,
			NOPrice:          eventA.NOPrice,
			Spread:           spread,
			Score:            decimal.Zero,
			FilterReason:     "spread_below_threshold",
			DetectedAt:       time.Now().UTC(),
			RelatedMarketID:  marketBID,
			RelationshipType: rel.RelationshipType,
		}
	}

	// Check near-resolution for both markets
	nearResA, factorA := d.nearResolution.Check(eventA.MarketID)
	nearResB, factorB := d.nearResolution.Check(eventB.MarketID)
	confidenceFactor := factorA
	if factorB < confidenceFactor {
		confidenceFactor = factorB
	}

	nearRes := nearResA || nearResB

	opp := &ports.Opportunity{
		ID:               uuid.New().String(),
		MarketID:         marketAID,
		YESPrice:         eventA.YESPrice,
		NOPrice:          eventA.NOPrice,
		Spread:           spread,
		DetectedAt:       time.Now().UTC(),
		LatencyMs:        0,
		RelatedMarketID:  marketBID,
		RelationshipType: rel.RelationshipType,
		NearResolution:   nearRes,
		ConfidenceFactor: confidenceFactor,
	}

	return opp
}

// DetectCascadeRisk checks if multiple correlated markets have concurrent opportunities.
// Returns true if 2+ correlated markets also have spreads above threshold.
func (d *CrossMarketDetector) DetectCascadeRisk(
	ctx context.Context,
	event ports.MarketPriceUpdated,
	prices map[string]ports.MarketPriceUpdated,
) (bool, []string) {
	related, err := d.registry.GetRelatedMarkets(ctx, event.MarketID)
	if err != nil {
		// #6: Log repository errors instead of swallowing
		d.logger.Error("failed to get related markets for cascade risk", zap.String("market_id", event.MarketID), zap.Error(err))
		return false, nil
	}
	if len(related) == 0 {
		return false, nil
	}

	one := decimal.RequireFromString("1.00")
	var correlatedIDs []string

	for _, rel := range related {
		relatedPrice, ok := prices[rel.MarketBID]
		if !ok {
			continue
		}

		var spread decimal.Decimal
		switch rel.RelationshipType {
		case "same_event":
			spread = one.Sub(relatedPrice.YESPrice).Sub(relatedPrice.NOPrice)
		case "date_variant":
			// #3: Align with evaluatePair — related market price - event price
			spread = relatedPrice.YESPrice.Sub(event.YESPrice)
		case "correlated_outcome":
			diff := event.YESPrice.Sub(relatedPrice.YESPrice).Abs()
			// #4: Clamp confidence to [0, 1]
			confidence := rel.Confidence
			if confidence < 0 {
				confidence = 0
			}
			if confidence > 1 {
				confidence = 1
			}
			spread = diff.Mul(decimal.NewFromFloat(confidence))
		}

		if spread.GreaterThan(d.minProfitThreshold) {
			correlatedIDs = append(correlatedIDs, rel.MarketBID)
		}
	}

	cascadeRisk := len(correlatedIDs) >= 2
	return cascadeRisk, correlatedIDs
}
