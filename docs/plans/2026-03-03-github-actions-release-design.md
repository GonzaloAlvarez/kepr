# GitHub Actions Release Workflow Design

**Date:** 2026-03-03
**Status:** Approved

## Overview

This design describes a GitHub Actions workflow that automatically builds cross-platform binaries and creates GitHub Releases when tags are pushed to the repository.

## Requirements

- Trigger on any tag matching `v*` pattern (e.g., `v1.0.0`, `v2.1.3-beta`, `v1.0.0-rc1`)
- Build binaries for three platforms:
  - Linux x86_64
  - macOS x86_64 (Intel)
  - macOS ARM64 (Apple Silicon)
- Package binaries as `.tar.gz` archives
- Generate SHA256 checksums for verification
- Create GitHub Release with all artifacts
- Use production build settings (no GitHub client secrets, PKCE auth only)

## Architecture

### Workflow Structure

The workflow consists of two jobs:

1. **`build`** (matrix job) - Builds binaries for all platforms in parallel
2. **`release`** (depends on build) - Aggregates artifacts and creates GitHub Release

### Trigger Configuration

```yaml
on:
  push:
    tags:
      - 'v*'
```

This matches:
- Semantic versions: `v1.0.0`, `v2.1.3`
- Pre-releases: `v1.0.0-beta`, `v2.0-rc1`, `v1.0.0-alpha.1`

This does not match:
- Non-v tags: `test-tag`, `release-1.0`

## Build Job Design

### Matrix Strategy

Uses GitHub Actions matrix to build all platforms in parallel:

```yaml
matrix:
  include:
    - goos: linux
      goarch: amd64
      platform: linux-amd64
    - goos: darwin
      goarch: amd64
      platform: darwin-amd64
    - goos: darwin
      goarch: arm64
      platform: darwin-arm64
```

### Build Process

For each platform:

1. **Setup:**
   - Checkout repository
   - Install Go 1.25.3 (matching go.mod requirement)
   - Extract version from tag (strip 'v' prefix)

2. **Cross-Compilation:**
   - Set environment: `GOOS=${{ matrix.goos }}`, `GOARCH=${{ matrix.goarch }}`
   - Build with ldflags injecting:
     - `Version` - from tag (e.g., "1.0.0")
     - `Commit` - from git SHA
     - `BuildTime` - ISO 8601 timestamp
     - `Env` - set to "prod"
   - No GitHub OAuth credentials injected (uses PKCE flow)
   - Output: `kepr-${{ matrix.platform }}`

3. **Packaging:**
   - Create archive: `tar -czf kepr-$VERSION-${{ matrix.platform }}.tar.gz kepr-${{ matrix.platform }}`
   - Upload to workflow artifacts storage

### Cross-Compilation Approach

Uses Go's native cross-compilation via `GOOS`/`GOARCH` environment variables. All builds run on Ubuntu runners - no platform-specific runners needed. Go's compiler handles generating macOS binaries from Linux without requiring macOS hardware.

## Release Job Design

### Artifact Aggregation

1. Download all three archives from workflow storage
2. Verify all expected files present

### Checksum Generation

Generate `checksums.txt` with SHA256 hashes:

```bash
sha256sum kepr-*.tar.gz > checksums.txt
```

Output format:
```
abc123def456...  kepr-v1.0.0-linux-amd64.tar.gz
789ghi012jkl...  kepr-v1.0.0-darwin-amd64.tar.gz
345mno678pqr...  kepr-v1.0.0-darwin-arm64.tar.gz
```

### GitHub Release Creation

Uses `softprops/action-gh-release@v2`:

- **Release title:** Tag name (e.g., "v1.0.0")
- **Draft:** No (publish immediately)
- **Prerelease detection:** Automatic (tags with `-alpha`, `-beta`, `-rc`, etc. marked as prerelease)
- **Assets uploaded:**
  - `kepr-v1.0.0-linux-amd64.tar.gz`
  - `kepr-v1.0.0-darwin-amd64.tar.gz`
  - `kepr-v1.0.0-darwin-arm64.tar.gz`
  - `checksums.txt`

### Required Permissions

```yaml
permissions:
  contents: write
```

Allows the workflow to create releases and upload assets to the repository.

## Workflow File Location

`.github/workflows/release.yml`

## Usage

To create a release:

```bash
git tag v1.0.0
git push --tags
```

The workflow automatically:
1. Detects the tag push
2. Builds binaries for all platforms (~3-5 minutes)
3. Creates GitHub Release with downloadable assets
4. Users can download platform-specific archives and verify with checksums

## Benefits

- **Simple:** Native GitHub Actions, no external dependencies
- **Fast:** Parallel builds complete in 3-5 minutes
- **Secure:** SHA256 checksums for download verification
- **Flexible:** Supports semantic versioning and pre-release tags
- **Maintainable:** Clear matrix structure makes adding platforms easy

## Future Enhancements (Not in Scope)

- Windows binaries
- Homebrew tap automation
- Docker image builds
- Automatic changelog generation
- Release notes from commit messages
