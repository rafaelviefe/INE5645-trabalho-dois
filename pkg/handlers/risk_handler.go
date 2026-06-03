package handlers

import (
	"encoding/json"
	"math/rand"
	"time"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
)

type RiskHandler struct {
	cfg *config.Config
}

func NewRiskHandler(cfg *config.Config) *RiskHandler {
	return &RiskHandler{cfg: cfg}
}

func (h *RiskHandler) Handle(req []byte) []byte {
	var rReq domain.RiskRequest
	if err := json.Unmarshal(req, &rReq); err != nil {
		return nil
	}

	sleepTime := rand.Intn(h.cfg.Risk.MaxSleepMs-h.cfg.Risk.MinSleepMs+1) + h.cfg.Risk.MinSleepMs
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)

	approved := (rand.Float64() * 100.0) <= h.cfg.Risk.SuccessRate

	rRes := domain.RiskResponse{
		Approved: approved,
	}

	res, _ := json.Marshal(rRes)
	return res
}
