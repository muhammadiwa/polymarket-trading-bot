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
	}
	st, err := decimal.NewFromString(scoreThreshold)
	if err != nil {
		st = decimal.NewFromFloat(0.01)
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

	// Strategy: Buy YES on A + Buy NO on B (or vice versa) when prices are misaligned
	// For same-event: if A.YES + B.NO < 1.00 - threshold, there's an arbitrage
	// For date-variant: if A.YES is significantly cheaper than B.YES (earlier deadline should be more expensive)
	// For correlated-outcome: if A.YES and B.YES diverge beyond correlation threshold

	var spread decimal.Decimal
	var marketAID, marketBID string

	switch rel.RelationshipType {
	case "same_event":
		// Buy YES_A + NO_B when YES_A + NO_B < 1.00
		sum := eventA.YESPrice.Add(eventB.NOPrice)
		spread = one.Sub(sum)
		marketAID = eventA.MarketID
		marketBID = eventB.MarketID

	case "date_variant":
		// Earlier deadline (A) should have higher YES price than later deadline (B)
		// If A.YES < B.YES, buy A.YES (cheaper) for same underlying question
		spread = eventB.YESPrice.Sub(eventA.YESPrice)
		marketAID = eventA.MarketID
		marketBID = eventB.MarketID

	case "correlated_outcome":
		// If A.YES and B.YES diverge significantly, there's an opportunity
		// Spread = |A.YES - B.YES| * correlation_confidence
		diff := eventA.YESPrice.Sub(eventB.YESPrice).Abs()
		spread = diff.Mul(decimal.NewFromFloat(rel.Confidence))
		marketAID = eventA.MarketID
		marketBID = eventB.MarketID

	default:
		return nil
	}

	if spread.LessThanOrEqual(d.minProfitThreshold) {
		return nil
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
