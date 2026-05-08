# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.24

FROM golang:${GO_VERSION}-bookworm AS build

WORKDIR /src

COPY go.mod ./
COPY cmd ./cmd

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/railtracks ./cmd/railtracks

FROM debian:bookworm-slim AS runtime

ARG RAILWAY_CLI_VERSION=latest

WORKDIR /app

RUN bash -c "$(curl -L https://setup.vector.dev)"

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    bash \
    ca-certificates \
    curl \
    gnupg \
  && bash -c "$(curl -L https://setup.vector.dev)" \
  && apt-get update \
  && apt-get install -y --no-install-recommends vector \
  && apt-get purge -y --auto-remove gnupg \
  && rm -rf /var/lib/apt/lists/* \
  && mkdir -p /app/tmp/vector /var/lib/vector

RUN curl -fsSL cli.new | bash

COPY --from=build /out/railtracks /usr/local/bin/railtracks
COPY railtracks.example.json /app/railtracks.example.json
COPY scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENV TRACK_CONFIG_PATH=/app/railtracks.json \
    VECTOR_CONFIG_PATH=/app/vector.toml \
    VECTOR_CONFIG_FORMAT=toml \
    RAILWAY_ENVIRONMENT=production

ENTRYPOINT ["docker-entrypoint.sh"]
CMD ["railtracks", "--config", "/app/railtracks.json"]
