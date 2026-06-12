package handlers

import (
	"encoding/json"
	"math/rand"
	"time"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
)

type TradeHandler struct {
	cfg *config.Config
}

func NewTradeHandler(cfg *config.Config) *TradeHandler {
	return &TradeHandler{cfg: cfg}
}

func (h *TradeHandler) Handle(req []byte) []byte {
	var pReq domain.TradeExecution
	if err := json.Unmarshal(req, &pReq); err != nil {
		return nil
	}

	sleepTime := rand.Intn(h.cfg.Purchase.MaxSleepMs-h.cfg.Purchase.MinSleepMs+1) + h.cfg.Purchase.MinSleepMs
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)

	success := true
	if pReq.Action == domain.ActionBuy {
		success = (rand.Float64() * 100.0) <= h.cfg.Purchase.SuccessRate
	}

	pRes := domain.TradeResponse{
		Success: success,
	}

	res, _ := json.Marshal(pRes)
	return res
}
