package domain

type Asset string

type OrderRequest struct {
	Asset1 Asset   `json:"asset_1"`
	Asset2 Asset   `json:"asset_2"`
	Qty    float64 `json:"qty"`
}

type QuotationRequest struct {
	Asset1 Asset `json:"asset_1"`
	Asset2 Asset `json:"asset_2"`
}

type QuotationResponse struct {
	Price1 float64 `json:"price_1"`
	Price2 float64 `json:"price_2"`
	TTLms  int     `json:"ttl_ms"`
}

type RiskRequest struct {
	Asset1 Asset   `json:"asset_1"`
	Price1 float64 `json:"price_1"`
	Asset2 Asset   `json:"asset_2"`
	Price2 float64 `json:"price_2"`
}

type RiskResponse struct {
	Approved bool `json:"approved"`
}

type ActionType string

const (
	ActionBuy  ActionType = "BUY"
	ActionSell ActionType = "SELL"
)

type PurchaseRequest struct {
	Asset  Asset      `json:"asset"`
	Price  float64    `json:"price"`
	Qty    float64    `json:"qty"`
	Action ActionType `json:"action"`
}

type PurchaseResponse struct {
	Success bool `json:"success"`
}