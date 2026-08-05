package producer

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	kafkatrace "github.com/nusiss-capstone-project/asset-mservice/server/kafka/trace"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

// KafkaProducer publishes raw Kafka messages.
type KafkaProducer interface {
	Publish(ctx context.Context, topic string, key, value []byte) error
}

type kafkaProducerImpl struct {
	client *kgo.Client
}

type nopKafkaProducer struct{}

func (nopKafkaProducer) Publish(context.Context, string, []byte, []byte) error {
	return nil
}

var (
	producerOnce sync.Once
	producerInst KafkaProducer
)

// Ensure initializes the singleton producer if it has not been created yet.
func Ensure() {
	GetKafkaProducer()
}

// GetKafkaProducer returns the singleton Kafka producer.
// When Kafka is disabled or misconfigured, a no-op implementation is returned.
func GetKafkaProducer() KafkaProducer {
	producerOnce.Do(func() {
		producerInst = buildProducer(config.Config.KafkaConfig)
	})
	return producerInst
}

func buildProducer(cfg *config.KafkaConfig) KafkaProducer {
	if cfg == nil || !cfg.Enabled {
		return nopKafkaProducer{}
	}
	if err := validateConfig(cfg); err != nil {
		log.Logger.Errorw("kafka producer config invalid, using no-op producer", "error", err)
		return nopKafkaProducer{}
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
	}
	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		log.Logger.Errorw("failed to create kafka producer, using no-op producer", "error", err)
		return nopKafkaProducer{}
	}

	log.Logger.Infow("kafka producer initialized", "brokers", cfg.Brokers)
	return &kafkaProducerImpl{client: client}
}

func (p *kafkaProducerImpl) Publish(ctx context.Context, topic string, key, value []byte) error {
	if topic == "" {
		return errors.New("topic is empty")
	}

	start := time.Now()
	ctx, span := kafkatrace.StartProduce(ctx, topic)
	var produceErr error
	defer func() {
		kafkatrace.Finish(span, produceErr)
	}()

	kafkatrace.LogProduceStart(ctx, "topic", topic)
	defer func() {
		kafkatrace.LogProduceFinish(ctx, float64(time.Since(start).Microseconds())/1000, produceErr, "topic", topic)
	}()

	record := &kgo.Record{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: kafkatrace.HeadersFromContext(ctx),
	}
	if produceErr = p.client.ProduceSync(ctx, record).FirstErr(); produceErr != nil {
		log.WithContext(ctx).Errorw("kafka message publish failed",
			"topic", topic,
			"error", produceErr,
		)
		return produceErr
	}
	return nil
}

func validateConfig(cfg *config.KafkaConfig) error {
	if cfg == nil {
		return errors.New("kafka config is nil")
	}
	if len(cfg.Brokers) == 0 {
		return errors.New("kafka brokers is empty")
	}
	return nil
}
