package riskmanager

import (
	"testing"
	"time"

	"github.com/pqap/services/risk-manager/internal/risk"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func newTestBatasiWinMonitor(threshold, cooldown int) *risk.BatasiWinMonitor {
	logger, _ := zap.NewDevelopment()
	return risk.NewBatasiWinMonitor(threshold, cooldown, nil, nil, logger)
}

func TestBatasiWin_RecordsWin(t *testing.T) {
	bw := newTestBatasiWinMonitor(5, 0)

	bw.RecordWin()
	state := bw.GetState()
	if state.CurrentStreak != 1 {
		t.Errorf("expected streak 1, got %d", state.CurrentStreak)
	}
}

func TestBatasiWin_ResetsOnLoss(t *testing.T) {
	bw := newTestBatasiWinMonitor(5, 0)

	bw.RecordWin()
	bw.RecordWin()
	bw.RecordWin()
	bw.RecordLoss()

	state := bw.GetState()
	if state.CurrentStreak != 0 {
		t.Errorf("expected streak 0 after loss, got %d", state.CurrentStreak)
	}
}

func TestBatasiWin_PausesAtThreshold(t *testing.T) {
	bw := newTestBatasiWinMonitor(5, 0)

	for i := 0; i < 5; i++ {
		bw.RecordWin()
	}

	if !bw.IsPaused() {
		t.Error("expected trading to be paused at threshold")
	}

	state := bw.GetState()
	if state.PausedAt == nil {
		t.Error("expected PaedAt to be set")
	}
}

func TestBatasiWin_DoesNotPauseBeforeThreshold(t *testing.T) {
	bw := newTestBatasiWinMonitor(5, 0)

	for i := 0; i < 4; i++ {
		bw.RecordWin()
	}

	if bw.IsPaused() {
		t.Error("should not be paused before threshold")
	}
}

func TestBatasiWin_ManualResume(t *testing.T) {
	bw := newTestBatasiWinMonitor(5, 0)

	for i := 0; i < 5; i++ {
		bw.RecordWin()
	}

	if !bw.IsPaused() {
		t.Fatal("expected to be paused")
	}

	resumed := bw.ManualResume()
	if !resumed {
		t.Error("expected manual resume to succeed")
	}

	if bw.IsPaused() {
		t.Error("expected to be resumed")
	}

	state := bw.GetState()
	if state.CurrentStreak != 0 {
		t.Errorf("expected streak reset to 0, got %d", state.CurrentStreak)
	}
}

func TestBatasiWin_ManualResumeFailsWhenNotPaused(t *testing.T) {
	bw := newTestBatasiWinMonitor(5, 0)

	resumed := bw.ManualResume()
	if resumed {
		t.Error("expected manual resume to fail when not paused")
	}
}

func TestBatasiWin_CooldownResume(t *testing.T) {
	bw := newTestBatasiWinMonitor(3, 1)

	for i := 0; i < 3; i++ {
		bw.RecordWin()
	}

	if !bw.IsPaused() {
		t.Fatal("expected to be paused")
	}

	state := bw.GetState()
	if state.ResumeAfter == nil {
		t.Fatal("expected resume_after to be set")
	}

	resumed := bw.CheckCooldownResume()
	if resumed {
		t.Error("should not resume before cooldown expires")
	}
}

func TestBatasiWin_CooldownDisabled(t *testing.T) {
	bw := newTestBatasiWinMonitor(3, 0)

	for i := 0; i < 3; i++ {
		bw.RecordWin()
	}

	state := bw.GetState()
	if state.ResumeAfter != nil {
		t.Error("expected no resume_after when cooldown is 0")
	}
}

func TestBatasiWin_IgnoresWinsWhilePaused(t *testing.T) {
	bw := newTestBatasiWinMonitor(3, 0)

	for i := 0; i < 3; i++ {
		bw.RecordWin()
	}

	bw.RecordWin()
	bw.RecordWin()

	state := bw.GetState()
	if state.CurrentStreak != 3 {
		t.Errorf("expected streak to stay at 3 while paused, got %d", state.CurrentStreak)
	}
}

func TestBatasiWin_EvaluateTrade(t *testing.T) {
	bw := newTestBatasiWinMonitor(5, 0)

	bw.EvaluateTrade(decimal.NewFromFloat(100))
	bw.EvaluateTrade(decimal.NewFromFloat(50))

	state := bw.GetState()
	if state.CurrentStreak != 2 {
		t.Errorf("expected streak 2, got %d", state.CurrentStreak)
	}

	bw.EvaluateTrade(decimal.NewFromFloat(-30))

	state = bw.GetState()
	if state.CurrentStreak != 0 {
		t.Errorf("expected streak 0 after loss, got %d", state.CurrentStreak)
	}
}

func TestBatasiWin_SetState(t *testing.T) {
	bw := newTestBatasiWinMonitor(5, 0)

	now := time.Now().UTC()
	bw.SetState(risk.BatasiWinState{
		CurrentStreak:   3,
		Threshold:       5,
		IsPaused:        false,
		CooldownMinutes: 0,
	})

	state := bw.GetState()
	if state.CurrentStreak != 3 {
		t.Errorf("expected streak 3, got %d", state.CurrentStreak)
	}
	_ = now
}

func TestBatasiWin_OnPauseCallback(t *testing.T) {
	bw := newTestBatasiWinMonitor(3, 0)

	paused := false
	bw.SetOnPause(func() { paused = true })

	for i := 0; i < 3; i++ {
		bw.RecordWin()
	}

	if !paused {
		t.Error("expected on_pause callback to be called")
	}
}
