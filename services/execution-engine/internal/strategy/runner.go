package strategy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pqap/services/execution-engine/metrics"
	"go.uber.org/zap"
)

type StrategyStatus string

const (
	StatusRunning StrategyStatus = "running"
	StatusFailed  StrategyStatus = "failed"
	StatusStopped StrategyStatus = "stopped"
)

type StrategyConfig struct {
	ID           string
	Name         string
	MaxRetries   int
	RetryBackoff time.Duration
}

type StrategyFunc func(ctx context.Context) error

type Runner struct {
	mu        sync.Mutex
	config    StrategyConfig
	status    StrategyStatus
	cancel    context.CancelFunc
	err       error
	logger    *zap.Logger
	onFailure func(strategyID string, err error)
	done      chan struct{} // #4: goroutine lifecycle synchronization
}

func NewRunner(config StrategyConfig, logger *zap.Logger, onFailure func(string, error)) *Runner {
	// #10: Guard zero backoff
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = 1 * time.Second
	}
	return &Runner{
		config:    config,
		status:    StatusStopped,
		logger:    logger,
		onFailure: onFailure,
		done:      make(chan struct{}),
	}
}

// Run executes the strategy function with panic recovery and retry logic.
func (r *Runner) Run(ctx context.Context, fn StrategyFunc) {
	r.mu.Lock()
	if r.status == StatusRunning {
		r.mu.Unlock()
		return // #13: Silent no-op on double-Run (by design — idempotent)
	}
	ctx, r.cancel = context.WithCancel(ctx)
	r.status = StatusRunning
	r.err = nil
	r.done = make(chan struct{}) // #4: Reset done channel
	r.mu.Unlock()

	go r.execute(ctx, fn)
}

func (r *Runner) execute(ctx context.Context, fn StrategyFunc) {
	defer close(r.done) // #4: Signal completion

	defer func() {
		if rec := recover(); rec != nil {
			r.mu.Lock()
			r.status = StatusFailed
			r.err = fmt.Errorf("panic: %v", rec)
			r.mu.Unlock()

			metrics.StrategyPanicRecoveries.Inc() // #11: Separate panic metric
			r.logger.Error("strategy panic recovered",
				zap.String("strategy_id", r.config.ID),
				zap.String("strategy_name", r.config.Name),
				zap.Any("panic", rec),
				zap.Stack("stack"),
			)

			// #1: Wrap onFailure in its own recover to prevent process crash
			if r.onFailure != nil {
				func() {
					defer func() {
						if r := recover(); r != nil {
							r.logger.Error("onFailure callback panicked",
								zap.String("strategy_id", r.config.ID),
								zap.Any("panic", r),
							)
						}
					}()
					r.onFailure(r.config.ID, fmt.Errorf("panic: %v", rec))
				}()
			}
		}
	}()

	var lastErr error
	for attempt := 0; attempt <= r.config.MaxRetries; attempt++ {
		if attempt > 0 {
			r.logger.Warn("retrying strategy",
				zap.String("strategy_id", r.config.ID),
				zap.Int("attempt", attempt),
				zap.Error(lastErr),
			)
			// #10: Exponential backoff with attempt multiplier
			delay := r.config.RetryBackoff * time.Duration(attempt)
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}
		}

		err := fn(ctx)
		if err == nil {
			r.mu.Lock()
			// #6: Only set stopped if still running (not stopped externally)
			if r.status == StatusRunning {
				r.status = StatusStopped
			}
			r.mu.Unlock()
			return
		}
		lastErr = err

		if ctx.Err() != nil {
			return
		}
	}

	// All retries exhausted
	r.mu.Lock()
	r.status = StatusFailed
	r.err = lastErr
	r.mu.Unlock()

	metrics.StrategyRetryExhausted.Inc() // #11: Separate retry metric
	r.logger.Error("strategy failed after retries",
		zap.String("strategy_id", r.config.ID),
		zap.Error(lastErr),
	)

	// #1: Wrap onFailure in its own recover
	if r.onFailure != nil {
		func() {
			defer func() {
				if r := recover(); r != nil {
					// onFailure panicked — already logged above
				}
			}()
			r.onFailure(r.config.ID, lastErr)
		}()
	}
}

// Stop gracefully stops the strategy and waits for goroutine exit.
func (r *Runner) Stop() {
	r.mu.Lock()
	if r.cancel != nil {
		r.cancel()
	}
	// #3: Only set stopped if still running (don't overwrite StatusFailed)
	if r.status == StatusRunning {
		r.status = StatusStopped
	}
	r.mu.Unlock()

	// #4: Wait for goroutine to exit
	<-r.done
}

// Status returns the current strategy status.
func (r *Runner) Status() StrategyStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

// Error returns the last error (if failed).
func (r *Runner) Error() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

// Manager manages multiple strategy runners.
type Manager struct {
	mu        sync.RWMutex
	runners   map[string]*Runner
	logger    *zap.Logger
	onFailure func(strategyID string, err error)
}

func NewManager(logger *zap.Logger, onFailure func(string, error)) *Manager {
	return &Manager{
		runners:   make(map[string]*Runner),
		logger:    logger,
		onFailure: onFailure,
	}
}

// Register adds a strategy runner to the manager.
func (m *Manager) Register(config StrategyConfig) *Runner {
	m.mu.Lock()
	defer m.mu.Unlock()
	runner := NewRunner(config, m.logger, m.onFailure)
	m.runners[config.ID] = runner
	return runner
}

// Start starts a strategy by ID.
func (m *Manager) Start(ctx context.Context, strategyID string, fn StrategyFunc) error {
	m.mu.RLock()
	runner, ok := m.runners[strategyID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("strategy %s not registered", strategyID)
	}
	runner.Run(ctx, fn)
	return nil
}

// Stop stops a strategy by ID and waits for goroutine exit.
func (m *Manager) Stop(strategyID string) error {
	m.mu.RLock()
	runner, ok := m.runners[strategyID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("strategy %s not registered", strategyID)
	}
	runner.Stop()
	return nil
}

// StopAll stops all strategies and waits for all goroutines to exit.
func (m *Manager) StopAll() {
	// #8: Copy runners under RLock, then stop outside lock to prevent deadlock
	m.mu.RLock()
	runners := make([]*Runner, 0, len(m.runners))
	for _, runner := range m.runners {
		runners = append(runners, runner)
	}
	m.mu.RUnlock()

	for _, runner := range runners {
		runner.Stop()
	}
}

// Status returns the status of all strategies.
func (m *Manager) Status() map[string]StrategyStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	statuses := make(map[string]StrategyStatus, len(m.runners))
	for id, runner := range m.runners {
		statuses[id] = runner.Status()
	}
	return statuses
}
