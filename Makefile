.PHONY: run-quotation run-risk run-purchase run-cli docker-up docker-down

run-quotation:
	go run cmd/quotation/main.go

run-risk:
	go run cmd/risk/main.go

run-purchase:
	go run cmd/purchase/main.go

run-cli:
	go run cmd/operation/main.go

docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down