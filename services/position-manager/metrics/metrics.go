package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PositionOpenTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_position_open_total",
		Help: "Total number of positions opened",
	})

	PositionClosedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_position_closed_total",
		Help: "Total number of positions closed",
	})

	PositionSettledTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_position_settled_total",
		Help: "Total number of positions settled via market resolution",
	})

	PositionActiveCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_position_active_count",
		Help: "Number of currently open positions",
	})

	PnLUpdateLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_position_pnl_update_latency_ms",
		Help:    "PnL recalculation latency in milliseconds",
		Buckets: []float64{10, 50, 100, 250, 500, 1000},
	})

	UnrealizedPnL = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_position_unrealized_pnl_usd",
		Help: "Total unrealized PnL across all positions in USD",
	})

	RealizedPnLTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_position_realized_pnl_usd_total",
		Help: "Cumulative realized PnL in USD",
	})

	ReconciliationTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_position_reconciliation_total",
		Help: "Total number of reconciliation runs",
	})

	ReconciliationMismatchesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_position_reconciliation_mismatches_total",
		Help: "Total number of reconciliation mismatches detected",
	})

	ReconciliationConsecutive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_position_reconciliation_consecutive",
		Help: "Current consecutive mismatch count",
	})

	ExitLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_position_exit_latency_ms",
		Help:    "Manual exit order latency in milliseconds",
		Buckets: []float64{10, 50, 100, 250, 500, 1000},
	})

	LimitBreachTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_position_limit_breach_total",
		Help: "Total number of position limit breach alerts",
	})

	NATSConnectionStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_position_nats_connection_status",
		Help: "NATS connection status (1=connected, 0=disconnected)",
	})
)
