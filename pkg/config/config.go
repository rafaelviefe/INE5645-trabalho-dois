package config

import (
	"encoding/json"
	"os"
)

type QuotationConfig struct {
	Port     string  `json:"port"`
	TTLMs    int     `json:"ttl_ms"`
	MinPrice float64 `json:"min_price"`
	MaxPrice float64 `json:"max_price"`
}

type RiskConfig struct {
	Port        string  `json:"port"`
	MinSleepMs  int     `json:"min_sleep_ms"`
	MaxSleepMs  int     `json:"max_sleep_ms"`
	SuccessRate float64 `json:"success_rate"`
}

type PurchaseConfig struct {
	Port        string  `json:"port"`
	MinSleepMs  int     `json:"min_sleep_ms"`
	MaxSleepMs  int     `json:"max_sleep_ms"`
	SuccessRate float64 `json:"success_rate"`
}

type BrokerConfig struct {
	Port string `json:"port"`
}

type OperationConfig struct {
	QuotationAddrs []string `json:"quotation_addrs"`
	RiskAddrs      []string `json:"risk_addrs"`
	PurchaseAddrs  []string `json:"purchase_addrs"`
	BrokerAddr     string   `json:"broker_addr"`
	PoolCooldownMs int      `json:"pool_cooldown_ms"`
}

type Config struct {
	Quotation QuotationConfig `json:"quotation"`
	Risk      RiskConfig      `json:"risk"`
	Purchase  PurchaseConfig  `json:"purchase"`
	Broker    BrokerConfig    `json:"broker"`
	Operation OperationConfig `json:"operation"`
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	if err := json.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
