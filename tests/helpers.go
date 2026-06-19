package tests

import (
	"fmt"
	"log"
	"time"
	"trading-saga/pkg/adapter/inbound"
	"trading-saga/pkg/adapter/outbound"
	"trading-saga/pkg/application/service"
	"trading-saga/pkg/config"
	"trading-saga/pkg/tcp"
)

type TestServerPorts struct {
	Quotation string
	Risk      string
	Purchase  string
}

func StartTestServices(ports TestServerPorts) (func(), error) {
	cfg := &config.Config{
		Quotation: config.QuotationConfig{
			Port:     ports.Quotation,
			TTLMs:    1000,
			MinPrice: 10.0,
			MaxPrice: 100.0,
		},
		Risk: config.RiskConfig{
			Port:        ports.Risk,
			MinSleepMs:  10,
			MaxSleepMs:  50,
			SuccessRate: 100.0,
		},
		Purchase: config.PurchaseConfig{
			Port:        ports.Purchase,
			MinSleepMs:  10,
			MaxSleepMs:  50,
			SuccessRate: 100.0,
		},
	}

	qs := service.NewQuotationService(cfg.Quotation)
	quotationHandler := inbound.NewQuotationHandler(qs)
	quotationServer := tcp.NewServer(cfg.Quotation.Port, 10, quotationHandler.Handle)
	go func() {
		if err := quotationServer.Start(); err != nil {
			log.Printf("Quotation server error: %v", err)
		}
	}()

	rs := service.NewRiskService(cfg.Risk)
	riskHandler := inbound.NewRiskHandler(rs)
	riskServer := tcp.NewServer(cfg.Risk.Port, 10, riskHandler.Handle)
	go func() {
		if err := riskServer.Start(); err != nil {
			log.Printf("Risk server error: %v", err)
		}
	}()

	ts := service.NewTradeService(cfg.Purchase)
	purchaseHandler := inbound.NewTradeHandler(ts)
	purchaseServer := tcp.NewServer(cfg.Purchase.Port, 10, purchaseHandler.Handle)
	go func() {
		if err := purchaseServer.Start(); err != nil {
			log.Printf("Purchase server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	cleanup := func() {
		fmt.Println("Test services cleanup completed")
	}

	return cleanup, nil
}

func GetTestClients(ports TestServerPorts) (*outbound.QuotationClient, *outbound.RiskClient, *outbound.TradeClient) {
	quotationAddr := "localhost" + ports.Quotation
	riskAddr := "localhost" + ports.Risk
	purchaseAddr := "localhost" + ports.Purchase

	quotationClient := outbound.NewQuotationClient(quotationAddr)
	riskClient := outbound.NewRiskClient(riskAddr)
	purchaseClient := outbound.NewTradeClient(purchaseAddr)

	return quotationClient, riskClient, purchaseClient
}
