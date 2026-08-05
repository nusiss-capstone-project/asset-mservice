package listener

import (
	"context"

	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	"github.com/nusiss-capstone-project/asset-mservice/server/kafka/producer"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
)

// Init starts the Kafka producer and consumer when enabled in config.
func Init(ctx context.Context) {
	cfg := config.Config.KafkaConfig
	producer.Ensure()
	if cfg == nil || !cfg.Enabled {
		log.Logger.Info("kafka disabled")
		return
	}
	Start(ctx, cfg)
}
