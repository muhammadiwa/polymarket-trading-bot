package tracker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/pqap/services/position-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type LimitAlert struct {
	repo             ports.PositionRepository
	publisher        ports.EventPublisher
	marketLimitPct   float64
	strategyLimitPct float64
	totalCapital     decimal.Decimal
	logger           *zap.Logger
}

func NewLimitAlert(
	repo ports.PositionRepository,
	publisher ports.EventPublisher,
	marketLimitPct float64,
	strategyLimitPct float64,
	totalCapital decimal.Decimal,
	logger *zap.Logger,
) *LimitAlert {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LimitAlert{
		repo:             repo,
		publisher:        publisher,
		marketLimitPct:   marketLimitPct,
		strategyLimitPct: strategyLimitPct,
		totalCapital:     totalCapital,
		logger:           logger,
	}
}

func (la *LimitAlert) CheckLimits(ctx context.Context, marketID string) {
	positions, err := la.repo.GetOpenPositionsByMarket(ctx, marketID)
	if err != nil {
		la.logger.Error("failed to get positions for limit check",
			zap.String("market_id", marketID),
			zap.Error(err),
		)
		return
	}

	for _, position := range positions {
		la.checkMarketLimit(ctx, position)
		la.checkStrategyLimit(ctx, position)
	}

	la.checkTotalCapitalLimit(ctx)
}

func (la *LimitAlert) checkMarketLimit(ctx context.Context, position *ports.Position) {
	positionValue := position.CurrentPrice.Mul(position.Quantity)
	limitValue := la.totalCapital.Mul(decimal.NewFromFloat(la.marketLimitPct))

	if positionValue.GreaterThan(limitValue) {
		la.sendAlert(ctx, "market_limit",
			"Position exceeds per-market limit",
			"Position value "+positionValue.String()+" exceeds market limit "+limitValue.String(),
			position.ID,
		)
	}
}

func (la *LimitAlert) checkStrategyLimit(ctx context.Context, position *ports.Position) {
	positions, err := la.repo.GetOpenPositionsByStrategy(ctx, position.StrategyID)
	if err != nil {
		la.logger.Error("failed to get positions by strategy", zap.Error(err))
		return
	}

	totalValue := decimal.Zero
	for _, p := range positions {
		totalValue = totalValue.Add(p.CurrentPrice.Mul(p.Quantity))
	}

	limitValue := la.totalCapital.Mul(decimal.NewFromFloat(la.strategyLimitPct))

	if totalValue.GreaterThan(limitValue) {
		la.sendAlert(ctx, "strategy_limit",
			"Strategy exposure exceeds per-strategy limit",
			"Strategy "+position.StrategyID+" total exposure "+totalValue.String()+" exceeds limit "+limitValue.String(),
			position.ID,
		)
	}
}

func (la *LimitAlert) checkTotalCapitalLimit(ctx context.Context) {
	positions, err := la.repo.GetOpenPositions(ctx)
	if err != nil {
		la.logger.Error("failed to get open positions for total capital check", zap.Error(err))
		return
	}

	totalExposure := decimal.Zero
	for _, p := range positions {
		totalExposure = totalExposure.Add(p.CurrentPrice.Mul(p.Quantity))
	}

	if totalExposure.GreaterThan(la.totalCapital) {
		la.sendAlert(ctx, "total_capital_limit",
			"Total capital utilization exceeds limit",
			"Total exposure "+totalExposure.String()+" exceeds total capital "+la.totalCapital.String(),
			"",
		)
	}
}

func (la *LimitAlert) sendAlert(ctx context.Context, alertType, title, message, positionID string) {
	metrics.LimitBreachTotal.Inc()

	riskAlert := ports.RiskAlert{
		EventID:   uuid.New().String(),
		EventType: "RiskAlert",
		Timestamp: time.Now().UTC(),
		Source:    "position-manager",
		Payload: ports.RiskAlertPayload{
			AlertType: alertType,
			Message:   message,
			Severity:  "warning",
		},
	}

	if err := la.publisher.PublishRiskAlert(ctx, riskAlert); err != nil {
		la.logger.Error("failed to publish risk alert", zap.Error(err))
	}

	notification := ports.NotificationRequest{
		EventID:   uuid.New().String(),
		EventType: "NotificationRequest",
		Timestamp: time.Now().UTC(),
		Source:    "position-manager",
		Payload: ports.NotificationRequestPayload{
			Category:       "limit_breach",
			Title:          title,
			Message:        message,
			Channel:        "telegram",
			Priority:       "warning",
			BypassThrottle: false,
		},
	}

	if err := la.publisher.PublishNotificationRequest(ctx, notification); err != nil {
		la.logger.Error("failed to send limit breach notification", zap.Error(err))
	}

	la.logger.Warn("position limit breach",
		zap.String("alert_type", alertType),
		zap.String("position_id", positionID),
		zap.String("message", message),
	)
}

func (la *LimitAlert) SetTotalCapital(capital decimal.Decimal) {
	la.totalCapital = capital
}
