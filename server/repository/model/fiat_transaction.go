package model

import (
	"time"

	"github.com/shopspring/decimal"
)

const (
	FiatTxnTypeDeposit = "DEPOSIT"

	FiatTxnStatusPending   = "PENDING"
	FiatTxnStatusSucceeded = "SUCCEEDED"
	FiatTxnStatusFailed    = "FAILED"
)

type FiatTransaction struct {
	ID                int64           `gorm:"primaryKey;autoIncrement"`
	TransactionNo     string          `gorm:"type:varchar(64);uniqueIndex;not null"`
	UserID            int64           `gorm:"not null;uniqueIndex:uk_fiat_user_idempotent;index"`
	IdempotentKey     string          `gorm:"type:varchar(128);uniqueIndex:uk_fiat_user_idempotent;not null"`
	AccountID         int64           `gorm:"not null;default:0"`
	Amount            decimal.Decimal `gorm:"type:decimal(36,18);not null"`
	Currency          string          `gorm:"type:varchar(16);not null"`
	TransactionType   string          `gorm:"type:varchar(32);not null"`
	Status            string          `gorm:"type:varchar(32);not null;index"`
	PaymentMethodID   int64           `gorm:"not null"`
	ExternalPaymentID string          `gorm:"type:varchar(128)"`
	FailureReason     string          `gorm:"type:varchar(512)"`
	CreatedAt         time.Time       `gorm:"autoCreateTime"`
	UpdatedAt         time.Time       `gorm:"autoUpdateTime"`
}

func (FiatTransaction) TableName() string {
	return "fiat_transactions"
}
