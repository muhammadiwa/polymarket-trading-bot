package scorer

import (
	"context"
	"time"

	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
)

var (
	defaultFillProb      = decimal.NewFromFloat(0.5)
	minHistoricalSamples = 100
)

type FillProbabilityEstimator struct {
	oppLogger         ports.OpportunityLogger
	orderbookWeight   decimal.Decimal
	historicalWeight  decimal.Decimal
	requiredDepth     decimal.Decimal
}

func NewFillProbabilityEstimator(oppLogger ports.OpportunityLogger, orderbookWeight, historicalWeight, requiredDepth float64) *FillProbabilityEstimator {
	return &FillProbabilityEstimator{
		oppLogger:        oppLogger,
		orderbookWeight:  decimal.NewFromFloat(orderbookWeight),
		historicalWeight: decimal.NewFromFloat(historicalWeight),
		requiredDepth:    decimal.NewFromFloat(requiredDepth),
	}
}

func (e *FillProbabilityEstimator) Estimate(ctx context.Context, liquidityDepth decimal.Decimal, marketID string) decimal.Decimal {
	orderbookEstimate := estimateFromOrderbook(liquidityDepth, e.requiredDepth)

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	historicalRate, sampleCount, err := e.oppLogger.GetHistoricalFillRate(ctx, marketID, 30)
	if err != nil || sampleCount < minHistoricalSamples {
		return orderbookEstimate
	}

	blended := e.orderbookWeight.Mul(orderbookEstimate).Add(e.historicalWeight.Mul(historicalRate))
	if blended.GreaterThan(decimal.NewFromFloat(1.0)) {
		return decimal.NewFromFloat(1.0)
	}
	if blended.IsNegative() {
		return decimal.Zero
	}
	return blended
}

func estimateFromOrderbook(depth, requiredDepth decimal.Decimal) decimal.Decimal {
	if depth.IsZero() || depth.IsNegative() {
		return defaultFillProb
	}

	if requiredDepth.IsZero() {
		return defaultFillProb
	}

	ratio := depth.Div(requiredDepth)
	if ratio.GreaterThan(decimal.NewFromFloat(1.0)) {
		return decimal.NewFromFloat(1.0)
	}
	return ratio
}
