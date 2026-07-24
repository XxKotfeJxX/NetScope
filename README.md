# NetScope

Concurrent network diagnostics dashboard built with Go and TypeScript.

[API docs](api/openapi.yaml) · [Architecture](#architecture) · [Roadmap](#roadmap)

NetScope accepts one explicit hostname, URL, or IP address and runs focused
diagnostic checks with bounded concurrency, per-check timeouts, cancellation,
and live progress. It is a diagnostics dashboard—not a subnet scanner,
vulnerability scanner, packet sniffer, or Nmap replacement.

> `main` currently contains the **v0.0.1 bootstrap**. The `dev` branch contains
> the v0.1.0 release candidate: target input → persisted run → concurrent DNS,
> TCP, HTTP, and TLS probes → SSE progress → browser result.

## Features

- Go `net/http` API with chi routing and graceful shutdown
- PostgreSQL 18.4 connection pooling through pgx
- Structured JSON logs with request IDs
- DNS A, AAAA, CNAME, MX, NS, TXT, and PTR inspection
- Explicit TCP connection checks with typed failure classification
- HTTP redirects, status, content metadata, hashing, and phase timings
- TLS protocol, cipher, certificate chain, hostname, and expiry validation
- Bounded run workers, per-probe timeout, cancellation, and SSE events
- Persisted run details, history, and JSON export
- Responsive React 19 + TypeScript diagnostic dashboard
- Docker Compose for database-only or full-stack development
- Backend and frontend test, lint, typecheck, race, and build checks

## Architecture

```text
React + TypeScript
        │
        │ REST + SSE
        ▼
Go HTTP API ──► diagnostic queue ──► bounded worker pool
        │                                  │
        │                                  ├─ DNS
        │                                  ├─ TCP
        │                                  ├─ HTTP
        │                                  └─ TLS
        ▼
PostgreSQL
```

The repository is a modular monolith. Transport, diagnostics, probes, target
policy, and storage have explicit package boundaries while deploying as one Go
process.

## Technology

| Layer | Stack |
| --- | --- |
| API | Go 1.26.5, net/http, chi, slog |
| Data | PostgreSQL 18.4, pgx/pgxpool |
| Web | Node.js 24 LTS, React, TypeScript, Vite, TanStack Query |
| Delivery | Docker, Docker Compose, GitHub Actions |

## Quick start

Requirements: Go 1.26.5, Node.js 24 LTS, pnpm 10, and Docker.

```sh
cp .env.example .env
docker compose up -d postgres
go run ./cmd/netscope
pnpm --dir web install
pnpm --dir web dev
```

Open `http://localhost:5173`. The API is available at
`http://localhost:8080`; `GET /healthz` checks the process and `GET /readyz`
checks PostgreSQL.

To build the full stack in containers:

```sh
docker compose --profile full up --build
```

## Configuration

Configuration is environment-only. Copy `.env.example`, provide
`DATABASE_URL`, and adjust concurrency or network-policy values as needed.
Invalid critical configuration stops startup. `.env` is ignored and must never
be committed.

Public deployments must set `NETWORK_POLICY=public`; this mode blocks private,
loopback, link-local, multicast, unspecified, and metadata targets. Local mode
is only suitable for a trusted development machine.

## API documentation

The OpenAPI 3.1 contract is maintained at [`api/openapi.yaml`](api/openapi.yaml)
and is the source of truth for endpoint schemas and enums.

## Testing

```sh
make check
```

Equivalent commands are `go test ./...`, `go test -race ./...`,
`pnpm --dir web lint`, `pnpm --dir web typecheck`, `pnpm --dir web test`, and
`pnpm --dir web build`.

## Git workflow

- `main` contains deployable versions only.
- `dev` is the integration branch.
- Feature branches start from `dev` and return through pull requests.
- Commits follow Conventional Commits; PRs are squash-merged after CI passes.
- Releases move from `dev` to `main` in a dedicated release PR.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the complete workflow.

## Known limitations

The current `dev` build implements the v0.1.0 core probes. ICMP ping and
traceroute remain intentionally disabled until v0.2.0 because their permission
and platform models differ. NetScope has no authentication yet, so a public
deployment still requires an external access-control layer.

## Roadmap

- **v0.1.0:** DNS, TCP, HTTP, TLS, worker pool, cancellation, SSE, history,
  JSON export, and PostgreSQL persistence
- **v0.2.0:** ICMP ping, traceroute, comparisons, CSV, capabilities, metrics
- **v1.0.0:** accounts, scheduling, notifications, public demo, telemetry

## License

[MIT](LICENSE)
