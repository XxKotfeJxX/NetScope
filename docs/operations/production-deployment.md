# Production deployment

This runbook deploys one NetScope API replica, the React web application behind
Caddy, and PostgreSQL with Docker Compose. It is the supported v1.0
single-host baseline.

## Prerequisites

- a Linux host with current Docker Engine and the Compose plugin;
- a DNS A/AAAA record for the NetScope hostname pointing to the host;
- inbound TCP 80 and TCP/UDP 443;
- outbound DNS, HTTP, HTTPS, SMTP (when configured), and ICMP (when enabled);
- an encrypted off-host location for database backups.

The production topology publishes only Caddy. PostgreSQL and the API use an
internal Docker network. Caddy obtains and renews TLS certificates
automatically.

## Configure

```sh
cp .env.production.example .env.production
chmod 600 .env.production
```

Set `NETSCOPE_DOMAIN`, generate a long URL-safe `POSTGRES_PASSWORD`, and put the
same value into `DATABASE_URL`. Do not commit `.env.production`.

The production compose file forces:

- HTTPS browser origin, Secure session cookies, and CSRF origin enforcement;
- public-only target policy with loopback, private, and link-local addresses
  blocked;
- a read-only API filesystem, dropped capabilities except `NET_RAW`, and
  `no-new-privileges`;
- trusted client-IP forwarding only from Caddy on the private backend network;
- bounded container logs and health-gated startup;
- private `/readyz` and `/metrics` paths at the public proxy;
- HSTS, CSP, anti-framing, MIME, referrer, and browser permissions headers.

## Deploy

```sh
docker compose \
  --env-file .env.production \
  -f compose.production.yml \
  config --quiet

docker compose \
  --env-file .env.production \
  -f compose.production.yml \
  build --pull

docker compose \
  --env-file .env.production \
  -f compose.production.yml \
  up -d
```

Verify:

```sh
curl --fail --silent --show-error https://netscope.example.com/healthz
curl --fail --silent --show-error https://netscope.example.com/
docker compose --env-file .env.production -f compose.production.yml ps
```

Replace the example hostname in the commands. The health response must report
the intended release version. `/readyz` and `/metrics` should return 404 at the
public hostname.

## Upgrade

1. Read the release notes and take a verified backup.
2. Pull the exact release tag, not `dev`.
3. Set `APP_VERSION` to that tag without the `v`.
4. Build with `--pull`, then run `up -d`.
5. Check container health, `/healthz`, login, one diagnostic, and one scheduled
   target.

Database migrations run before the API accepts traffic. Never downgrade the
application against a database that has received newer migrations unless the
release notes explicitly support it.

## Rollback

Application-only rollback:

1. check out the previous release tag;
2. rebuild and start the previous images;
3. verify health and a read-only workflow.

If a migration or data change is involved, restore the backup taken immediately
before the upgrade. The restore tool keeps the replaced database under a
timestamped name for a final manual rollback window.

## Operational checks

- probe public `/healthz` and the root document from outside the host;
- alert on container restarts and structured 5xx/panic metrics;
- review disk capacity for PostgreSQL, Caddy data, logs, and staged restores;
- verify SMTP or webhook notifications after configuration changes;
- review Dependabot and release-readiness checks before every release.
