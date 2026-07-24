# Demo workspace

`cmd/demo-seed` creates a reproducible workspace for product evaluation. It
uses only the public v1 API: registration or login, workspace-scoped target
creation, and diagnostic run creation.

The seed is idempotent for the configured account. It creates these examples
only when they are missing:

- `example.com` — DNS, TCP, TLS, and HTTP;
- `github.com` — DNS, TCP, TLS, and HTTP;
- `1.1.1.1` — DNS and TCP.

## Local seed

Start NetScope, then run:

```powershell
$env:DEMO_PASSWORD = "choose-at-least-12-characters"
$env:NETSCOPE_API_URL = "http://localhost:8080"
$env:NETSCOPE_WEB_ORIGIN = "http://localhost:5173"
go run ./cmd/demo-seed
```

For a shell:

```sh
DEMO_PASSWORD='choose-at-least-12-characters' \
NETSCOPE_API_URL='http://localhost:8080' \
NETSCOPE_WEB_ORIGIN='http://localhost:5173' \
go run ./cmd/demo-seed
```

The default login is `demo@netscope.local`. Override identity with
`DEMO_EMAIL`, `DEMO_DISPLAY_NAME`, and `DEMO_WORKSPACE_NAME`.

`NETSCOPE_WEB_ORIGIN` must exactly match the API's `WEB_ORIGIN`; the seeder
sends it for production CSRF/origin enforcement. The password is never logged.

## Safety

Treat the demo account as real access:

- never commit `DEMO_PASSWORD`;
- use a unique secret per environment;
- do not expose a shared demo account on a production instance containing real
  workspace data;
- remove the demo account or rotate its password after an evaluation;
- keep the public-target SSRF policy enabled for internet-facing demos.
