package inbound

import (
	"encoding/json"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
)

type RiskHandler struct {
	service ports.RiskService
}

func NewRiskHandler(svc ports.RiskService) *RiskHandler {
	return &RiskHandler{service: svc}
}

func (h *RiskHandler) Handle(raw []byte) []byte {
	var req domain.RiskRequest
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
