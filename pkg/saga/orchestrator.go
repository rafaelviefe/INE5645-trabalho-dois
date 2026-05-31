package saga

import (
	"errors"
	"fmt"
	"time"
	"trading-saga/pkg/domain"
)

var (
	ErrTTL       = errors.New("TTL excedido")
	ErrRisk      = errors.New("reprovado pela analise de risco")
	ErrPurchase1 = errors.New("falha na compra do ativo 1")
	ErrPurchase2 = errors.New("falha na compra do ativo 2")
)

type Orchestrator struct {
	quotationService domain.QuotationService
	riskService      domain.RiskService
	purchaseService  domain.PurchaseService
}

func NewOrchestrator(qs domain.QuotationService, rs domain.RiskService, ps domain.PurchaseService) *Orchestrator {
	return &Orchestrator{
		quotationService: qs,
		riskService:      rs,
		purchaseService:  ps,
	}
}

func checkTTL(deadline time.Time) error {
	if time.Now().After(deadline) {
		return ErrTTL
	}
	return nil
}

func (o *Orchestrator) ExecuteOrder(order domain.OrderRequest) error {
	quoteReq := domain.QuotationRequest{
		Asset1: order.Asset1,
		Asset2: order.Asset2,
	}

	quoteRes, err := o.quotationService.GetQuotation(quoteReq)
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

	riskRes, err := o.riskService.EvaluateRisk(riskReq)
	if err != nil {
		return err
	}

	if !riskRes.Approved {
		return ErrRisk
	}

	if err := checkTTL(deadline); err != nil {
		return err
	}

	purchReq1 := domain.PurchaseRequest{
		Asset:  order.Asset1,
		Price:  quoteRes.Price1,
		Qty:    order.Qty,
		Action: domain.ActionBuy,
	}

	purchRes1, err := o.purchaseService.ExecutePurchase(purchReq1)
	if err != nil || !purchRes1.Success {
		return ErrPurchase1
	}

	if err := checkTTL(deadline); err != nil {
		o.compensate(purchReq1)
		return err
	}

	purchReq2 := domain.PurchaseRequest{
		Asset:  order.Asset2,
		Price:  quoteRes.Price2,
		Qty:    order.Qty,
		Action: domain.ActionBuy,
	}

	purchRes2, err := o.purchaseService.ExecutePurchase(purchReq2)
	if err != nil || !purchRes2.Success {
		o.compensate(purchReq1)
		return ErrPurchase2
	}

	if err := checkTTL(deadline); err != nil {
		o.compensate(purchReq1)
		o.compensate(purchReq2)
		return err
	}

	return nil
}

func (o *Orchestrator) compensate(buyReq domain.PurchaseRequest) {
	compReq := domain.PurchaseRequest{
		Asset:  buyReq.Asset,
		Price:  buyReq.Price,
		Qty:    buyReq.Qty,
		Action: domain.ActionSell,
	}

	res, err := o.purchaseService.ExecutePurchase(compReq)
	if err != nil {
		fmt.Printf("\033[41m\033[37m[ALERTA CRÍTICO] ERRO DE REDE NA COMPENSAÇÃO DE %s: %v\033[0m\n", buyReq.Asset, err)
		return
	}

	if res == nil || !res.Success {
		fmt.Printf("\033[41m\033[37m[ALERTA CRÍTICO] FALHA DE CONSISTÊNCIA: COMPENSAÇÃO DE %s REJEITADA OU MAL-SUCEDIDA!\033[0m\n", buyReq.Asset)
		return
	}
}
