.PHONY: up down run-publisher run-consumer run-agent test-p0 build clean

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

test-p0:
	scripts/test-p0.sh

build:
	go build ./...

clean:
	rm -rf bin/
