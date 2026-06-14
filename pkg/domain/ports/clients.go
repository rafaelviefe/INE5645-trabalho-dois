package ports

import "trading-saga/pkg/domain"

type QuotationClient interface {
	Get(req domain.QuotationRequest) (*domain.QuotationResponse, error)
}

type RiskClient interface {
	Evaluate(req domain.RiskRequest) (*domain.RiskResponse, error)
}

type TradeClient interface {
	Execute(req domain.TradeExecution) (*domain.TradeResponse, error)
}
