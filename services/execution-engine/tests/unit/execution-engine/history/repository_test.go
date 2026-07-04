package history_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pqap/services/execution-engine/internal/history"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func TestPostgresRepository_Insert_Idempotent(t *testing.T) {
	t.Skip("requires database connection")
}

func TestPostgresRepository_Insert_Success(t *testing.T) {
	t.Skip("requires database connection")
}

func TestPostgresRepository_GetByClientOrderID_Found(t *testing.T) {
	t.Skip("requires database connection")
}

func TestPostgresRepository_GetByClientOrderID_NotFound(t *testing.T) {
	t.Skip("requires database connection")
}

func TestPostgresRepository_GetByClientOrderID_ParsesDecimals(t *testing.T) {
	t.Skip("requires database connection")
}

var _ history.Repository = (*history.PostgresRepository)(nil)
