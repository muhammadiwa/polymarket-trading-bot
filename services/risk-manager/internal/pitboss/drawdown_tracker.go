package pitboss

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/pqap/services/risk-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type DrawdownTracker struct {
	mu               sync.RWMutex // #29: RWMutex for RLock in GetState
	peakEquity       decimal.Decimal
	currentEquity    decimal.Decimal
	drawdownLimit    decimal.Decimal
	warningThreshold float64
	publisher        ports.EventPublisher
	logger           *zap.Logger
	onBreach         func()
	breached         bool // #6: prevent repeated breach fires
}

func NewDrawdownTracker(
	initialCapital decimal.Decimal,
	drawdownLimitPct float64,
	warningThreshold float64,
	publisher ports.EventPublisher,
	logger *zap.Logger,
) *DrawdownTracker {
	return &DrawdownTracker{
		peakEquity:       initialCapital,
		currentEquity:    initialCapital,
		drawdownLimit:    decimal.NewFromFloat(drawdownLimitPct),
		warningThreshold: warningThreshold,
		publisher:        publisher,
		logger:           logger,
	}
}

func (dt *DrawdownTracker) SetOnBreach(fn func()) {
	dt.mu.Lock() // #9: protect onBreach with mutex
	defer dt.mu.Unlock()
	dt.onBreach = fn
}

func (dt *DrawdownTracker) UpdateEquity(newEquity decimal.Decimal) {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	dt.currentEquity = newEquity

	if newEquity.GreaterThan(dt.peakEquity) {
		previousPeak := dt.peakEquity
		dt.peakEquity = newEquity
		metrics.DrawdownPeakEquityUSD.Set(mustFloat64(newEquity))
		dt.publishDrawdownReset(previousPeak, newEquity)
	}

	metrics.DrawdownCurrentEquityUSD.Set(mustFloat64(dt.currentEquity))

	drawdown := dt.calculateDrawdown()
	metrics.DrawdownCurrent.Set(mustFloat64(drawdown))

	dt.logger.Debug("drawdown updated",
		zap.String("peak_equity", dt.peakEquity.String()),
		zap.String("current_equity", dt.currentEquity.String()),
		zap.String("drawdown", drawdown.String()),
	)

	warningLevel := dt.drawdownLimit.Mul(decimal.NewFromFloat(dt.warningThreshold))
	if drawdown.GreaterThanOrEqual(warningLevel) && drawdown.LessThan(dt.drawdownLimit) {
		dt.publishDrawdownWarning(drawdown)
	}

	if drawdown.GreaterThanOrEqual(dt.drawdownLimit) {
		if !dt.breached { // #6: fire only once per breach
			dt.breached = true
			dt.logger.Error("drawdown limit breached",
				zap.String("drawdown", drawdown.String()),
				zap.String("limit", dt.drawdownLimit.String()),
			)
			if dt.onBreach != nil {
				go dt.onBreach()
			}
		}
	} else {
		dt.breached = false // reset when drawdown recovers
	}
}

func (dt *DrawdownTracker) UpdateCapital(capital decimal.Decimal) {
	dt.UpdateEquity(capital)
}

func (dt *DrawdownTracker) calculateDrawdown() decimal.Decimal {
	if dt.peakEquity.IsZero() {
		return decimal.Zero
	}
	drawdown := dt.peakEquity.Sub(dt.currentEquity).Div(dt.peakEquity)
	if drawdown.IsNegative() {
		return decimal.Zero
	}
	return drawdown
}

func (dt *DrawdownTracker) publishDrawdownWarning(drawdown decimal.Decimal) {
	metrics.DrawdownWarningTotal.Inc()

	utilization := float64(0)
	if !dt.drawdownLimit.IsZero() {
		utilization, _ = drawdown.Div(dt.drawdownLimit).Float64()
	}

	event := ports.DrawdownWarning{
		EventID:   uuid.New().String(),
		EventType: "DrawdownWarning",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.DrawdownWarningPayload{
			Drawdown:      drawdown,
			DrawdownLimit: dt.drawdownLimit,
			PeakEquity:    dt.peakEquity,
			CurrentEquity: dt.currentEquity,
			Utilization:   utilization,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dt.publisher.PublishDrawdownWarning(ctx, event); err != nil {
		dt.logger.Error("failed to publish drawdown warning", zap.Error(err))
	}
}

func (dt *DrawdownTracker) publishDrawdownReset(previousPeak, newPeak decimal.Decimal) {
	event := ports.DrawdownReset{
		EventID:   uuid.New().String(),
		EventType: "DrawdownReset",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.DrawdownResetPayload{
			NewPeakEquity: newPeak,
			PreviousPeak:  previousPeak,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dt.publisher.PublishDrawdownReset(ctx, event); err != nil {
		dt.logger.Error("failed to publish drawdown reset", zap.Error(err))
	}
}

func (dt *DrawdownTracker) GetState() (peak, current, drawdown, limit decimal.Decimal) {
	dt.mu.RLock() // #29: use RLock for read-only access
	defer dt.mu.RUnlock()
	return dt.peakEquity, dt.currentEquity, dt.calculateDrawdown(), dt.drawdownLimit
}

func (dt *DrawdownTracker) SetState(peak, current decimal.Decimal) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.peakEquity = peak
	dt.currentEquity = current
	metrics.DrawdownPeakEquityUSD.Set(mustFloat64(peak))
	metrics.DrawdownCurrentEquityUSD.Set(mustFloat64(current))
	drawdown := dt.calculateDrawdown()
	metrics.DrawdownCurrent.Set(mustFloat64(drawdown))
}

func mustFloat64(d decimal.Decimal) float64 {
	// #16: check ok, clamp on overflow
	f, ok := d.Float64()
	if !ok {
		if d.IsPositive() {
			return math.MaxFloat64
		}
		if d.IsNegative() {
			return -math.MaxFloat64
		}
		return 0
	}
	return f
}
