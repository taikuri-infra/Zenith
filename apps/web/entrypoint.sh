#!/bin/sh
# Runtime environment injection for Next.js standalone Docker.
# Writes NEXT_PUBLIC_* env vars to a JS file that the browser loads,
# allowing one Docker image to work across all environments.

ENV_FILE="/app/apps/web/.next/static/env.js"

cat > "$ENV_FILE" << EOF
window.__ENV = {
  NEXT_PUBLIC_API_URL: "${NEXT_PUBLIC_API_URL:-}",
  NEXT_PUBLIC_DEMO_MODE: "${NEXT_PUBLIC_DEMO_MODE:-false}",
  NEXT_PUBLIC_ZENITH_MODE: "${NEXT_PUBLIC_ZENITH_MODE:-standalone}",
  NEXT_PUBLIC_LANDING_URL: "${NEXT_PUBLIC_LANDING_URL:-https://freezenith.com}"
};
EOF

exec node apps/web/server.js
