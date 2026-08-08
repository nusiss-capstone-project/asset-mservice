package listener

import (
	"context"
	"encoding/json"

	"github.com/nusiss-capstone-project/asset-mservice/server/kafka"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/service"
)

const (
	PaymentTransactionUpdatedTopic = "payment.transaction.updated"

	PaymentSucceededEventType = "payment-succeeded"
	PaymentFailedEventType    = "payment-failed"
)

type PaymentTransactionUpdatedEvent struct {
	UserID    int64  `json:"user_id"`
	PaymentID string `json:"payment_id"`
	BizID     string `json:"biz_id"`
	EventType string `json:"event_type"`
	EventTime int64  `json:"event_time"`
	Provider  string `json:"provider"`
	Status    string `json:"status"`
}

func init() {
	kafka.RegisterHandler(PaymentTransactionUpdatedTopic, handlePaymentTransactionUpdated)
}

func handlePaymentTransactionUpdated(ctx context.Context, msg *kafka.Message) error {
	var event PaymentTransactionUpdatedEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		log.WithContext(ctx).Errorw("unmarshal payment.transaction.updated failed", "error", err)
		return err
	}
	log.WithContext(ctx).Infow("received payment.transaction.updated",
		"biz_id", event.BizID,
		"payment_id", event.PaymentID,
		"event_type", event.EventType,
		"status", event.Status,
		"user_id", event.UserID,
	)
	return service.GetOrderService().HandlePaymentResult(ctx, service.PaymentResultInput{
		BizID:     event.BizID,
		PaymentID: event.PaymentID,
		EventType: event.EventType,
		Status:    event.Status,
	})
}
