SHELL:=/bin/sh

PROJECT:=eon

.PHONY: up down run-ingress run-worker run-sample test

up:
	docker-compose up -d

down:
	docker-compose down

run-ingress:
	ADDR=:8080 DATABASE_URL="postgres://postgres:postgres@localhost:5432/eon?sslmode=disable" go run ./cmd/ingress

run-worker:
	DATABASE_URL="postgres://postgres:postgres@localhost:5432/eon?sslmode=disable" REDIS_URL="redis://localhost:6379" go run ./cmd/worker

run-sample: up
	# wait a bit for services to be healthy
	sleep 5
	ADDR=:8080 DATABASE_URL="postgres://postgres:postgres@localhost:5432/eon?sslmode=disable" go run ./cmd/ingress &
	INGRESS_PID=$$!
	DATABASE_URL="postgres://postgres:postgres@localhost:5432/eon?sslmode=disable" REDIS_URL="redis://localhost:6379" go run ./cmd/worker &
	WORKER_PID=$$!
	sleep 3
	curl -X POST http://localhost:8080/v1/events \
	  -H "Content-Type: application/json" \
	  -d '{"idempotency_key":"key-1","entity_key":"acct_123","type":"charge.created","payload":{"amount":100}}'
	curl -X POST http://localhost:8080/v1/events \
	  -H "Content-Type: application/json" \
	  -d '{"idempotency_key":"key-1","entity_key":"acct_123","type":"charge.created","payload":{"amount":100}}'
	kill $$INGRESS_PID $$WORKER_PID || true

test:
	go test ./...

