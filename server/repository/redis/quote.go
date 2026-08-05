package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nusiss-capstone-project/asset-mservice/server/log"
)

const (
	QuoteTTL     = 5 * time.Minute
	quoteKeyPref = "asset:quote:"
)

type QuoteCache struct {
	QuoteID   string `json:"quote_id"`
	AssetID   int64  `json:"asset_id"`
	Currency  string `json:"currency"`
	UnitPrice string `json:"unit_price"`
	UserID    int64  `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
}

func quoteKey(quoteID string) string {
	return quoteKeyPref + quoteID
}

func SaveQuote(ctx context.Context, quote *QuoteCache) error {
	if !Available() {
		return fmt.Errorf("redis unavailable")
	}
	payload, err := json.Marshal(quote)
	if err != nil {
		return err
	}
	if err := Client.Set(ctx, quoteKey(quote.QuoteID), payload, QuoteTTL).Err(); err != nil {
		log.WithContext(ctx).Errorw("save quote to redis failed", "quote_id", quote.QuoteID, "error", err)
		return err
	}
	log.WithContext(ctx).Infow("quote saved to redis",
		"quote_id", quote.QuoteID,
		"asset_id", quote.AssetID,
		"user_id", quote.UserID,
		"expires_at", quote.ExpiresAt,
	)
	return nil
}

func GetQuote(ctx context.Context, quoteID string) (*QuoteCache, error) {
	if !Available() {
		return nil, fmt.Errorf("redis unavailable")
	}
	raw, err := Client.Get(ctx, quoteKey(quoteID)).Bytes()
	if err != nil {
		return nil, err
	}
	var quote QuoteCache
	if err := json.Unmarshal(raw, &quote); err != nil {
		return nil, err
	}
	return &quote, nil
}
