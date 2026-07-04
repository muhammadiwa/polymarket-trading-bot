package pitboss

import (
	"context"
	"fmt"
	"time"

	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/pqap/services/risk-manager/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Reconstructor struct {
	repo           ports.RiskEventRepository
	dailyBudget    *DailyBudget
	marketLimits   *MarketExposure
	strategyLimits *StrategyExposure
	stateBuilder   *StateBuilder
	drawdownTracker *DrawdownTracker
	logger         *zap.Logger
}

func NewReconstructor(
	repo ports.RiskEventRepository,
	dailyBudget *DailyBudget,
	marketLimits *MarketExposure,
	strategyLimits *StrategyExposure,
	stateBuilder *StateBuilder,
	drawdownTracker *DrawdownTracker,
	logger *zap.Logger,
) *Reconstructor {
	return &Reconstructor{
		repo:            repo,
		dailyBudget:     dailyBudget,
		marketLimits:    marketLimits,
		strategyLimits:  strategyLimits,
		stateBuilder:    stateBuilder,
		drawdownTracker: drawdownTracker,
		logger:          logger,
	}
}

func (r *Reconstructor) Reconstruct(ctx context.Context) error {
	start := time.Now()

	r.logger.Info("starting state reconstruction from PostgreSQL")

	// #8, #15: Use positions table for current exposures instead of risk_events
	marketExposures, strategyExposures, err := r.repo.GetPositionExposures(ctx)
	if err != nil {
		r.logger.Warn("failed to get position exposures, falling back to decisions", zap.Error(err))
		decisions, decErr := r.repo.GetTodayDecisions(ctx)
		if decErr != nil {
			return fmt.Errorf("failed to get today's decisions: %w", decErr)
		}
		r.logger.Info("loaded decisions from PostgreSQL (fallback)",
			zap.Int("count", len(decisions)),
		)

		var hasEmergencyStop bool
		for _, d := range decisions {
			if d.Decision == "DENY" && d.Reason == "emergency_stop" {
				hasEmergencyStop = true
			}
			if d.MarketID != nil && d.CurrentExposure.GreaterThan(decimal.Zero) {
				r.marketLimits.UpdateExposure(*d.MarketID, d.CurrentExposure)
			}
			if d.StrategyID != nil && d.CurrentExposure.GreaterThan(decimal.Zero) {
				r.strategyLimits.UpdateExposure(*d.StrategyID, d.CurrentExposure)
			}
		}
		if hasEmergencyStop {
			r.stateBuilder.SetEmergencyStopWithReason(true, "reconstructed_from_db")
			r.logger.Warn("emergency stop flag reconstructed from PostgreSQL")
		}
	} else {
		for marketID, exposure := range marketExposures {
			r.marketLimits.UpdateExposure(marketID, exposure)
		}
		for strategyID, exposure := range strategyExposures {
			r.strategyLimits.UpdateExposure(strategyID, exposure)
		}
		r.logger.Info("loaded position exposures from positions table",
			zap.Int("market_count", len(marketExposures)),
			zap.Int("strategy_count", len(strategyExposures)),
		)
	}

	dailyLoss, err := r.repo.GetDailyLoss(ctx)
	if err != nil {
		r.logger.Warn("failed to get daily loss from repo, using zero", zap.Error(err))
	} else {
		r.dailyBudget.SetDailyLossFromDB(dailyLoss)
	}

	duration := time.Since(start)
	metrics.ReconstructionDuration.Observe(float64(duration.Milliseconds()))

	r.logger.Info("state reconstruction completed",
		zap.Duration("duration", duration),
		zap.String("daily_loss", r.dailyBudget.DailyLossValue().String()),
		zap.String("daily_budget_remaining", r.dailyBudget.BudgetRemaining().String()),
		zap.Int("market_count", len(r.marketLimits.GetAllExposures())),
		zap.Int("strategy_count", len(r.strategyLimits.GetAllExposures())),
	)

	return nil
}
