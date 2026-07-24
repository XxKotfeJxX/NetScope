#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
compose_file="${NETSCOPE_COMPOSE_FILE:-$repo_root/compose.production.yml}"
env_file="${NETSCOPE_ENV_FILE:-$repo_root/.env.production}"
database="${POSTGRES_DB:-netscope}"
database_user="${POSTGRES_USER:-netscope}"
output_dir="${1:-$repo_root/backups}"

if [[ ! "$database" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] ||
	[[ ! "$database_user" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
	echo "POSTGRES_DB and POSTGRES_USER must be simple PostgreSQL identifiers." >&2
	exit 1
fi

compose=(docker compose -f "$compose_file")
if [[ -f "$env_file" ]]; then
	compose=(docker compose --env-file "$env_file" -f "$compose_file")
fi

mkdir -p "$output_dir"
output_dir="$(cd "$output_dir" && pwd)"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
backup="$output_dir/netscope-$timestamp.dump"
temporary="$(mktemp "$output_dir/.netscope-$timestamp.XXXXXX")"
trap 'rm -f "$temporary"' EXIT
umask 077

"${compose[@]}" exec -T postgres pg_dump \
	--username "$database_user" \
	--dbname "$database" \
	--format custom \
	--compress 9 \
	--no-owner \
	--no-privileges >"$temporary"

if [[ ! -s "$temporary" ]]; then
	echo "Backup is empty; refusing to publish it." >&2
	exit 1
fi

"${compose[@]}" exec -T postgres pg_restore --list <"$temporary" >/dev/null
mv "$temporary" "$backup"
(cd "$output_dir" && sha256sum "$(basename "$backup")" >"$(basename "$backup").sha256")
trap - EXIT

echo "Verified backup written to $backup"
