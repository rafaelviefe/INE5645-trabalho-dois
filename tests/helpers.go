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
		Operation: config.OperationConfig{
			BrokerAddr: "localhost:0",
		},
	}

	qs := service.NewQuotationService(cfg.Quotation)
	publisher := outbound.NewPublisher(cfg.Operation.BrokerAddr)
	quotationHandler := inbound.NewQuotationHandler(qs, publisher)
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

func StartCustomServices(quotationCfg config.QuotationConfig, riskCfg config.RiskConfig, purchaseCfg config.PurchaseConfig) (func(), error) {
	qs := service.NewQuotationService(quotationCfg)
	publisher := outbound.NewPublisher("localhost:0")
	quotationHandler := inbound.NewQuotationHandler(qs, publisher)
	quotationServer := tcp.NewServer(quotationCfg.Port, 10, quotationHandler.Handle)
	go func() {
		if err := quotationServer.Start(); err != nil {
			log.Printf("Quotation server error: %v", err)
		}
	}()

	rs := service.NewRiskService(riskCfg)
	riskHandler := inbound.NewRiskHandler(rs)
	riskServer := tcp.NewServer(riskCfg.Port, 10, riskHandler.Handle)
	go func() {
		if err := riskServer.Start(); err != nil {
			log.Printf("Risk server error: %v", err)
		}
	}()

	ts := service.NewTradeService(purchaseCfg)
	purchaseHandler := inbound.NewTradeHandler(ts)
	purchaseServer := tcp.NewServer(purchaseCfg.Port, 10, purchaseHandler.Handle)
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

	quotationClient := outbound.NewQuotationClient([]string{quotationAddr}, time.Second)
	riskClient := outbound.NewRiskClient([]string{riskAddr}, time.Second)
	purchaseClient := outbound.NewTradeClient([]string{purchaseAddr}, time.Second)

	return quotationClient, riskClient, purchaseClient
}
