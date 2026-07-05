package pitboss

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/pqap/services/risk-manager/internal/risk"
	"github.com/pqap/services/risk-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type PitBoss struct {
	mu              sync.RWMutex
	dailyBudget     *DailyBudget
	marketLimits    *MarketExposure
	strategyLimits  *StrategyExposure
	stateBuilder    *StateBuilder
	logger          *Logger
	publisher       ports.EventPublisher
	zapLogger       *zap.Logger
	capital         decimal.Decimal
	drawdownTracker *DrawdownTracker
	publishCh       chan ports.RiskDecisionLogged // #18: buffered channel for async publish
	publishWg       sync.WaitGroup                // #18: for graceful shutdown
	correlationTracker *risk.CorrelationTracker        // #16: correlation check
	batasiMonitor      *risk.BatasiWinMonitor          // #16: batasi win check
	metabolicMonitor   *risk.MetabolicMonitor          // #16: metabolic check
}

func NewPitBoss(
	dailyBudget *DailyBudget,
	marketLimits *MarketExposure,
	strategyLimits *StrategyExposure,
	stateBuilder *StateBuilder,
	logger *Logger,
	publisher ports.EventPublisher,
	capital decimal.Decimal,
	drawdownTracker *DrawdownTracker,
	zapLogger *zap.Logger,
) *PitBoss {
	pb := &PitBoss{
		dailyBudget:     dailyBudget,
		marketLimits:    marketLimits,
		strategyLimits:  strategyLimits,
		stateBuilder:    stateBuilder,
		logger:          logger,
		publisher:       publisher,
		zapLogger:       zapLogger,
		capital:         capital,
		drawdownTracker: drawdownTracker,
		publishCh:       make(chan ports.RiskDecisionLogged, 256), // #18
	}
	// #18: Start worker pool for async event publishing
	const publishWorkerCount = 2
	pb.publishWg.Add(publishWorkerCount)
	for i := 0; i < publishWorkerCount; i++ {
		go pb.publishWorker()
	}
	return pb
}

// #18: Worker pool drains publishCh instead of goroutine-per-event.
func (pb *PitBoss) publishWorker() {
	defer pb.publishWg.Done()
	for event := range pb.publishCh {
		if pb.publisher == nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := pb.publisher.PublishRiskDecisionLogged(ctx, event); err != nil {
			pb.zapLogger.Error("failed to publish RiskDecisionLogged", zap.Error(err))
		}
		cancel()
	}
}

// Close drains the publish channel and waits for workers to finish.
func (pb *PitBoss) Close() {
	close(pb.publishCh)
	pb.publishWg.Wait()
}

func (pb *PitBoss) Evaluate(req ports.RiskCheckRequest) ports.RiskDecision {
	start := time.Now()

	decision := pb.evaluate(req)

	elapsed := time.Since(start)
	metrics.RiskCheckLatency.Observe(float64(elapsed.Milliseconds()))
	metrics.RiskCheckTotal.Inc()

	if decision.Decision == "DENY" {
		metrics.RiskCheckDeniedTotal.WithLabelValues(decision.Reason).Inc()
	}

	pb.logger.LogDecisionAsync(decision)

	// #2: Publish RiskDecisionLogged event to NATS via worker pool
	if pb.publisher != nil {
		event := ports.RiskDecisionLogged{
			EventID:   uuid.New().String(),
			EventType: "RiskDecisionLogged",
			Timestamp: time.Now().UTC(),
			Source:    "risk-manager",
			Payload: ports.RiskDecisionLoggedPayload{
				DecisionID:           decision.EventID,
				Decision:             decision.Decision,
				Reason:               decision.Reason,
				MarketID:             decision.MarketID,
				StrategyID:           decision.StrategyID,
				TradeSize:            decision.TradeSize,
				DailyBudgetRemaining: decision.DailyBudgetRemaining,
			},
		}
		select {
		case pb.publishCh <- event:
		default:
			pb.zapLogger.Warn("publish channel full, dropping event",
				zap.String("event_id", event.EventID))
		}
	}

	return decision
}

