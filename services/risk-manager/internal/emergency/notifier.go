package emergency

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pqap/services/risk-manager/internal/ports"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type Notifier struct {
	publisher ports.EventPublisher
	logger    *zap.Logger
}

func NewNotifier(publisher ports.EventPublisher, logger *zap.Logger) *Notifier {
	return &Notifier{
		publisher: publisher,
		logger:    logger,
	}
}

type EmergencyDetails struct {
	Reason          string
	Drawdown        *decimal.Decimal
	DrawdownLimit   *decimal.Decimal
	PeakEquity      *decimal.Decimal
	CurrentEquity   *decimal.Decimal
	DailyPnL        decimal.Decimal
	OpenOrdersCount int
	TriggeredAt     time.Time
}

func (n *Notifier) SendEmergencyAlert(ctx context.Context, details *EmergencyDetails) error {
	message := n.formatEmergencyMessage(details)

	event := ports.NotificationRequest{
		EventID:   uuid.New().String(),
		EventType: "NotificationRequest",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.NotificationRequestPayload{
			Category:       "risk",
			Title:          "EMERGENCY STOP TRIGGERED",
			Message:        message,
			Channel:        "telegram",
			Priority:       "critical",
			BypassThrottle: true,
		},
	}

	if err := n.publisher.PublishNotificationRequest(ctx, event); err != nil {
		n.logger.Error("failed to publish emergency notification", zap.Error(err))
		return err
	}

	n.logger.Info("emergency notification sent",
		zap.String("reason", details.Reason),
		zap.Bool("bypass_throttle", true),
	)
	return nil
}

func (n *Notifier) SendResumeAlert(ctx context.Context, previousReason, resumedBy string) error {
	message := fmt.Sprintf("Trading resumed by %s. Previous emergency reason: %s", resumedBy, previousReason)

	event := ports.NotificationRequest{
		EventID:   uuid.New().String(),
		EventType: "NotificationRequest",
		Timestamp: time.Now().UTC(),
		Source:    "risk-manager",
		Payload: ports.NotificationRequestPayload{
			Category:       "risk",
			Title:          "Trading Resumed",
			Message:        message,
			Channel:        "telegram",
			Priority:       "high",
			BypassThrottle: true,
		},
	}

	if err := n.publisher.PublishNotificationRequest(ctx, event); err != nil {
		n.logger.Error("failed to publish resume notification", zap.Error(err))
		return err
	}

	return nil
}

func (n *Notifier) formatEmergencyMessage(d *EmergencyDetails) string {
	// #23: nil pointer guard
	if d == nil {
		return "EMERGENCY STOP ACTIVATED (no details available)"
	}
	msg := fmt.Sprintf("Reason: %s\nTime: %s UTC", d.Reason, d.TriggeredAt.Format("2006-01-02 15:04:05"))

	if d.Drawdown != nil && d.DrawdownLimit != nil {
		drawdownPct, _ := d.Drawdown.Mul(decimal.NewFromInt(100)).Float64()
		limitPct, _ := d.DrawdownLimit.Mul(decimal.NewFromInt(100)).Float64()
		msg += fmt.Sprintf("\n\nDrawdown: %.2f%% (limit: %.2f%%)", drawdownPct, limitPct)
	}
	if d.PeakEquity != nil {
		msg += fmt.Sprintf("\nPeak Equity: $%s", d.PeakEquity.StringFixed(2))
	}
	if d.CurrentEquity != nil {
		msg += fmt.Sprintf("\nCurrent Equity: $%s", d.CurrentEquity.StringFixed(2))
	}
	msg += fmt.Sprintf("\nDaily PnL: $%s", d.DailyPnL.StringFixed(2))
	msg += fmt.Sprintf("\n\nOpen Orders Cancelled: %d", d.OpenOrdersCount)
	msg += "\n\nManual resume required via dashboard or API."

	return msg
}
