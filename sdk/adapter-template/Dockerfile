# The adapter process, ready for a sidecar next to ReputeCast.
#   docker build -t example-esp-adapter .
#   docker run -d -p 127.0.0.1:4600:4600 -e REPUTECAST_ADAPTER_SECRET=rcas_… example-esp-adapter
# Build it from your own copy of the template (the SDK comes from npm there).

FROM node:24-alpine AS build
WORKDIR /app
RUN corepack enable
COPY package.json pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile
COPY tsconfig.json tsconfig.build.json manifest.json ./
COPY src ./src
RUN pnpm build && pnpm prune --prod

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production HOST=0.0.0.0 PORT=4600
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/dist ./dist
COPY manifest.json package.json ./
# A non-root user, and Node's permission model: the process may read its
# own files only, and may not start child processes or workers. Limit its
# network egress to the hosts in manifest.json where your platform allows.
USER node
EXPOSE 4600
HEALTHCHECK --interval=30s --timeout=3s --retries=3 CMD wget -qO- http://127.0.0.1:4600/healthz || exit 1
CMD ["node", "--permission", "--allow-fs-read=/app", "dist/main.js"]
