#!/usr/bin/env bash
#
# Seed (create or reset) a login account. The password is hashed with bcrypt
# (cost 12, per-password salt) by the Go `user` command before it touches the DB —
# the plaintext is never stored.
#
# Usage:
#   DATABASE_URL="postgres://user:pass@host:5432/db?sslmode=disable" \
#     ./scripts/create-user.sh <username> <password> [USER|ADMIN]
#
# Examples:
#   ./scripts/create-user.sh alice 'S3cret!' ADMIN
#   DATABASE_URL=... ./scripts/create-user.sh bob 'hunter2'        # defaults to USER
set -euo pipefail

USERNAME="${1:-}"
PASSWORD="${2:-}"
ROLE="${3:-USER}"

if [[ -z "$USERNAME" || -z "$PASSWORD" ]]; then
  echo "usage: $0 <username> <password> [USER|ADMIN]" >&2
  exit 1
fi

: "${DATABASE_URL:?Set DATABASE_URL (env var)}"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

go run . user --username "$USERNAME" --password "$PASSWORD" --role "$ROLE"