func (pb *PitBoss) evaluate(req ports.RiskCheckRequest) ports.RiskDecision {
	// #5: Acquire RLock for consistent snapshot across subsystems
	pb.mu.RLock()
	defer pb.mu.RUnlock()

	// #16: Validate non-empty MarketID/StrategyID
	if req.MarketID == "" || req.StrategyID == "" {
		return ports.RiskDecision{
			EventID:              uuid.New().String(),
			Timestamp:            time.Now().UTC(),
			MarketID:             &req.MarketID,
			StrategyID:           &req.StrategyID,
			TradeSize:            req.TradeSize,
			Decision:             "DENY",
			Reason:               "invalid_request",
			CurrentExposure:      decimal.Zero,
			LimitValue:           decimal.Zero,
			DailyBudgetRemaining: pb.dailyBudget.BudgetRemaining(),
			Capital:              pb.capital,
			Context:              map[string]interface{}{"error": fmt.Sprintf("empty MarketID=%q or StrategyID=%q", req.MarketID, req.StrategyID)},
		}
	}

	base := ports.RiskDecision{
		EventID:              uuid.New().String(),
		Timestamp:            time.Now().UTC(),
		MarketID:             &req.MarketID,
		StrategyID:           &req.StrategyID,
		TradeSize:            req.TradeSize,
		DailyBudgetRemaining: pb.dailyBudget.BudgetRemaining(),
		Capital:              pb.capital,
		Context:              map[string]interface{}{"side": req.Side},
	}

	if pb.stateBuilder.IsEmergencyStop() {
		base.Decision = "DENY"
		base.Reason = "emergency_stop"
		base.CurrentExposure = decimal.Zero
		base.LimitValue = decimal.Zero
		return base
	}

	if pb.dailyBudget.IsExhausted() {
		base.Decision = "DENY"
		base.Reason = "daily_limit"
		base.CurrentExposure = pb.dailyBudget.DailyLossValue()
		base.LimitValue = pb.dailyBudget.DailyLossLimitValue()
		return base
	}

	// #16: Check batasi win pause
	if pb.batasiMonitor != nil && pb.batasiMonitor.IsPaused() {
		base.Decision = "DENY"
		base.Reason = "batasi_win_paused"
		base.CurrentExposure = decimal.Zero
		base.LimitValue = decimal.Zero
		return base
	}

	// #16: Check metabolic alert
	if pb.metabolicMonitor != nil && pb.metabolicMonitor.IsAlert() {
		base.Decision = "DENY"
		base.Reason = "metabolic_alert"
		base.CurrentExposure = decimal.Zero
		base.LimitValue = decimal.Zero
		return base
	}

	// #16: Check correlation limits
	if pb.correlationTracker != nil {
		corrResult := pb.correlationTracker.CheckMarket(req.MarketID)
		if !corrResult.Allowed {
			base.Decision = "DENY"
			base.Reason = corrResult.Reason
			base.CurrentExposure = decimal.Zero
			base.LimitValue = decimal.Zero
			base.Context["correlated_with"] = corrResult.CorrelatedWith
			base.Context["group_name"] = corrResult.GroupName
			return base
		}
	}

	// #2: Check per-strategy capital allocation
	if req.StrategyID != "" {
		// #7: Use dedicated getter instead of full BuildState()
		strategyWeight := pb.stateBuilder.GetStrategyWeight(req.StrategyID)

		// #5, #8: If weight is zero or negative, deny — no unconfigured access
		if strategyWeight.LessThanOrEqual(decimal.Zero) {
			base.Decision = "DENY"
			base.Reason = "strategy_weight_not_configured"
			base.CurrentExposure = decimal.Zero
			base.LimitValue = decimal.Zero
			base.Context["strategy_id"] = req.StrategyID
			return base
		}

		// Validate weight is in [0, 100]
		if strategyWeight.GreaterThan(decimal.NewFromInt(100)) {
			base.Decision = "DENY"
			base.Reason = "strategy_weight_invalid"
			base.CurrentExposure = decimal.Zero
			base.LimitValue = decimal.Zero
			return base
		}

		maxAllocation := pb.capital.Mul(strategyWeight).Div(decimal.NewFromInt(100))
		currentUsage := pb.strategyLimits.GetExposure(req.StrategyID)
		if currentUsage.Add(req.TradeSize).GreaterThan(maxAllocation) {
			base.Decision = "DENY"
			base.Reason = "strategy_allocation_exceeded"
			base.CurrentExposure = currentUsage
			base.LimitValue = maxAllocation
			base.Context["strategy_weight"] = strategyWeight.String()
			base.Context["max_allocation"] = maxAllocation.String()
			return base
		}
	}

	if pb.marketLimits.WouldExceed(req.MarketID, req.TradeSize) {
		base.Decision = "DENY"
		base.Reason = "market_limit"
		base.CurrentExposure = pb.marketLimits.GetExposure(req.MarketID)
		base.LimitValue = pb.marketLimits.GetLimit()

		// #9: Publish RiskAlert for market limit warning
		pb.publishRiskAlert("market_limit_warning",
			fmt.Sprintf("Market %s utilization high: %s/%s", req.MarketID, base.CurrentExposure.String(), base.LimitValue.String()),
			"warning")
		return base
	}

	if pb.strategyLimits.WouldExceed(req.StrategyID, req.TradeSize) {
		base.Decision = "DENY"
		base.Reason = "strategy_limit"
		base.CurrentExposure = pb.strategyLimits.GetExposure(req.StrategyID)
		base.LimitValue = pb.strategyLimits.GetLimit()

		// #9: Publish RiskAlert for strategy limit warning
		pb.publishRiskAlert("strategy_limit_warning",
			fmt.Sprintf("Strategy %s utilization high: %s/%s", req.StrategyID, base.CurrentExposure.String(), base.LimitValue.String()),
			"warning")
		return base
	}

	base.Decision = "ALLOW"
	base.Reason = "approved"
	base.CurrentExposure = decimal.Zero
	base.LimitValue = decimal.Zero
	return base
}

