package model

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	OrderStatusPending    = "pending"
	OrderStatusPaySucceed = "pay_succeed"
	OrderStatusPayFail    = "pay_fail"
)

type AssetOrder struct {
	ID             int64           `gorm:"primaryKey;autoIncrement"`
	UserID         int64           `gorm:"not null;index;uniqueIndex:uk_user_idempotency"`
	OrderNo        string          `gorm:"type:varchar(64);uniqueIndex;not null"`
	AssetID        int64           `gorm:"not null;index"`
	QuoteID        string          `gorm:"type:varchar(128);not null"`
	UnitPrice      string          `gorm:"type:varchar(64);not null"`
	Quantity       decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	PayCurrency    string          `gorm:"type:varchar(16);not null"`
	PayAmount      decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	PaymentID      string          `gorm:"type:varchar(128);index"`
	Status         string          `gorm:"type:varchar(32);not null;index"`
	IdempotencyKey string          `gorm:"type:varchar(128);uniqueIndex:uk_user_idempotency;not null"`
	CreatedAt      time.Time       `gorm:"autoCreateTime"`
	UpdatedAt      time.Time       `gorm:"autoUpdateTime"`
}

func (AssetOrder) TableName() string {
	return "asset_orders"
}
