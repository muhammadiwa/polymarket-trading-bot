package executor

import (
	"context"
	"time"

	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/pqap/services/execution-engine/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type RiskChecker struct {
	riskPort     ports.RiskPort
	riskEventRepo ports.RiskEventRepository
	logger       *zap.Logger
}

func NewRiskChecker(riskPort ports.RiskPort, riskEventRepo ports.RiskEventRepository, logger *zap.Logger) *RiskChecker {
	return &RiskChecker{
		riskPort:      riskPort,
		riskEventRepo: riskEventRepo,
		logger:        logger,
	}
}

func (rc *RiskChecker) Check(ctx context.Context, marketID, strategyID string, orderSize decimal.Decimal) (*ports.RiskDecision, int64, error) {
	start := time.Now()

	decision, err := rc.riskPort.CheckRisk(ctx, marketID, strategyID, orderSize)
	if err != nil {
		latencyMs := time.Since(start).Milliseconds()
		rc.logRiskEvent(ctx, marketID, strategyID, orderSize, false, "risk_check_error: "+err.Error(), latencyMs)
		return &ports.RiskDecision{
			Allowed: false,
			Reason:  "risk_check_error: " + err.Error(),
		}, latencyMs, err
	}

	latencyMs := time.Since(start).Milliseconds()
	metrics.RiskCheckLatency.Observe(float64(latencyMs))

	rc.logRiskEvent(ctx, marketID, strategyID, orderSize, decision.Allowed, decision.Reason, latencyMs)

	return decision, latencyMs, nil
}

func (rc *RiskChecker) logRiskEvent(ctx context.Context, marketID, strategyID string, orderSize decimal.Decimal, allowed bool, reason string, latencyMs int64) {
	if rc.riskEventRepo == nil {
		return
	}

	event := ports.RiskEvent{
		MarketID:   marketID,
		StrategyID: strategyID,
		OrderSize:  orderSize,
		Allowed:    allowed,
		Reason:     reason,
		LatencyMs:  latencyMs,
		CreatedAt:  time.Now().UTC(),
	}

	if err := rc.riskEventRepo.InsertRiskEvent(ctx, event); err != nil {
		rc.logger.Error("failed to log risk event",
			zap.String("market_id", marketID),
			zap.Error(err),
		)
	}
}
