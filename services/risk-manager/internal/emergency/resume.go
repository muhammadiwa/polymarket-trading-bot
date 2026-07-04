package emergency

import (
	"context"
	"time"

	"github.com/pqap/services/risk-manager/internal/pitboss"
	"go.uber.org/zap"
)

type ResumeHandler struct {
	emergencyStop *EmergencyStop
	stateBuilder  *pitboss.StateBuilder
	logger        *zap.Logger
	riskLogger    *pitboss.Logger // #11: log resume to risk_events
}

func NewResumeHandler(
	emergencyStop *EmergencyStop,
	stateBuilder *pitboss.StateBuilder,
	logger *zap.Logger,
	riskLogger *pitboss.Logger, // #11
) *ResumeHandler {
	return &ResumeHandler{
		emergencyStop: emergencyStop,
		stateBuilder:  stateBuilder,
		logger:        logger,
		riskLogger:    riskLogger,
	}
}

func (rh *ResumeHandler) HandleResume(ctx context.Context, resumedBy string) error {
	if !rh.emergencyStop.IsActive() {
		rh.logger.Warn("resume requested but emergency stop is not active")
		return nil
	}

	previousReason := rh.emergencyStop.Reason()

	if err := rh.emergencyStop.ResumeByUser(ctx, resumedBy); err != nil {
		return err
	}

	rh.stateBuilder.SetEmergencyStop(false)

	// #11: Log resume event to risk_events
	if rh.riskLogger != nil {
		if err := rh.riskLogger.LogEmergencyEvent(ctx, "resume", map[string]interface{}{
			"previous_reason": previousReason,
			"resumed_by":      resumedBy,
		}); err != nil {
			rh.logger.Error("failed to log resume event to risk_events", zap.Error(err))
		}
	}

	rh.logger.Info("resume completed - drawdown peak equity NOT reset",
		zap.String("previous_reason", previousReason),
		zap.String("resumed_by", resumedBy),
		zap.Time("resumed_at", time.Now().UTC()),
	)

	return nil
}
