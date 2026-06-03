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

	riskHandler := handlers.NewRiskHandler(cfg)

	log.Printf("Risk Service running on %s\n", cfg.Risk.Port)
	server := tcp.NewServer(cfg.Risk.Port, 100, riskHandler.Handle)
	if err := server.Start(); err != nil {
		log.Fatalf("%v\n", err)
	}
}
