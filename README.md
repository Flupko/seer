# Seer: Real-Time Prediction Market Platform

Seer is a full-stack prediction market platform where users can bet on real-world outcomes (sports, politics, finance, etc.) with automated liquidity and live pricing driven by the LS-LMSR market maker (see [LS-LMSR Algorithm](./LS-LMSR.pdf)). The backend is built for financial correctness under concurrent load.

The project was initially planned for public launch and is currently paused while I focus on other projects and an internship search (seeking a backend engineering internship starting September 2026). I may resume development in the future. The focus of this repository is the backend, the frontend is an unfinished prototype.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.24, Echo v4 |
| Database | PostgreSQL 17 + TimescaleDB |
| In-memory / Pub/Sub | Valkey 9 (Redis-compatible) |
| WebSockets | gorilla/websocket (custom hub) |
| Decimal arithmetic | ericlagergren/decimal (arbitrary precision) |
| Auth | OIDC (Google, Twitch), bcrypt, server-side sessions |
| Infrastructure | Docker, Docker Compose, golang-migrate |
| Frontend | Next.js 15, React Query, Zustand, TailwindCSS |

---

## Design Decisions

**LS-LMSR Automated Market Maker.** Pricing is driven by the Logarithmic Market Scoring Rule with Liquidity Sensitivity. Unlike an order book, it provides continuous guaranteed liquidity: any bet is filled instantly at a deterministic price computed from the current quantity vector, with no counterparty required. Since the algorithm relies on exponentials and logarithms over financial values, the pricing engine uses arbitrary-precision decimal arithmetic (`ericlagergren/decimal` at 30 significant digits) instead of `float64`. Rounding still happens, but at a level that cannot meaningfully affect payouts.

**Double-Entry Ledger.** All money movement is modelled as double-entry bookkeeping (`ledger_accounts`, `ledger_transfers`, `ledger_entries`). PostgreSQL triggers make ledger entries immutable, a row-level `CHECK` constraint enforces balance correctness on every insert, and a stored function locks both accounts `FOR UPDATE` before any transfer to prevent double-spend under concurrent requests.

**Serializable transactions with retry.** Bet placement and cashout run at PostgreSQL's Serializable isolation level. Serialization failures and deadlocks are caught and retried automatically with exponential backoff. Idempotency keys on both transfers and bets guarantee exactly-once semantics across retries and network failures.

**WebSocket hub from scratch.** Real-time delivery (live prices, bet feed, chat, online count) runs on a custom hub built on `gorilla/websocket`. A single goroutine owns all shared state and processes interactions through typed Go channels, which avoids mutex contention on hot paths. Each client runs a `readPump` and `writePump` goroutine pair, with the write pump batching outgoing messages to cut down on syscalls. Rooms bridge to Valkey Pub/Sub, opening the door to horizontal scaling across multiple backend instances (still requires some refactoring to be fully operational).

**Rate limiting in Valkey.** Chat rate limiting uses a token bucket implemented as a Lua script running atomically on Valkey. It runs on every message sent, so keeping it in-memory avoids a DB round-trip, and the atomic script removes any race between reading and updating the bucket. Comment posting uses a simpler time-gated key per user per market, also in Valkey.

**TimescaleDB for price history.** Price samples are stored as a time-series, with continuous aggregate policies bucketing them into `5m`, `1h`, `4h`, and `24h` intervals. The frontend can query any timeframe without the backend scanning raw tick data, reducing latency and database load.

**Auth & sessions.** OAuth 2.0/OIDC via Google and Twitch, credential login with bcrypt, and server-side sessions stored as hashed tokens in PostgreSQL with full geolocation metadata (IP2Location). Scoped single-use tokens cover email verification, profile completion, and password reset.

---

## Repository Structure

```
seer/
├── backend/
│   ├── cmd/server/          # Entry point, dependency wiring
│   ├── internal/
│   │   ├── market/          # LS-LMSR engine, transaction manager, bet/cashout logic
│   │   ├── ws/              # WebSocket hub, client, router, Valkey pub/sub bridge
│   │   ├── finance/         # Ledger transfer execution
│   │   ├── chat/            # Chat manager with Redis-backed rate limiting
│   │   ├── repos/           # Database repositories
│   │   ├── handlers/        # HTTP and WebSocket handlers
│   │   └── middlewares/     # Authentication, role enforcement
│   ├── migrations/          # PostgreSQL schema (ledger, triggers, TimescaleDB aggregates)
│   └── lua/                 # Valkey Lua scripts (token bucket, cache management)
└── frontend/
    └── seer/                # Next.js 15 application
```

---

## Running Locally

A single `docker-compose.dev.yml` starts the full backend environment: PostgreSQL with TimescaleDB, Valkey, and a `golang-migrate` container that waits for the database health check and applies all migrations automatically. No manual setup required.

```bash
git clone https://github.com/Flupko/seer.git
cd seer

docker compose -f docker-compose.dev.yml up -d

cd backend && air        # backend on :4000 (requires Air)
cd frontend/seer && pnpm install && pnpm dev  # frontend on :3000
```

Environment variables are loaded from `backend/.env` (database DSN, Valkey address, OAuth credentials, session secret).
