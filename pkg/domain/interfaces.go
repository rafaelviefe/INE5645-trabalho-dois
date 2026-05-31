package domain

type QuotationService interface {
	GetQuotation(req QuotationRequest) (*QuotationResponse, error)
}

type RiskService interface {
	EvaluateRisk(req RiskRequest) (*RiskResponse, error)
}

type PurchaseService interface {
	ExecutePurchase(req PurchaseRequest) (*PurchaseResponse, error)
}