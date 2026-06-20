.PHONY: run-quotation run-risk run-purchase run-cli run-dev run-broker run-monitor run-all docker-up docker-down docker-all

run-quotation:
	go run cmd/quotation/main.go

run-risk:
	go run cmd/risk/main.go

run-purchase:
	go run cmd/trade/main.go

run-broker:
	go run cmd/broker/main.go

run-monitor:
	go run cmd/monitor/main.go

run-cli:
	go run cmd/operation/main.go

run-dev:
	@echo "Iniciando servicos em background..."
	go run cmd/quotation/main.go &
	go run cmd/risk/main.go &
	go run cmd/trade/main.go &
	@sleep 2
	@echo "Servicos prontos. Iniciando CLI..."
	go run cmd/operation/main.go; \
		echo "Encerrando servicos..."; \
		kill %1 2>/dev/null; \
		kill %2 2>/dev/null; \
		kill %3 2>/dev/null; \
		wait 2>/dev/null; \
		echo "Servicos encerrados."

run-all:
	@echo "Iniciando broker, quotation, risk, purchase em background..."
	go run cmd/broker/main.go &
	go run cmd/quotation/main.go &
	go run cmd/risk/main.go &
	go run cmd/trade/main.go &
	@sleep 2
	@echo "Iniciando monitor em background (stdout -> monitor.log)..."
	go run cmd/monitor/main.go >> monitor.log 2>&1 &
	@sleep 1
	@echo "Servicos prontos. Iniciando CLI..."
	@echo "Monitor salvando em trace.jsonl e monitor.log"
	go run cmd/operation/main.go; \
		echo "Encerrando servicos..."; \
		kill %1 2>/dev/null; \
		kill %2 2>/dev/null; \
		kill %3 2>/dev/null; \
		kill %4 2>/dev/null; \
		kill %5 2>/dev/null; \
		kill %6 2>/dev/null; \
		wait 2>/dev/null; \
		echo "Servicos encerrados."

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down

docker-all:
	@echo "Subindo broker + servicos + replicas no Docker..."
	docker compose up --build -d
	@sleep 3
	@echo "Iniciando monitor local (stdout -> monitor.log)..."
	go run cmd/monitor/main.go >> monitor.log 2>&1 &
	@sleep 1
	@echo "Iniciando CLI..."
	go run cmd/operation/main.go; \
		echo "Encerrando..."; \
		kill %1 2>/dev/null; \
		docker compose down; \
		wait 2>/dev/null; \
		echo "Finalizado."
