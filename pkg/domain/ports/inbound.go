package ports

import "trading-saga/pkg/domain"

type QuotationService interface {
	Handle(req domain.QuotationRequest) (*domain.QuotationResponse, error)
}

type RiskService interface {
	Handle(req domain.RiskRequest) (*domain.RiskResponse, error)
}

type TradeService interface {
	Handle(req domain.TradeExecution) (*domain.TradeResponse, error)
}
