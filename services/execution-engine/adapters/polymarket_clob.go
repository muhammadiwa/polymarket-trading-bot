package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const maxResponseBodyBytes = 1 * 1024 * 1024 // 1MB

type PolymarketCLOB struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewPolymarketCLOB(baseURL string, logger *zap.Logger) *PolymarketCLOB {
	return &PolymarketCLOB{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

type placeOrderRequest struct {
	MarketID      string `json:"market_id"`
	Side          string `json:"side"`
	Price         string `json:"price"`
	Size          string `json:"size"`
	ClientOrderID string `json:"client_order_id"`
	TimeInForce   string `json:"time_in_force"`
}

type placeOrderResponse struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

type orderStatusResponse struct {
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
	FilledSize    string `json:"filled_size"`
	RemainingSize string `json:"remaining_size"`
	Price         string `json:"price"`
}

func (c *PolymarketCLOB) PlaceOrder(req ports.OrderRequest, clientOrderID string) (*ports.OrderResponse, error) {
	body := placeOrderRequest{
		MarketID:      req.MarketID,
		Side:          req.Side,
		Price:         req.Price.String(),
		Size:          req.Size.String(),
		ClientOrderID: clientOrderID,
		TimeInForce:   req.TimeInForce,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/order", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
		return nil, fmt.Errorf("CLOB API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result placeOrderResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBodyBytes)).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode order response: %w", err)
	}

	return &ports.OrderResponse{
		OrderID:       result.OrderID,
		ClientOrderID: clientOrderID,
		Status:        result.Status,
	}, nil
}

func (c *PolymarketCLOB) CancelOrder(ctx context.Context, orderID string) error {
	escapedOrderID := url.PathEscape(orderID)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", c.baseURL+"/order/"+escapedOrderID, nil)
	if err != nil {
		return fmt.Errorf("failed to create cancel request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
		return fmt.Errorf("CLOB API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (c *PolymarketCLOB) GetOrderStatus(orderID string) (*ports.OrderStatusResponse, error) {
	escapedOrderID := url.PathEscape(orderID)
	httpReq, err := http.NewRequest("GET", c.baseURL+"/order/"+escapedOrderID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create status request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
		return nil, fmt.Errorf("CLOB API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result orderStatusResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBodyBytes)).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	filledQty, err := decimal.NewFromString(result.FilledSize)
	if err != nil {
		return nil, fmt.Errorf("invalid filled_size %q: %w", result.FilledSize, err)
	}
	remainingQty, err := decimal.NewFromString(result.RemainingSize)
	if err != nil {
		return nil, fmt.Errorf("invalid remaining_size %q: %w", result.RemainingSize, err)
	}
	price, err := decimal.NewFromString(result.Price)
	if err != nil {
		return nil, fmt.Errorf("invalid price %q: %w", result.Price, err)
	}

	return &ports.OrderStatusResponse{
		OrderID:      result.OrderID,
		Status:       result.Status,
		FilledQty:    filledQty,
		RemainingQty: remainingQty,
		Price:        price,
	}, nil
}
