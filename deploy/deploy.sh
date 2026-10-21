#!/usr/bin/env bash
set -euo pipefail

: "${DOCKER_ACCOUNT:?DOCKER_ACCOUNT is required}"

VERSION=1.0.0
SERVER_IMAGE="docker.io/${DOCKER_ACCOUNT}/precedent-mcp-server:${VERSION}"
GATEWAY_IMAGE="docker.io/${DOCKER_ACCOUNT}/precedent-mcp-gateway:${VERSION}"

cd "$(dirname "${BASH_SOURCE[0]}")/.."

podman login docker.io
podman build -f deploy/Dockerfile --target server -t "$SERVER_IMAGE" .
podman build -f deploy/Dockerfile --target gateway -t "$GATEWAY_IMAGE" .
podman push "$SERVER_IMAGE"
podman push "$GATEWAY_IMAGE"
podman compose -f deploy/docker-compose.yml up -d
