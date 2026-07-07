package executor

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// PaperSimulator handles simulated fills for paper trading mode.
type PaperSimulator struct {
	riskPort        ports.RiskPort
	marketPricePort ports.MarketPricePort
	logger          *zap.Logger
	rng             *rand.Rand
	rngMu           sync.Mutex // #2: Protect rng from concurrent access
	slippagePct     float64
}

func NewPaperSimulator(riskPort ports.RiskPort, marketPricePort ports.MarketPricePort, logger *zap.Logger, slippagePct float64) *PaperSimulator {
	return &PaperSimulator{
		riskPort:        riskPort,
		marketPricePort: marketPricePort,
		logger:          logger,
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
		slippagePct:     slippagePct,
	}
}

type SimulatedFill struct {
	Filled    bool
	Status    string
	FillPrice decimal.Decimal
	Quantity  decimal.Decimal
	LatencyMs int64
	PnL       decimal.Decimal
}

// SimulateFill simulates a fill based on orderbook depth and configurable parameters.
func (ps *PaperSimulator) SimulateFill(ctx context.Context, order *ports.Order) *SimulatedFill {
	depth := ps.marketPricePort.GetLiquidityDepth(ctx, order.MarketID)
	fillProb := estimateFillProbability(depth)

	ps.rngMu.Lock()
	filled := ps.rng.Float64() < fillProb
	latencyMs := int64(ps.rng.Intn(100))
	ps.rngMu.Unlock()

	if !filled {
		return &SimulatedFill{
			Filled:    false,
			Status:    "SIMULATED_NO_FILL",
			FillPrice: decimal.Zero,
			Quantity:  decimal.Zero,
			LatencyMs: latencyMs,
			PnL:       decimal.Zero,
		}
	}

	// #3: Apply slippage in correct direction based on side
	slippage := order.Price.Mul(decimal.NewFromFloat(ps.slippagePct))
	var fillPrice decimal.Decimal
	if order.Side == "BUY" {
		fillPrice = order.Price.Sub(slippage) // BUY: lower is worse
	} else {
		fillPrice = order.Price.Add(slippage) // SELL: higher is worse
	}
	if fillPrice.IsNegative() {
		fillPrice = order.Price
	}

	// #9: PnL = slippage cost at entry (tracked separately from position PnL)
	pnl := slippage.Mul(order.Quantity).Neg()

	return &SimulatedFill{
		Filled:    true,
		Status:    "SIMULATED_FILL",
		FillPrice: fillPrice,
		Quantity:  order.Quantity,
		LatencyMs: latencyMs,
		PnL:       pnl,
	}
}

// GetExecutionMode reads execution mode from Redis with validation.
func GetExecutionMode(ctx context.Context, riskPort ports.RiskPort) string {
	mode, err := riskPort.GetExecutionMode(ctx)
	if err != nil || mode == "" {
		return "LIVE"
	}
	// #4: Validate mode is exactly LIVE or PAPER
	if mode != "LIVE" && mode != "PAPER" {
		return "LIVE"
	}
	return mode
}

func estimateFillProbability(depth decimal.Decimal) float64 {
	if depth.IsZero() || depth.IsNegative() {
		return 0.1
	}
	prob := depth.InexactFloat64() / 10000.0
	// #10: Handle NaN
	if prob != prob { // NaN check
		return 0.1
	}
	if prob > 0.95 {
		return 0.95
	}
	if prob < 0.1 {
		return 0.1
	}
	return prob
}
