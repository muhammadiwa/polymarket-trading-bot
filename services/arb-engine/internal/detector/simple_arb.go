package detector

import (
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/arb-engine/internal/ports"
	"github.com/shopspring/decimal"
)

const oneDollar = "1.00"

type SimpleArbDetector struct {
	minProfitThreshold decimal.Decimal
}

func NewSimpleArbDetector(minProfitThreshold string) *SimpleArbDetector {
	th, err := decimal.NewFromString(minProfitThreshold)
	if err != nil {
		th = decimal.NewFromFloat(0.01)
	}
	return &SimpleArbDetector{
		minProfitThreshold: th,
	}
}

func (d *SimpleArbDetector) Detect(event ports.MarketPriceUpdated) (*ports.Opportunity, int64) {
	start := time.Now()

	one := decimal.RequireFromString(oneDollar)
	spread := one.Sub(event.YESPrice).Sub(event.NOPrice)

	latencyMs := time.Since(start).Milliseconds()

	if spread.LessThanOrEqual(d.minProfitThreshold) {
		return nil, latencyMs
	}

	opp := &ports.Opportunity{
		ID:         uuid.New().String(),
		MarketID:   event.MarketID,
		YESPrice:   event.YESPrice,
		NOPrice:    event.NOPrice,
		Spread:     spread,
		DetectedAt: time.Now().UTC(),
		LatencyMs:  latencyMs,
	}

	return opp, latencyMs
}

func CalculateSpread(yesPrice, noPrice decimal.Decimal) decimal.Decimal {
	one := decimal.RequireFromString(oneDollar)
	return one.Sub(yesPrice).Sub(noPrice)
}
