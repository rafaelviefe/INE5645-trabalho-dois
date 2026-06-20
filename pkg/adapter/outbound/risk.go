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

var _ ports.RiskClient = (*RiskClient)(nil)

type RiskClient struct {
	pool *Pool
}

func NewRiskClient(addrs []string, cooldown time.Duration) *RiskClient {
	return &RiskClient{
		pool: NewPool(addrs, cooldown),
	}
}

func (c *RiskClient) Evaluate(req domain.RiskRequest) (*domain.RiskResponse, error) {
	fmt.Printf("\033[90m -> [RISCO] Analisando viabilidade da operação...\033[0m\n")

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

	var res domain.RiskResponse
	json.Unmarshal(resPayload, &res)
	if res.Approved {
		fmt.Printf("\033[90m <- [RISCO] Operação APROVADA\033[0m\n")
	} else {
		fmt.Printf("\033[90m <- [RISCO] Operação REPROVADA\033[0m\n")
	}
	return &res, nil
}
