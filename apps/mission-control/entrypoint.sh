#!/bin/sh
# Runtime environment injection for Next.js standalone Docker.

ENV_FILE="/app/apps/mission-control/.next/static/env.js"

cat > "$ENV_FILE" << EOF
window.__ENV = {
  NEXT_PUBLIC_API_URL: "${NEXT_PUBLIC_API_URL:-}",
  NEXT_PUBLIC_DEMO_MODE: "${NEXT_PUBLIC_DEMO_MODE:-false}",
  NEXT_PUBLIC_LANDING_URL: "${NEXT_PUBLIC_LANDING_URL:-https://freezenith.com}"
};
EOF

exec node apps/mission-control/server.js
