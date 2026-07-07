package executor

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/execution-engine/internal/history"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/pqap/services/execution-engine/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type PositionSizingConfig struct {
	LiquidityPct float64
	MinSize      int64
	MaxSize      int64
	DefaultSize  int64
}

func DefaultPositionSizingConfig() PositionSizingConfig {
	return PositionSizingConfig{
		LiquidityPct: 0.1,
		MinSize:      10,
		MaxSize:      1000,
		DefaultSize:  100,
	}
}

type Executor struct {
	orderPort       ports.OrderPort
	riskPort        ports.RiskPort
	publisher       ports.EventPublisher
	marketPricePort ports.MarketPricePort
	riskChecker     *RiskChecker
	slipProtector   *SlippageProtector
	idempotency     *IdempotencyChecker
	orderLogger     OrderLoggerAdapter
	fillMon         FillMonitorAdapter
	tradeHandler    TradeHistoryHandler
	logger          *zap.Logger
	timeInForce     string
	maxRetries      int
	retryBackoff    time.Duration
	retryBackoffMax time.Duration
	semaphore       chan struct{}
	posSizing       PositionSizingConfig
	paperSim        *PaperSimulator // #1: Paper trading simulator
}

type OrderLoggerAdapter interface {
	LogOrder(ctx context.Context, order *ports.Order) error
}

type FillMonitorAdapter interface {
	StartMonitoring(ctx context.Context, order *ports.Order)
}

type TradeHistoryHandler interface {
	HandleOrderResult(ctx context.Context, result *history.OrderResult) error
}

func NewExecutor(
	orderPort ports.OrderPort,
	riskPort ports.RiskPort,
	publisher ports.EventPublisher,
	marketPricePort ports.MarketPricePort,
	slippageTolerance float64,
	timeInForce string,
	maxRetries int,
	retryBackoff time.Duration,
	retryBackoffMax time.Duration,
	maxConcurrency int,
	orderLogger OrderLoggerAdapter,
	fillMon FillMonitorAdapter,
	riskEventRepo ports.RiskEventRepository,
	tradeHandler TradeHistoryHandler,
	paperSim *PaperSimulator,
	logger *zap.Logger,
) *Executor {
	return &Executor{
		orderPort:       orderPort,
		riskPort:        riskPort,
		publisher:       publisher,
		marketPricePort: marketPricePort,
		riskChecker:     NewRiskChecker(riskPort, riskEventRepo, logger),
		slipProtector:   NewSlippageProtector(slippageTolerance),
		idempotency:     NewIdempotencyChecker(),
		orderLogger:     orderLogger,
		fillMon:         fillMon,
		tradeHandler:    tradeHandler,
		logger:          logger,
		timeInForce:     timeInForce,
		maxRetries:      maxRetries,
		retryBackoff:    retryBackoff,
		retryBackoffMax: retryBackoffMax,
		semaphore:       make(chan struct{}, maxConcurrency),
		posSizing:       DefaultPositionSizingConfig(),
		paperSim:        paperSim,
	}
}

