#!/usr/bin/env bash
#
# Migrate + seed the database.
#
# Usage:
#   DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=disable" ./scripts/seed.sh
#   ./scripts/seed.sh "postgres://user:pass@host:5432/db?sslmode=disable"
#
# The URL can come from $DATABASE_URL or the first argument. SEED_CSV_PATH
# overrides the CSV location (defaults to data/mock_logistics_data.csv).
set -euo pipefail

# Allow passing the URL as the first positional argument.
if [[ $# -ge 1 && -n "${1:-}" ]]; then
  export DATABASE_URL="$1"
fi

: "${DATABASE_URL:?Set DATABASE_URL (env var or first argument)}"

# Run from the backend root regardless of where the script is invoked.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export SEED_CSV_PATH="${SEED_CSV_PATH:-data/mock_logistics_data.csv}"

echo "▶ Applying migrations (dbmate)…"
dbmate --no-dump-schema up

echo "▶ Seeding orders from ${SEED_CSV_PATH}…"
go run . seed

echo "✓ Done."
