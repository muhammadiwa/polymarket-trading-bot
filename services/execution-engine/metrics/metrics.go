package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	OrdersPlaced = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_orders_placed_total",
		Help: "Total number of orders placed via CLOB API",
	})

	OrdersFilled = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_orders_filled_total",
		Help: "Total number of orders fully filled",
	})

	OrdersFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_orders_failed_total",
		Help: "Total number of orders failed",
	})

	OrdersCancelled = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_orders_cancelled_total",
		Help: "Total number of orders cancelled",
	})

	RiskDenied = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_risk_denied_total",
		Help: "Total number of orders denied by Pit Boss risk check",
	})

	SlippageRejected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_slippage_rejected_total",
		Help: "Total number of orders rejected due to slippage",
	})

	OrderLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_execution_order_latency_ms",
		Help:    "Order placement latency in milliseconds",
		Buckets: []float64{10, 25, 50, 100, 150, 200, 300, 500, 1000},
	})

	RiskCheckLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_execution_risk_check_latency_ms",
		Help:    "Pit Boss risk check latency in milliseconds",
		Buckets: []float64{1, 2, 5, 10, 25, 50},
	})

	FillLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_execution_fill_latency_ms",
		Help:    "Time from order placement to fill in milliseconds",
		Buckets: []float64{100, 500, 1000, 5000, 10000, 30000},
	})

	CircuitBreakerState = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_execution_circuit_breaker_state",
		Help: "Circuit breaker state (0=closed, 1=open, 2=half-open)",
	})

	CircuitBreakerTrips = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_circuit_breaker_trips_total",
		Help: "Total number of circuit breaker trips",
	})

	ActiveOrders = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_execution_active_orders",
		Help: "Number of currently open orders",
	})

	PartialFills = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_partial_fills_total",
		Help: "Total number of partial fill events",
	})

	DuplicateRejected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_duplicate_rejected_total",
		Help: "Total number of duplicate client_order_id rejections",
	})

	NATSConnectionStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_execution_nats_connection_status",
		Help: "NATS connection status (1=connected, 0=disconnected)",
	})

	NATSPublisherConnectionStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_execution_nats_publisher_connection_status",
		Help: "NATS publisher connection status (1=connected, 0=disconnected)",
	})

	AtomicPairsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_atomic_pairs_total",
		Help: "Total number of atomic pair attempts",
	})

	AtomicPairsFilled = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_atomic_pairs_filled_total",
		Help: "Total number of fully filled atomic pairs",
	})

	AtomicPairsPartial = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_atomic_pairs_partial_total",
		Help: "Total number of partial fill atomic pairs",
	})

	AtomicPairsCancelled = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_atomic_pairs_cancelled_total",
		Help: "Total number of cancelled atomic pairs",
	})

	AtomicPairsFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_atomic_pairs_failed_total",
		Help: "Total number of failed atomic pairs",
	})

	AtomicPlacementLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_execution_atomic_placement_latency_ms",
		Help:    "Time to place both legs of atomic pair in milliseconds",
		Buckets: []float64{10, 25, 50, 100, 150, 200, 300, 500, 1000},
	})

	AtomicCancelLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_execution_atomic_cancel_latency_ms",
		Help:    "Time to cancel other leg on failure in milliseconds",
		Buckets: []float64{10, 25, 50, 100, 200, 500, 1000},
	})

	CircuitBreakerConsecutiveErrors = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_execution_circuit_breaker_consecutive_errors",
		Help: "Current consecutive error count",
	})

	CircuitBreakerCooldownRemaining = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_execution_circuit_breaker_cooldown_remaining_ms",
		Help: "Cooldown remaining in milliseconds",
	})

	TradeRecordsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pqap_execution_trade_records_total",
		Help: "Total trade records written by fill status",
	}, []string{"fill_status"})

	TradeRecordLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_execution_trade_record_latency_ms",
		Help:    "DB insert latency for trade records in milliseconds",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
	})

	TradeRecordErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_execution_trade_record_errors_total",
		Help: "Total failed trade record inserts",
	})

	// Strategy isolation metrics
	StrategyPanicRecoveries = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_strategy_panic_recoveries_total",
		Help: "Total strategy panics caught by recovery",
	})

	StrategyRetryExhausted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_strategy_retry_exhausted_total",
		Help: "Total strategies failed after all retries exhausted",
	})

	StrategyCapitalRejections = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_strategy_capital_rejections_total",
		Help: "Total trades rejected due to per-strategy capital allocation",
	})
)
