package adapters

import (
	"encoding/json"
	"fmt"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/tcp"
)

var _ domain.QuotationClient = (*QuotationClient)(nil)

type QuotationClient struct {
	address string
}

func NewQuotationClient(address string) *QuotationClient {
	return &QuotationClient{address: address}
}

func (c *QuotationClient) GetQuotation(req domain.QuotationRequest) (*domain.QuotationResponse, error) {
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

var _ domain.RiskClient = (*RiskClient)(nil)

type RiskClient struct {
	address string
}

func NewRiskClient(address string) *RiskClient {
	return &RiskClient{address: address}
}

func (c *RiskClient) EvaluateRisk(req domain.RiskRequest) (*domain.RiskResponse, error) {
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

var _ domain.PurchaseClient = (*PurchaseClient)(nil)

type PurchaseClient struct {
	address string
}

func NewPurchaseClient(address string) *PurchaseClient {
	return &PurchaseClient{address: address}
}

func (c *PurchaseClient) ExecutePurchase(req domain.PurchaseRequest) (*domain.PurchaseResponse, error) {
	if req.Action == domain.ActionBuy {
		fmt.Printf("\033[90m -> [COMPRA] Efetuando COMPRA de %.2f %s...\033[0m\n", req.Qty, req.Asset)
	} else {
		fmt.Printf("\033[93m -> [COMPENSAÇÃO] Efetuando VENDA (estorno) de %.2f %s...\033[0m\n", req.Qty, req.Asset)
	}

	payload, _ := json.Marshal(req)
	resPayload, err := tcp.SendRequest(c.address, payload)
	if err != nil {
		return nil, err
	}

	var res domain.PurchaseResponse
	json.Unmarshal(resPayload, &res)

	if res.Success {
		fmt.Printf("\033[90m <- [COMPRA] Ação %s para %s BEM-SUCEDIDA\033[0m\n", req.Action, req.Asset)
	} else {
		fmt.Printf("\033[90m <- [COMPRA] Ação %s para %s FALHOU\033[0m\n", req.Action, req.Asset)
	}
	return &res, nil
}
