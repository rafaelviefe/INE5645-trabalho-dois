package outbound

import (
	"encoding/json"
	"fmt"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
	"trading-saga/pkg/tcp"
)

var _ ports.TradeClient = (*TradeClient)(nil)

type TradeClient struct {
	address string
}

func NewTradeClient(address string) *TradeClient {
	return &TradeClient{address: address}
}

func (c *TradeClient) Execute(req domain.TradeExecution) (*domain.TradeResponse, error) {
	switch req.Action {
	case domain.ActionBuy:
		fmt.Printf("\033[90m -> [COMPRA] Efetuando COMPRA de %.2f %s...\033[0m\n", req.Quantity, req.Asset)
	case domain.ActionSell:
		fmt.Printf("\033[93m -> [COMPENSAÇÃO] Efetuando VENDA (estorno) de %.2f %s...\033[0m\n", req.Quantity, req.Asset)
	default:
		return nil, fmt.Errorf("invalid action: %s", req.Action)
	}

	payload, _ := json.Marshal(req)
	resPayload, err := tcp.SendRequest(c.address, payload)
	if err != nil {
		return nil, err
	}

	var res domain.TradeResponse
	json.Unmarshal(resPayload, &res)

	if res.Success {
		fmt.Printf("\033[90m <- [COMPRA] Ação %s para %s BEM-SUCEDIDA\033[0m\n", req.Action, req.Asset)
	} else {
		fmt.Printf("\033[90m <- [COMPRA] Ação %s para %s FALHOU\033[0m\n", req.Action, req.Asset)
	}
	return &res, nil
}
