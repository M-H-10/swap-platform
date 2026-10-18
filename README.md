# Reputation Swap Platform

A platform where a project owner and a freelancer sign a **reputation swap**:
both lock a money guarantee in escrow, the project's success probability
sets the terms, and the result of the project moves the money and the
reputation of both parties.

Graduation project, version 2: Go + PostgreSQL + RabbitMQ + Python ML + React.

![CI](https://github.com/M-H-10/swap-platform/actions/workflows/ci.yml/badge.svg)

## Architecture

```mermaid
flowchart LR
    UI["React + TypeScript<br/>(frontend)"] -- "REST /api<br/>JWT + refresh cookie" --> API["Go + Fiber<br/>(backend-go)"]
    API -- SQL --> DB[("PostgreSQL")]
    API -- "ml.predict" --> MQ[["RabbitMQ"]]
    MQ -- "ml.predict" --> ML["Python worker<br/>RandomForest"]
    ML -- "ml.result" --> MQ
    MQ -- "ml.result" --> API
```

The Go server follows Clean Architecture:
`handler` (HTTP) → `service` (business rules) → `repository` (SQL), with the
rules themselves in `domain`, which imports no framework at all.

| Part | Technology | What it does |
|---|---|---|
| `backend-go/` | Go, Fiber v3, pgx, golang-migrate | REST API, auth, scoring formula, wallets and escrow, swaps, reputation |
| `ml-service/` | Python, scikit-learn, pika | Second opinion on the success probability (RandomForest) |
| `frontend/` | React 19, TypeScript, Vite, Recharts | The website |
| `docker-compose.yml` | PostgreSQL 18, RabbitMQ 4 | Local infrastructure |

Key rules:

* Money is stored as integer **kopecks**, percentages as **basis points** (10000 = 100%): no rounding errors.
* Every money movement happens inside one database transaction and is written to an append-only ledger.
* Access token (15 min) in memory, refresh token (7 days) in an HttpOnly cookie, rotated on every use; reuse of an old one ends all sessions.
* Reviews are weighted by the author's trust (account age and completed swaps), so fake new accounts barely move a rating.

## Run it locally

You need Docker, Go 1.26+, Python 3.12+ and Node.js 22+.

```bash
# 1. Infrastructure (copy .env.example to .env and set passwords first)
docker compose up -d

# 2. Go server (copy backend-go/.env.example to backend-go/.env first)
cd backend-go
go run ./cmd/migrate up     # create the tables
go run ./cmd/seed           # optional: demo users, projects and swaps
go run ./cmd/api            # http://localhost:5002

# 3. ML worker (optional; copy ml-service/.env.example to ml-service/.env)
cd ml-service
python -m venv .venv && .venv/bin/pip install -r requirements.txt   # Windows: .venv\Scripts\pip
python train.py             # creates models/model.joblib
python worker.py

# 4. Website
cd frontend
npm install
npm run dev                 # http://localhost:5173
```

Demo accounts created by `go run ./cmd/seed` (development only):
`alice@example.com`, `boris@example.com`, `carol@example.com`, `dima@example.com`,
all with the password `demo12345`.

## API documentation

With the server running, open **http://localhost:5002/api/docs** (Swagger UI).
The description lives in `backend-go/internal/handler/openapi.yaml`, and a test
fails if a route is added without documenting it.

## Tests

```bash
cd backend-go && go test ./...      # unit tests
# integration test against a real (separate!) database:
TEST_DATABASE_URL=postgres://user:pass@localhost:5433/swap_test?sslmode=disable go test ./internal/app/

cd ml-service && pytest
cd frontend && npm test
```

GitHub Actions runs all of them, plus formatting, vet, lint and the build,
on every push and pull request (`.github/workflows/ci.yml`).