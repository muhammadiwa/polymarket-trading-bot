package pitboss

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/pqap/services/risk-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	logWorkerCount   = 4
	logChannelBuffer = 256
)

type Logger struct {
	repo   ports.RiskEventRepository
	logger *zap.Logger
	ch     chan ports.RiskDecision
	wg     sync.WaitGroup // #8: for graceful shutdown
}

func NewLogger(repo ports.RiskEventRepository, logger *zap.Logger) *Logger {
	l := &Logger{
		repo:   repo,
		logger: logger,
		ch:     make(chan ports.RiskDecision, logChannelBuffer),
	}
	l.wg.Add(logWorkerCount)
	for i := 0; i < logWorkerCount; i++ {
		go l.worker()
	}
	return l
}

func (l *Logger) worker() {
	defer l.wg.Done()
	for decision := range l.ch {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := l.LogDecision(ctx, decision); err != nil {
			l.logger.Error("async decision logging failed", zap.Error(err))
		}
		cancel()
	}
}

func (l *Logger) LogDecision(ctx context.Context, decision ports.RiskDecision) error {
	if err := l.repo.InsertRiskEvent(ctx, decision); err != nil {
		l.logger.Error("failed to log risk decision",
			zap.String("event_id", decision.EventID),
			zap.String("decision", decision.Decision),
			zap.String("reason", decision.Reason),
			zap.Error(err),
		)
		return err
	}

	metrics.EventsLoggedTotal.Inc()

	l.logger.Info("risk decision logged",
		zap.String("event_id", decision.EventID),
		zap.String("decision", decision.Decision),
		zap.String("reason", decision.Reason),
		zap.String("trade_size", decision.TradeSize.String()),
		zap.String("daily_budget_remaining", decision.DailyBudgetRemaining.String()),
	)

	return nil
}

func (l *Logger) LogDecisionAsync(decision ports.RiskDecision) {
	select {
	case l.ch <- decision:
	default:
		metrics.RiskEventsDroppedTotal.Inc() // #25: count dropped events
		l.logger.Warn("log decision channel full, dropping decision",
			zap.String("event_id", decision.EventID),
		)
	}
}

// Close drains the log channel and waits for all workers to finish. #8
func (l *Logger) Close() {
	close(l.ch)
	l.wg.Wait()
}

func (l *Logger) LogEmergencyEvent(ctx context.Context, reason string, contextData map[string]interface{}) error {
	decision := ports.RiskDecision{
		EventID:              uuid.New().String(),
		Timestamp:            time.Now().UTC(),
		Decision:             "DENY",
		Reason:               "emergency_stop",
		TradeSize:            decimal.Zero,
		CurrentExposure:      decimal.Zero,
		LimitValue:           decimal.Zero,
		DailyBudgetRemaining: decimal.Zero,
		Capital:              decimal.Zero,
		Context: map[string]interface{}{
			"emergency_reason": reason,
		},
	}

	if contextData != nil {
		for k, v := range contextData {
			decision.Context[k] = v
		}
	}

	return l.LogDecision(ctx, decision)
}
