.PHONY: build clean dev test

VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

-include github_app_credentials.env
export

build:
	@echo "Building kepr (production)..."
	@echo "  Environment: prod"
	@echo "  Version: $(VERSION)"
	@echo "  Commit: $(COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"
	@if [ -n "$(GITHUB_CLIENT_ID)" ] && [ -n "$(GITHUB_CLIENT_SECRET)" ]; then \
		echo "  Auth Method: PKCE (with custom credentials)"; \
		echo "  Client ID: $(GITHUB_CLIENT_ID)"; \
		echo "  Client Secret: [REDACTED]"; \
	else \
		echo "  Auth Method: Device Code Flow (default client ID)"; \
		echo "  Client ID: Ov23liaarzPv4HBvyPtW (default)"; \
	fi
	@LDFLAGS="-X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Env=prod'"; \
	LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Version=$(VERSION)'"; \
	LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Commit=$(COMMIT)'"; \
	LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.BuildTime=$(BUILD_TIME)'"; \
	if [ -n "$(GITHUB_CLIENT_ID)" ]; then \
		LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/init.githubClientID=$(GITHUB_CLIENT_ID)'"; \
	fi; \
	if [ -n "$(GITHUB_CLIENT_SECRET)" ]; then \
		LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/init.githubClientSecret=$(GITHUB_CLIENT_SECRET)'"; \
	fi; \
	go build -ldflags "$$LDFLAGS" -o kepr main.go
	@echo "Build complete: ./kepr"

dev:
	@echo "Building kepr (development)..."
	@echo "  Environment: dev"
	@echo "  Version: $(VERSION)-dev"
	@echo "  Commit: $(COMMIT)"
	@echo "  Build Time: $(BUILD_TIME)"
	@if [ -n "$(GITHUB_CLIENT_ID)" ] && [ -n "$(GITHUB_CLIENT_SECRET)" ]; then \
		echo "  Auth Method: PKCE (with custom credentials)"; \
		echo "  Client ID: $(GITHUB_CLIENT_ID)"; \
		echo "  Client Secret: [REDACTED]"; \
	else \
		echo "  Auth Method: Device Code Flow (default client ID)"; \
		echo "  Client ID: Ov23liaarzPv4HBvyPtW (default)"; \
	fi
	@LDFLAGS="-X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Env=dev'"; \
	LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Version=$(VERSION)-dev'"; \
	LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Commit=$(COMMIT)'"; \
	LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.BuildTime=$(BUILD_TIME)'"; \
	if [ -n "$(GITHUB_CLIENT_ID)" ]; then \
		LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/init.githubClientID=$(GITHUB_CLIENT_ID)'"; \
	fi; \
	if [ -n "$(GITHUB_CLIENT_SECRET)" ]; then \
		LDFLAGS="$$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/init.githubClientSecret=$(GITHUB_CLIENT_SECRET)'"; \
	fi; \
	go build -tags dev -ldflags "$$LDFLAGS" -o kepr main.go

test:
	@go test -v ./...

clean:
	@go clean
	@rm -f kepr
	@echo "Cleaned build artifacts"

