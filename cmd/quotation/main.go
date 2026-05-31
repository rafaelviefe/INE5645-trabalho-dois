package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/tcp"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	handler := func(req []byte) []byte {
		var qReq domain.QuotationRequest
		if err := json.Unmarshal(req, &qReq); err != nil {
			return nil
		}

		p1 := cfg.Quotation.MinPrice + rand.Float64()*(cfg.Quotation.MaxPrice-cfg.Quotation.MinPrice)
		p2 := cfg.Quotation.MinPrice + rand.Float64()*(cfg.Quotation.MaxPrice-cfg.Quotation.MinPrice)

		qRes := domain.QuotationResponse{
			Price1: p1,
			Price2: p2,
			TTLms:  cfg.Quotation.TTLMs,
		}

		res, _ := json.Marshal(qRes)
		return res
	}

	log.Printf("Quotation Service running on %s\n", cfg.Quotation.Port)
	server := tcp.NewServer(cfg.Quotation.Port, 100, handler)
	if err := server.Start(); err != nil {
		log.Fatalf("%v\n", err)
	}
}