package main

import (
	"log"
	"trading-saga/pkg/config"
	"trading-saga/pkg/handlers"
	"trading-saga/pkg/tcp"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	purchaseHandler := handlers.NewTradeHandler(cfg)

	log.Printf("Purchase Service running on %s\n", cfg.Purchase.Port)
	server := tcp.NewServer(cfg.Purchase.Port, 100, purchaseHandler.Handle)
	if err := server.Start(); err != nil {
		log.Fatalf("%v\n", err)
	}
}
