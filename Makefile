-include .env
export

.PHONY: run build lint smoke docker hooks

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

# Ставит git-хуки из lefthook.yml.
hooks:
	go tool -modfile=tools/go.mod lefthook install
