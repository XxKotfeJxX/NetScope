#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "${1:-}" != "--confirm" || -z "${2:-}" || -n "${3:-}" ]]; then
	echo "Usage: $0 --confirm /absolute/path/to/netscope-backup.dump" >&2
	exit 2
fi

backup="$2"
if [[ ! -f "$backup" ]]; then
	echo "Backup does not exist: $backup" >&2
	exit 1
fi
backup="$(cd "$(dirname "$backup")" && pwd)/$(basename "$backup")"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="${NETSCOPE_COMPOSE_FILE:-$repo_root/compose.production.yml}"
env_file="${NETSCOPE_ENV_FILE:-$repo_root/.env.production}"
database="${POSTGRES_DB:-netscope}"
database_user="${POSTGRES_USER:-netscope}"

if [[ ! "$database" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] ||
	[[ ! "$database_user" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
	echo "POSTGRES_DB and POSTGRES_USER must be simple PostgreSQL identifiers." >&2
	exit 1
fi

compose=(docker compose -f "$compose_file")
if [[ -f "$env_file" ]]; then
	compose=(docker compose --env-file "$env_file" -f "$compose_file")
fi

if [[ -f "$backup.sha256" ]]; then
	(cd "$(dirname "$backup")" && sha256sum --check "$(basename "$backup").sha256")
fi
"${compose[@]}" exec -T postgres pg_restore --list <"$backup" >/dev/null

timestamp="$(date -u +%Y%m%d_%H%M%S)"
restore_database="${database}_restore_${timestamp}"
previous_database="${database}_previous_${timestamp}"
api_stopped=false

start_api() {
	if [[ "$api_stopped" == true ]]; then
		"${compose[@]}" start api >/dev/null || true
	fi
}
trap start_api EXIT

"${compose[@]}" exec -T postgres createdb \
	--username "$database_user" \
	--owner "$database_user" \
	"$restore_database"

if ! "${compose[@]}" exec -T postgres pg_restore \
	--username "$database_user" \
	--dbname "$restore_database" \
	--exit-on-error \
	--no-owner \
	--no-privileges <"$backup"; then
	"${compose[@]}" exec -T postgres dropdb \
		--username "$database_user" \
		--if-exists \
		"$restore_database"
	exit 1
fi

"${compose[@]}" stop api
api_stopped=true
"${compose[@]}" exec -T postgres psql \
	--username "$database_user" \
	--dbname postgres \
	--set ON_ERROR_STOP=1 \
	--command "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$database' AND pid <> pg_backend_pid();"
"${compose[@]}" exec -T postgres psql \
	--username "$database_user" \
	--dbname postgres \
	--set ON_ERROR_STOP=1 \
	--command "ALTER DATABASE \"$database\" RENAME TO \"$previous_database\";"

if ! "${compose[@]}" exec -T postgres psql \
	--username "$database_user" \
	--dbname postgres \
	--set ON_ERROR_STOP=1 \
	--command "ALTER DATABASE \"$restore_database\" RENAME TO \"$database\";"; then
	"${compose[@]}" exec -T postgres psql \
		--username "$database_user" \
		--dbname postgres \
		--set ON_ERROR_STOP=1 \
		--command "ALTER DATABASE \"$previous_database\" RENAME TO \"$database\";" || true
	exit 1
fi

"${compose[@]}" start api >/dev/null
api_stopped=false
for _ in {1..30}; do
	if "${compose[@]}" exec -T api wget -qO- http://localhost:8080/readyz >/dev/null; then
		echo "Restore complete. Previous database retained as $previous_database."
		echo "Verify the application, then remove that database manually."
		trap - EXIT
		exit 0
	fi
	sleep 2
done

echo "Restored database did not become ready; previous data remains in $previous_database." >&2
exit 1
