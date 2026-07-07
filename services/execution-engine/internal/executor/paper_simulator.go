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
	Side      string // BUY or SELL — #2: needed for DB persistence
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
			Side:      order.Side,
		}
	}

	// #3: Apply slippage in correct direction based on side
	slippage := order.Price.Mul(decimal.NewFromFloat(ps.slippagePct))
	var fillPrice decimal.Decimal
	if order.Side == "BUY" {
		fillPrice = order.Price.Sub(slippage)
	} else {
		fillPrice = order.Price.Add(slippage)
	}
	if fillPrice.IsNegative() {
		fillPrice = order.Price
	}

	// #9: PnL = slippage cost (tracked as negative for entry, positive on exit)
	// For paper trading, we track the slippage as the cost of the trade
	quantity := order.Size // #1: Use Size field from Order struct
	pnl := slippage.Mul(quantity).Neg()

	return &SimulatedFill{
		Filled:    true,
		Status:    "SIMULATED_FILL",
		FillPrice: fillPrice,
		Quantity:  quantity,
		LatencyMs: latencyMs,
		PnL:       pnl,
		Side:      order.Side,
	}
}

// GetExecutionMode reads execution mode from Redis with validation.
func GetExecutionMode(ctx context.Context, riskPort ports.RiskPort) string {
	mode, err := riskPort.GetExecutionMode(ctx)
	if err != nil || mode == "" {
		return "PAPER" // #7: Default to PAPER (safe mode), matching API gateway
	}
	if mode != "LIVE" && mode != "PAPER" {
		return "PAPER"
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
