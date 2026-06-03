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

	quotationHandler := handlers.NewQuotationHandler(cfg)

	log.Printf("Quotation Service running on %s\n", cfg.Quotation.Port)
	server := tcp.NewServer(cfg.Quotation.Port, 100, quotationHandler.Handle)
	if err := server.Start(); err != nil {
		log.Fatalf("%v\n", err)
	}
}
