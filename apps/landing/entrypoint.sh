#!/bin/sh
# Runtime environment injection for Next.js standalone Docker.

ENV_FILE="/app/apps/landing/.next/static/env.js"

cat > "$ENV_FILE" << EOF
window.__ENV = {
  NEXT_PUBLIC_APP_URL: "${NEXT_PUBLIC_APP_URL:-https://app.freezenith.com}"
};
EOF

exec node apps/landing/server.js
