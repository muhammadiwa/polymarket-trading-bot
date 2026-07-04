package risk

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/pqap/services/risk-manager/metrics"
	"go.uber.org/zap"
)

// curatedKeywordMap groups common abbreviations and synonyms to canonical keywords.
// #25: Naive slug splitting doesn't group btc-100k with btc-eth.
var curatedKeywordMap = map[string]string{
	"btc":    "btc",
	"bitcoin": "btc",
	"eth":    "eth",
	"ethereum": "eth",
	"sol":    "sol",
	"solana":  "sol",
	"bnb":    "bnb",
	"xrp":    "xrp",
	"ripple":  "xrp",
	"ada":    "ada",
	"cardano": "ada",
	"doge":   "doge",
	"dogecoin": "doge",
	"avax":   "avax",
	"avalanche": "avax",
	"matic":  "matic",
	"polygon": "matic",
	"pol":    "matic",
	"dot":    "dot",
	"polkadot": "dot",
	"link":   "link",
	"chainlink": "link",
	"trump":  "trump",
	"biden":  "biden",
	"election": "election",
	"president": "election",
}

type CorrelationGroup struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	DetectionMethod string    `json:"detection_method"`
	MarketIDs       []string  `json:"market_ids"`
	MaxPositions    int       `json:"max_positions"`
	Confidence      float64   `json:"confidence"`
	LastUpdated     time.Time `json:"last_updated"`
}

// #14: Lock ordering documentation:
// CorrelationEngine.mu must be acquired before CorrelationTracker.mu when both are needed.
// Acquiring CorrelationTracker.mu first followed by CorrelationEngine.mu will deadlock.
// Current code: CorrelationTracker.CheckMarket holds ct.mu.RLock then calls ct.engine.GetGroupsForMarket
// which acquires ce.mu.RLock. This is safe since both are RLocks. If upgrading to Lock, break the chain.

type CorrelationEngine struct {
	mu                sync.RWMutex
	priceHistory      map[string][]float64
	correlationMatrix map[string]map[string]float64
	categoryGroups    map[string][]string
	keywordGroups     map[string][]string
	groups            map[string]*CorrelationGroup
	threshold         float64
	updateInterval    time.Duration
	maxPositions      int
	publisher         ports.EventPublisher
	repo              ports.CorrelationGroupRepository
	logger            *zap.Logger
	cancel            context.CancelFunc
}

func NewCorrelationEngine(threshold float64, updateInterval time.Duration, maxPositions int, publisher ports.EventPublisher, repo ports.CorrelationGroupRepository, logger *zap.Logger) *CorrelationEngine {
	return &CorrelationEngine{
		priceHistory:      make(map[string][]float64),
		correlationMatrix: make(map[string]map[string]float64),
		categoryGroups:    make(map[string][]string),
		keywordGroups:     make(map[string][]string),
		groups:            make(map[string]*CorrelationGroup),
		threshold:         threshold,
		updateInterval:    updateInterval,
		maxPositions:      maxPositions,
		publisher:         publisher,
		repo:              repo,
		logger:            logger,
	}
}

// #26: Start begins a background ticker that periodically calls UpdateGroups.
func (ce *CorrelationEngine) Start(ctx context.Context) {
	ce.mu.Lock()
	ctx, ce.cancel = context.WithCancel(ctx)
	ce.mu.Unlock()
	go ce.updateLoop(ctx)
}

func (ce *CorrelationEngine) Stop() {
	ce.mu.Lock()
	cancel := ce.cancel
	ce.cancel = nil
	ce.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (ce *CorrelationEngine) updateLoop(ctx context.Context) {
	ticker := time.NewTicker(ce.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ce.UpdateGroups()
		}
	}
}

func (ce *CorrelationEngine) UpdatePriceHistory(marketID string, price float64) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	history := ce.priceHistory[marketID]
	history = append(history, price)
	if len(history) > 288 {
		history = history[len(history)-288:]
	}
	ce.priceHistory[marketID] = history
}

