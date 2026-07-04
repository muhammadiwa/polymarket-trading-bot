package risk

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

type BatasiWinState struct {
	CurrentStreak   int        `json:"current_streak"`
	Threshold       int        `json:"threshold"`
	IsPaused        bool       `json:"is_paused"`
	PausedAt        *time.Time `json:"paused_at"`
	CooldownMinutes int        `json:"cooldown_minutes"`
	ResumeAfter     *time.Time `json:"resume_after"`
}

type BatasiWinMonitor struct {
	mu              sync.RWMutex
	state           BatasiWinState
	repo            ports.RiskEventRepository
	publisher       ports.EventPublisher
	logger          *zap.Logger
	onPause         func()
	onResume        func()
	stateBuilder    StateBuilderWriter
}

type StateBuilderWriter interface {
	SetBatasiWinPaused(paused bool)
}

func NewBatasiWinMonitor(threshold, cooldownMinutes int, repo ports.RiskEventRepository, publisher ports.EventPublisher, logger *zap.Logger) *BatasiWinMonitor {
	return &BatasiWinMonitor{
		state: BatasiWinState{
			CurrentStreak:   0,
			Threshold:       threshold,
			IsPaused:        false,
			CooldownMinutes: cooldownMinutes,
		},
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

func (bw *BatasiWinMonitor) SetStateBuilder(sb StateBuilderWriter) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.stateBuilder = sb
}

func (bw *BatasiWinMonitor) SetOnPause(fn func()) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.onPause = fn
}

func (bw *BatasiWinMonitor) SetOnResume(fn func()) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.onResume = fn
}

func (bw *BatasiWinMonitor) RecordWin() {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.recordWinLocked()
}

func (bw *BatasiWinMonitor) recordWinLocked() {
	if bw.state.IsPaused {
		return
	}

	bw.state.CurrentStreak++
	metrics.BatasiWinStreakCurrent.Set(float64(bw.state.CurrentStreak))

	bw.logger.Info("win recorded",
		zap.Int("current_streak", bw.state.CurrentStreak),
		zap.Int("threshold", bw.state.Threshold),
	)

	if bw.state.CurrentStreak >= bw.state.Threshold {
		bw.triggerPause()
	}
}

func (bw *BatasiWinMonitor) RecordLoss() {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.recordLossLocked()
}

func (bw *BatasiWinMonitor) recordLossLocked() {
	bw.state.CurrentStreak = 0
	metrics.BatasiWinStreakCurrent.Set(0)

	bw.logger.Info("loss recorded, streak reset")
}

func (bw *BatasiWinMonitor) triggerPause() {
	now := time.Now().UTC()
	bw.state.IsPaused = true
	bw.state.PausedAt = &now
	metrics.BatasiWinPausesTotal.Inc()

	if bw.state.CooldownMinutes > 0 {
		resumeAfter := now.Add(time.Duration(bw.state.CooldownMinutes) * time.Minute)
		bw.state.ResumeAfter = &resumeAfter
	}

	bw.logger.Warn("batasi win triggered - trading paused",
		zap.Int("streak", bw.state.CurrentStreak),
		zap.Int("threshold", bw.state.Threshold),
		zap.Int("cooldown_minutes", bw.state.CooldownMinutes),
	)

	if bw.stateBuilder != nil {
		bw.stateBuilder.SetBatasiWinPaused(true)
	}

	if bw.onPause != nil {
		bw.onPause()
	}

	if bw.publisher != nil {
		// #3: Capture state snapshot under lock before spawning goroutine
		snapshot := bw.state
		go bw.publishTriggered(snapshot)
		go bw.publishNotification()
	}
}

func (bw *BatasiWinMonitor) publishTriggered(state BatasiWinState) {
	event := ports.BatasiWinTriggered{
		EventID:   uuid.New().String(),
		EventType: "BatasiWinTriggered",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.BatasiWinPayload{
			CurrentStreak:   state.CurrentStreak,
			Threshold:       state.Threshold,
			IsPaused:        state.IsPaused,
			PausedAt:        state.PausedAt,
			CooldownMinutes: state.CooldownMinutes,
			ResumeAfter:     state.ResumeAfter,
		},
	}
	ctx, cancel := timeoutContext(5 * time.Second)
	defer cancel()
	if err := bw.publisher.PublishBatasiWinTriggered(ctx, event); err != nil {
		bw.logger.Error("failed to publish batasi win triggered", zap.Error(err))
	}
}

func (bw *BatasiWinMonitor) publishNotification() {
	// #10: Send notification on Batasi Win pause
	event := ports.NotificationRequest{
		EventID:   uuid.New().String(),
		EventType: "NotificationRequest",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.NotificationRequestPayload{
			Category:       "risk",
			Title:          "Batasi Win - Trading Paused",
			Message:        "Win streak threshold reached. Trading has been paused.",
			Channel:        "telegram",
			Priority:       "high",
			BypassThrottle: true,
		},
	}
	ctx, cancel := timeoutContext(5 * time.Second)
	defer cancel()
	if err := bw.publisher.PublishNotificationRequest(ctx, event); err != nil {
		bw.logger.Error("failed to publish batasi win notification", zap.Error(err))
	}
}

func (bw *BatasiWinMonitor) ManualResume() bool {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if !bw.state.IsPaused {
		return false
	}

	bw.state.IsPaused = false
	bw.state.PausedAt = nil
	bw.state.ResumeAfter = nil
	bw.state.CurrentStreak = 0
	metrics.BatasiWinStreakCurrent.Set(0)

	bw.logger.Info("batasi win manually resumed")

	if bw.stateBuilder != nil {
		bw.stateBuilder.SetBatasiWinPaused(false)
	}

	if bw.onResume != nil {
		bw.onResume()
	}

	return true
}

func (bw *BatasiWinMonitor) CheckCooldownResume() bool {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if !bw.state.IsPaused || bw.state.ResumeAfter == nil {
		return false
	}

	if time.Now().UTC().Before(*bw.state.ResumeAfter) {
		return false
	}

	bw.state.IsPaused = false
	bw.state.PausedAt = nil
	bw.state.ResumeAfter = nil
	bw.state.CurrentStreak = 0
	metrics.BatasiWinStreakCurrent.Set(0)

	bw.logger.Info("batasi win auto-resumed after cooldown")

	if bw.stateBuilder != nil {
		bw.stateBuilder.SetBatasiWinPaused(false)
	}

	if bw.onResume != nil {
		bw.onResume()
	}

	return true
}

func (bw *BatasiWinMonitor) IsPaused() bool {
	bw.mu.RLock()
	defer bw.mu.RUnlock()
	return bw.state.IsPaused
}

func (bw *BatasiWinMonitor) GetState() BatasiWinState {
	bw.mu.RLock()
	defer bw.mu.RUnlock()
	return bw.state
}

func (bw *BatasiWinMonitor) SetState(state BatasiWinState) {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	bw.state = state
	metrics.BatasiWinStreakCurrent.Set(float64(state.CurrentStreak))
}

// #5: EvaluateTrade is the single locked section instead of calling RecordWin/RecordLoss
func (bw *BatasiWinMonitor) EvaluateTrade(pnl decimal.Decimal) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if pnl.IsPositive() {
		bw.recordWinLocked()
	} else if pnl.IsNegative() {
		bw.recordLossLocked()
	}
}
