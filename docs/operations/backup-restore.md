# PostgreSQL backup and restore

NetScope's durable state lives in PostgreSQL. Caddy data contains replaceable
TLS state; application containers are disposable.

## Backup policy

For a small production deployment:

- run a logical backup every 24 hours;
- keep at least 7 daily, 4 weekly, and 6 monthly copies;
- copy backups off-host to encrypted, access-controlled storage;
- monitor backup age and size;
- perform a restore rehearsal at least monthly and before major upgrades.

Adjust recovery point and retention requirements for the deployment. A volume
snapshot is not a substitute for a tested PostgreSQL-aware backup.

## Create and verify

```sh
./scripts/backup-postgres.sh /secure/staging/netscope
```

The script:

1. writes a PostgreSQL custom-format dump to a private temporary file;
2. validates its catalog with `pg_restore --list`;
3. atomically publishes the dump;
4. writes a SHA-256 checksum.

Transfer both `.dump` and `.dump.sha256` files off-host. The default local
`backups/` directory is gitignored but is not automatically encrypted.

Use `NETSCOPE_COMPOSE_FILE` or `NETSCOPE_ENV_FILE` when the production files are
in non-default locations.

## Restore

Restoration changes live data and requires the explicit `--confirm` flag:

```sh
./scripts/restore-postgres.sh \
  --confirm \
  /secure/staging/netscope/netscope-YYYYMMDDTHHMMSSZ.dump
```

The script verifies the checksum when present, validates the dump, restores
into a staging database while the API remains online, briefly stops the API,
swaps database names, restarts the API, and waits for readiness.

The replaced database is retained as
`netscope_previous_YYYYMMDD_HHMMSS`. After application and data verification,
remove it manually:

```sh
docker compose --env-file .env.production -f compose.production.yml \
  exec -T postgres dropdb -U netscope netscope_previous_YYYYMMDD_HHMMSS
```

Do not remove it until login, diagnostics, monitoring history, members, API
keys, comments, and public reports have been checked.

## Restore rehearsal

Run a monthly rehearsal on an isolated host or Compose project:

1. provision empty storage;
2. restore the newest off-host backup;
3. start the matching NetScope release;
4. confirm readiness and record counts for core tables;
5. exercise login, historical reports, a diagnostic, and a monitored target;
6. record recovery time and any manual corrections;
7. destroy the isolated rehearsal environment.

Backups are only considered valid after this procedure succeeds.
