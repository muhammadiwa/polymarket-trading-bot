package executor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/pqap/services/execution-engine/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type PairStatus string

const (
	PairStatusPending    PairStatus = "PENDING"
	PairStatusPlacing    PairStatus = "PLACING"
	PairStatusOneLeg     PairStatus = "ONE_LEG"
	PairStatusFilled     PairStatus = "FILLED"
	PairStatusPartial    PairStatus = "PARTIAL"
	PairStatusCancelled  PairStatus = "CANCELLED"
	PairStatusFailed     PairStatus = "FAILED"
)

type AtomicPair struct {
	ID                 string          `json:"id"`
	OpportunityID      string          `json:"opportunity_id"`
	MarketID           string          `json:"market_id"`
	YesOrderID         string          `json:"yes_order_id"`
	NoOrderID          string          `json:"no_order_id"`
	YesClientOrderID   string          `json:"yes_client_order_id"`
	NoClientOrderID    string          `json:"no_client_order_id"`
	YesPrice           decimal.Decimal `json:"yes_price"`
	NoPrice            decimal.Decimal `json:"no_price"`
	YesSize            decimal.Decimal `json:"yes_size"`
	NoSize             decimal.Decimal `json:"no_size"`
	YesFilledQty       decimal.Decimal `json:"yes_filled_qty"`
	NoFilledQty        decimal.Decimal `json:"no_filled_qty"`
	Status             PairStatus      `json:"status"`
	PlacementLatencyMs int64           `json:"placement_latency_ms"`
	FailureReason      string          `json:"failure_reason"`
	FailedLeg          string          `json:"failed_leg"`
	StrategyID         string          `json:"strategy_id"`
	AccountID          *string         `json:"account_id"`
	CreatedAt          time.Time       `json:"created_at"`
	CompletedAt        *time.Time      `json:"completed_at"`
}

type AtomicExecutor struct {
	orderPort        ports.OrderPort
	riskPort         ports.RiskPort
	publisher        ports.EventPublisher
	marketPricePort  ports.MarketPricePort
	riskChecker      *RiskChecker
	slipProtector    *SlippageProtector
	idempotency      *IdempotencyChecker
	partialTracker   *PartialTracker
	legHandler       *LegHandler
	atomicLogger     AtomicLogger
	atomicTimeout    time.Duration
	legCancelTimeout time.Duration
	timeInForce      string
	logger           *zap.Logger
	posSizing        PositionSizingConfig
	mu               sync.RWMutex
}

type AtomicLogger interface {
	LogAtomicPair(ctx context.Context, record AtomicPairRecord) error
}

type AtomicPairRecord struct {
	ID                 string
	OpportunityID      string
	MarketID           string
	YesOrderID         string
	NoOrderID          string
	YesClientOrderID   string
	NoClientOrderID    string
	YesPrice           decimal.Decimal
	NoPrice            decimal.Decimal
	YesSize            decimal.Decimal
	NoSize             decimal.Decimal
	YesFilledQty       decimal.Decimal
	NoFilledQty        decimal.Decimal
	Status             string
	PlacementLatencyMs int64
	FailureReason      string
	FailedLeg          string
	StrategyID         string
	AccountID          *string
	CreatedAt          time.Time
	CompletedAt        *time.Time
}

func NewAtomicExecutor(
	orderPort ports.OrderPort,
	riskPort ports.RiskPort,
	publisher ports.EventPublisher,
	riskEventRepo ports.RiskEventRepository,
	marketPricePort ports.MarketPricePort,
	slippageTolerance float64,
	timeInForce string,
	atomicTimeout time.Duration,
	legCancelTimeout time.Duration,
	atomicLogger AtomicLogger,
	logger *zap.Logger,
) *AtomicExecutor {
	partialTracker := NewPartialTracker(logger)
	legHandler := NewLegHandler(orderPort, publisher, legCancelTimeout, logger)

	return &AtomicExecutor{
		orderPort:        orderPort,
		riskPort:         riskPort,
		publisher:        publisher,
		marketPricePort:  marketPricePort,
		riskChecker:      NewRiskChecker(riskPort, riskEventRepo, logger),
		slipProtector:    NewSlippageProtector(slippageTolerance),
		idempotency:      NewIdempotencyChecker(),
		partialTracker:   partialTracker,
		legHandler:       legHandler,
		atomicLogger:     atomicLogger,
		atomicTimeout:    atomicTimeout,
		legCancelTimeout: legCancelTimeout,
		timeInForce:      timeInForce,
		logger:           logger,
		posSizing:        DefaultPositionSizingConfig(),
	}
}

