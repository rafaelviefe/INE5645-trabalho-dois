package outbound

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
	"trading-saga/pkg/tcp"
)

var _ ports.TradeClient = (*TradeClient)(nil)

type TradeClient struct {
	pool *Pool
}

func NewTradeClient(addrs []string, cooldown time.Duration) *TradeClient {
	return &TradeClient{
		pool: NewPool(addrs, cooldown),
	}
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

	addr, err := c.pool.Next()
	if err != nil {
		return nil, err
	}

	payload, _ := json.Marshal(req)
	resPayload, err := tcp.SendRequest(addr, payload)
	if err != nil {
		if _, ok := err.(net.Error); ok {
			c.pool.MarkUnhealthy(addr)
		}
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
