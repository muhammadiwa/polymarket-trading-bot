package history

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/execution-engine/internal/ports"
	"github.com/pqap/services/execution-engine/metrics"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type NATSPublisher interface {
	PublishTradeRecorded(ctx context.Context, event ports.TradeRecorded) error
}

type Handler struct {
	repo      Repository
	publisher NATSPublisher
	logger    *zap.Logger
}

func NewHandler(repo Repository, publisher NATSPublisher, logger *zap.Logger) *Handler {
	return &Handler{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
	}
}

func (h *Handler) HandleOrderResult(ctx context.Context, result *OrderResult) error {
	startTime := time.Now()

	slippagePct := calculateSlippage(result.Price, result.SignalPrice)
	pnl := calculatePnL(result.FillStatus, result.FilledQuantity, result.Price, h.logger)

	record := &TradeRecord{
		ClientOrderID:   result.ClientOrderID,
		StrategyID:      result.StrategyID,
		MarketID:        result.MarketID,
		MarketSlug:      result.MarketSlug,
		Side:            result.Side,
		OrderType:       result.OrderType,
		Price:           result.Price,
		Quantity:        result.Quantity,
		FilledQuantity:  result.FilledQuantity,
		FillStatus:      result.FillStatus,
		LatencyMs:       result.LatencyMs,
		PnL:             pnl,
		Fee:             result.Fee,
		SlippagePct:     slippagePct,
		SignalTimestamp: result.SignalTimestamp,
		OrderTimestamp:  result.OrderTimestamp,
		FillTimestamp:   result.FillTimestamp,
		RiskDecision:    result.RiskDecision,
		FailureReason:   ptrString(result.FailureReason),
		AccountID:       result.AccountID,
	}

	if result.OpportunityID != "" {
		record.OpportunityID = &result.OpportunityID
	}

	if err := h.repo.Insert(ctx, record); err != nil {
		metrics.TradeRecordErrors.Inc()
		h.logger.Error("failed to insert trade record",
			zap.String("client_order_id", result.ClientOrderID),
			zap.Error(err),
		)
		return err
	}

	dbLatency := time.Since(startTime).Milliseconds()
	metrics.TradeRecordsTotal.WithLabelValues(string(result.FillStatus)).Inc()
	metrics.TradeRecordLatency.Observe(float64(dbLatency))

	h.logger.Info("trade recorded",
		zap.String("trade_id", record.ID),
		zap.String("client_order_id", result.ClientOrderID),
		zap.String("fill_status", string(result.FillStatus)),
		zap.Int64("db_latency_ms", dbLatency),
	)

	if h.publisher != nil {
		event := ports.TradeRecorded{
			EventID:   uuid.New().String(),
			EventType: "TradeRecorded",
			Timestamp: time.Now().UTC(),
			Source:    "execution-engine",
			Payload: ports.TradeRecordedPayload{
				TradeID:        record.ID,
				ClientOrderID:  record.ClientOrderID,
				StrategyID:     record.StrategyID,
				MarketID:       record.MarketID,
				Side:           record.Side,
				Price:          record.Price,
				FilledQuantity: record.FilledQuantity,
				FillStatus:     string(record.FillStatus),
				PnL:            record.PnL,
				LatencyMs:      record.LatencyMs,
			},
		}

		if err := h.publisher.PublishTradeRecorded(ctx, event); err != nil {
			h.logger.Error("failed to publish TradeRecorded event",
				zap.String("trade_id", record.ID),
				zap.Error(err),
			)
			return err
		}
	}

	return nil
}

func calculateSlippage(price, signalPrice decimal.Decimal) decimal.Decimal {
	if signalPrice.IsZero() {
		return decimal.Zero
	}
	diff := price.Sub(signalPrice).Abs()
	return diff.Div(signalPrice).Mul(decimal.NewFromInt(100))
}

func calculatePnL(status FillStatus, filledQuantity, price decimal.Decimal, logger *zap.Logger) decimal.Decimal {
	if status != FillStatusFilled && status != FillStatusPartialFill {
		return decimal.Zero
	}
	logger.Debug("PnL calculation deferred: requires entry price tracking for full implementation",
		zap.String("fill_status", string(status)),
		zap.String("filled_quantity", filledQuantity.String()),
		zap.String("price", price.String()),
	)
	return decimal.Zero
}

func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
