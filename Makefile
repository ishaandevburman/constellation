.PHONY: up down run-publisher run-consumer build clean

up:
	docker compose up -d

down:
	docker compose down

run-publisher:
	go run ./examples/publisher

run-consumer:
	go run ./examples/consumer

build:
	go build ./...

clean:
	rm -rf bin/