func (e *Executor) Execute(ctx context.Context, opp ports.OpportunityDetected) error {
	select {
	case e.semaphore <- struct{}{}:
		defer func() { <-e.semaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}

	startTime := time.Now()

	strategyID := opp.Payload.StrategyID
	if strategyID == "" {
		strategyID = "default"
	}

	e.logger.Info("executing opportunity",
		zap.String("opportunity_id", opp.Payload.OpportunityID),
		zap.String("market_id", opp.Payload.MarketID),
		zap.String("strategy_id", strategyID),
	)

	clientOrderID := uuid.New().String()

	if e.idempotency.IsDuplicate(clientOrderID) {
		metrics.DuplicateRejected.Inc()
		e.logger.Warn("duplicate client_order_id rejected",
			zap.String("client_order_id", clientOrderID),
		)
		return nil
	}

	orderSize := e.calculatePositionSize(opp)

	riskResult, riskLatencyMs, err := e.riskChecker.Check(ctx, opp.Payload.MarketID, strategyID, orderSize)
	if err != nil {
		e.logger.Error("risk check failed", zap.Error(err))
		return e.publishFailure(ctx, opp, clientOrderID, strategyID, "risk_check_error", err.Error())
	}

	if !riskResult.Allowed {
		metrics.RiskDenied.Inc()
		e.logger.Warn("risk check denied",
			zap.String("opportunity_id", opp.Payload.OpportunityID),
			zap.String("reason", riskResult.Reason),
		)
		return e.publishFailure(ctx, opp, clientOrderID, strategyID, "risk_denied", riskResult.Reason)
	}

	side := "BUY"
	price := opp.Payload.YESPrice

	currentPrice, err := e.marketPricePort.GetCurrentPrice(ctx, opp.Payload.MarketID)
	if err != nil {
		e.logger.Warn("failed to get current market price, using opportunity price",
			zap.String("market_id", opp.Payload.MarketID),
			zap.Error(err),
		)
		currentPrice = price
	}

	slippageResult := e.slipProtector.Check(price, currentPrice)
	slippageCheckStr := fmt.Sprintf("passed=%v delta=%s", slippageResult.Passed, slippageResult.Delta.String())
	if !slippageResult.Passed {
		metrics.SlippageRejected.Inc()
		e.logger.Warn("slippage check failed",
			zap.String("opportunity_id", opp.Payload.OpportunityID),
			zap.String("delta", slippageResult.Delta.String()),
		)
		return e.publishFailure(ctx, opp, clientOrderID, strategyID, "slippage_exceeded", fmt.Sprintf("delta: %s", slippageResult.Delta.String()))
	}

	e.idempotency.Mark(clientOrderID)

	// #1: Check execution mode — PAPER mode simulates fill, skips real order
	if e.paperSim != nil {
		mode := GetExecutionMode(ctx, e.riskPort)
		if mode == "PAPER" {
			order := &ports.Order{
				MarketID: opp.Payload.MarketID,
				Side:     side,
				Price:    price,
				Quantity: orderSize,
			}
			fill := e.paperSim.SimulateFill(ctx, order)

			latencyMs := time.Since(startTime).Milliseconds()
			e.logger.Info("paper trade simulated",
				zap.String("opportunity_id", opp.Payload.OpportunityID),
				zap.String("market_id", opp.Payload.MarketID),
				zap.Bool("filled", fill.Filled),
				zap.String("fill_price", fill.FillPrice.String()),
				zap.String("pnl", fill.PnL.String()),
				zap.Int64("latency_ms", latencyMs),
			)

			metrics.OrdersPlaced.Inc() // Count paper trades too
			return nil // Skip real order placement
		}
	}

	orderReq := ports.OrderRequest{
		OpportunityID: opp.Payload.OpportunityID,
		MarketID:      opp.Payload.MarketID,
		Side:          side,
		Price:         price,
		Size:          orderSize,
		TimeInForce:   e.timeInForce,
		StrategyID:    strategyID,
	}

	var orderResp *ports.OrderResponse
	for attempt := 0; attempt <= e.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(float64(e.retryBackoff) * math.Pow(2, float64(attempt-1)))
			if backoff > e.retryBackoffMax {
				backoff = e.retryBackoffMax
			}
			e.logger.Warn("retrying order placement",
				zap.Int("attempt", attempt),
				zap.Duration("backoff", backoff),
			)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		orderResp, err = e.orderPort.PlaceOrder(orderReq, clientOrderID)
		if err == nil {
			break
		}
		e.logger.Error("order placement attempt failed",
			zap.Int("attempt", attempt),
			zap.Error(err),
		)
	}

	if err != nil {
		e.logger.Error("order placement failed after retries", zap.Error(err))
		return e.publishFailure(ctx, opp, clientOrderID, strategyID, "api_error", err.Error())
	}

	latencyMs := time.Since(startTime).Milliseconds()

	metrics.OrdersPlaced.Inc()
	metrics.OrderLatency.Observe(float64(latencyMs))
	metrics.ActiveOrders.Inc()

	e.logger.Info("order placed",
		zap.String("order_id", orderResp.OrderID),
		zap.String("client_order_id", clientOrderID),
		zap.Int64("latency_ms", latencyMs),
		zap.Int64("risk_check_latency_ms", riskLatencyMs),
	)

	if e.fillMon != nil {
		order := &ports.Order{
			ID:            orderResp.OrderID,
			ClientOrderID: clientOrderID,
			OpportunityID: opp.Payload.OpportunityID,
			MarketID:      opp.Payload.MarketID,
			Side:          side,
			Price:         price,
			Size:          orderSize,
			FilledQty:     decimal.Zero,
			RemainingQty:  orderSize,
			Status:        ports.OrderStatusPlaced,
			TimeInForce:   e.timeInForce,
			StrategyID:    strategyID,
			PlacedAt:      time.Now().UTC(),
		}
		e.fillMon.StartMonitoring(ctx, order)
	}

	if e.orderLogger != nil {
		order := &ports.Order{
			ID:               orderResp.OrderID,
			ClientOrderID:    clientOrderID,
			OpportunityID:    opp.Payload.OpportunityID,
			MarketID:         opp.Payload.MarketID,
			Side:             side,
			Price:            price,
			Size:             orderSize,
			FilledQty:        decimal.Zero,
			RemainingQty:     orderSize,
			Status:           ports.OrderStatusPlaced,
			TimeInForce:      e.timeInForce,
			LatencyMs:        latencyMs,
			RiskCheckResult:  riskResult.Reason,
			SlippageCheck:    slippageCheckStr,
			StrategyID:       strategyID,
			PlacedAt:         time.Now().UTC(),
		}
		if logErr := e.orderLogger.LogOrder(ctx, order); logErr != nil {
			e.logger.Error("failed to log order audit trail",
				zap.String("order_id", orderResp.OrderID),
				zap.Error(logErr),
			)
		}
	}

	if e.tradeHandler != nil {
		tradeResult := &history.OrderResult{
			ClientOrderID:   clientOrderID,
			OrderID:         orderResp.OrderID,
			OpportunityID:   opp.Payload.OpportunityID,
			MarketID:        opp.Payload.MarketID,
			Side:            side,
			OrderType:       e.timeInForce,
			Price:           price,
			SignalPrice:     currentPrice,
			Quantity:        orderSize,
			FilledQuantity:  decimal.Zero,
			FillStatus:      history.FillStatusPlaced,
			LatencyMs:       int(latencyMs),
			SignalTimestamp: time.Now().UTC(),
			OrderTimestamp:  time.Now().UTC(),
			RiskDecision:    riskResult.Reason,
			StrategyID:      strategyID,
		}
		if tradeErr := e.tradeHandler.HandleOrderResult(ctx, tradeResult); tradeErr != nil {
			e.logger.Error("failed to record trade history",
				zap.String("order_id", orderResp.OrderID),
				zap.Error(tradeErr),
			)
		}
	}

	placedEvent := ports.OrderPlaced{
		EventID:   uuid.New().String(),
		EventType: "OrderPlaced",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderPlacedPayload{
			OrderID:       orderResp.OrderID,
			ClientOrderID: clientOrderID,
			OpportunityID: opp.Payload.OpportunityID,
			MarketID:      opp.Payload.MarketID,
			Side:          side,
			Price:         price,
			CurrentPrice:  currentPrice,
			Size:          orderSize,
			StrategyID:    strategyID,
		},
	}

	if err := e.publisher.PublishOrderPlaced(ctx, placedEvent); err != nil {
		e.logger.Error("failed to publish OrderPlaced event", zap.Error(err))
	}

	return nil
}

