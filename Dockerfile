# syntax=docker/dockerfile:1.7

FROM --platform=$TARGETPLATFORM golang:1.25.0-bookworm AS build

ARG VERSION=dev

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w -checklinkname=0 -X github.com/fumiama/WireGold/config.Version=${VERSION}" \
    -o /out/wg .

FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

LABEL org.opencontainers.image.title="WireGold" \
      org.opencontainers.image.description="Container image for WireGold, a pure-Go Layer 3 VPN inspired by WireGuard." \
      org.opencontainers.image.source="https://github.com/fumiama/WireGold"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates iproute2 tini \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /config

COPY --from=build /out/wg /usr/local/bin/wg
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/wg /usr/local/bin/docker-entrypoint.sh \
    && mkdir -p /config

VOLUME ["/config"]

ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/docker-entrypoint.sh"]
CMD ["-c", "/config/config.yaml"]
