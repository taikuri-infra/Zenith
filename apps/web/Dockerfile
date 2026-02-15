# ---- Base ----
FROM node:20-alpine AS base
RUN corepack enable && corepack prepare pnpm@10.29.3 --activate
RUN apk add --no-cache libc6-compat

# ---- Dependencies ----
FROM base AS deps
WORKDIR /app

# Copy workspace root files needed by pnpm
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml turbo.json ./

# Copy package.json for the app and its workspace dependencies
COPY apps/web/package.json ./apps/web/
COPY packages/ui/package.json ./packages/ui/

# Install dependencies with frozen lockfile for reproducible builds
RUN pnpm install --frozen-lockfile

# ---- Builder ----
FROM base AS builder
WORKDIR /app

# NEXT_PUBLIC_* must be set at build time for client components
ARG NEXT_PUBLIC_DEMO_MODE=false
ARG NEXT_PUBLIC_API_URL=https://api.freezenith.com
ENV NEXT_PUBLIC_DEMO_MODE=$NEXT_PUBLIC_DEMO_MODE
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL

# Copy installed dependencies from deps stage
COPY --from=deps /app/ ./

# Copy source code
COPY packages/ui/ ./packages/ui/
COPY apps/web/ ./apps/web/

# Build the application
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm --filter @zenith/web build

# ---- Runner ----
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

LABEL org.opencontainers.image.title="zenith-web"
LABEL org.opencontainers.image.description="Zenith Web Platform - User-facing dashboard"
LABEL org.opencontainers.image.source="https://github.com/dotechhq/zenith"
LABEL org.opencontainers.image.vendor="DoTech"

# Create non-root user
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Copy standalone output
COPY --from=builder --chown=nextjs:nodejs /app/apps/web/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/apps/web/.next/static ./apps/web/.next/static

# Copy public assets if they exist (using a no-fail pattern)
# If your app has a public/ directory, uncomment the next line:
# COPY --from=builder --chown=nextjs:nodejs /app/apps/web/public ./apps/web/public

USER nextjs

EXPOSE 3000

CMD ["node", "apps/web/server.js"]
