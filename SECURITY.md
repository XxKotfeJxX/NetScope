# Security policy

## Supported versions

| Version | Supported |
| --- | --- |
| 1.0.x | Yes |
| 0.x | No |

## Reporting a vulnerability

Please report security issues privately to the repository owner instead of
opening a public issue. Include reproduction steps, affected versions, and the
expected impact when possible.

## Network target policy

NetScope connects to user-supplied targets and therefore treats SSRF controls
as a product boundary. Public deployments must use `NETWORK_POLICY=public`.
Private, loopback, link-local, multicast, unspecified, and cloud metadata
addresses are rejected in that mode.

Local mode is intentionally more permissive for development and must not be
used for an internet-facing deployment.

Production deployments must also use HTTPS, Secure session cookies, CSRF
origin enforcement, and a private API/database network. The supported
production Compose topology enables these controls and exposes only Caddy.
