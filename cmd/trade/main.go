package main

import (
	"log"
	"trading-saga/pkg/adapter/inbound"
	"trading-saga/pkg/application/service"
	"trading-saga/pkg/config"
	"trading-saga/pkg/tcp"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	svc := service.NewTradeService(cfg.Purchase)
	handler := inbound.NewTradeHandler(svc)

	log.Printf("Purchase Service running on %s\n", cfg.Purchase.Port)
	server := tcp.NewServer(cfg.Purchase.Port, 100, handler.Handle)
	if err := server.Start(); err != nil {
		log.Fatalf("%v\n", err)
	}
}
