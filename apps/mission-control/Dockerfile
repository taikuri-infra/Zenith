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
COPY apps/mission-control/package.json ./apps/mission-control/
COPY packages/ui/package.json ./packages/ui/

# Install dependencies with frozen lockfile for reproducible builds
RUN pnpm install --frozen-lockfile

# ---- Builder ----
FROM base AS builder
WORKDIR /app

# NEXT_PUBLIC_* must be set at build time for client components
ARG NEXT_PUBLIC_DEMO_MODE=false
ARG NEXT_PUBLIC_API_URL=https://api.freezenith.com
ARG NEXT_PUBLIC_LANDING_URL=https://freezenith.com
ENV NEXT_PUBLIC_DEMO_MODE=$NEXT_PUBLIC_DEMO_MODE
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_LANDING_URL=$NEXT_PUBLIC_LANDING_URL

# Copy installed dependencies from deps stage
COPY --from=deps /app/ ./

# Copy source code
COPY packages/ui/ ./packages/ui/
COPY apps/mission-control/ ./apps/mission-control/

# Build the application
ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm --filter @zenith/mission-control build

# ---- Runner ----
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3100
ENV HOSTNAME="0.0.0.0"

LABEL org.opencontainers.image.title="zenith-mission-control"
LABEL org.opencontainers.image.description="Zenith Mission Control - Operator dashboard"
LABEL org.opencontainers.image.source="https://github.com/dotechhq/zenith"
LABEL org.opencontainers.image.vendor="DoTech"

# Create non-root user
RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Copy standalone output
COPY --from=builder --chown=nextjs:nodejs /app/apps/mission-control/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/apps/mission-control/.next/static ./apps/mission-control/.next/static

# Copy entrypoint script for runtime env injection
COPY --chown=nextjs:nodejs apps/mission-control/entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh

USER nextjs

EXPOSE 3100

ENTRYPOINT ["./entrypoint.sh"]
