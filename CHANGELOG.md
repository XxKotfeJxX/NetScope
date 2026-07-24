# Changelog

All notable changes are documented here. The project follows Semantic
Versioning and uses the Keep a Changelog format.

## [Unreleased]

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
- Dashboard, live run details, typed result cards, history, and runtime settings.
- PostgreSQL integration workflow.

## [0.0.1] - 2026-07-23

### Added

- Repository governance, local configuration, and development documentation.
- Go API and React application bootstrap.
- PostgreSQL Docker Compose service and continuous integration.
