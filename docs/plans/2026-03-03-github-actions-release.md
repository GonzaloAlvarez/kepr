# GitHub Actions Release Workflow Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a GitHub Actions workflow that automatically builds cross-platform binaries (Linux x86_64, macOS x86_64, macOS ARM64) and creates GitHub Releases when version tags are pushed.

**Architecture:** Two-job workflow using matrix strategy for parallel builds, followed by artifact aggregation and release creation. Uses Go's native cross-compilation to build all platforms from a single Ubuntu runner. No external dependencies beyond official GitHub Actions.

**Tech Stack:** GitHub Actions, Go 1.25.3, softprops/action-gh-release@v2

---

## Task 1: Create GitHub Actions Directory Structure

**Files:**
- Create: `.github/workflows/release.yml`

**Step 1: Create directory structure**

Run:
```bash
mkdir -p .github/workflows
```

Expected: Directory created successfully

**Step 2: Verify directory exists**

Run:
```bash
ls -la .github/workflows
```

Expected: Empty directory listing

---

## Task 2: Write Release Workflow - Trigger and Build Job

**Files:**
- Create: `.github/workflows/release.yml`

**Step 1: Write workflow header and trigger**

Create `.github/workflows/release.yml` with:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  build:
    name: Build ${{ matrix.platform }}
    runs-on: ubuntu-latest
    strategy:
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

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.25.3'

      - name: Extract version from tag
        id: version
        run: |
          VERSION=${GITHUB_REF#refs/tags/v}
          echo "version=$VERSION" >> $GITHUB_OUTPUT
          echo "Building version: $VERSION"

      - name: Build binary
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
        run: |
          VERSION="${{ steps.version.outputs.version }}"
          COMMIT="${{ github.sha }}"
          BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

          LDFLAGS="-X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Env=prod'"
          LDFLAGS="$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Version=$VERSION'"
          LDFLAGS="$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.Commit=$COMMIT'"
          LDFLAGS="$LDFLAGS -X 'github.com/gonzaloalvarez/kepr/internal/buildflags.BuildTime=$BUILD_TIME'"

          go build -ldflags "$LDFLAGS" -o kepr-${{ matrix.platform }} main.go

          echo "Built kepr-${{ matrix.platform }}"
          file kepr-${{ matrix.platform }}

      - name: Create archive
        run: |
          VERSION="${{ steps.version.outputs.version }}"
          tar -czf kepr-v$VERSION-${{ matrix.platform }}.tar.gz kepr-${{ matrix.platform }}
          ls -lh kepr-v$VERSION-${{ matrix.platform }}.tar.gz

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: kepr-${{ matrix.platform }}
          path: kepr-v${{ steps.version.outputs.version }}-${{ matrix.platform }}.tar.gz
          retention-days: 1
```

**Step 2: Verify YAML syntax**

Run:
```bash
# Check if file exists and has content
cat .github/workflows/release.yml | head -20
```

Expected: Workflow YAML displayed with correct indentation

---

## Task 3: Write Release Workflow - Release Job

**Files:**
- Modify: `.github/workflows/release.yml` (append to end)

**Step 1: Add release job to workflow**

Append to `.github/workflows/release.yml`:

```yaml

  release:
    name: Create Release
    needs: build
    runs-on: ubuntu-latest

    steps:
      - name: Extract version from tag
        id: version
        run: |
          VERSION=${GITHUB_REF#refs/tags/v}
          echo "version=$VERSION" >> $GITHUB_OUTPUT
          echo "Creating release for version: $VERSION"

      - name: Download all artifacts
        uses: actions/download-artifact@v4
        with:
          path: artifacts

      - name: Prepare release assets
        run: |
          mkdir -p release
          find artifacts -name "*.tar.gz" -exec mv {} release/ \;
          ls -lh release/

      - name: Generate checksums
        working-directory: release
        run: |
          sha256sum *.tar.gz > checksums.txt
          cat checksums.txt

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: release/*
          draft: false
          prerelease: false
          generate_release_notes: true
          fail_on_unmatched_files: true
```

**Step 2: Verify complete workflow**

Run:
```bash
cat .github/workflows/release.yml | wc -l
```

Expected: File should have approximately 100+ lines

**Step 3: Check workflow is valid YAML**

Run:
```bash
# Use yamllint if available, or just check syntax with grep
grep -E "^[[:space:]]*-[[:space:]]*name:" .github/workflows/release.yml | wc -l
```

Expected: Should find multiple step names (at least 10)

---

## Task 4: Commit Workflow File

**Files:**
- Staged: `.github/workflows/release.yml`

**Step 1: Review changes**

Run:
```bash
git status
git diff --cached .github/workflows/release.yml || git diff .github/workflows/release.yml
```

Expected: Shows new workflow file

**Step 2: Stage workflow file**

Run:
```bash
git add .github/workflows/release.yml
```

Expected: File staged for commit

**Step 3: Commit workflow**

Run:
```bash
git commit -m "$(cat <<'EOF'
Add GitHub Actions workflow for automated cross-platform releases

Implements tag-triggered release workflow with:
- Matrix build strategy for Linux/macOS (x86_64/ARM64)
- Go cross-compilation from Ubuntu runner
- Tar.gz archives with SHA256 checksums
- Automatic GitHub Release creation on v* tags

Triggered by: git push --tags (any tag matching v*)
Artifacts: kepr-vX.Y.Z-{linux,darwin}-{amd64,arm64}.tar.gz + checksums.txt

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

Expected: Commit created successfully

**Step 4: Verify commit**

Run:
```bash
git log --oneline -1
git show --stat
```

Expected: Shows commit with .github/workflows/release.yml

---

## Task 5: Documentation and Testing Guidance

**Files:**
- Modify: `README.md` (if exists, add release section)

**Step 1: Check if README exists**

Run:
```bash
ls -la README.md
```

Expected: File exists or "No such file or directory"

**Step 2: Document release process**

If README.md exists, add a section (manually or suggest to user):

```markdown
## Releases

To create a new release:

1. Create and push a version tag:
   ```bash
   git tag v1.0.0
   git push --tags
   ```

2. GitHub Actions will automatically:
   - Build binaries for Linux (x86_64), macOS (x86_64), and macOS (ARM64)
   - Create release archives (.tar.gz)
   - Generate SHA256 checksums
   - Create a GitHub Release with all artifacts

3. Download binaries from the GitHub Releases page

To verify downloads:
```bash
sha256sum -c checksums.txt
```

Tags matching `v*` trigger releases (e.g., v1.0.0, v2.1.3-beta, v1.0.0-rc1).
```

**Step 3: Inform user about testing**

Testing options:

1. **Test with a pre-release tag:**
   ```bash
   git tag v0.0.1-test
   git push --tags
   ```
   Then watch: https://github.com/gonzaloalvarez/kepr/actions

2. **Delete test tag after verification:**
   ```bash
   git tag -d v0.0.1-test
   git push --delete origin v0.0.1-test
   # Also delete the test release from GitHub UI
   ```

3. **Monitor workflow:**
   - Check Actions tab in GitHub
   - Verify all 3 builds complete
   - Verify release is created with 4 files (3 archives + checksums)

---

## Task 6: Final Verification Checklist

**Manual verification steps (user or engineer should perform):**

**Step 1: Push workflow to GitHub**

Run:
```bash
git push
```

Expected: Workflow file now in GitHub repository

**Step 2: Verify workflow appears in GitHub**

Action: Visit https://github.com/gonzaloalvarez/kepr/actions
Expected: "Release" workflow should appear (may show as disabled until first tag)

**Step 3: Create test tag (optional but recommended)**

Run:
```bash
git tag v0.0.1-test
git push --tags
```

Expected: Tag pushed successfully

**Step 4: Monitor workflow execution**

Action: Visit https://github.com/gonzaloalvarez/kepr/actions
Expected:
- Release workflow running
- Build job shows 3 parallel runs (linux-amd64, darwin-amd64, darwin-arm64)
- Release job runs after builds complete
- Release created at https://github.com/gonzaloalvarez/kepr/releases

**Step 5: Verify release artifacts**

Expected release should contain:
- `kepr-v0.0.1-test-linux-amd64.tar.gz`
- `kepr-v0.0.1-test-darwin-amd64.tar.gz`
- `kepr-v0.0.1-test-darwin-arm64.tar.gz`
- `checksums.txt`

**Step 6: Clean up test release (optional)**

```bash
# Delete local tag
git tag -d v0.0.1-test
# Delete remote tag
git push --delete origin v0.0.1-test
# Manually delete release from GitHub UI
```

---

## Implementation Complete

The GitHub Actions workflow is now ready to use. On every tag push matching `v*`, it will automatically:

1. Build binaries for all three platforms in parallel (~3-5 minutes)
2. Package them as tar.gz archives
3. Generate SHA256 checksums
4. Create a GitHub Release with all artifacts

**Next production release:**
```bash
git tag v1.0.0
git push --tags
```

**Workflow will not run on:**
- Regular commits
- Branch pushes
- Non-v tags (e.g., "test", "staging")