func (ae *AtomicExecutor) ExecuteAtomic(ctx context.Context, opp ports.OpportunityDetected) error {
	startTime := time.Now()

	strategyID := opp.Payload.StrategyID
	if strategyID == "" {
		strategyID = "default"
	}

	ae.logger.Info("executing atomic pair",
		zap.String("opportunity_id", opp.Payload.OpportunityID),
		zap.String("market_id", opp.Payload.MarketID),
		zap.String("strategy_id", strategyID),
	)

	metrics.AtomicPairsTotal.Inc()

	yesClientOrderID := uuid.New().String()
	noClientOrderID := uuid.New().String()
	pairID := uuid.New().String()

	if ae.idempotency.IsDuplicate(yesClientOrderID) || ae.idempotency.IsDuplicate(noClientOrderID) {
		metrics.DuplicateRejected.Inc()
		ae.logger.Warn("duplicate client_order_id rejected",
			zap.String("yes_client_order_id", yesClientOrderID),
			zap.String("no_client_order_id", noClientOrderID),
		)
		return nil
	}

	orderSize := ae.calculatePositionSize(opp)

	riskResult, _, err := ae.riskChecker.Check(ctx, opp.Payload.MarketID, strategyID, orderSize)
	if err != nil {
		ae.logger.Error("risk check failed", zap.Error(err))
		return ae.publishAtomicFailure(ctx, opp, pairID, strategyID, "risk_check_error", err.Error())
	}

	if !riskResult.Allowed {
		metrics.RiskDenied.Inc()
		ae.logger.Warn("risk check denied",
			zap.String("opportunity_id", opp.Payload.OpportunityID),
			zap.String("reason", riskResult.Reason),
		)
		return ae.publishAtomicFailure(ctx, opp, pairID, strategyID, "risk_denied", riskResult.Reason)
	}

	currentPrice, err := ae.marketPricePort.GetCurrentPrice(ctx, opp.Payload.MarketID)
	if err != nil {
		ae.logger.Warn("failed to get current market price for slippage check, using opportunity price",
			zap.String("market_id", opp.Payload.MarketID),
			zap.Error(err),
		)
		currentPrice = opp.Payload.YESPrice
	}

	yesSlippage := ae.slipProtector.Check(opp.Payload.YESPrice, currentPrice)
	if !yesSlippage.Passed {
		metrics.SlippageRejected.Inc()
		return ae.publishAtomicFailure(ctx, opp, pairID, strategyID, "slippage_exceeded", "YES leg slippage exceeded")
	}

	noSlippage := ae.slipProtector.Check(opp.Payload.NOPrice, currentPrice)
	if !noSlippage.Passed {
		metrics.SlippageRejected.Inc()
		return ae.publishAtomicFailure(ctx, opp, pairID, strategyID, "slippage_exceeded", "NO leg slippage exceeded")
	}

	ae.idempotency.Mark(yesClientOrderID)
	ae.idempotency.Mark(noClientOrderID)

	pair := &AtomicPair{
		ID:               pairID,
		OpportunityID:    opp.Payload.OpportunityID,
		MarketID:         opp.Payload.MarketID,
		YesClientOrderID: yesClientOrderID,
		NoClientOrderID:  noClientOrderID,
		YesPrice:         opp.Payload.YESPrice,
		NoPrice:          opp.Payload.NOPrice,
		YesSize:          orderSize,
		NoSize:           orderSize,
		YesFilledQty:     decimal.Zero,
		NoFilledQty:      decimal.Zero,
		Status:           PairStatusPending,
		StrategyID:       strategyID,
		CreatedAt:        time.Now().UTC(),
	}

	pair.Status = PairStatusPlacing

	atomicCtx, atomicCancel := context.WithTimeout(ctx, ae.atomicTimeout)
	defer atomicCancel()

	var (
		yesResp *ports.OrderResponse
		noResp  *ports.OrderResponse
		yesErr  error
		noErr   error
		mu      sync.Mutex
	)

	g, _ := errgroup.WithContext(atomicCtx)

	g.Go(func() error {
		yesOrderReq := ports.OrderRequest{
			OpportunityID: opp.Payload.OpportunityID,
			MarketID:      opp.Payload.MarketID,
			Side:          "BUY",
			Price:         opp.Payload.YESPrice,
			Size:          orderSize,
			TimeInForce:   ae.timeInForce,
			StrategyID:    strategyID,
		}

		resp, err := ae.orderPort.PlaceOrder(yesOrderReq, yesClientOrderID)
		mu.Lock()
		yesResp = resp
		yesErr = err
		mu.Unlock()

		if err != nil {
			return fmt.Errorf("YES leg placement failed: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		noOrderReq := ports.OrderRequest{
			OpportunityID: opp.Payload.OpportunityID,
			MarketID:      opp.Payload.MarketID,
			Side:          "BUY",
			Price:         opp.Payload.NOPrice,
			Size:          orderSize,
			TimeInForce:   ae.timeInForce,
			StrategyID:    strategyID,
		}

		resp, err := ae.orderPort.PlaceOrder(noOrderReq, noClientOrderID)
		mu.Lock()
		noResp = resp
		noErr = err
		mu.Unlock()

		if err != nil {
			return fmt.Errorf("NO leg placement failed: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		placementLatencyMs := time.Since(startTime).Milliseconds()
		ae.logger.Error("atomic pair placement failed",
			zap.String("pair_id", pairID),
			zap.Error(err),
			zap.Int64("placement_latency_ms", placementLatencyMs),
		)

		mu.Lock()
		yesPlaced := yesErr == nil && yesResp != nil
		noPlaced := noErr == nil && noResp != nil
		yesOrderID := ""
		noOrderID := ""
		if yesResp != nil {
			yesOrderID = yesResp.OrderID
		}
		if noResp != nil {
			noOrderID = noResp.OrderID
		}
		mu.Unlock()

		if yesPlaced {
			pair.YesOrderID = yesOrderID
		}
		if noPlaced {
			pair.NoOrderID = noOrderID
		}

		failedLeg := "YES"
		if yesErr == nil {
			failedLeg = "NO"
		}

		pair.FailedLeg = failedLeg
		pair.FailureReason = err.Error()
		pair.PlacementLatencyMs = placementLatencyMs

		if yesPlaced && noPlaced {
			pair.Status = PairStatusFailed
		} else if yesPlaced || noPlaced {
			pair.Status = PairStatusOneLeg
			cancelledLeg := "NO"
			cancelledOrderID := noOrderID
			if !noPlaced {
				cancelledLeg = "YES"
				cancelledOrderID = yesOrderID
			}

			ae.legHandler.HandleLegFailure(ctx, &LegFailureContext{
				PairID:           pairID,
				OpportunityID:    opp.Payload.OpportunityID,
				MarketID:         opp.Payload.MarketID,
				FailedLeg:        failedLeg,
				CancelledLeg:     cancelledLeg,
				CancelledOrderID: cancelledOrderID,
				StrategyID:       strategyID,
			})
		} else {
			pair.Status = PairStatusCancelled
		}

		metrics.AtomicPairsFailed.Inc()

		if ae.atomicLogger != nil {
			completedAt := time.Now().UTC()
			if logErr := ae.atomicLogger.LogAtomicPair(ctx, AtomicPairRecord{
				ID:                 pair.ID,
				OpportunityID:      pair.OpportunityID,
				MarketID:           pair.MarketID,
				YesOrderID:         pair.YesOrderID,
				NoOrderID:          pair.NoOrderID,
				YesClientOrderID:   pair.YesClientOrderID,
				NoClientOrderID:    pair.NoClientOrderID,
				YesPrice:           pair.YesPrice,
				NoPrice:            pair.NoPrice,
				YesSize:            pair.YesSize,
				NoSize:             pair.NoSize,
				YesFilledQty:       pair.YesFilledQty,
				NoFilledQty:        pair.NoFilledQty,
				Status:             string(pair.Status),
				PlacementLatencyMs: pair.PlacementLatencyMs,
				FailureReason:      pair.FailureReason,
				FailedLeg:          pair.FailedLeg,
				StrategyID:         pair.StrategyID,
				AccountID:          pair.AccountID,
				CreatedAt:          pair.CreatedAt,
				CompletedAt:        &completedAt,
			}); logErr != nil {
				ae.logger.Error("failed to log atomic pair", zap.Error(logErr))
			}
		}

		return fmt.Errorf("atomic pair failed: %s", err.Error())
	}

	mu.Lock()
	pair.YesOrderID = yesResp.OrderID
	pair.NoOrderID = noResp.OrderID
	yesFilledQty := yesResp.FilledQty
	noFilledQty := noResp.FilledQty
	mu.Unlock()

	if yesFilledQty.IsPositive() && yesFilledQty.LessThan(orderSize) {
		ae.partialTracker.RecordPartialFill(PartialFillRecord{
			PairID:        pairID,
			Leg:           "YES",
			FilledQty:     yesFilledQty,
			RemainingQty:  orderSize.Sub(yesFilledQty),
			FillPrice:     opp.Payload.YESPrice,
			OrderID:       yesResp.OrderID,
			ClientOrderID: yesClientOrderID,
			MarketID:      opp.Payload.MarketID,
			StrategyID:    strategyID,
		})
		metrics.PartialFills.Inc()
	}
	if noFilledQty.IsPositive() && noFilledQty.LessThan(orderSize) {
		ae.partialTracker.RecordPartialFill(PartialFillRecord{
			PairID:        pairID,
			Leg:           "NO",
			FilledQty:     noFilledQty,
			RemainingQty:  orderSize.Sub(noFilledQty),
			FillPrice:     opp.Payload.NOPrice,
			OrderID:       noResp.OrderID,
			ClientOrderID: noClientOrderID,
			MarketID:      opp.Payload.MarketID,
			StrategyID:    strategyID,
		})
		metrics.PartialFills.Inc()
	}

	placementLatencyMs := time.Since(startTime).Milliseconds()
	pair.Status = PairStatusFilled
	pair.PlacementLatencyMs = placementLatencyMs
	pair.YesFilledQty = yesFilledQty
	pair.NoFilledQty = noFilledQty
	now := time.Now().UTC()
	pair.CompletedAt = &now

	if ae.atomicLogger != nil {
		if logErr := ae.atomicLogger.LogAtomicPair(ctx, AtomicPairRecord{
			ID:                 pair.ID,
			OpportunityID:      pair.OpportunityID,
			MarketID:           pair.MarketID,
			YesOrderID:         pair.YesOrderID,
			NoOrderID:          pair.NoOrderID,
			YesClientOrderID:   pair.YesClientOrderID,
			NoClientOrderID:    pair.NoClientOrderID,
			YesPrice:           pair.YesPrice,
			NoPrice:            pair.NoPrice,
			YesSize:            pair.YesSize,
			NoSize:             pair.NoSize,
			YesFilledQty:       pair.YesFilledQty,
			NoFilledQty:        pair.NoFilledQty,
			Status:             string(pair.Status),
			PlacementLatencyMs: pair.PlacementLatencyMs,
			StrategyID:         pair.StrategyID,
			AccountID:          pair.AccountID,
			CreatedAt:          pair.CreatedAt,
			CompletedAt:        pair.CompletedAt,
		}); logErr != nil {
			ae.logger.Error("failed to log atomic pair", zap.Error(logErr))
		}
	}

	metrics.AtomicPairsFilled.Inc()
	metrics.AtomicPlacementLatency.Observe(float64(placementLatencyMs))

	ae.logger.Info("atomic pair filled",
		zap.String("pair_id", pairID),
		zap.String("yes_order_id", pair.YesOrderID),
		zap.String("no_order_id", pair.NoOrderID),
		zap.Int64("placement_latency_ms", placementLatencyMs),
	)

	return nil
}

func (ae *AtomicExecutor) publishAtomicFailure(ctx context.Context, opp ports.OpportunityDetected, pairID, strategyID, reason, detail string) error {
	metrics.AtomicPairsFailed.Inc()

	if ae.atomicLogger != nil {
		completedAt := time.Now().UTC()
		if logErr := ae.atomicLogger.LogAtomicPair(ctx, AtomicPairRecord{
			ID:            pairID,
			OpportunityID: opp.Payload.OpportunityID,
			MarketID:      opp.Payload.MarketID,
			YesPrice:      opp.Payload.YESPrice,
			NoPrice:       opp.Payload.NOPrice,
			Status:        string(PairStatusFailed),
			FailureReason: detail,
			StrategyID:    strategyID,
			CreatedAt:     time.Now().UTC(),
			CompletedAt:   &completedAt,
		}); logErr != nil {
			ae.logger.Error("failed to log atomic pair", zap.Error(logErr))
		}
	}

	failedEvent := ports.OrderFailed{
		EventID:   uuid.New().String(),
		EventType: "OrderFailed",
		Timestamp: time.Now().UTC(),
		Source:    "execution-engine",
		Payload: ports.OrderFailedPayload{
			OpportunityID: opp.Payload.OpportunityID,
			MarketID:      opp.Payload.MarketID,
			Reason:        reason,
			ErrorDetail:   detail,
			StrategyID:    strategyID,
		},
	}

	if err := ae.publisher.PublishOrderFailed(ctx, failedEvent); err != nil {
		ae.logger.Error("failed to publish OrderFailed event", zap.Error(err))
	}

	return fmt.Errorf("atomic pair failed: %s - %s", reason, detail)
}

func (ae *AtomicExecutor) calculatePositionSize(opp ports.OpportunityDetected) decimal.Decimal {
	return calculatePositionSizeFromConfig(opp, ae.posSizing)
}
