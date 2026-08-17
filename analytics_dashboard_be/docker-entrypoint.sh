#!/bin/sh
# Runs pending migrations (idempotent) then execs the requested command.
# `serve` (default) starts the API; `seed` / `user ...` run one-shot commands.
set -e

if [ -n "$DATABASE_URL" ]; then
  echo "▶ applying migrations…"
  dbmate --no-dump-schema --migrations-dir /app/db/migrations up || {
    echo "migration failed" >&2; exit 1;
  }
fi

exec /app/analytics "$@"
