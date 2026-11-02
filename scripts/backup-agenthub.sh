#!/usr/bin/env bash
# Back up AgentHub MySQL and (optionally) the artifact volume.
# Credentials come from the environment; never pass them as CLI arguments.
set -euo pipefail

: "${MYSQL_HOST:=127.0.0.1}"
: "${MYSQL_PORT:=3306}"
: "${MYSQL_USER:?MYSQL_USER is required}"
: "${MYSQL_PASSWORD:?MYSQL_PASSWORD is required}"
: "${MYSQL_DATABASE:=agenthub}"
: "${BACKUP_DIR:=/var/backups/agenthub}"
: "${RETENTION_DAYS:=7}"
: "${MYSQLDUMP_BIN:=mysqldump}"

if ! command -v "$MYSQLDUMP_BIN" >/dev/null 2>&1; then
  echo "backup: $MYSQLDUMP_BIN not found" >&2
  exit 1
fi

umask 077
mkdir -p "$BACKUP_DIR"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
sql_file="$BACKUP_DIR/agenthub-mysql-$timestamp.sql.gz"
sql_tmp="$sql_file.tmp"
trap 'rm -f "$sql_tmp"' EXIT

# MYSQL_PWD keeps the password out of the process list.
MYSQL_PWD="$MYSQL_PASSWORD" "$MYSQLDUMP_BIN" \
  --host="$MYSQL_HOST" --port="$MYSQL_PORT" --user="$MYSQL_USER" \
  --single-transaction --quick --no-tablespaces \
  --routines --triggers --events --set-gtid-purged=OFF \
  "$MYSQL_DATABASE" | gzip -c >"$sql_tmp"
mv "$sql_tmp" "$sql_file"
echo "backup: mysql $sql_file"

if [[ -n "${ARTIFACT_BACKUP_PATH:-}" ]]; then
  if [[ ! -d "$ARTIFACT_BACKUP_PATH" ]]; then
    echo "backup: artifact path does not exist: $ARTIFACT_BACKUP_PATH" >&2
    exit 1
  fi
  artifact_file="$BACKUP_DIR/agenthub-artifacts-$timestamp.tgz"
  artifact_tmp="$artifact_file.tmp"
  trap 'rm -f "$sql_tmp" "$artifact_tmp"' EXIT
  tar -czf "$artifact_tmp" -C "$ARTIFACT_BACKUP_PATH" .
  mv "$artifact_tmp" "$artifact_file"
  echo "backup: artifacts $artifact_file"
fi

find "$BACKUP_DIR" -maxdepth 1 -type f \( -name 'agenthub-mysql-*.sql.gz' -o -name 'agenthub-artifacts-*.tgz' \) \
  -mtime +"$RETENTION_DAYS" -print -delete
echo "backup: complete (retention ${RETENTION_DAYS}d)"
