package tests

import (
	"fmt"
	"log"
	"time"
	"trading-saga/pkg/adapters"
	"trading-saga/pkg/config"
	"trading-saga/pkg/handlers"
	"trading-saga/pkg/tcp"
)

// TestServerPorts defines ports for test services
type TestServerPorts struct {
	Quotation string
	Risk      string
	Purchase  string
}

// StartTestServices starts all three test services and returns cleanup function
func StartTestServices(ports TestServerPorts) (func(), error) {
	cfg := &config.Config{
		Quotation: config.QuotationConfig{
			Port:     ports.Quotation,
			TTLMs:    1000, // 1 second TTL for tests
			MinPrice: 10.0,
			MaxPrice: 100.0,
		},
		Risk: config.RiskConfig{
			Port:        ports.Risk,
			MinSleepMs:  10,
			MaxSleepMs:  50,
			SuccessRate: 100.0, // Always approve by default
		},
		Purchase: config.PurchaseConfig{
			Port:        ports.Purchase,
			MinSleepMs:  10,
			MaxSleepMs:  50,
			SuccessRate: 100.0, // Always succeed by default
		},
	}

	// Start quotation service
	quotationHandler := handlers.NewQuotationHandler(cfg)
	quotationServer := tcp.NewServer(cfg.Quotation.Port, 10, quotationHandler.Handle)
	go func() {
		if err := quotationServer.Start(); err != nil {
			log.Printf("Quotation server error: %v", err)
		}
	}()

	// Start risk service
	riskHandler := handlers.NewRiskHandler(cfg)
	riskServer := tcp.NewServer(cfg.Risk.Port, 10, riskHandler.Handle)
	go func() {
		if err := riskServer.Start(); err != nil {
			log.Printf("Risk server error: %v", err)
		}
	}()

	// Start purchase service
	purchaseHandler := handlers.NewTradeHandler(cfg)
	purchaseServer := tcp.NewServer(cfg.Purchase.Port, 10, purchaseHandler.Handle)
	go func() {
		if err := purchaseServer.Start(); err != nil {
			log.Printf("Purchase server error: %v", err)
		}
	}()

	// Give servers time to start
	time.Sleep(100 * time.Millisecond)

	// Return cleanup function
	cleanup := func() {
		// Note: In a real scenario, we would add Stop() methods to servers
		// For now, servers will be stopped when the test process exits
		fmt.Println("Test services cleanup completed")
	}

	return cleanup, nil
}

// GetTestClients returns initialized clients pointing to test services
func GetTestClients(ports TestServerPorts) (*adapters.QuotationClient, *adapters.RiskClient, *adapters.TradeClient) {
	// Extract host:port from addresses like ":9091" -> "localhost:9091"
	quotationAddr := "localhost" + ports.Quotation
	riskAddr := "localhost" + ports.Risk
	purchaseAddr := "localhost" + ports.Purchase

	quotationClient := adapters.NewQuotationClient(quotationAddr)
	riskClient := adapters.NewRiskClient(riskAddr)
	purchaseClient := adapters.NewTradeClient(purchaseAddr)

	return quotationClient, riskClient, purchaseClient
}
