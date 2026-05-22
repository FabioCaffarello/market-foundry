# syntax=docker/dockerfile:1.7
#
# Multi-stage Go service builder.
#
# GO_VERSION must match the toolchain declared in go.work and all 17
# cmd/*/go.mod + internal/*/go.mod files. A Go bump requires a
# coordinated update across all of them (no toolchain directive yet,
# so go.work is the source of truth).
#
# Phase 2 hardening (P2.5): patch-pinned runtime alpine; OCI labels.

ARG GO_VERSION=1.25.7
ARG ALPINE_VERSION=3.20.3

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /src

ARG SERVICE
ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN test -n "${SERVICE}"

RUN apk add --no-cache ca-certificates

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/${SERVICE} \
    && test -x /out/service

FROM alpine:${ALPINE_VERSION}

ARG SERVICE
ENV SERVICE=${SERVICE}

LABEL org.opencontainers.image.title="market-foundry-${SERVICE}"
LABEL org.opencontainers.image.description="market-foundry Go service binary (${SERVICE})"
LABEL org.opencontainers.image.source="https://github.com/FabioCaffarello/market-foundry"
LABEL org.opencontainers.image.licenses="proprietary"

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /etc/market-foundry \
    && chown -R app:app /etc/market-foundry

COPY --from=builder /out/service /usr/local/bin/service

USER app:app
ENTRYPOINT ["/usr/local/bin/service"]
