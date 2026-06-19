package inbound

import (
	"encoding/json"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
)

type TradeHandler struct {
	service ports.TradeService
}

func NewTradeHandler(svc ports.TradeService) *TradeHandler {
	return &TradeHandler{service: svc}
}

func (h *TradeHandler) Handle(raw []byte) []byte {
	var req domain.TradeExecution
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil
	}

	res, err := h.service.Handle(req)
	if err != nil {
		return nil
	}

	data, _ := json.Marshal(res)
	return data
}
