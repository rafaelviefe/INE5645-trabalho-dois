package inbound

import (
	"encoding/json"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
)

type QuotationHandler struct {
	service ports.QuotationService
}

func NewQuotationHandler(svc ports.QuotationService) *QuotationHandler {
	return &QuotationHandler{service: svc}
}

func (h *QuotationHandler) Handle(raw []byte) []byte {
	var req domain.QuotationRequest
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
