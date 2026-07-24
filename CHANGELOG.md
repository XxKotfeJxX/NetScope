# Changelog

All notable changes are documented here. The project follows Semantic
Versioning and uses the Keep a Changelog format.

## [Unreleased]

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
