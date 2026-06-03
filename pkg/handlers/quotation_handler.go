package handlers

import (
	"encoding/json"
	"math/rand"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
)

type QuotationHandler struct {
	cfg *config.Config
}

func NewQuotationHandler(cfg *config.Config) *QuotationHandler {
	return &QuotationHandler{cfg: cfg}
}

func (h *QuotationHandler) Handle(req []byte) []byte {
	var qReq domain.QuotationRequest
	if err := json.Unmarshal(req, &qReq); err != nil {
		return nil
	}

	p1 := h.cfg.Quotation.MinPrice + rand.Float64()*(h.cfg.Quotation.MaxPrice-h.cfg.Quotation.MinPrice)
	p2 := h.cfg.Quotation.MinPrice + rand.Float64()*(h.cfg.Quotation.MaxPrice-h.cfg.Quotation.MaxPrice)

	qRes := domain.QuotationResponse{
		Price1: p1,
		Price2: p2,
		TTLms:  h.cfg.Quotation.TTLMs,
	}

	res, err := json.Marshal(qRes)
	if err != nil {
		return nil
	}
	return res
}
