package model

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	LedgerBusinessTypePurchase = "PURCHASE"
	LedgerBusinessTypeReward   = "REWARD"
)

type AccountLedger struct {
	ID           int64           `gorm:"primaryKey;autoIncrement"`
	LedgerNo     string          `gorm:"type:varchar(64);uniqueIndex;not null"`
	UserID       int64           `gorm:"not null;index"`
	AssetCode    string          `gorm:"type:varchar(32);not null;uniqueIndex:uk_ledger_biz"`
	ChangeAmount decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	BusinessType string          `gorm:"type:varchar(32);not null;uniqueIndex:uk_ledger_biz"`
	BusinessID   string          `gorm:"type:varchar(128);not null;uniqueIndex:uk_ledger_biz"`
	BalanceAfter decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	CreatedAt    time.Time       `gorm:"autoCreateTime"`
}

func (AccountLedger) TableName() string {
	return "account_ledger"
}
