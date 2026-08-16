#!/usr/bin/env bash
#
# Deploy the frontend to Netlify.
#
# Prerequisites (one-time):
#   1. A Netlify account and a personal access token:
#        https://app.netlify.com/user/applications#personal-access-tokens
#   2. A Netlify site. Either create one in the UI and copy its API ID
#      (Site settings → General → Site information → "Site ID"), or let the
#      first `netlify deploy` prompt you to create/link one.
#   3. Point the build at your deployed backend:
#        netlify env:set NEXT_PUBLIC_API_URL "https://your-api.example.com"
#      (or set it in the Netlify UI). NEXT_PUBLIC_* is inlined at build time.
#
# Usage:
#   NETLIFY_AUTH_TOKEN=xxxxx NETLIFY_SITE_ID=yyyyy ./scripts/deploy-netlify.sh          # production
#   NETLIFY_AUTH_TOKEN=xxxxx NETLIFY_SITE_ID=yyyyy ./scripts/deploy-netlify.sh --draft  # preview URL only
#
# Without NETLIFY_SITE_ID the CLI will prompt to select/create a site.
# Without NETLIFY_AUTH_TOKEN the CLI will open a browser to log in.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# Production by default; pass --draft for a preview deploy that isn't promoted.
PROD_FLAG="--prod"
if [[ "${1:-}" == "--draft" ]]; then
  PROD_FLAG=""
  echo "▶ Draft deploy (preview URL, not promoted to production)"
fi

SITE_ARGS=()
if [[ -n "${NETLIFY_SITE_ID:-}" ]]; then
  SITE_ARGS=(--site "$NETLIFY_SITE_ID")
fi

echo "▶ Deploying to Netlify (runs the build via netlify.toml)…"
# --build runs the `pnpm build` command from netlify.toml with the Next.js
# runtime plugin, then uploads the result. netlify-cli is fetched on demand.
pnpm dlx netlify-cli deploy --build ${PROD_FLAG} "${SITE_ARGS[@]}"

echo "✓ Done."
