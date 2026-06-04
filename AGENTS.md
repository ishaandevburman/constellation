# AGENTS.md

## Project
Constellation — a distributed multi-agent runtime built on NATS (Go). Event-driven, NATS-first, observable by default.

## Commands
- `make up` — start NATS via Docker Compose
- `make down` — stop NATS
- `make run-publisher` — run the example publisher
- `make run-consumer` — run the example consumer
- `make run-agent` — run generic agent (NATS ↔ Ollama)
- `make run-ingress` — run CLI ingress (stdin → NATS → stdout)
- `make run-orchestrator` — run orchestrator (routes tasks between agents)
- `make run-observer` — run event observer (replay + live tail)
- `make run-p0test` — run P0 test binary directly
- `make test-p0` — run P0 messaging foundation test
- `make test-p1` — run P1 single-agent test
- `make test-p2` — run P2 multi-agent (router→worker→critic)
- `make build` — build all binaries
- `make clean` — clean build artifacts

## Ollama Agent Models (local)
| Model | Base | Role | Temp | Tokens |
|---|---|---|---|---|---|
| `constellation-router` | qwen2.5:0.5b-instruct | routing/coordination | 0.1 | 100 |
| `constellation-worker` | qwen2.5:0.5b-instruct | task execution | 0.2 | 500 |
| `constellation-critic` | qwen2.5:0.5b-instruct | verification | 0.1 | 300 |

Modelfiles in `models/agents/`. Created via `ollama create <name> -f models/agents/<name>`.

## Module
`github.com/ishaan/constellation`

## Key Packages
| Package | Path | Purpose |
|---|---|---|
| event | `pkg/event` | Core event model (Event struct, JSON serialization, type constants) |
| natsx | `pkg/natsx` | NATS client wrapper (connect, reconnect, JetStream stream management) |
| service | `pkg/service` | Service bootstrap (signal handling, graceful shutdown) |
| subjects | `internal/subjects` | Subject naming conventions and helpers |

## Conventions
- Events serialized as compact JSON via `encoding/json`
- Subject format: `constellation.event.<type>` or `constellation.event.<type>.<source>`
- JetStream streams use FileStorage for durability
- All timestamps in UTC
- UUID v4 for event IDs and correlation IDs

## Test Scripts (planned per phase)
| Phase | Script | What it tests |
|---|---|---|
| P0 | `scripts/test-p0.sh` | Messaging pipeline with router agent |
| P1 | `scripts/test-p1.sh` | Single request-response through worker |
| P2 | `scripts/test-p2.sh` | Multi-agent (router→worker→critic) |
| P3 | `scripts/test-p3.sh` | Fan-out/fan-in orchestration |
| P4 | `scripts/test-p4.sh` | Memory persistence across restarts |
| P5 | `scripts/test-p5.sh` | Tool invocation with calculator |
| P6 | `scripts/test-p6.sh` | Full observability + replay |
