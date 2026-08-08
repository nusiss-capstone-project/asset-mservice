package producer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

const OrderPaymentResultTopic = "asset.order.payment_result"

// OrderPaymentResultEvent is published after an order payment status transition.
type OrderPaymentResultEvent struct {
	UserID        int64  `json:"user_id"`
	OrderID       int64  `json:"order_id"`
	OrderNo       string `json:"order_no"`
	PaymentID     string `json:"payment_id"`
	AssetID       int64  `json:"asset_id"`
	AssetSymbol   string `json:"asset_symbol"`
	Quantity      string `json:"quantity"`
	UnitPrice     string `json:"unit_price"`
	PaymentAmount string `json:"payment_amount"`
	PayCurrency   string `json:"pay_currency"`
	Status        string `json:"status"`
	EventTime     int64  `json:"event_time"`
}

type OrderPaymentResultProducer interface {
	PublishOrderPaymentResult(ctx context.Context, event OrderPaymentResultEvent) error
}

type orderPaymentResultProducerImpl struct {
	producer KafkaProducer
	topic    string
}

var (
	orderPaymentResultProducerOnce sync.Once
	orderPaymentResultProducerInst OrderPaymentResultProducer
)

func GetOrderPaymentResultProducer() OrderPaymentResultProducer {
	orderPaymentResultProducerOnce.Do(func() {
		orderPaymentResultProducerInst = &orderPaymentResultProducerImpl{
			producer: GetKafkaProducer(),
			topic:    OrderPaymentResultTopic,
		}
	})
	return orderPaymentResultProducerInst
}

func (p *orderPaymentResultProducerImpl) PublishOrderPaymentResult(
	ctx context.Context,
	event OrderPaymentResultEvent,
) error {
	if event.UserID <= 0 {
		return errors.New("user_id must be positive")
	}
	if event.OrderNo == "" {
		return errors.New("order_no is required")
	}
	if event.EventTime == 0 {
		event.EventTime = time.Now().Unix()
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal order payment result event: %w", err)
	}
	return p.producer.Publish(ctx, p.topic, []byte(strconv.FormatInt(event.UserID, 10)), payload)
}
