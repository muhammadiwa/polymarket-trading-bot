package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	OpportunitiesDetected = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_arb_opportunities_detected_total",
		Help: "Total number of arbitrage opportunities detected (before filter)",
	})

	OpportunitiesEmitted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_arb_opportunities_emitted_total",
		Help: "Total number of opportunities emitted to NATS (above threshold)",
	})

	OpportunitiesFiltered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_arb_opportunities_filtered_total",
		Help: "Total number of opportunities filtered (below threshold)",
	})

	DetectionLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_arb_detection_latency_ms",
		Help:    "Detection latency in milliseconds",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
	})

	ScoreDistribution = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_arb_score_distribution",
		Help:    "Opportunity score distribution",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
	})

	FillProbabilityEstimate = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pqap_arb_fill_probability_estimate",
		Help:    "Fill probability estimates",
		Buckets: []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
	})

	NATSConnectionStatus = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pqap_arb_nats_connection_status",
		Help: "NATS connection status (1=connected, 0=disconnected)",
	})

	StaleMarketIgnored = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pqap_arb_stale_market_ignored_total",
		Help: "Total number of stale market events ignored",
	})
)
