/*
Copyright © 2025 Gonzalo Alvarez

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package gpg

import (
	"fmt"
	"log/slog"
	"strings"
)

type GPGKey struct {
	Fingerprint string
	UserID      string
	Email       string
	Name        string
}

func (g *GPG) GenerateKeys(name, email string) (string, error) {
	slog.Debug("generating master key", "name", name, "email", email)

	keyTemplate := fmt.Sprintf(`Key-Type: EDDSA
Key-Curve: ed25519
Key-Usage: cert
Name-Real: %s
Name-Email: %s
Expire-Date: 0
%%no-protection
%%commit
`, name, email)

	_, stderr, err := g.execute(keyTemplate, "--batch", "--gen-key")
	if err != nil {
		return "", fmt.Errorf("failed to generate master key: %w, stderr: %s", err, stderr)
	}

	slog.Debug("master key generated, retrieving fingerprint")

	keys, err := g.ListPublicKeys()
	if err != nil {
		return "", err
	}

	if len(keys) == 0 {
		return "", fmt.Errorf("no keys found after generation")
	}

	fingerprint := keys[0].Fingerprint

	slog.Debug("adding encryption subkey", "fingerprint", fingerprint)

	_, stderr, err = g.execute("", "--batch", "--pinentry-mode", "loopback", "--passphrase", "", "--quick-add-key", fingerprint, "cv25519", "encr", "0")
	if err != nil {
		return "", fmt.Errorf("failed to generate encryption subkey: %w, stderr: %s", err, stderr)
	}

	slog.Debug("encryption subkey generated")
	return fingerprint, nil
}

func (g *GPG) BackupMasterKey(fingerprint string) error {
	slog.Debug("exporting master key for backup", "fingerprint", fingerprint)

	stdout, stderr, err := g.execute("", "--armor", "--export-secret-key", fingerprint)
	if err != nil {
		return fmt.Errorf("failed to export secret key: %w, stderr: %s", err, stderr)
	}

	secretKey := strings.Replace(stdout, "\n\n", "\n", -1)
	if secretKey == "" {
		return fmt.Errorf("exported secret key is empty")
	}

	g.io.Warning("WARNING: The Master Key below will be DELETED from this machine immediately after this step.")
	g.io.Infoln("")
	g.io.Info(secretKey)
	g.io.Infoln("")
	g.io.Infoln("Copy the key block above and save it to a secure location (e.g., Bitwarden, 1Password, or a Gmail Draft).")

	confirmed, err := g.io.Confirm("Have you saved the Private Key safely? This action cannot be undone.")
	if err != nil {
		return fmt.Errorf("confirmation failed: %w", err)
	}

	if !confirmed {
		return fmt.Errorf("master key backup cancelled by user")
	}

	slog.Debug("user confirmed backup, deleting master key from keyring")

	return nil
}

func (g *GPG) ExportPublicKey(fingerprint string) ([]byte, error) {
	slog.Debug("exporting public key", "fingerprint", fingerprint)

	stdout, stderr, err := g.execute("", "--armor", "--export", fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to export public key: %w, stderr: %s", err, stderr)
	}

	if stdout == "" {
		return nil, fmt.Errorf("exported public key is empty for fingerprint %s", fingerprint)
	}

	return []byte(stdout), nil
}

func (g *GPG) ImportPublicKey(keyData []byte) error {
	slog.Debug("importing public key")

	_, stderr, err := g.execute(string(keyData), "--import")
	if err != nil {
		return fmt.Errorf("failed to import public key: %w, stderr: %s", err, stderr)
	}

	return nil
}

func (g *GPG) ListPublicKeys() ([]GPGKey, error) {
	slog.Debug("listing public keys")

	stdout, _, err := g.execute("", "--list-keys", "--with-colons")
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	var keys []GPGKey
	var currentFingerprint string

	lines := strings.Split(stdout, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "fpr:") {
			fields := strings.Split(line, ":")
			if len(fields) >= 10 {
				currentFingerprint = fields[9]
			}
		} else if strings.HasPrefix(line, "uid:") && currentFingerprint != "" {
			fields := strings.Split(line, ":")
			if len(fields) >= 10 {
				uid := fields[9]
				key := GPGKey{
					Fingerprint: currentFingerprint,
					UserID:      uid,
				}

				if strings.Contains(uid, "<") && strings.Contains(uid, ">") {
					emailStart := strings.Index(uid, "<")
					emailEnd := strings.Index(uid, ">")
					if emailStart < emailEnd {
						key.Email = uid[emailStart+1 : emailEnd]
						key.Name = strings.TrimSpace(uid[:emailStart])
					}
				}

				keys = append(keys, key)
			}
		}
	}

	slog.Debug("found keys", "count", len(keys))
	return keys, nil
}
