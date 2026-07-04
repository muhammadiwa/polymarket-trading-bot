package position_manager

import (
	"testing"

	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/pqap/services/position-manager/internal/tracker"
	"github.com/shopspring/decimal"
)

func TestPnL_YESPositionProfit(t *testing.T) {
	position := &ports.Position{
		Side:       "YES",
		EntryPrice: decimal.NewFromFloat(0.6000),
		Quantity:   decimal.NewFromFloat(100),
	}

	currentPrice := decimal.NewFromFloat(0.7500)
	pnl := tracker.CalculateUnrealizedPnL(position, currentPrice)

	expected := decimal.NewFromFloat(15.0)
	if !pnl.Equal(expected) {
		t.Errorf("expected PnL %s, got %s", expected, pnl)
	}
}

func TestPnL_YESPositionLoss(t *testing.T) {
	position := &ports.Position{
		Side:       "YES",
		EntryPrice: decimal.NewFromFloat(0.7000),
		Quantity:   decimal.NewFromFloat(100),
	}

	currentPrice := decimal.NewFromFloat(0.5500)
	pnl := tracker.CalculateUnrealizedPnL(position, currentPrice)

	expected := decimal.NewFromFloat(-15.0)
	if !pnl.Equal(expected) {
		t.Errorf("expected PnL %s, got %s", expected, pnl)
	}
}

func TestPnL_NOPositionProfit(t *testing.T) {
	position := &ports.Position{
		Side:       "NO",
		EntryPrice: decimal.NewFromFloat(0.4000),
		Quantity:   decimal.NewFromFloat(100),
	}

	currentPrice := decimal.NewFromFloat(0.5500)
	pnl := tracker.CalculateUnrealizedPnL(position, currentPrice)

	expected := decimal.NewFromFloat(15.0)
	if !pnl.Equal(expected) {
		t.Errorf("expected PnL %s, got %s", expected, pnl)
	}
}

func TestPnL_NOPositionLoss(t *testing.T) {
	position := &ports.Position{
		Side:       "NO",
		EntryPrice: decimal.NewFromFloat(0.5000),
		Quantity:   decimal.NewFromFloat(200),
	}

	currentPrice := decimal.NewFromFloat(0.3000)
	pnl := tracker.CalculateUnrealizedPnL(position, currentPrice)

	expected := decimal.NewFromFloat(-40.0)
	if !pnl.Equal(expected) {
		t.Errorf("expected PnL %s, got %s", expected, pnl)
	}
}

func TestPnL_PriceAtZero(t *testing.T) {
	position := &ports.Position{
		Side:       "YES",
		EntryPrice: decimal.NewFromFloat(0.5000),
		Quantity:   decimal.NewFromFloat(100),
	}

	currentPrice := decimal.NewFromFloat(0.0000)
	pnl := tracker.CalculateUnrealizedPnL(position, currentPrice)

	expected := decimal.NewFromFloat(-50.0)
	if !pnl.Equal(expected) {
		t.Errorf("expected PnL %s, got %s", expected, pnl)
	}
}

func TestPnL_PriceAtOne(t *testing.T) {
	position := &ports.Position{
		Side:       "YES",
		EntryPrice: decimal.NewFromFloat(0.5000),
		Quantity:   decimal.NewFromFloat(100),
	}

	currentPrice := decimal.NewFromFloat(1.0000)
	pnl := tracker.CalculateUnrealizedPnL(position, currentPrice)

	expected := decimal.NewFromFloat(50.0)
	if !pnl.Equal(expected) {
		t.Errorf("expected PnL %s, got %s", expected, pnl)
	}
}

func TestPnL_ZeroQuantity(t *testing.T) {
	position := &ports.Position{
		Side:       "YES",
		EntryPrice: decimal.NewFromFloat(0.5000),
		Quantity:   decimal.Zero,
	}

	currentPrice := decimal.NewFromFloat(0.8000)
	pnl := tracker.CalculateUnrealizedPnL(position, currentPrice)

	if !pnl.IsZero() {
		t.Errorf("expected zero PnL, got %s", pnl)
	}
}

func TestPnL_DecimalPrecision(t *testing.T) {
	position := &ports.Position{
		Side:       "YES",
		EntryPrice: decimal.NewFromFloat(0.6543),
		Quantity:   decimal.NewFromFloat(123.45678901),
	}

	currentPrice := decimal.NewFromFloat(0.7890)
	pnl := tracker.CalculateUnrealizedPnL(position, currentPrice)

	expected := currentPrice.Sub(position.EntryPrice).Mul(position.Quantity)
	if !pnl.Equal(expected) {
		t.Errorf("expected PnL %s, got %s", expected, pnl)
	}
}

func TestPnL_RealizedPnL(t *testing.T) {
	entryPrice := decimal.NewFromFloat(0.6000)
	exitPrice := decimal.NewFromFloat(0.8000)
	quantity := decimal.NewFromFloat(100)

	pnl := tracker.CalculateRealizedPnL(entryPrice, exitPrice, quantity)

	expected := decimal.NewFromFloat(20.0)
	if !pnl.Equal(expected) {
		t.Errorf("expected realized PnL %s, got %s", expected, pnl)
	}
}

func TestPnL_RealizedPnLLoss(t *testing.T) {
	entryPrice := decimal.NewFromFloat(0.7000)
	exitPrice := decimal.NewFromFloat(0.5000)
	quantity := decimal.NewFromFloat(100)

	pnl := tracker.CalculateRealizedPnL(entryPrice, exitPrice, quantity)

	expected := decimal.NewFromFloat(-20.0)
	if !pnl.Equal(expected) {
		t.Errorf("expected realized PnL %s, got %s", expected, pnl)
	}
}

func TestPnL_UpdatePnL(t *testing.T) {
	position := &ports.Position{
		Side:         "YES",
		EntryPrice:   decimal.NewFromFloat(0.6000),
		CurrentPrice: decimal.NewFromFloat(0.6000),
		Quantity:     decimal.NewFromFloat(100),
		UnrealizedPnL: decimal.Zero,
	}

	currentPrice := decimal.NewFromFloat(0.7500)
	tracker.UpdatePnL(position, currentPrice)

	if !position.CurrentPrice.Equal(currentPrice) {
		t.Errorf("expected current price %s, got %s", currentPrice, position.CurrentPrice)
	}

	expectedPnL := decimal.NewFromFloat(15.0)
	if !position.UnrealizedPnL.Equal(expectedPnL) {
		t.Errorf("expected unrealized PnL %s, got %s", expectedPnL, position.UnrealizedPnL)
	}
}
