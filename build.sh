#!/bin/bash
set -e

if [ -f github_app_credentials.env ]; then
    source github_app_credentials.env
fi

GITHUB_CLIENT_ID=${GITHUB_CLIENT_ID:-""}
GITHUB_CLIENT_SECRET=${GITHUB_CLIENT_SECRET:-""}

VERSION=${VERSION:-"dev"}
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

ENV=${ENV:-prod}

LDFLAGS="-X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Env=${ENV}'"
LDFLAGS="${LDFLAGS} -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Version=${VERSION}'"
LDFLAGS="${LDFLAGS} -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Commit=${COMMIT}'"
LDFLAGS="${LDFLAGS} -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.BuildTime=${BUILD_TIME}'"

if [ -n "$GITHUB_CLIENT_ID" ]; then
    LDFLAGS="${LDFLAGS} -X 'github.com/gonzaloalvarez/kepr/internal/init.githubClientID=${GITHUB_CLIENT_ID}'"
fi

if [ -n "$GITHUB_CLIENT_SECRET" ]; then
    LDFLAGS="${LDFLAGS} -X 'github.com/gonzaloalvarez/kepr/internal/init.githubClientSecret=${GITHUB_CLIENT_SECRET}'"
fi

echo "Building kepr..."
echo "  Environment: ${ENV}"
echo "  Version: ${VERSION}"
echo "  Commit: ${COMMIT}"
echo "  Build Time: ${BUILD_TIME}"

if [ -n "$GITHUB_CLIENT_ID" ] && [ -n "$GITHUB_CLIENT_SECRET" ]; then
    echo "  Auth Method: PKCE (with custom credentials)"
    echo "  Client ID: ${GITHUB_CLIENT_ID}"
    echo "  Client Secret: [REDACTED]"
else
    echo "  Auth Method: Device Code Flow (default client ID)"
    echo "  Client ID: Ov23liaarzPv4HBvyPtW (default)"
fi

go build -ldflags "${LDFLAGS}" -o kepr main.go

echo "Build complete: ./kepr"

