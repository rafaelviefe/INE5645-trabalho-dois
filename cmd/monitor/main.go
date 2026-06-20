package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"trading-saga/pkg/config"
	"trading-saga/pkg/tcp"
)

type Event struct {
	Seq       int    `json:"seq"`
	Channel   string `json:"channel"`
	Data      any    `json:"data"`
	Timestamp string `json:"timestamp"`
}

type pullResponse struct {
	Events []Event `json:"events"`
}

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Erro ao carregar config: %v", err)
	}

	brokerAddr := cfg.Operation.BrokerAddr

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	file, err := os.Create("trace.jsonl")
	if err != nil {
		log.Fatalf("Erro ao criar trace.jsonl: %v", err)
	}
	defer file.Close()

	since := 0
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fmt.Println("\033[36m[MONITOR] Conectado ao broker. Pressione Ctrl+C para parar.\033[0m")

	for {
		select {
		case <-sigCh:
			fmt.Println("\n\033[36m[MONITOR] Encerrando...\033[0m")
			return
		case <-ticker.C:
			req := map[string]any{
				"action":  "pull",
				"channel": "saga",
				"since":   since,
			}

			payload, _ := json.Marshal(req)
			resPayload, err := tcp.SendRequest(brokerAddr, payload)
			if err != nil {
				continue
			}

			var resp pullResponse
			if err := json.Unmarshal(resPayload, &resp); err != nil {
				continue
			}

			for _, evt := range resp.Events {
				line, _ := json.Marshal(evt)
				file.Write(line)
				file.Write([]byte("\n"))
				fmt.Printf("\033[32m%s\033[0m\n", string(line))
				if evt.Seq > since {
					since = evt.Seq
				}
			}
		}
	}
}
