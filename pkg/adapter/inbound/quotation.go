package inbound

import (
	"encoding/json"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
)

type QuotationHandler struct {
	service   ports.QuotationService
	publisher ports.EventPublisher
}

func NewQuotationHandler(svc ports.QuotationService, publisher ports.EventPublisher) *QuotationHandler {
	return &QuotationHandler{service: svc, publisher: publisher}
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

	event := map[string]any{
		"type":   "quotation.received",
		"asset1": req.Asset1,
		"price1": res.Price1,
		"asset2": req.Asset2,
		"price2": res.Price2,
		"ttl_ms": res.TTLms,
	}
	_ = h.publisher.Publish("saga", event)

	data, _ := json.Marshal(res)
	return data
}
