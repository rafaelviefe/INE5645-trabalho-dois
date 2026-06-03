package handlers

import (
	"encoding/json"
	"math/rand"
	"time"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
)

type PurchaseHandler struct {
	cfg *config.Config
}

func NewPurchaseHandler(cfg *config.Config) *PurchaseHandler {
	return &PurchaseHandler{cfg: cfg}
}

func (h *PurchaseHandler) Handle(req []byte) []byte {
	var pReq domain.PurchaseRequest
	if err := json.Unmarshal(req, &pReq); err != nil {
		return nil
	}

	sleepTime := rand.Intn(h.cfg.Purchase.MaxSleepMs-h.cfg.Purchase.MinSleepMs+1) + h.cfg.Purchase.MinSleepMs
	time.Sleep(time.Duration(sleepTime) * time.Millisecond)

	success := true
	if pReq.Action == domain.ActionBuy {
		success = (rand.Float64() * 100.0) <= h.cfg.Purchase.SuccessRate
	}

	pRes := domain.PurchaseResponse{
		Success: success,
	}

	res, _ := json.Marshal(pRes)
	return res
}
