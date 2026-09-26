-include .env
export

.PHONY: run build lint smoke

run:
	go run ./cmd

build:
	go build -o bin/teriyaki-sauce-service ./cmd

lint:
	golangci-lint run --new ./...

smoke:
	./scripts/smoke.sh
