-include .env
export

.PHONY: run build lint smoke docker

IMAGE ?= teriyaki-sauce-service

run:
	go run ./cmd

build:
	go build -o bin/teriyaki-sauce-service ./cmd

lint:
	golangci-lint run --new ./...

smoke:
	./scripts/smoke.sh

docker:
	docker build -t $(IMAGE) .
