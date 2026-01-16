<div align="center">
  <img src="assets/img/kepr_icon.jpg" alt="kepr icon" width="200"/>
</div>

# kepr

**kepr** (Key Encrypted Private Ring) is an opinionated, serverless, and secure command-line secret manager. It uses industry-standard tools—`gpg` and `git`—to provide a seamless experience for managing encrypted secrets backed by hardware security modules (YubiKeys).

## Motivation

Managing secrets securely across teams and remote systems often requires setting up complex infrastructure (like HashiCorp Vault) or relying on purely software-based keys which are prone to theft.

`kepr` was built to solve specific pain points:
1.  **Serverless:** It relies entirely on public/private Git repositories (GitHub) for storage and distribution. No self-hosted servers required.
2.  **Hardware Enforced:** Decryption keys are generated on or moved to a YubiKey. If you don't have the physical token, you can't access the secrets.
3.  **Git-Native:** History, versioning, and rollback of secrets come for free via Git.
4.  **Remote Friendly:** It solves the "Bootstrapping Trust" problem for CI/CD and remote servers using a GitOps-style request/approval workflow.
5.  **Orchestration:** It does not roll its own crypto. It automates the complex configuration of GnuPG, providing a "porcelain" interface over the underlying plumbing.

## Prerequisites

*   **OS:** Linux or macOS
*   **Hardware:** A YubiKey (Series 5 or compatible)
*   **Account:** A GitHub account
*   **Dependencies:** `gpg` (v2.2+), `git`

## Installation

```bash
# Using Go (Recommended)
go install github.com/gonzaloalvarez/kepr@latest

# Ensure dependencies are in your path
# kepr will verify these during the first run
which gpg git
```

### Building from Source

If you're building from source (e.g., for development or forking):

1. Clone the repository
2. Build using make:
   ```bash
   make build          # Production build
   make dev            # Development build (with dev features)
   ```

**Optional: Enable PKCE Authentication (Recommended for Desktop Users)**

By default, kepr uses GitHub's device code flow for authentication. To enable PKCE authentication for a smoother UX (no copy-pasting codes):

1. Create `github_app_credentials.env` from the example file:
   ```bash
   cp github_app_credentials.env.example github_app_credentials.env
   ```
2. Add your GitHub OAuth App credentials (client ID and client secret) to `github_app_credentials.env`
3. Rebuild - credentials will be embedded at compile time

The `github_app_credentials.env` file is gitignored and will never be committed. If credentials are not provided, kepr automatically falls back to device code authentication.

## Usage

### Initialization (Onboarding)
The init process handles GitHub authentication, GPG key generation, YubiKey provisioning, and secure cold-storage backup automatically.

```bash
$ kepr init [github_username]/kepr-secrets
```

### Managing Secrets

```bash
# Add a secret (interactive)
$ kepr add prod/db/password
> Enter Value: *********

# Add a secret (one-liner)
$ kepr add prod/api-key "super-secret-value"

# Retrieve a secret
$ kepr get prod/db/password
> correct-horse-battery-staple
```

### Remote Machine Access (GitOps Flow)

`kepr` allows remote servers (which lack YubiKeys) to request access via ephemeral Git branches.

**On the Remote Server:**
```bash
$ kepr init --remote
# Generates a local soft-key and pushes an access request to the repo
```

**On Your Admin Machine:**
```bash
$ kepr review-requests
# Lists pending requests, validates fingerprints, and re-encrypts secrets for the new host
```

## Security Model

*   **Cryptography:** Uses Ed25519 (Edwards-curve Digital Signature Algorithm) via GnuPG.
*   **Identity:**
    *   **Master Key:** Kept in "Cold Storage" (encrypted AES-256 backup in a private GitHub repo), deleted from local disk, never touches the YubiKey.
    *   **Subkeys:** Moved to the YubiKey (Encryption/Signing).
*   **Isolation:** Runs with a custom `GNUPGHOME` (`~/.kepr/gpg`) to avoid interfering with your personal GPG configuration.

## License

MIT License. See [LICENSE](LICENSE) for details.

## Author

Gonzalo Alvarez