// #9: Helper to publish RiskAlert via bounded worker pool (prevents goroutine leak)
func (pb *PitBoss) publishRiskAlert(alertType, message, severity string) {
	if pb.publisher == nil {
		return
	}
	// #4: Use a dedicated goroutine with timeout rather than unbounded spawn.
	// publishRiskAlert is fire-and-forget but bounded.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		event := ports.RiskAlert{
			EventID:   uuid.New().String(),
			EventType: "RiskAlert",
			Timestamp: time.Now().UTC(),
			Source:    "risk-manager",
			Payload: ports.RiskAlertPayload{
				AlertType: alertType,
				Message:   message,
				Severity:  severity,
			},
		}
		if err := pb.publisher.PublishRiskAlert(ctx, event); err != nil {
			pb.zapLogger.Error("failed to publish RiskAlert", zap.Error(err))
		}
	}()
}

func (pb *PitBoss) HandlePositionOpened(event ports.PositionOpened) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	// #14: Guard against zero or negative exposure
	exposure := event.Payload.EntryPrice.Mul(event.Payload.Quantity)
	if exposure.LessThanOrEqual(decimal.Zero) {
		pb.zapLogger.Warn("ignoring zero/negative exposure position",
			zap.String("market_id", event.Payload.MarketID),
			zap.String("exposure", exposure.String()),
		)
		return
	}

	pb.marketLimits.AddExposure(event.Payload.MarketID, exposure)
	pb.strategyLimits.AddExposure(event.Payload.StrategyID, exposure)

	pb.zapLogger.Info("position opened - exposure updated",
		zap.String("market_id", event.Payload.MarketID),
		zap.String("strategy_id", event.Payload.StrategyID),
		zap.String("exposure", exposure.String()),
	)
}

func (pb *PitBoss) HandlePositionClosed(event ports.PositionClosed) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	exposure := event.Payload.EntryPrice.Mul(event.Payload.Quantity)
	pb.marketLimits.RemoveExposure(event.Payload.MarketID, exposure)
	pb.strategyLimits.RemoveExposure(event.Payload.StrategyID, exposure)

	if event.Payload.RealizedPnL.IsNegative() {
		pb.dailyBudget.RecordLoss(event.Payload.RealizedPnL)
	}

	pb.zapLogger.Info("position closed - exposure updated",
		zap.String("market_id", event.Payload.MarketID),
		zap.String("strategy_id", event.Payload.StrategyID),
		zap.String("realized_pnl", event.Payload.RealizedPnL.String()),
	)
}

func (pb *PitBoss) HandlePositionUpdated(event ports.PositionUpdated) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	exposure := event.Payload.CurrentPrice.Mul(event.Payload.Quantity)
	pb.marketLimits.UpdateExposure(event.Payload.MarketID, exposure)
	if event.Payload.StrategyID != "" {
		pb.strategyLimits.UpdateExposure(event.Payload.StrategyID, exposure)
	}
}

func (pb *PitBoss) HandleCapitalUpdated(event ports.CapitalUpdated) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if pb.drawdownTracker != nil {
		pb.drawdownTracker.UpdateCapital(event.Payload.TotalCapital)
		peak, current, drawdown, limit := pb.drawdownTracker.GetState()
		pb.stateBuilder.SetDrawdownState(peak, current, drawdown, limit)
	}
}

func (pb *PitBoss) DrawdownTracker() *DrawdownTracker {
	return pb.drawdownTracker
}

func (pb *PitBoss) SetEmergencyStop(val bool) {
	pb.stateBuilder.SetEmergencyStop(val)
}

func (pb *PitBoss) IsEmergencyStop() bool {
	return pb.stateBuilder.IsEmergencyStop()
}

func (pb *PitBoss) DailyBudget() *DailyBudget {
	return pb.dailyBudget
}

func (pb *PitBoss) MarketLimits() *MarketExposure {
	return pb.marketLimits
}

func (pb *PitBoss) StrategyLimits() *StrategyExposure {
	return pb.strategyLimits
}

func (pb *PitBoss) StateBuilder() *StateBuilder {
	return pb.stateBuilder
}

func (pb *PitBoss) Capital() decimal.Decimal {
	return pb.capital
}

// #16: Wire correlation tracker
func (pb *PitBoss) SetCorrelationTracker(ct *risk.CorrelationTracker) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.correlationTracker = ct
}

// #16: Wire batasi win monitor
func (pb *PitBoss) SetBatasiMonitor(bw *risk.BatasiWinMonitor) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.batasiMonitor = bw
}

// #16: Wire metabolic monitor
func (pb *PitBoss) SetMetabolicMonitor(mm *risk.MetabolicMonitor) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	pb.metabolicMonitor = mm
}
