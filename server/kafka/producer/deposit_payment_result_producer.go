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

const DepositPaymentResultTopic = "deposit.transaction.payment_result"

// DepositPaymentResultEvent is published after a fiat deposit payment status transition.
type DepositPaymentResultEvent struct {
	UserID        int64  `json:"user_id"`
	TransactionID int64  `json:"transaction_id"`
	Status        string `json:"status"`
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	EventTime     int64  `json:"event_time"`
}

type DepositPaymentResultProducer interface {
	PublishDepositPaymentResult(ctx context.Context, event DepositPaymentResultEvent) error
}

type depositPaymentResultProducerImpl struct {
	producer KafkaProducer
	topic    string
}

var (
	depositPaymentResultProducerOnce sync.Once
	depositPaymentResultProducerInst DepositPaymentResultProducer
)

func GetDepositPaymentResultProducer() DepositPaymentResultProducer {
	depositPaymentResultProducerOnce.Do(func() {
		depositPaymentResultProducerInst = &depositPaymentResultProducerImpl{
			producer: GetKafkaProducer(),
			topic:    DepositPaymentResultTopic,
		}
	})
	return depositPaymentResultProducerInst
}

func (p *depositPaymentResultProducerImpl) PublishDepositPaymentResult(
	ctx context.Context,
	event DepositPaymentResultEvent,
) error {
	if event.UserID <= 0 {
		return errors.New("user_id must be positive")
	}
	if event.TransactionID <= 0 {
		return errors.New("transaction_id must be positive")
	}
	if event.EventTime == 0 {
		event.EventTime = time.Now().Unix()
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal deposit payment result event: %w", err)
	}
	return p.producer.Publish(ctx, p.topic, []byte(strconv.FormatInt(event.UserID, 10)), payload)
}
