.PHONY: run-quotation run-risk run-purchase run-cli run-dev docker-up docker-down

run-quotation:
	go run cmd/quotation/main.go

run-risk:
	go run cmd/risk/main.go

run-purchase:
	go run cmd/trade/main.go

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

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
