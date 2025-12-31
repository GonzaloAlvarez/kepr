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
	Login              string
	SignatureOccupied  bool
	EncryptionOccupied bool
	gpg                *GPG
}

func NewYubikey(g *GPG) *Yubikey {
	return &Yubikey{
		gpg: g,
	}
}

func (y *Yubikey) killSCDaemon() {
	if y.gpg.GPGConfPath == "" {
		slog.Debug("gpgconf not available, skipping scdaemon kill")
		return
	}

	slog.Debug("killing scdaemon")
	cmd := y.gpg.executor.Command(y.gpg.GPGConfPath, "--kill", "all")
	if err := cmd.Run(); err != nil {
		slog.Debug("failed to kill scdaemon", "error", err)
	}
}

func (y *Yubikey) CheckCardPresent() error {
	if err := y.gpg.replaceSCDaemonConf(); err != nil {
		slog.Debug("failed to replace scdaemon.conf", "error", err)
	}
	y.killSCDaemon()
	slog.Debug("checking if card is present")

	stdout, stderr, err := y.gpg.execute("", "--card-status")
	if err != nil {
		slog.Debug("card status command failed", "error", err, "stderr", stderr)
		if revertErr := y.gpg.revertSCDaemonConf(); revertErr != nil {
			slog.Debug("failed to revert scdaemon.conf", "error", revertErr)
		}
		y.killSCDaemon()
		return fmt.Errorf("no card found")
	}

	if strings.Contains(stdout, "no card") || strings.Contains(stderr, "no card") ||
		strings.Contains(stdout, "Card not present") || strings.Contains(stderr, "Card not present") {
		slog.Debug("card not present in output")
		if revertErr := y.gpg.revertSCDaemonConf(); revertErr != nil {
			slog.Debug("failed to revert scdaemon.conf", "error", revertErr)
		}
		y.killSCDaemon()
		return fmt.Errorf("no card found")
	}

	slog.Debug("card is present")
	if err := y.gpg.revertSCDaemonConf(); err != nil {
		slog.Debug("failed to revert scdaemon.conf", "error", err)
	}
	y.killSCDaemon()
	return nil
}

func (y *Yubikey) checkCardStatus() error {
	if err := y.gpg.replaceSCDaemonConf(); err != nil {
		slog.Debug("failed to replace scdaemon.conf", "error", err)
	}
	y.killSCDaemon()
	slog.Debug("checking card status")

	stdout, stderr, err := y.gpg.execute("", "--card-status")
	if err != nil {
		slog.Debug("card status command failed", "error", err, "stderr", stderr)
		if revertErr := y.gpg.revertSCDaemonConf(); revertErr != nil {
			slog.Debug("failed to revert scdaemon.conf", "error", revertErr)
		}
		y.killSCDaemon()
		return fmt.Errorf("no card found")
	}

	if strings.Contains(stdout, "no card") || strings.Contains(stderr, "no card") ||
		strings.Contains(stdout, "Card not present") || strings.Contains(stderr, "Card not present") {
		slog.Debug("card not present in output")
		if revertErr := y.gpg.revertSCDaemonConf(); revertErr != nil {
			slog.Debug("failed to revert scdaemon.conf", "error", revertErr)
		}
		y.killSCDaemon()
		return fmt.Errorf("no card found")
	}

	slog.Debug("card status output", "stdout", stdout, "stderr", stderr)

	if err := y.parseCardStatus(stdout); err != nil {
		if revertErr := y.gpg.revertSCDaemonConf(); revertErr != nil {
			slog.Debug("failed to revert scdaemon.conf", "error", revertErr)
		}
		y.killSCDaemon()
		return fmt.Errorf("failed to parse card status: %w", err)
	}

	slog.Debug("yubikey status retrieved", "serial", y.SerialNumber,
		"sig_occupied", y.SignatureOccupied,
		"enc_occupied", y.EncryptionOccupied)

	if err := y.gpg.revertSCDaemonConf(); err != nil {
		slog.Debug("failed to revert scdaemon.conf", "error", err)
	}
	y.killSCDaemon()
	return nil
}

