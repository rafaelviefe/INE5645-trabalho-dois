package outbound

import (
	"encoding/json"
	"fmt"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
	"trading-saga/pkg/tcp"
)

var _ ports.RiskClient = (*RiskClient)(nil)

type RiskClient struct {
	address string
}

func NewRiskClient(address string) *RiskClient {
	return &RiskClient{address: address}
}

func (c *RiskClient) Evaluate(req domain.RiskRequest) (*domain.RiskResponse, error) {
	fmt.Printf("\033[90m -> [RISCO] Analisando viabilidade da operação...\033[0m\n")

	payload, _ := json.Marshal(req)
	resPayload, err := tcp.SendRequest(c.address, payload)
	if err != nil {
		return nil, err
	}

	var res domain.RiskResponse
	json.Unmarshal(resPayload, &res)
	if res.Approved {
		fmt.Printf("\033[90m <- [RISCO] Operação APROVADA\033[0m\n")
	} else {
		fmt.Printf("\033[90m <- [RISCO] Operação REPROVADA\033[0m\n")
	}
	return &res, nil
}
