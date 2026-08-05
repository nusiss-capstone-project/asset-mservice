package data

type AssetVO struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Symbol       string `json:"symbol"`
	IconURL      string `json:"icon_url"`
	Currency     string `json:"currency"`
	CurrentPrice string `json:"current_price"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}
