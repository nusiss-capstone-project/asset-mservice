package redis

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"github.com/nusiss-capstone-project/asset-mservice/server/log"
)

var Client *goredis.Client

// Init connects to Redis when REDIS_ADDR is set. Connection failures disable the client
// so callers can degrade intentionally.
func Init() {
	addr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if addr == "" {
		Client = nil
		log.Logger.Info("redis disabled: REDIS_ADDR not set")
		return
	}

	db := 0
	if raw := strings.TrimSpace(os.Getenv("REDIS_DB")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			db = parsed
		}
	}

	client := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Logger.Warnw("redis unavailable", "addr", addr, "error", err)
		_ = client.Close()
		Client = nil
		return
	}
	Client = client
	log.Logger.Infow("redis connected", "addr", addr, "db", db)
}

// Available reports whether a live Redis client is configured.
func Available() bool {
	return Client != nil
}
