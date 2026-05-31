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
		var rReq domain.RiskRequest
		if err := json.Unmarshal(req, &rReq); err != nil {
			return nil
		}

		sleepTime := rand.Intn(cfg.Risk.MaxSleepMs-cfg.Risk.MinSleepMs+1) + cfg.Risk.MinSleepMs
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)

		approved := (rand.Float64() * 100.0) <= cfg.Risk.SuccessRate

		rRes := domain.RiskResponse{
			Approved: approved,
		}

		res, _ := json.Marshal(rRes)
		return res
	}

	log.Printf("Risk Service running on %s\n", cfg.Risk.Port)
	server := tcp.NewServer(cfg.Risk.Port, 100, handler)
	if err := server.Start(); err != nil {
		log.Fatalf("%v\n", err)
	}
}