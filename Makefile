.PHONY: build clean dev test nuke

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

nuke:
	@if [ ! -f ./kepr ]; then \
		echo "Error: ./kepr binary not found. Run 'make dev' first."; \
		exit 1; \
	fi
	@if ! ./kepr --version 2>&1 | head -n1 | grep -q "dev"; then \
		echo "Error: kepr is not a dev build. This command only works with dev builds."; \
		exit 1; \
	fi
	@if [ -d "$$HOME/Library/Application Support/kepr" ]; then \
		CONFIG_DIR="$$HOME/Library/Application Support/kepr"; \
	elif [ -d "$$HOME/.config/kepr" ]; then \
		CONFIG_DIR="$$HOME/.config/kepr"; \
	else \
		echo "Error: kepr config directory not found"; \
		exit 1; \
	fi; \
	CONFIG_FILE="$$CONFIG_DIR/config.json"; \
	if [ ! -f "$$CONFIG_FILE" ]; then \
		echo "Error: config.json not found"; \
		exit 1; \
	fi; \
	TOKEN=$$(grep '"github_token"' "$$CONFIG_FILE" | sed 's/.*"github_token": *"\([^"]*\)".*/\1/'); \
	REPO=$$(grep '"github_repo"' "$$CONFIG_FILE" | sed 's/.*"github_repo": *"\([^"]*\)".*/\1/'); \
	if [ -z "$$TOKEN" ] || [ -z "$$REPO" ]; then \
		echo "Error: github_token or github_repo not found in config"; \
		exit 1; \
	fi; \
	echo "Deleting repository: $$REPO"; \
	curl -s -X DELETE \
		-H "Authorization: Bearer $$TOKEN" \
		-H "Accept: application/vnd.github+json" \
		"https://api.github.com/repos/$$REPO"; \
	echo "Repository deleted"; \
	echo "Resetting YubiKey GPG card..."; \
	gpg-card factory-reset || echo "GPG card reset failed"; \
	echo "Nuke complete"

