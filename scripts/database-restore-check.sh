#!/usr/bin/env bash
set -euo pipefail
umask 077
: "${BACKUP_IDENTITY:?Set BACKUP_IDENTITY to an age private identity file}"
command -v age >/dev/null || { echo 'age is required.' >&2; exit 1; }
archive="${1:?Usage: database-restore-check.sh BACKUP.dump.age}"
[ -f "$archive" ] && [ -f "$BACKUP_IDENTITY" ] || { echo 'Backup or identity file not found.' >&2; exit 1; }
project_dir="$(cd "$(dirname "$0")/.." && pwd)"
scratch_db="restore_21mobile_$(date -u +%Y%m%d%H%M%S)_$$"
created=false
compose=(docker compose -f "$project_dir/compose.yaml")
cleanup() {
  if [ "$created" = true ]; then
    "${compose[@]}" exec -T postgres sh -c 'dropdb -U "$POSTGRES_USER" "$1"' sh "$scratch_db" || echo "Could not remove scratch database: $scratch_db" >&2
  fi
}
trap cleanup EXIT
"${compose[@]}" exec -T postgres sh -c 'createdb -U "$POSTGRES_USER" "$1"' sh "$scratch_db"
created=true
age -d -i "$BACKUP_IDENTITY" "$archive" |
  "${compose[@]}" exec -T postgres sh -c 'pg_restore -U "$POSTGRES_USER" -d "$1" --no-owner --no-acl --exit-on-error' sh "$scratch_db"
verified="$("${compose[@]}" exec -T postgres sh -c 'psql -X -U "$POSTGRES_USER" -d "$1" -v ON_ERROR_STOP=1 -At -c "SELECT to_regclass('\''public.api_credentials'\'') IS NOT NULL AND to_regclass('\''public.mo_events'\'') IS NOT NULL AND to_regclass('\''public.mo_callback_jobs'\'') IS NOT NULL"' sh "$scratch_db")"
[ "$verified" = t ] || { echo 'Restored schema verification failed.' >&2; exit 1; }
echo 'Backup restored and basic schema checked in an isolated database. Existing database unchanged.'
