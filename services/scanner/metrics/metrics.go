package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	MarketsTracked = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_scanner_markets_tracked",
		Help: "Total number of active markets being tracked",
	})

	UpdateLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_scanner_update_latency_ms",
		Help:    "Price update processing latency in milliseconds",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
	})

	WSConnectionStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_scanner_ws_connected",
		Help: "WebSocket connection status (1=connected, 0=disconnected)",
	})

	StaleMarkets = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_scanner_stale_markets",
		Help: "Total number of stale markets",
	})

	WSReconnectTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_scanner_ws_reconnect_total",
		Help: "Total number of WebSocket reconnection attempts",
	})

	EventsPublished = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pqap_scanner_events_published_total",
		Help: "Total number of events published to NATS",
	}, []string{"event_type"})

	ReconciliationTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_scanner_reconciliation_total",
		Help: "Total number of reconciliation runs completed",
	})

	PriceDiscrepanciesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_scanner_price_discrepancies_total",
		Help: "Total number of price discrepancies detected during reconciliation",
	})

	RestBatchTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_scanner_rest_batch_total",
		Help: "Total number of batch API calls made",
	})

	RestRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_scanner_rest_requests_total",
		Help: "Total number of REST API requests",
	})

	RestErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_scanner_rest_errors_total",
		Help: "Total number of REST API errors",
	})

	RestLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_scanner_rest_latency_ms",
		Help:    "REST API latency in milliseconds",
		Buckets: []float64{5, 10, 25, 50, 100, 250, 500, 1000},
	})
)
