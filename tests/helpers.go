package tests

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
	"trading-saga/pkg/adapter/inbound"
	"trading-saga/pkg/adapter/outbound"
	"trading-saga/pkg/application/service"
	"trading-saga/pkg/config"
	"trading-saga/pkg/tcp"
)

var nextBrokerPort int32 = 9900

type TestServerPorts struct {
	Quotation string
	Risk      string
	Purchase  string
	Broker    string
}

type testBroker struct {
	mu     sync.Mutex
	events []map[string]any
	seq    int
}

func newTestBroker() *testBroker {
	return &testBroker{events: make([]map[string]any, 0, 100)}
}

func (b *testBroker) Handle(raw []byte) []byte {
	var req map[string]any
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil
	}

	action, _ := req["action"].(string)
	switch action {
	case "pub":
		b.mu.Lock()
		b.seq++
		evt := map[string]any{
			"seq":       b.seq,
			"channel":   req["channel"],
			"data":      req["data"],
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		}
		b.events = append(b.events, evt)
		b.mu.Unlock()
		res, _ := json.Marshal(map[string]any{"ok": true, "seq": b.seq})
		return res
	case "pull":
		channel, _ := req["channel"].(string)
		since := 0
		if s, ok := req["since"].(float64); ok {
			since = int(s)
		}
		b.mu.Lock()
		var result []map[string]any
		for _, e := range b.events {
			if e["channel"] == channel {
				if seq, _ := e["seq"].(int); seq > since {
					result = append(result, e)
				}
			}
		}
		b.mu.Unlock()
		res, _ := json.Marshal(map[string]any{"events": result})
		return res
	}
	return nil
}

func startTestBroker(port string) (*testBroker, string, func()) {
	broker := newTestBroker()
	server := tcp.NewServer(port, 10, broker.Handle)
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Test broker error: %v", err)
		}
	}()
	return broker, port, func() {}
}

func StartTestServices(ports TestServerPorts) (func(), error) {
	_, _, stopBroker := startTestBroker(ports.Broker)

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
			BrokerAddr: "localhost" + ports.Broker,
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
		stopBroker()
		fmt.Println("Test services cleanup completed")
	}

	return cleanup, nil
}

func StartCustomServices(quotationCfg config.QuotationConfig, riskCfg config.RiskConfig, purchaseCfg config.PurchaseConfig) (func(), error) {
	brokerPort := atomic.AddInt32(&nextBrokerPort, 1)
	brokerAddr := fmt.Sprintf(":%d", brokerPort)
	_, _, stopBroker := startTestBroker(brokerAddr)

	qs := service.NewQuotationService(quotationCfg)
	publisher := outbound.NewPublisher("localhost" + brokerAddr)
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
		stopBroker()
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
