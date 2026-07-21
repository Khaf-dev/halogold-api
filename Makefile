.PHONY: run build test tidy up down fmt

run:
	go run ./cmd/api

build:
	go build -o bin/halogold-api ./cmd/api

test:
	go test ./... -v

tidy:
	go mod tidy

fmt:
	gofmt -w .

up:
	docker compose up --build

down:
	docker compose down -v
