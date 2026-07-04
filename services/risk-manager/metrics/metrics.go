package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const maxLabelCardinality = 100

var (
	labelGuard   sync.RWMutex
	allowedIDs   = make(map[string]struct{})
	labelCounter int
)

// IsAllowedLabel checks if a label value is within the cardinality cap.
func IsAllowedLabel(id string) bool {
	if id == "" {
		return false
	}
	labelGuard.RLock()
	_, ok := allowedIDs[id]
	labelGuard.RUnlock()
	return ok
}

// RegisterLabel attempts to register a new label value. Returns false if cap is reached.
func RegisterLabel(id string) bool {
	if id == "" {
		return false
	}
	labelGuard.Lock()
	defer labelGuard.Unlock()
	if _, ok := allowedIDs[id]; ok {
		return true
	}
	if labelCounter >= maxLabelCardinality {
		return false
	}
	allowedIDs[id] = struct{}{}
	labelCounter++
	return true
}

func SafeLabel(id string) string {
	if RegisterLabel(id) {
		return id
	}
	return "_other"
}

var (
	RiskCheckLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_risk_check_latency_ms",
		Help:    "Risk check latency in milliseconds",
		Buckets: []float64{1, 2, 5, 10, 25, 50},
	})

	RiskCheckTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_check_total",
		Help: "Total risk checks performed",
	})

	RiskCheckDeniedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pqap_risk_check_denied_total",
		Help: "Total risk checks denied by reason",
	}, []string{"reason"})

	DailyBudgetRemaining = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_daily_budget_remaining_usd",
		Help: "Daily budget remaining in USD",
	})

	DailyLoss = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_daily_loss_usd",
		Help: "Daily realized loss in USD",
	})

	DailyLossLimit = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_daily_loss_limit_usd",
		Help: "Daily loss limit in USD",
	})

	MarketExposure = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pqap_risk_market_exposure_usd",
		Help: "Per-market exposure in USD",
	}, []string{"market_id"})

	StrategyExposure = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pqap_risk_strategy_exposure_usd",
		Help: "Per-strategy exposure in USD",
	}, []string{"strategy_id"})

	MarketUtilization = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pqap_risk_market_utilization",
		Help: "Per-market utilization ratio",
	}, []string{"market_id"})

	StrategyUtilization = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pqap_risk_strategy_utilization",
		Help: "Per-strategy utilization ratio",
	}, []string{"strategy_id"})

	StateRefreshTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_state_refresh_total",
		Help: "Total Redis state refreshes",
	})

	StateRefreshLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_risk_state_refresh_latency_ms",
		Help:    "Redis state write latency in milliseconds",
		Buckets: []float64{1, 2, 5, 10, 25, 50, 100},
	})

	ReconstructionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_risk_reconstruction_duration_ms",
		Help:    "State reconstruction time on startup in milliseconds",
		Buckets: []float64{10, 50, 100, 250, 500, 1000, 5000},
	})

	EmergencyStopTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_emergency_stop_total",
		Help: "Total emergency stops triggered",
	})

	EventsLoggedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_events_logged_total",
		Help: "Total risk events logged to PostgreSQL",
	})

	NATSConnectionStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_nats_connection_status",
		Help: "NATS connection status (1=connected, 0=disconnected)",
	})

	DrawdownCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_drawdown_current",
		Help: "Current drawdown ratio (0.0-1.0)",
	})

	DrawdownLimit = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_drawdown_limit",
		Help: "Drawdown threshold ratio",
	})

	DrawdownPeakEquityUSD = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_drawdown_peak_equity_usd",
		Help: "Peak equity in USD",
	})

	DrawdownCurrentEquityUSD = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_drawdown_current_equity_usd",
		Help: "Current equity in USD",
	})

	EmergencyStopActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_emergency_stop_active",
		Help: "1 if emergency stop active, 0 if not",
	})

	EmergencyStopDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_emergency_stop_duration_seconds",
		Help: "Seconds since emergency stop activated",
	})

	OrderCancelLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_risk_order_cancel_latency_ms",
		Help:    "Order cancellation latency in milliseconds",
		Buckets: []float64{10, 50, 100, 250, 500, 1000, 5000},
	})

	OrderCancelTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_order_cancel_total",
		Help: "Total orders cancelled by emergency stop",
	})

	ResumeTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_resume_total",
		Help: "Total trading resumes",
	})

	DrawdownWarningTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_drawdown_warning_total",
		Help: "Total drawdown warnings",
	})

	// #25: Counter for risk events dropped due to full channel
	RiskEventsDroppedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_events_dropped_total",
		Help: "Total risk events dropped due to full log channel",
	})

	// #3: Gauge for consecutive state write failures (death spiral detection)
	StateWriteConsecutiveFailures = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_state_write_consecutive_failures",
		Help: "Number of consecutive Redis state write failures",
	})

	CorrelatedPositionsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pqap_risk_correlated_positions_total",
		Help: "Current correlated position count per group",
	}, []string{"group_id"})

	CorrelationRejectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_correlation_rejections_total",
		Help: "Total correlation limit rejections",
	})

	BatasiWinStreakCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_batasi_win_streak_current",
		Help: "Current win streak count",
	})

	BatasiWinPausesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_batasi_win_pauses_total",
		Help: "Total Batasi Win pause events",
	})

	MetabolicCPUPercent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_metabolic_cpu_percent",
		Help: "Current CPU usage",
	})

	MetabolicMemoryBytes = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_metabolic_memory_bytes",
		Help: "Current memory usage",
	})

	MetabolicGoroutines = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_risk_metabolic_goroutines_total",
		Help: "Current goroutine count",
	})

	MetabolicAlertsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_risk_metabolic_alerts_total",
		Help: "Total metabolic rate alerts",
	})
)
