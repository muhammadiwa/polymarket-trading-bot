package adapters_test

import (
	"testing"
)

func TestNATSPublisher_PublishOrderPlaced(t *testing.T) {
	t.Skip("requires NATS connection")
}

func TestNATSPublisher_PublishOrderPartialFill_Subject(t *testing.T) {
	t.Skip("requires NATS connection")
}

func TestNATSPublisher_PublishCircuitBreakerTripped_Subject(t *testing.T) {
	t.Skip("requires NATS connection")
}

func TestNATSPublisher_PublishCircuitBreakerResumed_Subject(t *testing.T) {
	t.Skip("requires NATS connection")
}

func TestNATSPublisher_Conn_NotExposed(t *testing.T) {
	t.Skip("compile-time check: Conn() method removed")
}

func TestNATSPublisher_IsConnected(t *testing.T) {
	t.Skip("requires NATS connection")
}
