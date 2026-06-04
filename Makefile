.PHONY: up down run-publisher run-consumer run-agent run-ingress run-orchestrator run-p0test test-p0 test-p1 test-p2 build clean

up:
	docker compose up -d

down:
	docker compose down

run-publisher:
	go run ./examples/publisher

run-consumer:
	go run ./examples/consumer

run-agent:
	go run ./cmd/agent

run-ingress:
	go run ./cmd/ingress

run-orchestrator:
	go run ./cmd/orchestrator

run-p0test:
	go run ./cmd/p0test

test-p0:
	scripts/test-p0.sh

test-p1:
	scripts/test-p1.sh

test-p2:
	scripts/test-p2.sh

build:
	go build ./...

clean:
	rm -rf bin/
