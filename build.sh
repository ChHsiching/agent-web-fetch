#!/usr/bin/env bash
# build.sh — cross-compile agent-web-fetch for all four target platforms.
#
# Works in any bash (Git Bash on Windows, WSL, Linux, macOS). Produces a
# single static binary per platform in dist/. CGO is disabled so each binary
# is statically linked with no external runtime dependency (ADR-0001).
#
# Usage: ./build.sh
#        VERSION=vX.Y.Z ./build.sh   # stamps the version instead of "dev"
set -euo pipefail

BINARY="agent-web-fetch"
CMD="./cmd/agent-web-fetch"
DIST="dist"
# VERSION is stamped into the server binary via -X, the same linker injection
# the release workflow uses, so `VERSION=vX.Y.Z ./build.sh` reports the same
# version as a CI-built release. The default reports "dev".
VERSION="${VERSION:-dev}"
LDFLAGS="-s -w -X github.com/ChHsiching/agent-web-fetch/internal/mcpserver.Version=${VERSION}"

rm -rf "$DIST"
mkdir -p "$DIST"

build_one() {
	local goos="$1" goarch="$2" out="$3"
	echo "building $goos/$goarch -> $DIST/$out"
	GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
		go build -trimpath -ldflags "$LDFLAGS" -o "$DIST/$out" "$CMD"
}

build_one windows amd64 "$BINARY-windows-amd64.exe"
build_one darwin  amd64 "$BINARY-darwin-amd64"
build_one darwin  arm64 "$BINARY-darwin-arm64"
build_one linux   amd64 "$BINARY-linux-amd64"

echo
echo "built into $DIST/:"
ls -la "$DIST"
