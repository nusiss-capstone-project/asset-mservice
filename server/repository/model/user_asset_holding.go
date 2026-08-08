package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type UserAssetHolding struct {
	ID        int64           `gorm:"primaryKey;autoIncrement"`
	UserID    int64           `gorm:"not null;uniqueIndex:uk_user_asset"`
	AssetID   int64           `gorm:"not null;uniqueIndex:uk_user_asset;index"`
	Quantity  decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	CreatedAt time.Time       `gorm:"autoCreateTime"`
	UpdatedAt time.Time       `gorm:"autoUpdateTime"`
}

func (UserAssetHolding) TableName() string {
	return "user_asset_holdings"
}
