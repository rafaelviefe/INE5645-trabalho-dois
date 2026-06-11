package domain

type QuotationClient interface {
	GetQuotation(req QuotationRequest) (*QuotationResponse, error)
}

type RiskClient interface {
	EvaluateRisk(req RiskRequest) (*RiskResponse, error)
}

type PurchaseClient interface {
	ExecutePurchase(req PurchaseRequest) (*PurchaseResponse, error)
}
