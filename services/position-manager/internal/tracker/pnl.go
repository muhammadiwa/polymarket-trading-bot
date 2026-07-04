package tracker

import (
	"github.com/pqap/services/position-manager/internal/ports"
	"github.com/shopspring/decimal"
)

func CalculateUnrealizedPnL(position *ports.Position, currentPrice decimal.Decimal) decimal.Decimal {
	return currentPrice.Sub(position.EntryPrice).Mul(position.Quantity)
}

func CalculateRealizedPnL(entryPrice, exitPrice, quantity decimal.Decimal) decimal.Decimal {
	return exitPrice.Sub(entryPrice).Mul(quantity)
}

func UpdatePnL(position *ports.Position, currentPrice decimal.Decimal) {
	position.CurrentPrice = currentPrice
	position.UnrealizedPnL = CalculateUnrealizedPnL(position, currentPrice)
}
