# kepr Design Document

## 1. Overview
`kepr` is a command-line application designed to act as a wrapper (or "porcelain") around the GPG/Git ecosystem. It provides an opinionated workflow for managing encrypted secrets, prioritizing **hardware security** (YubiKey) and **distributed storage** (Git) without requiring a central secret management server.

## 2. Core Philosophy
1.  **Trust Hardware, Not Files:** Private keys for human users must reside on a hardware token (YubiKey). They should not persist on the local filesystem.
2.  **Serverless Distribution:** The application relies on existing public infrastructure (GitHub) for:
    *   Identity Discovery (User Name/Email).
    *   Disaster Recovery (Encrypted Identity Backup).
    *   Data Synchronization (Git Repositories).
3.  **No "Rolled" Crypto:** `kepr` does not implement encryption primitives. It delegates all cryptographic operations to GnuPG, using a custom UUID-based encrypted storage system.
4.  **Isolation:** `kepr` maintains its own state, ensuring it does not conflict with the user's existing `~/.gnupg` or git configurations. The state directory can be customized via the `KEPR_HOME` environment variable (defaults to `~/.config/kepr` on Linux/Unix, `~/Library/Application Support/kepr` on macOS).

## 3. System Architecture

### 3.1 Directory Structure
`kepr` separates application state from data storage. All paths are relative to `KEPR_HOME` (customizable via environment variable, defaults to system config directory).

*   **Application State (`{KEPR_HOME}/`)**
    *   `config.json`: Local configuration (Current user fingerprint, path to data repo, GitHub tokens).
    *   `gpg/`: A custom GnuPG home directory. Contains keyrings and `gpg-agent.conf`.
*   **Data Storage (`{KEPR_HOME}/{owner}/{repo}/`)**
    *   UUID-based directory structure where each secret and directory has a unique identifier.
    *   `.git/`: The git repository tracking changes.
    *   `.gpg.id`: The GPG key fingerprint authorized for decryption.

### 3.2 Identity & Key Management
`kepr` enforces a strict **Master Key vs. Subkey** architecture to ensure long-term identity security.

#### The "Cold Storage" Model
During `kepr init` (optional repo name; owner is the authenticated GitHub user):
1.  **Generation:** An Ed25519 Master Key (Certify capability only) is generated locally.
2.  **Subkeys:** Ed25519 Signing and Encryption subkeys are attached to the Master.
3.  **Backup:** The Master Key is exported, encrypted symmetrically (AES-256) with a user passphrase, and uploaded to a private GitHub repository (`username/kepr-identity-backup`).
4.  **Provisioning:** The Subkeys are moved to the YubiKey (Slots: Signature and Encryption).
5.  **Sanitization:** The Master Key is **deleted** from the local filesystem.

*Result:* If the YubiKey is lost, the identity can be recovered from the GitHub backup using the passphrase. If the laptop is stolen, the attacker cannot clone the identity because the Master Key is not present and the Subkeys are trapped in the hardware token.

### 3.3 Remote/Machine Access (The "GitOps" Flow)
Servers (e.g., EC2, Kubernetes workers) cannot use YubiKeys. `kepr` implements a specific flow for "Machine Users":

1.  **Request:** The remote machine generates a local, file-based key pair. It creates a new git branch `access-request/<hostname>` and pushes its public key to a `requests/` directory on that branch.
2.  **Review:** An admin (with a YubiKey) runs `kepr review-requests`. This fetches the branch, displays the machine's key fingerprint for verification, and asks for approval.
3.  **Approval:** If approved, the admin imports the machine's key, adds it to the `.gpg-id` recipients list (scoped to specific folders if needed), re-encrypts the secrets, and pushes the changes to `main`.

## 4. Technical Stack
*   **Language:** Go (Golang)
*   **CLI Framework:** Cobra
*   **Git Interface:** `os/exec` (wrapping system git)
*   **GPG Interface:** `os/exec` (wrapping system gpg 2.2+)
*   **Secret Management Engine:** Custom UUID-based encrypted store (`pkg/store`)
*   **GitHub API:** `google/go-github` (OAuth Device Flow)

## 5. Security Considerations
*   **Agent Forwarding:** `kepr` explicitly discourages SSH/GPG agent forwarding. Remote machines must have their own identities.
*   **Write Access:** Remote machines authenticate via GitHub Device Flow. While they have write access to the repo, the "Request" flow isolates their input to ephemeral branches to prevent destruction of the `main` history.
*   **Man-in-the-Middle:** The `review-requests` flow relies on Out-of-Band verification. The admin must verify the fingerprint displayed on the server matches the one displayed on their laptop before approving.
