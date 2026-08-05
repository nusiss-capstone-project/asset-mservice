package data

type HoldingAssetVO struct {
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	IconURL string `json:"icon_url"`
}

type HoldingValuationVO struct {
	Currency  string `json:"currency"`
	UnitPrice string `json:"unit_price"`
	Amount    string `json:"amount"`
}

type HoldingVO struct {
	AssetID   string             `json:"asset_id"`
	Quantity  string             `json:"quantity"`
	UpdatedAt int64              `json:"updated_at"`
	Asset     HoldingAssetVO     `json:"asset"`
	Valuation HoldingValuationVO `json:"valuation"`
}

type HoldingListVO struct {
	Items []*HoldingVO `json:"items"`
}
