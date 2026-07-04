package scorer

import (
	"context"

	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
)

var (
	defaultLiquidity = decimal.NewFromFloat(0.5)
)

type Scorer struct {
	fillProbEstimator *FillProbabilityEstimator
	maxDepth          decimal.Decimal
}

func NewScorer(fillProbEstimator *FillProbabilityEstimator, maxDepth float64) *Scorer {
	return &Scorer{
		fillProbEstimator: fillProbEstimator,
		maxDepth:          decimal.NewFromFloat(maxDepth),
	}
}

func (s *Scorer) Score(ctx context.Context, opp *ports.Opportunity, liquidityDepth decimal.Decimal, marketID string) {
	liquidity := normalizeLiquidity(liquidityDepth, s.maxDepth)
	fillProb := s.fillProbEstimator.Estimate(ctx, liquidityDepth, marketID)

	opp.Liquidity = liquidity
	opp.FillProbability = fillProb
	opp.Score = opp.Spread.Mul(liquidity).Mul(fillProb)
}

func normalizeLiquidity(depth, maxDepth decimal.Decimal) decimal.Decimal {
	if depth.IsZero() || depth.IsNegative() {
		return defaultLiquidity
	}

	normalized := depth.Div(maxDepth)
	if normalized.GreaterThan(decimal.NewFromFloat(1.0)) {
		return decimal.NewFromFloat(1.0)
	}
	return normalized
}

func CalculateScore(spread, liquidity, fillProbability decimal.Decimal) decimal.Decimal {
	return spread.Mul(liquidity).Mul(fillProbability)
}
