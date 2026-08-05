package service

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/nusiss-capstone-project/asset-mservice/server/config"
	"github.com/nusiss-capstone-project/asset-mservice/server/log"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository"
	cacheredis "github.com/nusiss-capstone-project/asset-mservice/server/repository/redis"
	"github.com/nusiss-capstone-project/asset-mservice/server/repository/model"
	goredis "github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func initServiceTestEnv(t *testing.T) {
	t.Helper()
	config.Config = &config.Conf{
		LogConfig: &config.LogConfig{Level: "error", FilePath: ""},
	}
	log.InitLogger()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	repository.DB = db
}

func startTestRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(func() {
		mr.Close()
		cacheredis.Client = nil
	})
	cacheredis.Client = goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	return mr
}

func sampleAsset(id int64) *model.Asset {
	now := time.Now()
	return &model.Asset{
		ID:           id,
		Name:         "Bitcoin",
		Symbol:       "BTC",
		IconURL:      "https://example.com/btc.png",
		Currency:     "USD",
		CurrentPrice: "100.00",
		Status:       model.AssetStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func sampleOrder(id int64, userID int64, status string) *model.AssetOrder {
	now := time.Now()
	return &model.AssetOrder{
		ID:             id,
		UserID:         userID,
		OrderNo:        "AO20260805001",
		AssetID:        1,
		QuoteID:        "q_test",
		UnitPrice:      "100.00",
		Quantity:       decimal.RequireFromString("0.5"),
		PayCurrency:    "USD",
		PayAmount:      decimal.RequireFromString("50.00"),
		PaymentID:      "",
		Status:         status,
		IdempotencyKey: "idem-1",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}
