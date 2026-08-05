package data

type GenQuoteRequest struct {
	AssetID  string `json:"asset_id" binding:"required"`
	Currency string `json:"currency"`
}

type QuoteVO struct {
	QuoteID   string `json:"quote_id"`
	AssetID   string `json:"asset_id"`
	Currency  string `json:"currency"`
	UnitPrice string `json:"unit_price"`
	ExpiresAt int64  `json:"expires_at"`
}
