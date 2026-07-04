package pitboss

import (
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
)

type Checker struct {
	pitboss *PitBoss
}

func NewChecker(pb *PitBoss) *Checker {
	return &Checker{pitboss: pb}
}

func (c *Checker) Check(req ports.RiskCheckRequest) ports.RiskCheckResponse {
	decision := c.pitboss.Evaluate(req)
	return ports.RiskCheckResponse{
		Decision: decision.Decision,
		Reason:   decision.Reason,
	}
}

func (c *Checker) CheckFromState(state *ports.PitBossState, req ports.RiskCheckRequest) ports.RiskCheckResponse {
	if state.EmergencyStop {
		return ports.RiskCheckResponse{
			Decision: "DENY",
			Reason:   "emergency_stop",
		}
	}

	if state.DailyBudgetRemaining.LessThanOrEqual(decimal.Zero) {
		return ports.RiskCheckResponse{
			Decision: "DENY",
			Reason:   "daily_limit",
		}
	}

	if marketEntry, ok := state.MarketLimits[req.MarketID]; ok {
		if marketEntry.Exposure.Add(req.TradeSize).GreaterThan(marketEntry.Limit) {
			return ports.RiskCheckResponse{
				Decision: "DENY",
				Reason:   "market_limit",
			}
		}
	}

	if strategyEntry, ok := state.StrategyLimits[req.StrategyID]; ok {
		if strategyEntry.Exposure.Add(req.TradeSize).GreaterThan(strategyEntry.Limit) {
			return ports.RiskCheckResponse{
				Decision: "DENY",
				Reason:   "strategy_limit",
			}
		}
	}

	return ports.RiskCheckResponse{
		Decision: "ALLOW",
		Reason:   "approved",
	}
}
