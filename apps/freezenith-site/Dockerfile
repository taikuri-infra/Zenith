# syntax=docker/dockerfile:1
# ---- Base ----
FROM node:20-alpine AS base
RUN corepack enable && corepack prepare pnpm@10.29.3 --activate
RUN apk add --no-cache libc6-compat

# ---- Dependencies ----
FROM base AS deps
WORKDIR /app

# Workspace root files needed by pnpm
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml turbo.json ./

# Package manifest for this app (self-contained — no workspace deps)
COPY apps/freezenith-site/package.json ./apps/freezenith-site/

RUN pnpm install --frozen-lockfile

# ---- Builder ----
FROM base AS builder
WORKDIR /app
COPY --from=deps /app/ ./
COPY apps/freezenith-site/ ./apps/freezenith-site/

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
RUN pnpm --filter @zenith/freezenith-site build

# ---- Runner ----
FROM node:20-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

LABEL org.opencontainers.image.title="freezenith-site"
LABEL org.opencontainers.image.description="FreeZenith open-source marketing site — freezenith.com"
LABEL org.opencontainers.image.source="https://github.com/taikuri-infra/Zenith"

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

# Standalone Next.js output
COPY --from=builder --chown=nextjs:nodejs /app/apps/freezenith-site/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/apps/freezenith-site/.next/static ./apps/freezenith-site/.next/static

USER nextjs
EXPOSE 3000

CMD ["node", "apps/freezenith-site/server.js"]
