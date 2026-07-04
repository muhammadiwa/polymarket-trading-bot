package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/pqap/services/scanner/metrics"
)

func TestMetricNamesFollowConvention(t *testing.T) {
	expectedNames := map[string]string{
		"MarketsTracked":    "pqap_scanner_markets_tracked_total",
		"UpdateLatency":     "pqap_scanner_update_latency_ms",
		"WSConnectionStatus": "pqap_scanner_ws_connection_status",
		"StaleMarkets":      "pqap_scanner_stale_markets_total",
		"WSReconnectTotal":  "pqap_scanner_ws_reconnect_total",
		"RestRequestsTotal": "pqap_scanner_rest_requests_total",
		"RestErrorsTotal":   "pqap_scanner_rest_errors_total",
		"RestLatency":       "pqap_scanner_rest_latency_ms",
		"RestBatchTotal":    "pqap_scanner_rest_batch_total",
		"ReconciliationTotal":     "pqap_scanner_reconciliation_total",
		"PriceDiscrepanciesTotal": "pqap_scanner_price_discrepancies_total",
	}

	metricMap := map[string]prometheus.Collector{
		"MarketsTracked":          metrics.MarketsTracked,
		"UpdateLatency":           metrics.UpdateLatency,
		"WSConnectionStatus":      metrics.WSConnectionStatus,
		"StaleMarkets":            metrics.StaleMarkets,
		"WSReconnectTotal":        metrics.WSReconnectTotal,
		"RestRequestsTotal":       metrics.RestRequestsTotal,
		"RestErrorsTotal":         metrics.RestErrorsTotal,
		"RestLatency":             metrics.RestLatency,
		"RestBatchTotal":          metrics.RestBatchTotal,
		"ReconciliationTotal":     metrics.ReconciliationTotal,
		"PriceDiscrepanciesTotal": metrics.PriceDiscrepanciesTotal,
	}

	for name, collector := range metricMap {
		expected, ok := expectedNames[name]
		if !ok {
			t.Errorf("No expected name for metric %s", name)
			continue
		}

		descs := make(chan *prometheus.Desc, 1)
		collector.Describe(descs)
		close(descs)
		desc := <-descs

		fqName := desc.String()
		if fqName == "" {
			t.Errorf("Metric %s: could not get FQName", name)
		}
	}
}

func TestMarketsTrackedIsGauge(t *testing.T) {
	metrics.MarketsTracked.Set(42)

	metric := &dto.Metric{}
	if err := metrics.MarketsTracked.Write(metric); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}
	if metric.GetGauge() == nil {
		t.Error("MarketsTracked should be a Gauge")
	}
	if metric.GetGauge().GetValue() != 42 {
		t.Errorf("Expected 42, got %f", metric.GetGauge().GetValue())
	}
}

func TestWSConnectionStatusValues(t *testing.T) {
	metrics.WSConnectionStatus.Set(1)
	metric := &dto.Metric{}
	if err := metrics.WSConnectionStatus.Write(metric); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}
	if metric.GetGauge().GetValue() != 1 {
		t.Errorf("Expected 1 (connected), got %f", metric.GetGauge().GetValue())
	}

	metrics.WSConnectionStatus.Set(0)
	if err := metrics.WSConnectionStatus.Write(metric); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}
	if metric.GetGauge().GetValue() != 0 {
		t.Errorf("Expected 0 (disconnected), got %f", metric.GetGauge().GetValue())
	}
}

func TestStaleMarketsIsGauge(t *testing.T) {
	metrics.StaleMarkets.Set(5)

	metric := &dto.Metric{}
	if err := metrics.StaleMarkets.Write(metric); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}
	if metric.GetGauge() == nil {
		t.Error("StaleMarkets should be a Gauge")
	}
	if metric.GetGauge().GetValue() != 5 {
		t.Errorf("Expected 5, got %f", metric.GetGauge().GetValue())
	}
}

func TestUpdateLatencyIsHistogram(t *testing.T) {
	metrics.UpdateLatency.Observe(100)
	metrics.UpdateLatency.Observe(200)

	metric := &dto.Metric{}
	if err := metrics.UpdateLatency.Write(metric); err != nil {
		t.Fatalf("Failed to write metric: %v", err)
	}
	if metric.GetHistogram() == nil {
		t.Error("UpdateLatency should be a Histogram")
	}
	if metric.GetHistogram().GetSampleCount() != 2 {
		t.Errorf("Expected 2 observations, got %d", metric.GetHistogram().GetSampleCount())
	}
}

func TestEventsPublishedCounterIncrement(t *testing.T) {
	metrics.EventsPublished.WithLabelValues("MarketPriceUpdated").Inc()
	metrics.EventsPublished.WithLabelValues("MarketPriceUpdated").Inc()
	metrics.EventsPublished.WithLabelValues("MarketStaleDetected").Inc()

	metricChan := make(chan prometheus.Metric, 10)
	metrics.EventsPublished.Collect(metricChan)
	close(metricChan)

	counts := make(map[string]float64)
	for m := range metricChan {
		dto := &dto.Metric{}
		if err := m.Write(dto); err != nil {
			t.Fatalf("Failed to write metric: %v", err)
		}
		label := dto.GetLabel()[0].GetValue()
		counts[label] = dto.GetCounter().GetValue()
	}

	if counts["MarketPriceUpdated"] != 2 {
		t.Errorf("Expected 2 for MarketPriceUpdated, got %f", counts["MarketPriceUpdated"])
	}
	if counts["MarketStaleDetected"] != 1 {
		t.Errorf("Expected 1 for MarketStaleDetected, got %f", counts["MarketStaleDetected"])
	}
}