func (y *Yubikey) Init(name, email, fingerprint string) error {
	slog.Debug("initializing yubikey", "fingerprint", fingerprint)

	if err := y.checkCardStatus(); err != nil {
		return err
	}

	if y.IsOccupied() {
		return fmt.Errorf("yubikey slots are occupied")
	}

	if err := y.configureCard(name, email); err != nil {
		return fmt.Errorf("failed to configure card: %w", err)
	}

	if err := y.encryptionKeyToYubikey(fingerprint); err != nil {
		return fmt.Errorf("failed to move encryption key to yubikey: %w", err)
	}

	slog.Debug("yubikey initialized successfully")
	return nil
}

func (y *Yubikey) parseCardStatus(output string) error {
	lines := strings.Split(output, "\n")
	fingerprintRegex := regexp.MustCompile(`[0-9A-Fa-f\s]{40,}`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Serial number") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				y.SerialNumber = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Manufacturer") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				y.Manufacturer = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "Name of cardholder") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				y.CardholderName = strings.TrimSpace(parts[1])
				slog.Debug("found cardholder name", "name", y.CardholderName)
			}
		} else if strings.HasPrefix(line, "Login name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				y.Login = strings.TrimSpace(parts[1])
				slog.Debug("found login name", "login", y.Login)
			}
		} else if strings.Contains(line, "Signature key") {
			slog.Debug("found signature key line", "line", line)
			if strings.Contains(line, "[none]") {
				y.SignatureOccupied = false
				y.SignatureKey = ""
			} else if fingerprintRegex.MatchString(line) {
				y.SignatureOccupied = true
				y.SignatureKey = extractFingerprint(line)
			}
		} else if strings.Contains(line, "Encryption key") {
			slog.Debug("found encryption key line", "line", line)
			if strings.Contains(line, "[none]") {
				y.EncryptionOccupied = false
				y.EncryptionKey = ""
			} else if fingerprintRegex.MatchString(line) {
				y.EncryptionOccupied = true
				y.EncryptionKey = extractFingerprint(line)
			}
		} else if strings.Contains(line, "Authentication key") {
			slog.Debug("found authentication key line", "line", line)
			if !strings.Contains(line, "[none]") && fingerprintRegex.MatchString(line) {
				y.AuthenticationKey = extractFingerprint(line)
			}
		}
	}

	slog.Debug("parsed yubikey info",
		"serial", y.SerialNumber,
		"manufacturer", y.Manufacturer,
		"sig_occupied", y.SignatureOccupied,
		"enc_occupied", y.EncryptionOccupied)

	return nil
}

func extractFingerprint(line string) string {
	fingerprintRegex := regexp.MustCompile(`[0-9A-Fa-f]{40}`)
	match := fingerprintRegex.FindString(strings.ReplaceAll(line, " ", ""))
	return match
}

func (y *Yubikey) IsOccupied() bool {
	return y.SignatureOccupied || y.EncryptionOccupied
}

func (y *Yubikey) cardEdit(attribute string, values []string, adminPin string) error {
	if err := y.gpg.replaceSCDaemonConf(); err != nil {
		slog.Debug("failed to replace scdaemon.conf", "error", err)
	}
	y.killSCDaemon()

	if adminPin != "manual" {
		err := y.tryAutomatedCardEdit(attribute, values, adminPin)
		if err == nil {
			if revertErr := y.gpg.revertSCDaemonConf(); revertErr != nil {
				slog.Debug("failed to revert scdaemon.conf", "error", revertErr)
			}
			y.killSCDaemon()
			return nil
		}
		if adminPin == "" && strings.Contains(err.Error(), "bad PIN") {
			slog.Debug("default pin failed, falling back to interactive")
			config.SaveYubikeyAdminPin("manual")
		} else if adminPin != "" {
			if revertErr := y.gpg.revertSCDaemonConf(); revertErr != nil {
				slog.Debug("failed to revert scdaemon.conf", "error", revertErr)
			}
			y.killSCDaemon()
			return err
		}
	}

	return y.cardEditInteractive(attribute, values)
}