func (e *Executor) publishFailure(ctx context.Context, opp ports.OpportunityDetected, clientOrderID, strategyID, reason, detail string) error {
	metrics.OrdersFailed.Inc()

	if e.tradeHandler != nil {
		tradeResult := &history.OrderResult{
			ClientOrderID:  clientOrderID,
			OpportunityID:  opp.Payload.OpportunityID,
			MarketID:       opp.Payload.MarketID,
			Side:           "BUY",
			OrderType:      e.timeInForce,
			Price:          opp.Payload.YESPrice,
			Quantity:       decimal.Zero,
			FilledQuantity: decimal.Zero,
			FillStatus:     history.FillStatusFailed,
			LatencyMs:      0,
			SignalTimestamp: time.Now().UTC(),
			OrderTimestamp:  time.Now().UTC(),
			RiskDecision:   reason,
			FailureReason:  detail,
			StrategyID:     strategyID,
		}
		if tradeErr := e.tradeHandler.HandleOrderResult(ctx, tradeResult); tradeErr != nil {
			e.logger.Error("failed to record failed trade history",
				zap.String("client_order_id", clientOrderID),
				zap.Error(tradeErr),
			)
		}
	}

	failedEvent := ports.OrderFailed{
		EventID:   uuid.New().String(),
		EventType: "OrderFailed",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderFailedPayload{
			ClientOrderID: clientOrderID,
			OpportunityID: opp.Payload.OpportunityID,
			MarketID:      opp.Payload.MarketID,
			Reason:        reason,
			ErrorDetail:   detail,
			StrategyID:    strategyID,
		},
	}

	if err := e.publisher.PublishOrderFailed(ctx, failedEvent); err != nil {
		e.logger.Error("failed to publish OrderFailed event", zap.Error(err))
	}

	return fmt.Errorf("order failed: %s - %s", reason, detail)
}

func (e *Executor) calculatePositionSize(opp ports.OpportunityDetected) decimal.Decimal {
	return calculatePositionSizeFromConfig(opp, e.posSizing)
}

func calculatePositionSizeFromConfig(opp ports.OpportunityDetected, cfg PositionSizingConfig) decimal.Decimal {
	liquidity := opp.Payload.Liquidity
	score := opp.Payload.Score

	defaultSize := decimal.NewFromInt(cfg.DefaultSize)

	if liquidity.IsZero() || liquidity.IsNegative() {
		return defaultSize
	}

	baseSize := liquidity.Mul(decimal.NewFromFloat(cfg.LiquidityPct))

	if score.IsPositive() && score.LessThanOrEqual(decimal.NewFromInt(1)) {
		baseSize = baseSize.Mul(score)
	}

	minSize := decimal.NewFromInt(cfg.MinSize)
	maxSize := decimal.NewFromInt(cfg.MaxSize)

	if baseSize.LessThan(minSize) {
		baseSize = minSize
	}
	if baseSize.GreaterThan(maxSize) {
		baseSize = maxSize
	}

	return baseSize.Round(0)
}
