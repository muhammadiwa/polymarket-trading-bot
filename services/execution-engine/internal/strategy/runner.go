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
	StatusRunning  StrategyStatus = "running"
	StatusFailed   StrategyStatus = "failed"
	StatusStopped  StrategyStatus = "stopped"
)

type StrategyConfig struct {
	ID           string
	Name         string
	MaxRetries   int
	RetryBackoff time.Duration
}

type StrategyFunc func(ctx context.Context) error

type Runner struct {
	mu       sync.Mutex
	config   StrategyConfig
	status   StrategyStatus
	cancel   context.CancelFunc
	err      error
	logger   *zap.Logger
	onFailure func(strategyID string, err error) // callback on failure
}

func NewRunner(config StrategyConfig, logger *zap.Logger, onFailure func(string, error)) *Runner {
	return &Runner{
		config:    config,
		status:    StatusStopped,
		logger:    logger,
		onFailure: onFailure,
	}
}

// Run executes the strategy function with panic recovery and retry logic.
// It runs in a goroutine and returns immediately.
func (r *Runner) Run(ctx context.Context, fn StrategyFunc) {
	r.mu.Lock()
	if r.status == StatusRunning {
		r.mu.Unlock()
		return
	}
	ctx, r.cancel = context.WithCancel(ctx)
	r.status = StatusRunning
	r.err = nil
	r.mu.Unlock()

	go r.execute(ctx, fn)
}

func (r *Runner) execute(ctx context.Context, fn StrategyFunc) {
	defer func() {
		if rec := recover(); rec != nil {
			r.mu.Lock()
			r.status = StatusFailed
			r.err = fmt.Errorf("panic: %v", rec)
			r.mu.Unlock()

			metrics.StrategyIsolationFailures.Inc()
			r.logger.Error("strategy panic recovered",
				zap.String("strategy_id", r.config.ID),
				zap.String("strategy_name", r.config.Name),
				zap.Any("panic", rec),
				zap.Stack("stack"),
			)

			if r.onFailure != nil {
				r.onFailure(r.config.ID, fmt.Errorf("panic: %v", rec))
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
			select {
			case <-ctx.Done():
				return
			case <-time.After(r.config.RetryBackoff * time.Duration(attempt)):
			}
		}

		err := fn(ctx)
		if err == nil {
			r.mu.Lock()
			r.status = StatusStopped
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

	metrics.StrategyIsolationFailures.Inc()
	r.logger.Error("strategy failed after retries",
		zap.String("strategy_id", r.config.ID),
		zap.Error(lastErr),
	)

	if r.onFailure != nil {
		r.onFailure(r.config.ID, lastErr)
	}
}

// Stop gracefully stops the strategy.
func (r *Runner) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
	}
	r.status = StatusStopped
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
	mu       sync.RWMutex
	runners  map[string]*Runner
	logger   *zap.Logger
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

// Stop stops a strategy by ID.
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

// StopAll stops all strategies.
func (m *Manager) StopAll() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, runner := range m.runners {
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
