package main

import (
	"log"
	"trading-saga/pkg/adapter/inbound"
	"trading-saga/pkg/adapter/outbound"
	"trading-saga/pkg/application/service"
	"trading-saga/pkg/config"
	"trading-saga/pkg/tcp"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	svc := service.NewQuotationService(cfg.Quotation)
	publisher := outbound.NewPublisher(cfg.Operation.BrokerAddr)
	handler := inbound.NewQuotationHandler(svc, publisher)

	log.Printf("Quotation Service running on %s\n", cfg.Quotation.Port)
	server := tcp.NewServer(cfg.Quotation.Port, 100, handler.Handle)
	if err := server.Start(); err != nil {
		log.Fatalf("%v\n", err)
	}
}
