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

	"github.com/gonzaloalvarez/kepr/pkg/config"
)

func (g *GPG) Encrypt(data []byte, recipient string) ([]byte, error) {
	slog.Debug("encrypting data", "recipient", recipient, "size", len(data))

	args := []string{
		"--encrypt",
		"--armor",
		"--batch",
		"--trust-model", "always",
		"-r", recipient,
	}

	stdout, stderr, err := g.executeBytes(data, args...)
	if err != nil {
		slog.Debug("encryption failed", "error", err, "stderr", stderr)
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	slog.Debug("encryption successful", "output_size", len(stdout))
	return stdout, nil
}

func (g *GPG) Decrypt(data []byte) ([]byte, error) {
	slog.Debug("decrypting data", "size", len(data))

	userPin := config.GetYubikeyUserPin()

	if userPin != "" && userPin != "manual" {
		slog.Debug("using automated decryption with loopback pinentry")
		args := []string{
			"--decrypt",
			"--batch",
			"--pinentry-mode", "loopback",
			"--passphrase", userPin,
		}

		stdout, stderr, err := g.executeBytes(data, args...)
		if err != nil {
			slog.Debug("decryption failed", "error", err, "stderr", stderr)
			return nil, fmt.Errorf("failed to decrypt data: %w", err)
		}

		slog.Debug("decryption successful", "output_size", len(stdout))
		return stdout, nil
	}

	slog.Debug("using interactive pinentry for decryption")
	args := []string{
		"--decrypt",
	}

	stdout, stderr, err := g.executeBytesWithPinentry(data, args...)
	if err != nil {
		slog.Debug("decryption failed", "error", err, "stderr", stderr)
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	slog.Debug("decryption successful", "output_size", len(stdout))
	return stdout, nil
}
