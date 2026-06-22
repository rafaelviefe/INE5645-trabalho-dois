.PHONY: run-quotation run-risk run-purchase run-cli run-broker run-monitor run-all docker-up docker-down

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

run-all:
	@echo "Iniciando broker, quotation, risk, purchase em background..."
	go run cmd/broker/main.go &
	go run cmd/quotation/main.go &
	go run cmd/risk/main.go &
	go run cmd/trade/main.go &
	@sleep 2
	@echo "Iniciando monitor em background..."
	go run cmd/monitor/main.go >> monitor.log 2>&1 &
	@sleep 1
	@echo "Servicos prontos. Iniciando CLI..."
	@echo "trace.jsonl e monitor.log sendo escritos em tempo real"
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