func (ce *CorrelationEngine) SetCategoryGroup(category string, marketIDs []string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	ce.categoryGroups[category] = marketIDs
}

// #22: Clear stale groups before rebuilding to avoid stale entries persisting.
func (ce *CorrelationEngine) UpdateGroups() {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Clear only auto-detected groups (category, keyword, correlation) before rebuild
	for id, g := range ce.groups {
		if g.DetectionMethod != "manual" {
			delete(ce.groups, id)
		}
	}

	ce.detectCategoryGroups()
	ce.detectKeywordGroups()
	ce.detectCorrelationGroups()

	// #8: Persist groups to repository
	if ce.repo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, g := range ce.groups {
			data := ports.CorrelationGroupData{
				ID:              g.ID,
				Name:            g.Name,
				DetectionMethod: g.DetectionMethod,
				MarketIDs:       g.MarketIDs,
				MaxPositions:    g.MaxPositions,
				Confidence:      g.Confidence,
				LastUpdated:     g.LastUpdated,
			}
			if err := ce.repo.UpsertCorrelationGroup(ctx, data); err != nil {
				ce.logger.Error("failed to persist correlation group", zap.String("id", g.ID), zap.Error(err))
			}
		}
	}
}

func (ce *CorrelationEngine) RestoreGroups() {
	if ce.repo == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	groups, err := ce.repo.GetCorrelationGroups(ctx)
	if err != nil {
		ce.logger.Error("failed to restore correlation groups", zap.Error(err))
		return
	}

	ce.mu.Lock()
	defer ce.mu.Unlock()
	for _, g := range groups {
		g := g
		ce.groups[g.ID] = &CorrelationGroup{
			ID:              g.ID,
			Name:            g.Name,
			DetectionMethod: g.DetectionMethod,
			MarketIDs:       g.MarketIDs,
			MaxPositions:    g.MaxPositions,
			Confidence:      g.Confidence,
			LastUpdated:     g.LastUpdated,
		}
	}
	ce.logger.Info("restored correlation groups from database", zap.Int("count", len(groups)))
}

func (ce *CorrelationEngine) detectCategoryGroups() {
	for category, marketIDs := range ce.categoryGroups {
		if len(marketIDs) < 2 {
			continue
		}
		groupID := "cat_" + sanitizeID(category)
		ce.groups[groupID] = &CorrelationGroup{
			ID:              groupID,
			Name:            "Category: " + category,
			DetectionMethod: "category",
			MarketIDs:       marketIDs,
			MaxPositions:    ce.maxPositions,
			Confidence:      0.9,
			LastUpdated:     time.Now().UTC(),
		}
	}
}

func (ce *CorrelationEngine) detectKeywordGroups() {
	keywordMarkets := make(map[string][]string)
	for keyword, marketIDs := range ce.keywordGroups {
		if len(marketIDs) < 2 {
			continue
		}
		keywordMarkets[keyword] = marketIDs
	}

	for keyword, marketIDs := range keywordMarkets {
		groupID := "kw_" + sanitizeID(keyword)
		ce.groups[groupID] = &CorrelationGroup{
			ID:              groupID,
			Name:            "Keyword: " + keyword,
			DetectionMethod: "keyword",
			MarketIDs:       marketIDs,
			MaxPositions:    ce.maxPositions,
			Confidence:      0.7,
			LastUpdated:     time.Now().UTC(),
		}
	}
}

