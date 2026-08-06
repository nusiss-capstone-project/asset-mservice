package data

type CreateDepositRequest struct {
	PaymentMethodID int64  `json:"payment_method_id" binding:"required"`
	Currency        string `json:"currency" binding:"required"`
	Amount          string `json:"amount" binding:"required"`
	IdempotentKey   string `json:"idempotent_key" binding:"required"`
}

type FiatTransactionVO struct {
	TransactionID     string `json:"transaction_id"`
	TransactionNo     string `json:"transaction_no"`
	AccountID         string `json:"account_id"`
	UserID            string `json:"user_id"`
	Currency          string `json:"currency"`
	Amount            string `json:"amount"`
	TransactionType   string `json:"transaction_type"`
	Status            string `json:"status"`
	PaymentMethodID   string `json:"payment_method_id"`
	ExternalPaymentID string `json:"external_payment_id"`
	FailureReason     string `json:"failure_reason"`
	CreatedAt         int64  `json:"created_at"`
	UpdatedAt         int64  `json:"updated_at"`
}

type FiatAccountVO struct {
	AccountID string `json:"account_id"`
	Currency  string `json:"currency"`
	Balance   string `json:"balance"`
	UpdatedAt int64  `json:"updated_at"`
}

type FiatAccountListVO struct {
	Items []*FiatAccountVO `json:"items"`
}

type LedgerVO struct {
	LedgerID     string `json:"ledger_id"`
	LedgerNo     string `json:"ledger_no"`
	AssetCode    string `json:"asset_code"`
	ChangeAmount string `json:"change_amount"`
	BusinessType string `json:"business_type"`
	BusinessID   string `json:"business_id"`
	BalanceAfter string `json:"balance_after"`
	CreatedAt    int64  `json:"created_at"`
}

type LedgerListVO struct {
	NextCursor string      `json:"next_cursor"`
	Items      []*LedgerVO `json:"items"`
}
