# Changelog

All notable changes are documented here. The project follows Semantic
Versioning and uses the Keep a Changelog format.

## [Unreleased]

## [1.0.0] - 2026-07-24

### Added

- Production-safe API configuration validation, strict browser-origin
  protection, security headers, and scoped per-client rate limits.
- Opt-in Prometheus metrics plus structured unexpected-error and recovered-panic
  tracking correlated by request ID.
- Persisted light and dark Technical Atlas themes, compact mobile navigation,
  skip navigation, reduced-motion support, and a keyboard command palette.
- First-run target presets for `example.com`, `github.com`, and `1.1.1.1`.
- Public `/docs` quickstart, API, authentication, limits, and operations manual.
- Idempotent demo workspace seeder using only the public v1 API.
- Hardened single-host production Compose deployment with Caddy automatic
  HTTPS, private backend networking, public-only SSRF policy, health gates,
  read-only API filesystem, reduced capabilities, and bounded container logs.
- Verified PostgreSQL custom-format backup and staged restore tools with
  checksums, rollback database retention, and readiness verification.
- Structural OpenAPI contract tests and a release-readiness workflow that
  builds images and rehearses backup/restore against ephemeral PostgreSQL.

### Changed

- The documented API contract is now the stable v1 contract.
- Production startup now rejects non-HTTPS browser origins, insecure session
  cookies, and disabled CSRF protection.
- CORS preflights from foreign origins are rejected instead of receiving
  permissive response headers.
- Public deployment guidance now treats Caddy as the only exposed service and
  keeps readiness and metrics endpoints private.

### Security

- Forwarded client IPs are ignored unless an explicitly trusted proxy topology
  is configured.
- Rate-limit storage is bounded and exposes standard limit/reset/retry headers.
- Public report tokens remain redacted from structured error and request logs.
- Production target and webhook resolution blocks private, loopback,
  link-local, multicast, unspecified, and metadata destinations.

## [0.4.0] - 2026-07-24

### Added

- Argon2id account registration and login with revocable opaque sessions.
- Isolated workspaces with Owner, Admin, Operator, and Viewer roles.
- Workspace member administration with transactional last-owner protection.
- Immutable audit events for membership, API-key, comment, and publication
  changes.
- Workspace-scoped Viewer/Operator API keys with one-time secret disclosure,
  expiry, and immediate revocation.
- Team report comments and revocable, expiring public read-only report links.

### Security

- Diagnostic runs, monitoring resources, comments, links, keys, and audit
  records are tenant-scoped at transport, service, and repository boundaries.
- Session, API-key, and public-link plaintext tokens are never stored.
- Public report token paths are redacted from access logs and public responses
  are non-cacheable.

## [0.3.0] - 2026-07-24

### Added

- Saved monitoring targets with names, tags, configurable checks, intervals,
  and consecutive-failure thresholds.
- A scheduler that claims due targets safely, launches diagnostic runs, and
  records availability, latency, failure context, and TLS expiry.
- Target pause/resume controls and scheduled maintenance windows.
- Targets atlas, detailed status timeline, latency trend, and live monitoring
  journal in the Technical Atlas interface.
- Per-target email and webhook notification channels for outage and recovery
  events.
- SMTP delivery with STARTTLS, implicit TLS, optional authentication, bounded
  timeouts, and sanitized message headers.

### Security

- Webhook delivery rejects redirects and credentials in URLs and uses the
  SSRF-resistant secure dialer with the configured network policy.

## [0.2.0] - 2026-07-24

### Added

- IPv4 and IPv6 preference for connection-oriented probes.
- Capability-detected ICMP ping and traceroute probes with bounded packet and
  hop counts.
- Structured connection, HTTP, and route-probe options in the diagnostic form.
- Hop-by-hop traceroute records and compact ping loss and latency metrics.
- Side-by-side comparison of persisted diagnostic runs.
- Formula-safe CSV report export alongside JSON.
- Exact reruns that preserve the original target, checks, and options.

### Changed

- The Linux container grants `CAP_NET_RAW` only to the non-root NetScope
  executable so ICMP probes work without running the service as root.

## [0.1.0] - 2026-07-24

### Added

- Target parsing, URL normalization, and local/public network policies.
- PostgreSQL migrations and persisted diagnostic run history.
- Bounded worker queue with cancellation and race-tested SSE subscriptions.
- DNS A, AAAA, CNAME, MX, NS, TXT, and PTR lookups.
- TCP connection checks with explicit ports and typed network errors.
- HTTP redirect tracking, response metadata, body hashing, and phase timings.
- TLS protocol, cipher, certificate chain, hostname, and expiry validation.
- SSRF-resistant dialing that pins connections to policy-approved IP addresses.
- Per-client creation rate limits and an overall run timeout.
- Run creation, detail, history, cancellation, SSE, capabilities, and JSON export APIs.
- Technical Atlas interface with a diagnostic route, live run details, typed
  results, field-log history, and runtime reference.
- PostgreSQL integration workflow.

## [0.0.1] - 2026-07-23

### Added

- Repository governance, local configuration, and development documentation.
- Go API and React application bootstrap.
- PostgreSQL Docker Compose service and continuous integration.