func (ce *CorrelationEngine) detectCorrelationGroups() {
	ce.computeCorrelationMatrix()

	visited := make(map[string]bool)
	for marketA := range ce.correlationMatrix {
		if visited[marketA] {
			continue
		}
		group := []string{marketA}
		visited[marketA] = true

		for marketB, corr := range ce.correlationMatrix[marketA] {
			if visited[marketB] {
				continue
			}
			if corr >= ce.threshold {
				group = append(group, marketB)
				visited[marketB] = true
			}
		}

		if len(group) >= 2 {
			sort.Strings(group)
			groupID := "corr_" + sanitizeID(strings.Join(group, "_"))
			ce.groups[groupID] = &CorrelationGroup{
				ID:              groupID,
				Name:            "Correlation: " + strings.Join(group, ", "),
				DetectionMethod: "correlation",
				MarketIDs:       group,
				MaxPositions:    ce.maxPositions,
				Confidence:      0.8,
				LastUpdated:     time.Now().UTC(),
			}
		}
	}
}

func (ce *CorrelationEngine) computeCorrelationMatrix() {
	ce.correlationMatrix = make(map[string]map[string]float64)
	markets := make([]string, 0, len(ce.priceHistory))
	for m := range ce.priceHistory {
		markets = append(markets, m)
	}
	sort.Strings(markets)

	for i := 0; i < len(markets); i++ {
		ce.correlationMatrix[markets[i]] = make(map[string]float64)
		for j := i + 1; j < len(markets); j++ {
			corr := pearsonCorrelation(ce.priceHistory[markets[i]], ce.priceHistory[markets[j]])
			ce.correlationMatrix[markets[i]][markets[j]] = corr
		}
	}
}

func (ce *CorrelationEngine) GetGroups() []CorrelationGroup {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	groups := make([]CorrelationGroup, 0, len(ce.groups))
	for _, g := range ce.groups {
		groups = append(groups, *g)
	}
	return groups
}

func (ce *CorrelationEngine) GetGroupsForMarket(marketID string) []CorrelationGroup {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	var result []CorrelationGroup
	for _, g := range ce.groups {
		for _, id := range g.MarketIDs {
			if id == marketID {
				result = append(result, *g)
				break
			}
		}
	}
	return result
}

// #13: Deduplicate keyword mappings to prevent unbounded growth.
func (ce *CorrelationEngine) AddKeywordMapping(keyword string, marketID string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	existing := ce.keywordGroups[keyword]
	for _, id := range existing {
		if id == marketID {
			return
		}
	}
	ce.keywordGroups[keyword] = append(existing, marketID)
}

// #25: Add curated keyword map and improved extraction.
func (ce *CorrelationEngine) ExtractKeywordsFromSlug(slug string) []string {
	parts := strings.Split(slug, "-")
	keywords := make([]string, 0)
	for _, p := range parts {
		lower := strings.ToLower(p)
		if len(lower) >= 3 {
			if canonical, ok := curatedKeywordMap[lower]; ok {
				lower = canonical
			}
			keywords = append(keywords, lower)
		}
	}
	return keywords
}

func pearsonCorrelation(x, y []float64) float64 {
	n := len(x)
	if n != len(y) || n < 2 {
		return 0
	}

	var sumX, sumY, sumXY, sumX2, sumY2 float64
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	denom := math.Sqrt((float64(n)*sumX2 - sumX*sumX) * (float64(n)*sumY2 - sumY*sumY))
	if denom == 0 {
		return 0
	}
	return (float64(n)*sumXY - sumX*sumY) / denom
}

func sanitizeID(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ToLower(s)
	// #18: Use SHA-256 hash prefix instead of truncation to avoid collisions
	hash := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", hash[:8])
}

type CorrelationTracker struct {
	mu               sync.RWMutex
	engine           *CorrelationEngine
	openPositions    map[string]bool
	maxPositions     int
	rejectionCounter uint64
	publisher        ports.EventPublisher
	logger           *zap.Logger
}

func NewCorrelationTracker(engine *CorrelationEngine, maxPositions int, publisher ports.EventPublisher, logger *zap.Logger) *CorrelationTracker {
	return &CorrelationTracker{
		engine:        engine,
		openPositions: make(map[string]bool),
		maxPositions:  maxPositions,
		publisher:     publisher,
		logger:        logger,
	}
}

