package adapters

import (
	"encoding/json"
	"fmt"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"

	"trading-saga/pkg/tcp"
)

var _ ports.QuotationClient = (*QuotationClient)(nil)

type QuotationClient struct {
	address string
}

func NewQuotationClient(address string) *QuotationClient {
	return &QuotationClient{address: address}
}

func (c *QuotationClient) Get(req domain.QuotationRequest) (*domain.QuotationResponse, error) {
	fmt.Printf("\033[90m -> [COTAÇÃO] Solicitando preços e TTL para %s e %s...\033[0m\n", req.Asset1, req.Asset2)

	payload, _ := json.Marshal(req)
	resPayload, err := tcp.SendRequest(c.address, payload)
	if err != nil {
		return nil, err
	}

	var res domain.QuotationResponse
	json.Unmarshal(resPayload, &res)
	fmt.Printf("\033[90m <- [COTAÇÃO] %s=$%.2f | %s=$%.2f | TTL=%dms\033[0m\n", req.Asset1, res.Price1, req.Asset2, res.Price2, res.TTLms)
	return &res, nil
}
