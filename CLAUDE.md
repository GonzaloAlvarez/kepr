# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

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

## Architecture

### Package Structure

- **cmd/** - Cobra CLI commands (init, add, get, list, request)
- **pkg/** - Public packages:
  - `store/` - UUID-based encrypted secret storage engine; also contains `access.go` (`.gpg-id`-aware traversal), `requests.go` (pending access requests), `rekey.go`, `scan.go`
  - `gpg/` - GPG/YubiKey wrapper and key management
  - `github/` - GitHub API client (OAuth, repo operations)
  - `git/` - Git operations using go-git library (pure Go, no shell dependency)
  - `pass/` - High-level password store API orchestrating store+gpg+git
  - `config/` - JSON-based configuration (config.json in KEPR_HOME)
  - `cout/` - Console I/O interface using pterm
  - `shell/` - Shell execution abstraction
- **internal/** - Private workflows:
  - `workflow/` - Shared state machine engine wrapping `qmuntal/stateless`; all workflows use `State`/`Trigger` types and a `Workflow` struct with `Run(ctx)`
  - `init/` - Initialization workflow (GPG setup, YubiKey provisioning)
  - `add/` - Add secret workflow
  - `get/` - Get secret workflow
  - `list/` - List secrets workflow
  - `request/` - GitOps access request workflow (submit, list, approve by UUID prefix, approve by email with `--from`)
  - `common/` - Shared input validation helpers
  - `buildflags/` - Build-time variables (version, commit, env)
- **tests/** - E2E tests, mocks, and `fakeghserver/` (in-process fake GitHub API server used in E2E tests)

### Key Patterns

**Dependency Injection**: The `cmd.App` struct holds Shell, UI, GitHub dependencies, enabling mock substitution in tests.

**Interface-Based Design**: All external dependencies are interfaces (`shell.Executor`, `cout.IO`, `github.Client`). Mocks are in `tests/mocks/` (including a `MockGit`).

**State Machine Workflows**: Every internal workflow (`init`, `add`, `get`, `list`, `request`) is implemented as a `qmuntal/stateless` state machine. Each workflow package has three files: `states.go` (State/Trigger constants), `steps.go` (step functions that close over a shared state struct), and `workflow.go` (wires states→triggers→steps and exposes `NewXxxWorkflow(...) *workflow.Workflow`).

**Isolated GPG Home**: Uses custom `GNUPGHOME` to avoid interfering with user's GPG config. Defaults to `{KEPR_HOME}/gpg` where `KEPR_HOME` is the environment variable or system config directory.

**UUID-Based Storage**: Secrets stored at `{KEPR_HOME}/{owner}/{repo}` with UUIDs instead of readable paths. Each secret has encrypted metadata (`uuid_md.gpg`).

**Cold Storage Model**: Master key backed up encrypted to GitHub, then deleted locally. Only subkeys remain on YubiKey.

**Access Requests (GitOps)**: `kepr request <path>` pushes an ephemeral branch with an encrypted request file (`requests/<uuid>.json.gpg`). The store owner runs `kepr request --approve <uuid-prefix>` or `kepr request --approve --from <email>` to re-encrypt the relevant secret for the requester and merge.

**Configurable Home**: Set `KEPR_HOME` environment variable to override default config directory location. All kepr state (config, GPG home, secrets) will be stored under this directory.

### Data Flow

1. `kepr init [repo]` (optional repo name, default kepr-store) → GitHub OAuth → GPG key generation → YubiKey provisioning → master key backup → store init
2. `kepr add` → encrypt with GPG → store in UUID dir → git commit → push
3. `kepr get` → git pull → decrypt with YubiKey → output
4. `kepr list` → git pull → scan metadata → display secret paths
5. `kepr request <path>` → generate GPG keypair → push encrypted request to ephemeral branch
6. `kepr request --approve <prefix>` / `--approve --from <email>` → decrypt request → rekey secret → push merge

## Testing

Tests use mocks in `tests/mocks/` for Shell, GitHub, UI, and Git. The mock interfaces must match their corresponding real interfaces:
- `MockIO` must implement `cout.IO`
- `MockGitHub` must implement `github.Client`
- `MockCmd` must implement `shell.Cmd`
- `MockGit` must implement `git.Client`

E2E tests in `tests/` use `fakeghserver/` — an in-process HTTP server that stubs the GitHub API — so they run without real GitHub credentials.

Run a single test:
```bash
go test -v -run TestName ./path/to/package
```
