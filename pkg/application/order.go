package application

import (
	"context"
	"errors"
	"fmt"
	"time"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
	"trading-saga/pkg/saga"
)

var (
	ErrTTLExceeded      = errors.New("TTL excedido")
	ErrRiskReproved     = errors.New("reprovado pela analise de risco")
	ErrPurchasingAsset1 = errors.New("falha na compra do ativo 1")
	ErrPurchasingAsset2 = errors.New("falha na compra do ativo 2")
)

func ExecuteOrder(order domain.OrderRequest, quotation ports.QuotationClient, risk ports.RiskClient, trade ports.TradeClient) error {
	sg := saga.New()

	var (
		quoteRes *domain.QuotationResponse
		deadline time.Time
	)

	checkTTL := func(ctx context.Context) error {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return ErrTTLExceeded
		}
		return nil
	}

	sg.AddStep(
		func(ctx context.Context) error {
			res, err := quotation.Get(domain.QuotationRequest{
				Asset1: order.Asset1,
				Asset2: order.Asset2,
			})
			if err != nil {
				return err
			}
			quoteRes = res
			deadline = time.Now().Add(time.Duration(res.TTLms) * time.Millisecond)
			return nil
		},
		nil,
	)

	sg.AddStep(checkTTL, nil)

	sg.AddStep(
		func(ctx context.Context) error {
			res, err := risk.Evaluate(domain.RiskRequest{
				Asset1: order.Asset1,
				Price1: quoteRes.Price1,
				Asset2: order.Asset2,
				Price2: quoteRes.Price2,
			})
			if err != nil {
				return err
			}
			if !res.Approved {
				return ErrRiskReproved
			}
			return nil
		},
		nil,
	)

	sg.AddStep(checkTTL, nil)

	sg.AddStep(
		func(ctx context.Context) error {
			buy := domain.TradeExecution{
				Asset:    order.Asset1,
				Price:    quoteRes.Price1,
				Quantity: order.Qty,
				Action:   domain.ActionBuy,
			}
			res, err := trade.Execute(buy)
			if err != nil || !res.Success {
				return fmt.Errorf("%w: %s", ErrPurchasingAsset1, order.Asset1)
			}
			return nil
		},
		func(ctx context.Context) error {
			sell := domain.TradeExecution{
				Asset:    order.Asset1,
				Price:    quoteRes.Price1,
				Quantity: order.Qty,
				Action:   domain.ActionSell,
			}
			res, err := trade.Execute(sell)
			if err != nil {
				fmt.Printf("\033[41m\033[37m[ALERTA CRÍTICO] ERRO DE REDE NA COMPENSAÇÃO DE %s: %v\033[0m\n", order.Asset1, err)
			} else if !res.Success {
				fmt.Printf("\033[41m\033[37m[ALERTA CRÍTICO] FALHA DE CONSISTÊNCIA: COMPENSAÇÃO DE %s REJEITADA!\033[0m\n", order.Asset1)
			}
			return nil
		},
	)

	sg.AddStep(checkTTL, nil)

	sg.AddStep(
		func(ctx context.Context) error {
			buy := domain.TradeExecution{
				Asset:    order.Asset2,
				Price:    quoteRes.Price2,
				Quantity: order.Qty,
				Action:   domain.ActionBuy,
			}
			res, err := trade.Execute(buy)
			if err != nil || !res.Success {
				return fmt.Errorf("%w: %s", ErrPurchasingAsset2, order.Asset2)
			}
			return nil
		},
		func(ctx context.Context) error {
			sell := domain.TradeExecution{
				Asset:    order.Asset2,
				Price:    quoteRes.Price2,
				Quantity: order.Qty,
				Action:   domain.ActionSell,
			}
			res, err := trade.Execute(sell)
			if err != nil {
				fmt.Printf("\033[41m\033[37m[ALERTA CRÍTICO] ERRO DE REDE NA COMPENSAÇÃO DE %s: %v\033[0m\n", order.Asset2, err)
			} else if !res.Success {
				fmt.Printf("\033[41m\033[37m[ALERTA CRÍTICO] FALHA DE CONSISTÊNCIA: COMPENSAÇÃO DE %s REJEITADA!\033[0m\n", order.Asset2)
			}
			return nil
		},
	)

	return sg.Execute(context.Background())
}
