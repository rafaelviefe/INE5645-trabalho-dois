FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY pkg/ ./pkg/
COPY cmd/ ./cmd/

ARG SERVICE_NAME

RUN go build -o /app/main ./cmd/${SERVICE_NAME}/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .
COPY config.docker.json ./config.json

CMD ["./main"]