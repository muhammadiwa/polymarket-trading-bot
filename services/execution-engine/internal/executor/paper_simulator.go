package executor

import (
	"context"
	"math/rand"
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
	// Get real orderbook depth from Redis
	depth := ps.marketPricePort.GetLiquidityDepth(ctx, order.MarketID)

	// Estimate fill probability based on depth
	fillProb := estimateFillProbability(depth)

	// Simulate with randomness
	filled := ps.rng.Float64() < fillProb

	if !filled {
		return &SimulatedFill{
			Filled:    false,
			Status:    "SIMULATED_NO_FILL",
			FillPrice: decimal.Zero,
			Quantity:  decimal.Zero,
			LatencyMs: int64(ps.rng.Intn(50)),
			PnL:       decimal.Zero,
		}
	}

	// Apply slippage
	slippage := order.Price.Mul(decimal.NewFromFloat(ps.slippagePct))
	fillPrice := order.Price.Sub(slippage)
	if fillPrice.IsNegative() {
		fillPrice = order.Price
	}

	// Simulate latency
	latencyMs := int64(ps.rng.Intn(100))

	// Calculate PnL (spread - slippage as profit potential)
	pnl := slippage.Mul(order.Quantity).Neg() // Paper PnL = -slippage * quantity (cost)

	return &SimulatedFill{
		Filled:    true,
		Status:    "SIMULATED_FILL",
		FillPrice: fillPrice,
		Quantity:  order.Quantity,
		LatencyMs: latencyMs,
		PnL:       pnl,
	}
}

// GetExecutionMode reads execution mode from Redis.
func GetExecutionMode(ctx context.Context, riskPort ports.RiskPort) string {
	// Read from Redis via risk port
	mode, err := riskPort.GetExecutionMode(ctx)
	if err != nil || mode == "" {
		return "LIVE" // Default to LIVE for safety
	}
	return mode
}

func estimateFillProbability(depth decimal.Decimal) float64 {
	// Simple model: deeper orderbook = higher fill probability
	if depth.IsZero() || depth.IsNegative() {
		return 0.1 // Minimum 10% for any order
	}
	// Cap at 95% — never guarantee fills
	prob := depth.InexactFloat64() / 10000.0
	if prob > 0.95 {
		return 0.95
	}
	if prob < 0.1 {
		return 0.1
	}
	return prob
}
