
# Eon - Distributed Events-Driven Payments Platform


## Overview   


Eon is my implementation of a distributed payment platform via events designed with the intent of preventing duplicate transactions. I made sure to build metrics and observability tools as well.

I was inspired by watching a video about how Uber handles millions of transactions, and wanted to build a small-scale ledger to do the same.

I created this project out of boredom after my Networks and Distributed processing class, in an effort to apply concepts like distributed event processing and fault tolerance by handling crashes and duplicates.  
  
If you want an in-depth explanation of how it works, I am in the process of writing a paper and creating a YouTube video to showcase it.

## Key features
- Dockerized infrastructure makes it easily reproducible
- Event-driven architecture with smart partitioned ordering for every account
- Processing is crash-safe by using checkpoints and a durable state
- PostgresSQL and Redis provide us with a robust deduplication layer
- A transactional outbox (think email) provides us with guaranteed delivery
- We can replay events and recovery tools to view failures and fix
- Observability tools and metrics are also used. ex: uptime stats 

## Architecture 
- Client → Ingress API → Event Log → Workers → Ledger + Outbox → Downstream systems
## Tech stack



Language: Go

  

Databases: PostgreSQL, Redis (this was my first time using redis)

  

Infrastructure: Docker, Docker Compose used

  

Observability: Prometheus via Telemetry

  

APIs: HTTP REST (pinging back and forth between logs, testing dedupe)

  

AI: GPT5 and Claude Opus 4.5 assisted well with bug fixes, feature requests

  

Concurrency: Goroutines

## Compile instructions

 Start dependencies:

```
docker-compose up -d
```

Run services:

```
make run-ingress
make run-worker
```

Send a sample event:

```
curl -X POST http://localhost:8080/v1/events \
  -H "Content-Type: application/json" \
  -d '{"idempotency_key":"key-1","entity_key":"acct_123","type":"charge.created","payload":{"amount":100}}'
```

