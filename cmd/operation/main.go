package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"trading-saga/pkg/adapter/outbound"
	"trading-saga/pkg/application/saga"
	"trading-saga/pkg/config"
	"trading-saga/pkg/domain"
	"trading-saga/pkg/domain/ports"
)

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Fatalf("Erro ao carregar config: %v", err)
	}

	poolCooldown := time.Duration(cfg.Operation.PoolCooldownMs) * time.Millisecond

	var quotationClient ports.QuotationClient = outbound.NewQuotationClient(cfg.Operation.QuotationAddrs, poolCooldown)
	var riskClient ports.RiskClient = outbound.NewRiskClient(cfg.Operation.RiskAddrs, poolCooldown)
	var purchaseClient ports.TradeClient = outbound.NewTradeClient(cfg.Operation.PurchaseAddrs, poolCooldown)

	orchestrator := saga.NewOrchestrator(quotationClient, riskClient, purchaseClient)

	fmt.Println("\033[36m===================================================\033[0m")
	fmt.Println("\033[36m   SISTEMA DE OPERAÇÃO - TRADING SAGA (CLI)        \033[0m")
	fmt.Println("\033[36m   Arthur Schurhaus, Rafael Vieira e Uriel Jaloto  \033[0m")
	fmt.Println("\033[36m===================================================\033[0m")
	fmt.Println("Comandos disponíveis:")
	fmt.Println("  order <ativo1> <ativo2> <quantidade>")
	fmt.Println("  batch <ativo1> <ativo2> <quantidade> <n_ordens>")
	fmt.Println("  exemplo: batch ETH/USDT USD/BRL 1.5 5")
	fmt.Println("  exit")
	fmt.Println("\033[36m===================================================\033[0m")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n\033[33m> \033[0m")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		cmd := parts[0]

		if cmd == "exit" {
			fmt.Println("Encerrando Sistema de Operação...")
			break
		}

		if cmd == "order" {
			if len(parts) != 4 {
				fmt.Println("\033[31mUso incorreto. Exemplo: order ETH/USDT USD/BRL 1.5\033[0m")
				continue
			}

			asset1 := domain.Asset(parts[1])
			asset2 := domain.Asset(parts[2])
			qty, err := strconv.ParseFloat(parts[3], 64)
			if err != nil {
				fmt.Println("\033[31mQuantidade inválida.\033[0m")
				continue
			}

			req := domain.OrderRequest{
				Asset1: asset1,
				Asset2: asset2,
				Qty:    qty,
			}

			fmt.Println("\033[34m[SAGA] Iniciando transação distribuída...\033[0m")
			err = orchestrator.ExecuteOrder(req)

			if err != nil {
				fmt.Printf("\033[31m[SAGA] OPERAÇÃO ABORTADA / ROLLBACK: %v\033[0m\n", err)
			} else {
				fmt.Println("\033[32m[SAGA] OPERAÇÃO CONCLUÍDA COM SUCESSO!\033[0m")
			}
		} else if cmd == "batch" {
			if len(parts) != 5 {
				fmt.Println("\033[31mUso incorreto. Exemplo: batch ETH/USDT USD/BRL 1.5 5\033[0m")
				continue
			}

			asset1 := domain.Asset(parts[1])
			asset2 := domain.Asset(parts[2])
			qty, err := strconv.ParseFloat(parts[3], 64)
			if err != nil {
				fmt.Println("\033[31mQuantidade inválida.\033[0m")
				continue
			}

			count, err := strconv.Atoi(parts[4])
			if err != nil || count <= 0 {
				fmt.Println("\033[31mQuantidade de ordens inválida.\033[0m")
				continue
			}

			fmt.Printf("\033[34m[SAGA] Iniciando lote de %d transações concorrentes...\033[0m\n", count)

			var wg sync.WaitGroup
			for i := 0; i < count; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					req := domain.OrderRequest{
						Asset1: asset1,
						Asset2: asset2,
						Qty:    qty,
					}
					err := orchestrator.ExecuteOrder(req)
					if err != nil {
						fmt.Printf("\033[31m[SAGA - Ordem %d] ABORTADA: %v\033[0m\n", id, err)
					} else {
						fmt.Printf("\033[32m[SAGA - Ordem %d] SUCESSO!\033[0m\n", id)
					}
				}(i + 1)
			}
			wg.Wait()
			fmt.Println("\033[34m[SAGA] Lote de transações finalizado.\033[0m")
		} else {
			fmt.Println("\033[31mComando desconhecido. Use 'order', 'batch' ou 'exit'.\033[0m")
		}
	}
}
