# API security and observability

NetScope applies the controls in this document at the API process boundary.
The reverse proxy remains responsible for TLS termination, request-size limits
for static assets, and restricting operational endpoints.

## Production invariants

Set `APP_ENV=production`. The API refuses to start unless:

- `WEB_ORIGIN` is a path-free HTTPS origin, for example
  `https://netscope.example`;
- `SESSION_COOKIE_SECURE=true`;
- `CSRF_PROTECTION_ENABLED=true`.

Unsafe requests authenticated by the `netscope_session` cookie must carry an
`Origin` header that exactly matches `WEB_ORIGIN`. API-key and other Bearer
requests do not use browser cookies and are exempt from this check. Cross-origin
preflights from any other origin are rejected.

Every response receives API-appropriate content security, framing, MIME
sniffing, referrer, browser permission, and resource policy headers. Production
responses additionally receive HSTS. The web reverse proxy must set a separate
content security policy suitable for the React application.

## Rate limits

The API uses bounded, in-memory, fixed-window limiters. Defaults are per client
IP and per one-minute window:

| Scope | Environment variable | Default |
| --- | --- | ---: |
| All `/api/v1` traffic | `RATE_LIMIT_GENERAL` | 300 |
| Registration and login | `RATE_LIMIT_AUTH` | 10 |
| POST, PUT, PATCH, and DELETE requests | `RATE_LIMIT_MUTATION` | 60 |
| Public shared report reads | `RATE_LIMIT_PUBLIC` | 60 |

Change the window with `RATE_LIMIT_WINDOW`. Responses expose
`RateLimit-Limit`, `RateLimit-Remaining`, and `RateLimit-Reset`. A rejected
request returns HTTP 429, a stable `rate_limit_exceeded` API error, and
`Retry-After`.

Limits are local to each API replica. Use an edge or distributed limiter when a
deployment has multiple replicas or needs globally coordinated quotas.

`X-Forwarded-For` and `X-Real-IP` are ignored by default so a client cannot
evade limits by spoofing them. Set `TRUST_PROXY_HEADERS=true` only when NetScope
is directly behind a trusted reverse proxy that removes inbound forwarding
headers and writes its own values.

## Metrics and error tracking

Set `METRICS_ENABLED=true` to expose Prometheus text metrics at `/metrics`.
Metrics contain aggregate request, server-error, recovered-panic, in-flight,
and cumulative duration values without user, workspace, target, or route
labels. Keep `/metrics` private with the reverse proxy or network policy; it is
disabled by default.

Unexpected API errors and recovered panics are emitted as structured logs with
the request ID, method, redacted path, and error context. Public report tokens
are always replaced with `[redacted]`. The aggregate 5xx and panic counters can
drive alerts, while `X-Request-ID` correlates an API response with its log
event.

At minimum, alert on:

- any sustained increase in `netscope_http_server_errors_total`;
- any increase in `netscope_http_panics_total`;
- readiness failures on `/readyz`;
- process or container restarts.