func (ct *CorrelationTracker) SetOpenPositions(positions []string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.openPositions = make(map[string]bool, len(positions))
	for _, p := range positions {
		ct.openPositions[p] = true
	}
}

func (ct *CorrelationTracker) AddOpenPosition(marketID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.openPositions[marketID] = true
}

func (ct *CorrelationTracker) RemoveOpenPosition(marketID string) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	delete(ct.openPositions, marketID)
}

type CorrelationCheckResult struct {
	Allowed        bool
	Reason         string
	CorrelatedWith []string
	GroupName      string
}

// #14: CheckMarket holds ct.mu.RLock then calls ct.engine.GetGroupsForMarket which acquires ce.mu.RLock.
// Both are read locks so no deadlock risk. Documented for future refactoring safety.
func (ct *CorrelationTracker) CheckMarket(marketID string) CorrelationCheckResult {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	groups := ct.engine.GetGroupsForMarket(marketID)
	for _, group := range groups {
		correlated := make([]string, 0)
		for _, id := range group.MarketIDs {
			if id != marketID && ct.openPositions[id] {
				correlated = append(correlated, id)
			}
		}
		if len(correlated) >= ct.maxPositions {
			ct.logger.Warn("correlation limit exceeded",
				zap.String("market_id", marketID),
				zap.String("group", group.Name),
				zap.Strings("correlated_with", correlated),
			)
			result := CorrelationCheckResult{
				Allowed:        false,
				Reason:         "correlation_limit_exceeded",
				CorrelatedWith: correlated,
				GroupName:      group.Name,
			}
			// #7: Publish CorrelationRejection to NATS
			ct.publishRejection(marketID, result)
			return result
		}
	}

	return CorrelationCheckResult{Allowed: true, Reason: "approved"}
}

func (ct *CorrelationTracker) publishRejection(marketID string, result CorrelationCheckResult) {
	if ct.publisher == nil {
		return
	}
	metrics.CorrelationRejectionsTotal.Inc()

	event := ports.CorrelationRejectionEvent{
		EventID:   uuid.New().String(),
		EventType: "CorrelationRejection",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.CorrelationRejectionPayload{
			MarketID:       marketID,
			CorrelatedWith: result.CorrelatedWith,
			Reason:         result.Reason,
			GroupName:      result.GroupName,
		},
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ct.publisher.PublishCorrelationRejection(ctx, event); err != nil {
			ct.logger.Error("failed to publish correlation rejection", zap.Error(err))
		}
	}()
}

func (ct *CorrelationTracker) BuildCorrelationRejection(marketID string, result CorrelationCheckResult) ports.CorrelationRejectionEvent {
	return ports.CorrelationRejectionEvent{
		EventID:   uuid.New().String(),
		EventType: "CorrelationRejection",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.CorrelationRejectionPayload{
			MarketID:       marketID,
			CorrelatedWith: result.CorrelatedWith,
			Reason:         result.Reason,
			GroupName:      result.GroupName,
		},
	}
}

func (ct *CorrelationTracker) GetCorrelationState() map[string]CorrelationState {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	states := make(map[string]CorrelationState)
	for _, group := range ct.engine.GetGroups() {
		openCount := 0
		for _, id := range group.MarketIDs {
			if ct.openPositions[id] {
				openCount++
			}
		}
		states[group.ID] = CorrelationState{
			MarketGroup:   group.Name,
			MarketIDs:     group.MarketIDs,
			OpenPositions: openCount,
			MaxPositions:  group.MaxPositions,
			IsExceeded:    openCount >= group.MaxPositions,
		}
	}
	return states
}

type CorrelationState struct {
	MarketGroup   string   `json:"market_group"`
	MarketIDs     []string `json:"market_ids"`
	OpenPositions int      `json:"open_positions"`
	MaxPositions  int      `json:"max_positions"`
	IsExceeded    bool     `json:"is_exceeded"`
}
