package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"
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
		var pReq domain.PurchaseRequest
		if err := json.Unmarshal(req, &pReq); err != nil {
			return nil
		}

		sleepTime := rand.Intn(cfg.Purchase.MaxSleepMs-cfg.Purchase.MinSleepMs+1) + cfg.Purchase.MinSleepMs
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)

		success := true
		if pReq.Action == domain.ActionBuy {
			success = (rand.Float64() * 100.0) <= cfg.Purchase.SuccessRate
		}

		pRes := domain.PurchaseResponse{
			Success: success,
		}

		res, _ := json.Marshal(pRes)
		return res
	}

	log.Printf("Purchase Service running on %s\n", cfg.Purchase.Port)
	server := tcp.NewServer(cfg.Purchase.Port, 100, handler)
	if err := server.Start(); err != nil {
		log.Fatalf("%v\n", err)
	}
}