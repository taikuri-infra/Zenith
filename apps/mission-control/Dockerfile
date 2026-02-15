FROM node:20-alpine AS base
RUN npm install -g pnpm@10.29.3

FROM base AS deps
WORKDIR /app
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY apps/mission-control/package.json ./apps/mission-control/
COPY packages/ui/package.json ./packages/ui/
RUN pnpm install --frozen-lockfile

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/apps/mission-control/node_modules ./apps/mission-control/node_modules
COPY --from=deps /app/packages/ui/node_modules ./packages/ui/node_modules
COPY . .
RUN pnpm build:mc

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/apps/mission-control/.next/standalone ./
COPY --from=builder /app/apps/mission-control/.next/static ./apps/mission-control/.next/static
COPY --from=builder /app/apps/mission-control/public ./apps/mission-control/public
EXPOSE 3100
CMD ["node", "apps/mission-control/server.js"]
