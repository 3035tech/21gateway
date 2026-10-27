#!/usr/bin/env bash
set -euo pipefail
umask 077
: "${BACKUP_RECIPIENT:?Set the age public recipient in BACKUP_RECIPIENT}"
command -v age >/dev/null || { echo 'age is required.' >&2; exit 1; }
project_dir="$(cd "$(dirname "$0")/.." && pwd)"
backup_dir="${1:?Usage: database-backup.sh OUTPUT_DIRECTORY}"
mkdir -p "$backup_dir"
archive="$backup_dir/21mobile-$(date -u +%Y%m%dT%H%M%SZ)-$$.dump.age"
[ ! -e "$archive" ] || { echo 'Backup destination already exists.' >&2; exit 1; }
temporary="$(mktemp "$backup_dir/.21mobile.XXXXXX")"
trap 'rm -f "$temporary"' EXIT
# No plaintext dump is written to disk and no password is passed on the host CLI.
docker compose -f "$project_dir/compose.yaml" exec -T postgres sh -c 'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --format=custom --no-owner --no-acl' |
  age -r "$BACKUP_RECIPIENT" > "$temporary"
[ -s "$temporary" ] || { echo 'Empty backup.' >&2; exit 1; }
mv "$temporary" "$archive"
echo "Encrypted backup created: $archive"
