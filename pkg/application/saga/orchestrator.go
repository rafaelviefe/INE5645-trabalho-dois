package saga

import (
	"errors"
	"fmt"
	"time"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
)

var (
	ErrTTLExceeded      = errors.New("TTL excedido")
	ErrRiskReproved     = errors.New("reprovado pela analise de risco")
	ErrPurchasingAsset1 = errors.New("falha na compra do ativo 1")
	ErrPurchasingAsset2 = errors.New("falha na compra do ativo 2")
)

type Orchestrator struct {
	quotation ports.QuotationClient
	risk      ports.RiskClient
	order     ports.TradeClient
}

func NewOrchestrator(qs ports.QuotationClient, rs ports.RiskClient, ps ports.TradeClient) *Orchestrator {
	return &Orchestrator{
		quotation: qs,
		risk:      rs,
		order:     ps,
	}
}

func checkTTL(deadline time.Time) error {
	if time.Now().After(deadline) {
		return ErrTTLExceeded
	}
	return nil
}

func (o *Orchestrator) ExecuteOrder(order domain.OrderRequest) error {
	var rollbacks []domain.TradeExecution

	defer func() {
		for i := len(rollbacks) - 1; i >= 0; i-- {
			o.rollback(rollbacks[i])
		}
	}()

	quoteReq := domain.QuotationRequest{
		Asset1: order.Asset1,
		Asset2: order.Asset2,
	}

	quoteRes, err := o.quotation.Get(quoteReq)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(time.Duration(quoteRes.TTLms) * time.Millisecond)

	if err := checkTTL(deadline); err != nil {
		return err
	}

	riskReq := domain.RiskRequest{
		Asset1: order.Asset1,
		Price1: quoteRes.Price1,
		Asset2: order.Asset2,
		Price2: quoteRes.Price2,
	}

	riskRes, err := o.risk.Evaluate(riskReq)
	if err != nil {
		return err
	}

	if !riskRes.Approved {
		return ErrRiskReproved
	}

	if err := checkTTL(deadline); err != nil {
		return err
	}

	purchaseResponseErrors := map[int]error{
		0: ErrPurchasingAsset1,
		1: ErrPurchasingAsset2,
	}
	prices := []float64{quoteRes.Price1, quoteRes.Price2}
	for i, asset := range []domain.Asset{order.Asset1, order.Asset2} {
		if err := checkTTL(deadline); err != nil {
			return err
		}

		purchaseReq := domain.TradeExecution{
			Asset:    asset,
			Price:    prices[i],
			Quantity: order.Qty,
			Action:   domain.ActionBuy,
		}

		purchaseRes, err := o.order.Execute(purchaseReq)
		if err != nil || !purchaseRes.Success {
			return purchaseResponseErrors[i]
		}

		rollbacks = append(rollbacks, purchaseReq)
	}

	rollbacks = nil
	return nil
}

func (o *Orchestrator) rollback(purchaseReq domain.TradeExecution) {
	compReq := domain.TradeExecution{
		Asset:    purchaseReq.Asset,
		Price:    purchaseReq.Price,
		Quantity: purchaseReq.Quantity,

		Action: domain.ActionSell,
	}

	res, err := o.order.Execute(compReq)
	if err != nil {
		fmt.Printf("\033[41m\033[37m[ALERTA CRÍTICO] ERRO DE REDE NA COMPENSAÇÃO DE %s: %v\033[0m\n", purchaseReq.Asset, err)
		return
	}

	if res == nil || !res.Success {
		fmt.Printf("\033[41m\033[37m[ALERTA CRÍTICO] FALHA DE CONSISTÊNCIA: COMPENSAÇÃO DE %s REJEITADA OU MAL-SUCEDIDA!\033[0m\n", purchaseReq.Asset)
		return
	}
}
