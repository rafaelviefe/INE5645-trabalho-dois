package service

import (
	"math/rand"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
)

var _ ports.QuotationService = (*QuotationService)(nil)

type QuotationService struct {
	minPrice float64
	maxPrice float64
	ttlMs    int
}

func NewQuotationService(cfg config.QuotationConfig) *QuotationService {
	return &QuotationService{
		minPrice: cfg.MinPrice,
		maxPrice: cfg.MaxPrice,
		ttlMs:    cfg.TTLMs,
	}
}

func (s *QuotationService) Handle(req domain.QuotationRequest) (*domain.QuotationResponse, error) {
	p1 := s.minPrice + rand.Float64()*(s.maxPrice-s.minPrice)
	p2 := s.minPrice + rand.Float64()*(s.maxPrice-s.maxPrice)

	return &domain.QuotationResponse{
		Price1: p1,
		Price2: p2,
		TTLms:  s.ttlMs,
	}, nil
}
