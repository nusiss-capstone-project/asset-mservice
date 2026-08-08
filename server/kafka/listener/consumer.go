package listener

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	"github.com/nusiss-capstone-project/asset-mservice/server/kafka"
	kafkatrace "github.com/nusiss-capstone-project/asset-mservice/server/kafka/trace"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/twmb/franz-go/pkg/kgo"
)

type consumer struct {
	client *kgo.Client
}

// Start launches the Kafka consumer when config is valid.
func Start(ctx context.Context, cfg *config.KafkaConfig) {
	if err := validateConfig(cfg); err != nil {
		log.Logger.Errorw("kafka consumer config invalid", "error", err)
		return
	}

	c, err := newConsumer(cfg)
	if err != nil {
		log.Logger.Errorw("failed to create kafka consumer", "error", err)
		return
	}

	log.Logger.Infow("kafka consumer started",
		"brokers", cfg.Brokers,
		"group_id", cfg.GroupID,
		"topics", cfg.Topics,
		"registered_topics", kafka.RegisteredTopics(),
	)
	go c.run(ctx)
}

func newConsumer(cfg *config.KafkaConfig) (*consumer, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumerGroup(cfg.GroupID),
		kgo.ConsumeTopics(cfg.Topics...),
		kgo.DisableAutoCommit(),
	}
	if cfg.ClientID != "" {
		opts = append(opts, kgo.ClientID(cfg.ClientID))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &consumer{client: client}, nil
}

func (c *consumer) run(ctx context.Context) {
	defer c.client.Close()

	for {
		select {
		case <-ctx.Done():
			log.Logger.Infow("kafka consumer stopped", "reason", ctx.Err())
			return
		default:
		}

		fetches := c.client.PollFetches(ctx)
		if err := ctx.Err(); err != nil {
			log.Logger.Infow("kafka consumer poll stopped", "reason", err)
			return
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, ferr := range errs {
				log.Logger.Errorw("kafka fetch error",
					"topic", ferr.Topic,
					"partition", ferr.Partition,
					"error", ferr.Err,
				)
			}
			continue
		}

		fetches.EachRecord(func(record *kgo.Record) {
			c.handleRecord(ctx, record)
		})
	}
}

func (c *consumer) handleRecord(parentCtx context.Context, record *kgo.Record) {
	start := time.Now()
	topicHandlers := kafka.HandlersForTopic(record.Topic)
	if len(topicHandlers) == 0 {
		log.Logger.Warnw("no handlers registered for topic, skipping commit",
			"topic", record.Topic,
			"partition", record.Partition,
			"offset", record.Offset,
		)
		return
	}

	ctx, span := kafkatrace.StartConsume(parentCtx, record)
	var err error
	defer func() {
		kafkatrace.Finish(span, err)
	}()

	kafkatrace.LogConsumeStart(ctx, record, len(topicHandlers))

	err = invokeHandlersParallel(ctx, topicHandlers, toMessage(record))
	kafkatrace.LogConsumeFinish(ctx, record, float64(time.Since(start).Microseconds())/1000, err)
	if err != nil {
		return
	}

	if commitErr := c.client.CommitRecords(ctx, record); commitErr != nil {
		log.WithContext(ctx).Errorw("kafka manual commit failed",
			"topic", record.Topic,
			"partition", record.Partition,
			"offset", record.Offset,
			"error", commitErr,
		)
		err = commitErr
	}
}

func invokeHandlersParallel(ctx context.Context, handlers []kafka.Handler, msg *kafka.Message) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	wg.Add(len(handlers))
	for _, handler := range handlers {
		go func(h kafka.Handler) {
			defer wg.Done()
			if err := h(ctx, msg); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(handler)
	}
	wg.Wait()
	return joinErrors(errs)
}

func toMessage(record *kgo.Record) *kafka.Message {
	headers := make(map[string]string, len(record.Headers))
	for _, header := range record.Headers {
		headers[string(header.Key)] = string(header.Value)
	}
	return &kafka.Message{
		Topic:     record.Topic,
		Partition: record.Partition,
		Offset:    record.Offset,
		Key:       record.Key,
		Value:     record.Value,
		Headers:   headers,
	}
}

func validateConfig(cfg *config.KafkaConfig) error {
	if cfg == nil {
		return errors.New("kafka config is nil")
	}
	if len(cfg.Brokers) == 0 {
		return errors.New("kafka brokers is empty")
	}
	if cfg.GroupID == "" {
		return errors.New("kafka group_id is empty")
	}
	if len(cfg.Topics) == 0 {
		return errors.New("kafka topics is empty")
	}
	return nil
}

func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			parts = append(parts, err.Error())
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(parts, "; "))
}
