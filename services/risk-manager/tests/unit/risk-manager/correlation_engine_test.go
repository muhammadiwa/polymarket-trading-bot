package riskmanager

import (
	"testing"
	"time"

	"github.com/pqap/services/risk-manager/internal/risk"
	"go.uber.org/zap"
)

func newTestCorrelationEngine() *risk.CorrelationEngine {
	logger, _ := zap.NewDevelopment()
	return risk.NewCorrelationEngine(0.7, 1*time.Hour, 3, nil, nil, logger)
}

func TestCorrelationEngine_CategoryDetection(t *testing.T) {
	engine := newTestCorrelationEngine()

	engine.SetCategoryGroup("election-2026", []string{"market_1", "market_2", "market_3"})
	engine.UpdateGroups()

	groups := engine.GetGroups()
	if len(groups) == 0 {
		t.Fatal("expected at least one group from category detection")
	}

	found := false
	for _, g := range groups {
		if g.DetectionMethod == "category" && len(g.MarketIDs) == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected category group with 3 markets")
	}
}

func TestCorrelationEngine_KeywordDetection(t *testing.T) {
	engine := newTestCorrelationEngine()

	engine.AddKeywordMapping("btc", "btc-100k")
	engine.AddKeywordMapping("btc", "btc-ath")
	engine.AddKeywordMapping("btc", "btc-etf")
	engine.UpdateGroups()

	groups := engine.GetGroups()
	found := false
	for _, g := range groups {
		if g.DetectionMethod == "keyword" && len(g.MarketIDs) == 3 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected keyword group with 3 markets")
	}
}

func TestCorrelationEngine_PriceCorrelationDetection(t *testing.T) {
	engine := newTestCorrelationEngine()

	for i := 0; i < 50; i++ {
		engine.UpdatePriceHistory("market_a", float64(i))
		engine.UpdatePriceHistory("market_b", float64(i)+0.1)
		engine.UpdatePriceHistory("market_c", float64(100-i))
	}

	engine.UpdateGroups()

	groups := engine.GetGroups()
	foundCorrelated := false
	for _, g := range groups {
		if g.DetectionMethod == "correlation" {
			foundCorrelated = true
			hasA := false
			hasB := false
			for _, id := range g.MarketIDs {
				if id == "market_a" {
					hasA = true
				}
				if id == "market_b" {
					hasB = true
				}
			}
			if hasA && hasB {
				break
			}
		}
	}
	if !foundCorrelated {
		t.Error("expected correlation group between market_a and market_b")
	}
}

func TestCorrelationEngine_GetGroupsForMarket(t *testing.T) {
	engine := newTestCorrelationEngine()

	engine.SetCategoryGroup("election", []string{"market_1", "market_2"})
	engine.UpdateGroups()

	groups := engine.GetGroupsForMarket("market_1")
	if len(groups) == 0 {
		t.Error("expected groups for market_1")
	}

	groups = engine.GetGroupsForMarket("market_unknown")
	if len(groups) != 0 {
		t.Error("expected no groups for unknown market")
	}
}

func TestCorrelationEngine_ExtractKeywordsFromSlug(t *testing.T) {
	engine := newTestCorrelationEngine()

	keywords := engine.ExtractKeywordsFromSlug("btc-100k-by-december")
	expected := []string{"btc", "100k", "december"}
	if len(keywords) != len(expected) {
		t.Errorf("expected %d keywords, got %d", len(expected), len(keywords))
	}
	for i, kw := range keywords {
		if kw != expected[i] {
			t.Errorf("expected keyword[%d] = %q, got %q", i, expected[i], kw)
		}
	}
}

func TestCorrelationEngine_MinimumGroupSize(t *testing.T) {
	engine := newTestCorrelationEngine()

	engine.SetCategoryGroup("single", []string{"market_1"})
	engine.UpdateGroups()

	groups := engine.GetGroups()
	for _, g := range groups {
		if g.DetectionMethod == "category" && len(g.MarketIDs) < 2 {
			t.Error("should not create group with less than 2 markets")
		}
	}
}
