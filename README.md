# NetScope

Technical network diagnostics built with Go and TypeScript.

[API docs](api/openapi.yaml) · [v1.0.0 release notes](docs/releases/v1.0.0.md) ·
[Architecture](#architecture) ·
[API security](docs/operations/api-security.md) ·
[Production deployment](docs/operations/production-deployment.md) ·
[Backup and restore](docs/operations/backup-restore.md) ·
[Demo workspace](docs/operations/demo-workspace.md) · [Roadmap](#roadmap)

NetScope accepts one explicit hostname, URL, or IP address and runs focused
diagnostic checks with bounded concurrency, per-check timeouts, cancellation,
and live progress. It is a focused diagnostic tool—not a subnet scanner,
vulnerability scanner, packet sniffer, or Nmap replacement.

`main` contains deployable releases. Ongoing integration happens in `dev` and
reaches `main` only through a dedicated release pull request.

## Features

- Go `net/http` API with chi routing and graceful shutdown
- PostgreSQL 18.4 connection pooling through pgx
- Structured JSON logs with request IDs
- DNS A, AAAA, CNAME, MX, NS, TXT, and PTR inspection
- Explicit TCP connection checks with typed failure classification
- HTTP redirects, status, content metadata, hashing, and phase timings
- TLS protocol, cipher, certificate chain, hostname, and expiry validation
- Capability-detected ICMP ping and hop-by-hop traceroute
- IPv4/IPv6 preference, custom TCP ports, HTTP method, and redirect policy
- Bounded run workers, per-probe timeout, cancellation, and SSE events
- Persisted run details, history, JSON/CSV export, exact reruns, and comparison
- Named monitoring targets with tags, configurable checks, and intervals
- Scheduled availability, latency, and TLS-expiry history
- Consecutive-failure thresholds, pause/resume, and maintenance windows
- Recovery and outage alerts through SSRF-protected webhooks or SMTP email
- Argon2id accounts, opaque sessions, and isolated multi-workspace tenancy
- Owner, Admin, Operator, and Viewer access control with shared target history
- Workspace members, scoped API keys, and an immutable control-plane audit log
- Team report comments and revocable, expiring public read-only report links
- Responsive React 19 + TypeScript interface with persisted light/dark themes
- Keyboard command palette, compact mobile navigation, and first-run presets
- Public product/API documentation and an idempotent demo workspace seeder
- Production Compose with automatic HTTPS and private backend networking
- Verified PostgreSQL backup and staged restore tooling
- Backend, frontend, integration, contract, image, and recovery CI gates

## Architecture

```text
React + TypeScript
        │
        │ REST + SSE
        ▼
Go HTTP API ──► diagnostic queue ──► bounded worker pool
        │                                  │
        │                                  ├─ DNS
        │                                  ├─ Ping / Traceroute
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

| Layer    | Stack                                                   |
| -------- | ------------------------------------------------------- |
| API      | Go 1.26.5, net/http, chi, slog                          |
| Data     | PostgreSQL 18.4, pgx/pgxpool                            |
| Web      | Node.js 24 LTS, React, TypeScript, Vite, TanStack Query |
| Delivery | Docker, Docker Compose, GitHub Actions                  |

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

Webhook notifications use the same network policy and secure dialer as
diagnostic probes. Email notifications are optional: set `SMTP_HOST`,
`SMTP_FROM`, and, when required, the matching username/password pair. SMTP
supports `starttls` (default), implicit `tls`, and `none` for trusted local
development. See `.env.example` for every setting.

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

ICMP probes require raw-socket permission and are capability-detected at
runtime. The supplied Compose stack grants only `CAP_NET_RAW` to the non-root
API executable. The supported production baseline is a single-host deployment;
rate-limit state is local to each API replica. Multi-replica installations
need an edge or distributed limiter. Metrics are intentionally disabled by
default and must remain private when enabled.

## Roadmap

- **v0.1.0:** DNS, TCP, HTTP, TLS, worker pool, cancellation, SSE, history,
  JSON export, and PostgreSQL persistence
- **v0.2.0:** ICMP ping, traceroute, comparisons, CSV, capabilities, exact reruns
- **v0.3.0:** saved targets, schedules, status history, and notifications
- **v0.4.0:** accounts, workspaces, roles, shared reports, and API keys
- **v1.0.0:** stable public release with production hardening, public docs,
  accessibility, themes, onboarding, demo data, and recovery tooling

## License

[MIT](LICENSE)
