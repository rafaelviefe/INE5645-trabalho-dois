package domain

type QuotationClient interface {
	Get(req QuotationRequest) (*QuotationResponse, error)
}

type RiskClient interface {
	Evaluate(req RiskRequest) (*RiskResponse, error)
}

type TradeClient interface {
	Execute(req TradeExecution) (*TradeResponse, error)
}
