package data

type CreateOrderRequest struct {
	QuoteID         string `json:"quote_id" binding:"required"`
	Quantity        string `json:"quantity" binding:"required"`
	PaymentMethodID int64  `json:"payment_method_id" binding:"required"`
	IdempotencyKey  string `json:"idempotency_key" binding:"required"`
}

type OrderVO struct {
	OrderID     string `json:"order_id"`
	OrderNo     string `json:"order_no"`
	AssetID     string `json:"asset_id,omitempty"`
	QuoteID     string `json:"quote_id,omitempty"`
	UnitPrice   string `json:"unit_price"`
	Quantity    string `json:"quantity"`
	PayCurrency string `json:"pay_currency"`
	PayAmount   string `json:"pay_amount"`
	PaymentID   string `json:"payment_id"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type OrderAssetVO struct {
	AssetID string `json:"asset_id"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	IconURL string `json:"icon_url"`
}

type OrderDetailVO struct {
	OrderID     string       `json:"order_id"`
	OrderNo     string       `json:"order_no"`
	Asset       OrderAssetVO `json:"asset"`
	UnitPrice   string       `json:"unit_price"`
	Quantity    string       `json:"quantity"`
	PayCurrency string       `json:"pay_currency"`
	PayAmount   string       `json:"pay_amount"`
	PaymentID   string       `json:"payment_id"`
	Status      string       `json:"status"`
	CreatedAt   int64        `json:"created_at"`
	UpdatedAt   int64        `json:"updated_at"`
}

type OrderListVO struct {
	NextCursor string     `json:"next_cursor"`
	Items      []*OrderVO `json:"items"`
}
