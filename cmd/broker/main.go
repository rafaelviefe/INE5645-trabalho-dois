package main

import (
	"encoding/json"
	"log"
	"sync"
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

type Broker struct {
	mu     sync.RWMutex
	events []Event
	seq    int
}

const maxEvents = 10000

func (b *Broker) publish(channel string, data any) Event {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.seq++
	evt := Event{
		Seq:       b.seq,
		Channel:   channel,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	b.events = append(b.events, evt)
	if len(b.events) > maxEvents {
		b.events = b.events[len(b.events)-maxEvents:]
	}
	return evt
}

func (b *Broker) pull(channel string, since int) []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []Event
	for _, e := range b.events {
		if e.Channel == channel && e.Seq > since {
			result = append(result, e)
		}
	}
	return result
}

func (b *Broker) Handle(raw []byte) []byte {
	var req map[string]any
	if err := json.Unmarshal(raw, &req); err != nil {
		return nil
	}

	action, _ := req["action"].(string)
	switch action {
	case "pub":
		channel, _ := req["channel"].(string)
		data := req["data"]
		evt := b.publish(channel, data)
		res, _ := json.Marshal(map[string]any{"ok": true, "seq": evt.Seq})
		return res

	case "pull":
		channel, _ := req["channel"].(string)
		since := 0
		if s, ok := req["since"].(float64); ok {
			since = int(s)
		}
		events := b.pull(channel, since)
		res, _ := json.Marshal(map[string]any{"events": events})
		return res
	}

	return nil
}

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Erro ao carregar config: %v", err)
	}

	broker := &Broker{}
	server := tcp.NewServer(cfg.Broker.Port, 100, broker.Handle)

	log.Printf("Broker running on %s\n", cfg.Broker.Port)
	if err := server.Start(); err != nil {
		log.Fatalf("%v\n", err)
	}
}
