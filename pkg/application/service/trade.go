package service

import (
	"math/rand"
	"time"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
)

var _ ports.TradeService = (*TradeService)(nil)

type TradeService struct {
	minSleepMs  int
	maxSleepMs  int
	successRate float64
}

func NewTradeService(cfg config.PurchaseConfig) *TradeService {
	return &TradeService{
		minSleepMs:  cfg.MinSleepMs,
		maxSleepMs:  cfg.MaxSleepMs,
		successRate: cfg.SuccessRate,
	}
}

func (s *TradeService) Handle(req domain.TradeExecution) (*domain.TradeResponse, error) {
	sleepTime := rand.Intn(s.maxSleepMs-s.minSleepMs+1) + s.minSleepMs
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)

	success := true
	if req.Action == domain.ActionBuy {
		success = (rand.Float64() * 100.0) <= s.successRate
	}

	return &domain.TradeResponse{
		Success: success,
	}, nil
}
