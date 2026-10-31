# syntax=docker/dockerfile:1.7
# Multi-stage build: web app (Node 24) → static Go binary → distroless.

FROM node:24-alpine AS web
WORKDIR /src/web
RUN corepack enable
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

FROM golang:1.27-alpine AS build
WORKDIR /src
ARG VERSION=dev
ARG COMMIT=
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/web/dist ./web/dist
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X github.com/hostrepute/reputecast/internal/version.Version=${VERSION} -X github.com/hostrepute/reputecast/internal/version.Commit=${COMMIT}" \
      -o /out/reputecast ./cmd/reputecast
# The data directory (secret key) must exist and belong to the runtime user,
# or a fresh volume mounted there comes up owned by root.
RUN mkdir -p /out/data

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/reputecast /usr/local/bin/reputecast
COPY --from=build --chown=65532:65532 /out/data /var/lib/reputecast
ENV REPUTECAST_HTTP__ADDR=:8080 \
    REPUTECAST_SECRETS__KEY_FILE=/var/lib/reputecast/secret.key \
    REPUTECAST_LOG__FORMAT=json
VOLUME ["/var/lib/reputecast"]
EXPOSE 8080 9464
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/reputecast"]
CMD ["serve", "--auto-migrate", "--auto-install"]
