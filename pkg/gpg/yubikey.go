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
	"regexp"
	"strings"

	"github.com/gonzaloalvarez/kepr/pkg/config"
)

type Yubikey struct {
	SerialNumber       string
	Manufacturer       string
	CardholderName     string
	SignatureKey       string
	EncryptionKey      string
	AuthenticationKey  string
	SignatureOccupied  bool
	EncryptionOccupied bool
}

func (g *GPG) CheckCardPresent() error {
	slog.Debug("checking if card is present")

	stdout, stderr, err := g.execute("", "--card-status")
	if err != nil {
		slog.Debug("card status command failed", "error", err, "stderr", stderr)
		return fmt.Errorf("no card found")
	}

	if strings.Contains(stdout, "no card") || strings.Contains(stderr, "no card") ||
		strings.Contains(stdout, "Card not present") || strings.Contains(stderr, "Card not present") {
		slog.Debug("card not present in output")
		return fmt.Errorf("no card found")
	}

	slog.Debug("card is present")
	return nil
}

func (g *GPG) InitYubikey() error {
	slog.Debug("initializing yubikey")

	stdout, stderr, err := g.execute("", "--card-status")
	if err != nil {
		slog.Debug("card status command failed", "error", err, "stderr", stderr)
		return fmt.Errorf("no card found")
	}

	if strings.Contains(stdout, "no card") || strings.Contains(stderr, "no card") ||
		strings.Contains(stdout, "Card not present") || strings.Contains(stderr, "Card not present") {
		slog.Debug("card not present in output")
		return fmt.Errorf("no card found")
	}

	slog.Debug("card status output", "stdout", stdout, "stderr", stderr)

	yubikey, err := parseCardStatus(stdout)
	if err != nil {
		return fmt.Errorf("failed to parse card status: %w", err)
	}

	g.Yubikey = yubikey
	slog.Debug("yubikey initialized", "serial", yubikey.SerialNumber,
		"sig_occupied", yubikey.SignatureOccupied,
		"enc_occupied", yubikey.EncryptionOccupied)

	if err := g.configureYubikey(); err != nil {
		return fmt.Errorf("failed to configure yubikey: %w", err)
	}

	return nil
}

func parseCardStatus(output string) (*Yubikey, error) {
	lines := strings.Split(output, "\n")
	yubikey := &Yubikey{}

	fingerprintRegex := regexp.MustCompile(`[0-9A-Fa-f\s]{40,}`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Serial number") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				yubikey.SerialNumber = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Manufacturer") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				yubikey.Manufacturer = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Name of cardholder") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				yubikey.CardholderName = strings.TrimSpace(parts[1])
				slog.Debug("found cardholder name", "name", yubikey.CardholderName)
			}
		} else if strings.Contains(line, "Signature key") {
			slog.Debug("found signature key line", "line", line)
			if strings.Contains(line, "[none]") {
				yubikey.SignatureOccupied = false
				yubikey.SignatureKey = ""
			} else if fingerprintRegex.MatchString(line) {
				yubikey.SignatureOccupied = true
				yubikey.SignatureKey = extractFingerprint(line)
			}
		} else if strings.Contains(line, "Encryption key") {
			slog.Debug("found encryption key line", "line", line)
			if strings.Contains(line, "[none]") {
				yubikey.EncryptionOccupied = false
				yubikey.EncryptionKey = ""
			} else if fingerprintRegex.MatchString(line) {
				yubikey.EncryptionOccupied = true
				yubikey.EncryptionKey = extractFingerprint(line)
			}
		} else if strings.Contains(line, "Authentication key") {
			slog.Debug("found authentication key line", "line", line)
			if !strings.Contains(line, "[none]") && fingerprintRegex.MatchString(line) {
				yubikey.AuthenticationKey = extractFingerprint(line)
			}
		}
	}

	slog.Debug("parsed yubikey info",
		"serial", yubikey.SerialNumber,
		"manufacturer", yubikey.Manufacturer,
		"sig_occupied", yubikey.SignatureOccupied,
		"enc_occupied", yubikey.EncryptionOccupied)

	return yubikey, nil
}

func extractFingerprint(line string) string {
	fingerprintRegex := regexp.MustCompile(`[0-9A-Fa-f]{40}`)
	match := fingerprintRegex.FindString(strings.ReplaceAll(line, " ", ""))
	return match
}

func (y *Yubikey) IsOccupied() bool {
	return y.SignatureOccupied || y.EncryptionOccupied
}

func (g *GPG) cardEdit(attribute, value string) error {
	stdin := fmt.Sprintf("admin\n%s\n%s\nquit\n", attribute, value)

	slog.Debug("editing card", "attribute", attribute)

	_, stderr, err := g.executeWithPinentry(stdin, "--quiet", "--card-edit", "--expert", "--batch", "--display-charset", "utf-8", "--command-fd", "3")
	if err != nil {
		slog.Debug("card edit failed", "error", err, "stderr", stderr)
		return fmt.Errorf("failed to edit card attribute %s: %w", attribute, err)
	}

	return nil
}

func (g *GPG) configureYubikey() error {
	if g.Yubikey.CardholderName != "" {
		slog.Debug("yubikey already configured", "cardholder", g.Yubikey.CardholderName)
		return nil
	}

	slog.Debug("configuring yubikey")

	name := config.GetUserName()
	if name == "" {
		slog.Debug("no user name in config, skipping configuration")
		return nil
	} else {
		slog.Debug("user name in config", "name", name)
	}

	if err := g.cardEdit("name", name); err != nil {
		return err
	}

	g.Yubikey.CardholderName = name
	slog.Debug("yubikey configured", "cardholder", name)

	return nil
}
