package adapters

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pqap/services/position-manager/internal/ports"
	"go.uber.org/zap"
)

type PolymarketAccount struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewPolymarketAccount(baseURL string, logger *zap.Logger) *PolymarketAccount {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &PolymarketAccount{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

func (p *PolymarketAccount) GetPositions() ([]*ports.APIPosition, error) {
	reqURL := fmt.Sprintf("%s/positions", p.baseURL)

	resp, err := p.httpClient.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch positions from Polymarket API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("polymarket API returned status %d: %s", resp.StatusCode, string(body))
	}

	var apiPositions []*ports.APIPosition
	if err := json.NewDecoder(resp.Body).Decode(&apiPositions); err != nil {
		return nil, fmt.Errorf("failed to decode positions response: %w", err)
	}

	return apiPositions, nil
}

func (p *PolymarketAccount) GetMarketResolution(marketID string) (resolved bool, outcome string, err error) {
	escapedMarketID := url.PathEscape(marketID)
	reqURL := fmt.Sprintf("%s/markets/%s", p.baseURL, escapedMarketID)

	resp, err := p.httpClient.Get(reqURL)
	if err != nil {
		return false, "", fmt.Errorf("failed to fetch market from Polymarket API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, "", fmt.Errorf("polymarket API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Resolved bool   `json:"resolved"`
		Outcome  string `json:"outcome"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "", fmt.Errorf("failed to decode market response: %w", err)
	}

	return result.Resolved, result.Outcome, nil
}
