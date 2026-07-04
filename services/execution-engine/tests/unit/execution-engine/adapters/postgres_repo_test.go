package adapters_test

import (
	"testing"
)

func TestPostgresRepo_EnsureSchema_NoTradesTable(t *testing.T) {
	t.Skip("requires database connection")
}

func TestPostgresRepo_InsertTrade(t *testing.T) {
	t.Skip("requires database connection")
}

func TestPostgresRepo_InsertRiskEvent(t *testing.T) {
	t.Skip("requires database connection")
}

func TestPostgresRepo_InsertCircuitBreakerEvent(t *testing.T) {
	t.Skip("requires database connection")
}
