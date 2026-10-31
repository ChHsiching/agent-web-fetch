# agent-web-fetch — build & release
#
# `make release` cross-compiles a single static binary for each target
# platform into dist/. CGO is disabled so every binary is statically linked
# with no external runtime dependency (ADR-0001, ADR-0004).

BINARY   := agent-web-fetch
CMD      := ./cmd/agent-web-fetch
DIST     := dist
# VERSION is stamped into the server binary via -X, the same linker injection
# the release workflow uses, so `make release VERSION=vX.Y.Z` reports the same
# version as a CI-built release. The default reports "dev".
VERSION  ?= dev
LDFLAGS  := -s -w -X github.com/ChHsiching/agent-web-fetch/internal/mcpserver.Version=$(VERSION)

TARGETS := \
	windows-amd64 \
	darwin-amd64 \
	darwin-arm64 \
	linux-amd64

.PHONY: release clean test vet

release: $(TARGETS)

# Each target cross-compiles one binary and depends on the dist/ directory
# existing (order-only prerequisite). CGO_ENABLED=0 guarantees a static binary
# with no C-runtime dependency (ADR-0001).
$(TARGETS): | $(DIST)

$(DIST):
	mkdir -p $(DIST)

windows-amd64:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-windows-amd64.exe $(CMD)

darwin-amd64:
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-amd64 $(CMD)

darwin-arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-darwin-arm64 $(CMD)

linux-amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY)-linux-amd64 $(CMD)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -rf $(DIST)
