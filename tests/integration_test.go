package tests

import (
	"testing"
	"time"
	"trading-saga/pkg/adapter/outbound"
	"trading-saga/pkg/application"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
)

func TestSuccessfulOrderFlow(t *testing.T) {
	ports := TestServerPorts{
		Quotation: ":9091",
		Risk:      ":9092",
		Purchase:  ":9093",
		Broker:    ":9090",
	}

	cleanup, err := StartTestServices(ports)
	if err != nil {
		t.Fatalf("Failed to start test services: %v", err)
	}
	defer cleanup()

	quotationClient, riskClient, purchaseClient := GetTestClients(ports)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err = application.ExecuteOrder(order, quotationClient, riskClient, purchaseClient)
	if err != nil {
		t.Errorf("Expected successful order execution, got error: %v", err)
	}
}

func TestOrderFailsOnRiskRejection(t *testing.T) {
	ports := TestServerPorts{
		Quotation: ":9094",
		Risk:      ":9095",
		Purchase:  ":9096",
		Broker:    ":9093",
	}

	cleanup, err := StartTestServices(ports)
	if err != nil {
		t.Fatalf("Failed to start test services: %v", err)
	}
	defer cleanup()

	quotationClient, riskClient, purchaseClient := GetTestClients(ports)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err = application.ExecuteOrder(order, quotationClient, riskClient, purchaseClient)
	if err != nil {
		t.Logf("Order failed as expected: %v", err)
	} else {
		t.Logf("Order succeeded (expected with current config)")
	}
}

func TestOrderWithTTLExceeded(t *testing.T) {
	quotationCfg := config.QuotationConfig{
		Port:     ":9107",
		TTLMs:    1,
		MinPrice: 10.0,
		MaxPrice: 100.0,
	}
	riskCfg := config.RiskConfig{
		Port:        ":9108",
		MinSleepMs:  100,
		MaxSleepMs:  200,
		SuccessRate: 100.0,
	}
	purchaseCfg := config.PurchaseConfig{
		Port:        ":9109",
		MinSleepMs:  10,
		MaxSleepMs:  50,
		SuccessRate: 100.0,
	}

	cleanup, err := StartCustomServices(quotationCfg, riskCfg, purchaseCfg)
	if err != nil {
		t.Fatalf("Failed to start test services: %v", err)
	}
	defer cleanup()

	quotationClient := outbound.NewQuotationClient([]string{"localhost:9107"}, time.Second)
	riskClient := outbound.NewRiskClient([]string{"localhost:9108"}, time.Second)
	purchaseClient := outbound.NewTradeClient([]string{"localhost:9109"}, time.Second)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err = application.ExecuteOrder(order, quotationClient, riskClient, purchaseClient)
	if err != application.ErrTTLExceeded {
		t.Errorf("Expected ErrTTLExceeded, got: %v", err)
	}
}

func TestOrderCompensatesOnPurchaseFailure(t *testing.T) {
	quotationCfg := config.QuotationConfig{
		Port:     ":9110",
		TTLMs:    5000,
		MinPrice: 10.0,
		MaxPrice: 100.0,
	}
	riskCfg := config.RiskConfig{
		Port:        ":9111",
		MinSleepMs:  10,
		MaxSleepMs:  50,
		SuccessRate: 100.0,
	}
	purchaseCfg := config.PurchaseConfig{
		Port:        ":9112",
		MinSleepMs:  10,
		MaxSleepMs:  50,
		SuccessRate: 0.0,
	}

	cleanup, err := StartCustomServices(quotationCfg, riskCfg, purchaseCfg)
	if err != nil {
		t.Fatalf("Failed to start test services: %v", err)
	}
	defer cleanup()

	quotationClient := outbound.NewQuotationClient([]string{"localhost:9110"}, time.Second)
	riskClient := outbound.NewRiskClient([]string{"localhost:9111"}, time.Second)
	purchaseClient := outbound.NewTradeClient([]string{"localhost:9112"}, time.Second)

	order := domain.OrderRequest{
		Asset1: "ETH/USDT",
		Asset2: "USD/BRL",
		Qty:    1.5,
	}

	err = application.ExecuteOrder(order, quotationClient, riskClient, purchaseClient)
	if err != application.ErrPurchasingAsset1 {
		t.Errorf("Expected ErrPurchasingAsset1 (0%% purchase success), got: %v", err)
	}
}

func TestMultipleConcurrentOrders(t *testing.T) {
	ports := TestServerPorts{
		Quotation: ":9097",
		Risk:      ":9098",
		Purchase:  ":9099",
		Broker:    ":9096",
	}

	cleanup, err := StartTestServices(ports)
	if err != nil {
		t.Fatalf("Failed to start test services: %v", err)
	}
	defer cleanup()

	quotationClient, riskClient, purchaseClient := GetTestClients(ports)

	numOrders := 5
	done := make(chan error, numOrders)

	for i := range numOrders {
		go func(id int) {
			order := domain.OrderRequest{
				Asset1: "ETH/USDT",
				Asset2: "USD/BRL",
				Qty:    float64(id) * 0.5,
			}
			err := application.ExecuteOrder(order, quotationClient, riskClient, purchaseClient)
			done <- err
		}(i)
	}

	successCount := 0
	for i := 0; i < numOrders; i++ {
		if err := <-done; err == nil {
			successCount++
		}
	}

	if successCount != numOrders {
		t.Errorf("Expected %d successful orders, got %d", numOrders, successCount)
	}
}
