# AGENTS.md

This file provides guidance to WARP (warp.dev) when working with code in this repository.

## Code Style

- Follow [Google's Go Style Guide](https://google.github.io/styleguide/go/)
- **No comments in code** - Do not generate any comments unless it is the license header with the author's name at the top of the file

## Project Overview

kepr (Key Encrypted Private Ring) is a Go CLI secret manager that wraps GPG and Git to provide hardware-backed (YubiKey) encrypted secret storage using GitHub as the backend.

## Build Commands

```bash
make build      # Production build with version info
make dev        # Development build with dev tag
make test       # Run all tests (go test -v ./...)
make clean      # Clean build artifacts
make nuke       # Full reset - deletes GitHub repo, resets YubiKey (dev builds only)
```

Run a single test:
```bash
go test -v -run TestName ./path/to/package
```

Run E2E tests:
```bash
make e2e        # Runs shell-based E2E tests with fake GitHub server
make e2e-go     # Runs Go-based E2E tests
```

## Architecture

### Package Structure

- **cmd/** - Cobra CLI commands (init, add, get). `cmd.App` struct holds injected dependencies.
- **pkg/** - Public packages:
  - `store/` - UUID-based encrypted secret storage engine
  - `gpg/` - GPG/YubiKey wrapper and key management
  - `github/` - GitHub API client (OAuth, repo operations)
  - `git/` - Git operations using go-git library (pure Go, no shell dependency)
  - `pass/` - High-level password store API orchestrating store+gpg+git
  - `config/` - JSON-based configuration (config.json in KEPR_HOME)
  - `cout/` - Console I/O interface using pterm
  - `shell/` - Shell execution abstraction
- **internal/** - Private workflows:
  - `init/` - Initialization workflow (GPG setup, YubiKey provisioning)
  - `add/` - Add secret workflow
  - `get/` - Get secret workflow
  - `buildflags/` - Build-time variables (version, commit, env)
- **tests/** - E2E tests and mocks

### Key Patterns

**Dependency Injection**: The `cmd.App` struct holds Shell, UI, GitHub dependencies, enabling mock substitution in tests.

**Interface-Based Design**: All external dependencies are interfaces (`shell.Executor`, `cout.IO`, `github.Client`). Mocks are in `tests/mocks/`.

**Isolated GPG Home**: Uses custom `GNUPGHOME` to avoid interfering with user's GPG config. Defaults to `{KEPR_HOME}/gpg`.

**UUID-Based Storage**: Secrets stored at `{KEPR_HOME}/{owner}/{repo}` with UUIDs instead of readable paths. Each secret has encrypted metadata (`uuid_md.gpg`).

**Cold Storage Model**: Master key backed up encrypted to GitHub, then deleted locally. Only subkeys remain on YubiKey.

**Configurable Home**: Set `KEPR_HOME` environment variable to override default config directory location.

### Data Flow

1. `kepr init` → GitHub OAuth → GPG key generation → YubiKey provisioning → master key backup → store init
2. `kepr add` → encrypt with GPG → store in UUID dir → git commit → push
3. `kepr get` → git pull → decrypt with YubiKey → output

## Testing

Tests use mocks in `tests/mocks/` for Shell, GitHub, and UI. The mock interfaces must match their corresponding real interfaces:
- `MockIO` must implement `cout.IO`
- `MockGitHub` must implement `github.Client`
- `MockCmd` must implement `shell.Cmd`
