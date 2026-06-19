package service

import (
	"math/rand"
	"time"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
)

var _ ports.RiskService = (*RiskService)(nil)

type RiskService struct {
	minSleepMs  int
	maxSleepMs  int
	successRate float64
}

func NewRiskService(cfg config.RiskConfig) *RiskService {
	return &RiskService{
		minSleepMs:  cfg.MinSleepMs,
		maxSleepMs:  cfg.MaxSleepMs,
		successRate: cfg.SuccessRate,
	}
}

func (s *RiskService) Handle(req domain.RiskRequest) (*domain.RiskResponse, error) {
	sleepTime := rand.Intn(s.maxSleepMs-s.minSleepMs+1) + s.minSleepMs
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)

	approved := (rand.Float64() * 100.0) <= s.successRate

	return &domain.RiskResponse{
		Approved: approved,
	}, nil
}
