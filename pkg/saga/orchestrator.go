package saga

import (
	"errors"
	"fmt"
	"time"
	"trading-saga/pkg/domain"
)

var (
	ErrTTLExceeded      = errors.New("TTL excedido")
	ErrRiskReproved     = errors.New("reprovado pela analise de risco")
	ErrPurchasingAsset1 = errors.New("falha na compra do ativo 1")
	ErrPurchasingAsset2 = errors.New("falha na compra do ativo 2")
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
		return ErrTTLExceeded
	}
	return nil
}

func (o *Orchestrator) ExecuteOrder(order domain.OrderRequest) error {
	var rollbacks []domain.PurchaseRequest

	defer func() {
		for i := len(rollbacks) - 1; i >= 0; i-- {
			o.rollback(rollbacks[i])
		}
	}()

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

		purchaseReq := domain.PurchaseRequest{
			Asset:  asset,
			Price:  prices[i],
			Qty:    order.Qty,
			Action: domain.ActionBuy,
		}

		purchaseRes, err := o.purchaseService.ExecutePurchase(purchaseReq)
		if err != nil || !purchaseRes.Success {
			return purchaseResponseErrors[i]
		}

		rollbacks = append(rollbacks, purchaseReq)
	}

	rollbacks = nil
	return nil
}

func (o *Orchestrator) rollback(buyReq domain.PurchaseRequest) {
	compReq := domain.PurchaseRequest{
		Asset: buyReq.Asset,
		Price: buyReq.Price,
		Qty:   buyReq.Qty,

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