func (y *Yubikey) tryAutomatedCardEdit(attribute string, values []string, pin string) error {
	pinToUse := pin
	if pinToUse == "" {
		pinToUse = "12345678"
	}

	valuesStr := strings.Join(values, "\n")
	stdin := fmt.Sprintf("admin\n%s\n%s\n%s\nquit\n", attribute, valuesStr, pinToUse)

	slog.Debug("editing card with automated pin", "attribute", attribute, "values", values, "stdin", stdin)

	_, stderr, err := y.gpg.execute(stdin, "--pinentry-mode", "loopback", "--command-fd", "0", "--batch", "--expert", "--quiet", "--display-charset", "utf-8", "--card-edit")
	slog.Debug("automated card edit output", "stderr", stderr)
	if err != nil {
		slog.Debug("automated card edit failed", "error", err, "stderr", stderr)
		if strings.Contains(stderr, "Bad PIN") {
			return fmt.Errorf("bad PIN")
		}
		return fmt.Errorf("failed to edit card attribute %s: %w", attribute, err)
	}

	return nil
}

func (y *Yubikey) cardEditInteractive(attribute string, values []string) error {
	valuesStr := strings.Join(values, "\n")
	stdin := fmt.Sprintf("admin\n%s\n%s\nquit\n", attribute, valuesStr)

	slog.Debug("editing card", "attribute", attribute, "values", values)
	slog.Debug("stdin", "stdin", stdin)

	_, stderr, err := y.gpg.executeWithPinentry(stdin, "--quiet", "--card-edit", "--expert", "--batch", "--display-charset", "utf-8", "--command-fd", "3")
	if err != nil {
		slog.Debug("card edit failed", "error", err, "stderr", stderr)
		if revertErr := y.gpg.revertSCDaemonConf(); revertErr != nil {
			slog.Debug("failed to revert scdaemon.conf", "error", revertErr)
		}
		y.killSCDaemon()
		return fmt.Errorf("failed to edit card attribute %s: %w", attribute, err)
	}

	if err := y.gpg.revertSCDaemonConf(); err != nil {
		slog.Debug("failed to revert scdaemon.conf", "error", err)
	}
	y.killSCDaemon()
	return nil
}

func splitName(name string) (string, string) {
	parts := strings.Fields(name)
	var firstName, lastName string
	if len(parts) > 0 {
		firstName = parts[0]
		if len(parts) > 1 {
			lastName = strings.Join(parts[1:], " ")
		}
	}
	return firstName, lastName
}

func (y *Yubikey) configureCard(name, email string) error {
	adminPin := config.GetYubikeyAdminPin()

	slog.Debug("configuring card", "name", name, "email", email)
	slog.Debug("cardholder name", "cardholder", y.CardholderName)
	slog.Debug("login", "login", y.Login)
	if name != "" && (y.CardholderName == "" || y.CardholderName == "[not set]") {
		firstName, lastName := splitName(name)
		slog.Debug("split name", "firstName", firstName, "lastName", lastName)

		if err := y.cardEdit("name", []string{lastName, firstName}, adminPin); err != nil {
			return err
		}
	}

	if email != "" && (y.Login == "" || y.Login == "[not set]") {
		if err := y.cardEdit("login", []string{email}, adminPin); err != nil {

			return err
		}
	}

	return nil
}

func (y *Yubikey) encryptionKeyToYubikey(fingerprint string) error {
	if err := y.gpg.replaceSCDaemonConf(); err != nil {
		slog.Debug("failed to replace scdaemon.conf", "error", err)
	}
	y.killSCDaemon()
	slog.Debug("moving encryption key to yubikey", "fingerprint", fingerprint)

	stdin := "key 1\nkeytocard\n2\nsave\n"

	_, stderr, err := y.gpg.executeWithPinentry(stdin, "--command-fd", "3", "--batch", "--edit-key", fingerprint)
	if err != nil {
		slog.Debug("failed to move key to card", "error", err, "stderr", stderr)
		if revertErr := y.gpg.revertSCDaemonConf(); revertErr != nil {
			slog.Debug("failed to revert scdaemon.conf", "error", revertErr)
		}
		y.killSCDaemon()
		return fmt.Errorf("failed to move encryption key to yubikey: %w", err)
	}

	slog.Debug("encryption key moved to yubikey successfully")
	if err := y.gpg.revertSCDaemonConf(); err != nil {
		slog.Debug("failed to revert scdaemon.conf", "error", err)
	}
	y.killSCDaemon()
	return nil
}
