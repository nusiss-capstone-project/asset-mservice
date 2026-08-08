package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type UserFiatAccount struct {
	ID        int64           `gorm:"primaryKey;autoIncrement"`
	UserID    int64           `gorm:"not null;uniqueIndex:uk_user_fiat_currency;index"`
	Currency  string          `gorm:"type:varchar(16);not null;uniqueIndex:uk_user_fiat_currency"`
	Balance   decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	CreatedAt time.Time       `gorm:"autoCreateTime"`
	UpdatedAt time.Time       `gorm:"autoUpdateTime"`
}

func (UserFiatAccount) TableName() string {
	return "user_fiat_accounts"
}
