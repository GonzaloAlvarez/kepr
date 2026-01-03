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
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/gonzaloalvarez/kepr/pkg/config"
)

type PinType int

const (
	PinTypeAdmin PinType = iota
	PinTypeUser
)

var (
	ErrManualModeRequired = errors.New("manual mode required")
	ErrBadPIN             = errors.New("bad PIN")
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

func (y *Yubikey) KillSCDaemon() {
	if y.gpg.GPGConfPath == "" {
		slog.Debug("gpgconf not available, skipping scdaemon kill")
		return
	}

	slog.Debug("killing scdaemon")
	cmd := y.gpg.executor.Command(y.gpg.GPGConfPath, "--kill", "scdaemon")
	if err := cmd.Run(); err != nil {
		slog.Debug("failed to kill scdaemon", "error", err)
	}
}

func (y *Yubikey) CheckCardPresent() error {
	slog.Debug("checking if card is present")

	stdout, stderr, err := y.gpg.execute("", "--card-status")
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

func (y *Yubikey) checkCardStatus() error {
	slog.Debug("checking card status")

	stdout, stderr, err := y.gpg.execute("", "--card-status")
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

	if err := y.parseCardStatus(stdout); err != nil {
		return fmt.Errorf("failed to parse card status: %w", err)
	}

	slog.Debug("yubikey status retrieved", "serial", y.SerialNumber,
		"sig_occupied", y.SignatureOccupied,
		"enc_occupied", y.EncryptionOccupied)

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
	baseArgs := []string{"--card-edit"}
	commands := []string{"admin", attribute}
	commands = append(commands, values...)
	commands = append(commands, "quit")

	if adminPin != "manual" {
		err := y.automatedYubikey(baseArgs, commands, PinTypeAdmin)
		if err == nil {
			return nil
		}
		if errors.Is(err, ErrBadPIN) && adminPin == "" {
			slog.Debug("default pin failed, falling back to manual")
			config.SaveYubikeyAdminPin("manual")
		} else if adminPin != "" {
			return err
		}
	}

	return y.manualYubikey(baseArgs, commands)
}

func (y *Yubikey) automatedYubikey(baseArgs []string, commands []string, pinType PinType) error {
	var pinToUse string

	if pinType == PinTypeUser {
		userPin := config.GetYubikeyUserPin()
		pinToUse = userPin
		if pinToUse == "" {
			pinToUse = "123456"
		}
	} else {
		adminPin := config.GetYubikeyAdminPin()
		if adminPin == "manual" {
			return ErrManualModeRequired
		}

		pinToUse = adminPin
		if pinToUse == "" {
			pinToUse = "12345678"
		}
	}

	args := []string{
		"--pinentry-mode", "loopback",
		"--command-fd", "0",
		"--status-fd", "3",
		"--display-charset", "utf-8",
		"--batch",
		"--with-colons",
		"--expert",
	}

	if slog.Default().Enabled(context.TODO(), slog.LevelDebug) {
		args = append(args, "--debug-all")
	} else {
		args = append(args, "--quiet")
	}

	args = append(args, baseArgs...)

	slog.Debug("starting automated yubikey operation", "args", args, "commands", commands)

	session, err := y.gpg.ExecuteInteractive(args...)
	if err != nil {
		return fmt.Errorf("failed to start interactive session: %w", err)
	}

	handlerErr := make(chan error, 1)
	go y.gpgInputHandler(session, commands, pinToUse, handlerErr)

	err = <-session.Done

	select {
	case hErr := <-handlerErr:
		if hErr != nil {
			slog.Debug("automated yubikey operation failed in handler", "error", hErr)
			return hErr
		}
	default:
	}

	if err != nil {
		slog.Debug("automated yubikey operation failed", "error", err)
		if errors.Is(err, ErrBadPIN) {
			return ErrBadPIN
		}
		return fmt.Errorf("automated yubikey operation failed: %w", err)
	}

	slog.Debug("automated yubikey operation completed successfully")
	return nil
}

func (y *Yubikey) manualYubikey(baseArgs []string, commands []string) error {
	args := []string{
		"--command-fd", "3",
		"--batch",
		"--display-charset", "utf-8",
		"--expert",
	}

	if slog.Default().Enabled(context.TODO(), slog.LevelDebug) {
		args = append(args, "--debug-all")
	} else {
		args = append(args, "--quiet")
	}

	args = append(args, baseArgs...)

	stdin := strings.Join(commands, "\n") + "\n"

	slog.Debug("starting manual yubikey operation", "args", args, "stdin", stdin)

	_, stderr, err := y.gpg.executeWithPinentry(stdin, args...)
	if err != nil {
		slog.Debug("manual yubikey operation failed", "error", err, "stderr", stderr)
		return fmt.Errorf("manual yubikey operation failed: %w", err)
	}

	slog.Debug("manual yubikey operation completed successfully")
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

	if email != "" && (y.Login == "" || y.Login == "[not set]") {
		if err := y.cardEdit("login", []string{email}, adminPin); err != nil {
			return err
		}
	}

	if name != "" && (y.CardholderName == "" || y.CardholderName == "[not set]") {
		firstName, lastName := splitName(name)
		slog.Debug("split name", "firstName", firstName, "lastName", lastName)

		if err := y.cardEdit("name", []string{lastName, firstName}, adminPin); err != nil {
			return err
		}
	}

	return nil
}

func (y *Yubikey) encryptionKeyToYubikey(fingerprint string) error {
	slog.Debug("moving encryption key to yubikey", "fingerprint", fingerprint)

	adminPin := config.GetYubikeyAdminPin()
	baseArgs := []string{"--edit-key", fingerprint}
	commands := []string{"key 1", "keytocard", "2", "save"}

	if adminPin != "manual" {
		err := y.automatedYubikey(baseArgs, commands, PinTypeAdmin)
		if err == nil {
			slog.Debug("encryption key moved to yubikey successfully")
			return nil
		}
		if errors.Is(err, ErrBadPIN) && adminPin == "" {
			slog.Debug("default pin failed, falling back to manual")
			config.SaveYubikeyAdminPin("manual")
		} else if adminPin != "" {
			return fmt.Errorf("failed to move encryption key to yubikey: %w", err)
		}
	}

	err := y.manualYubikey(baseArgs, commands)
	if err != nil {
		return fmt.Errorf("failed to move encryption key to yubikey: %w", err)
	}

	slog.Debug("encryption key moved to yubikey successfully")
	return nil
}

func (y *Yubikey) VerifyUserPin() error {
	slog.Debug("verifying yubikey user pin")

	baseArgs := []string{"--card-edit"}
	commands := []string{"verify", "quit"}

	err := y.automatedYubikey(baseArgs, commands, PinTypeUser)
	if err != nil {
		slog.Debug("user pin verification failed", "error", err)
		return err
	}

	slog.Debug("user pin verification successful")
	return nil
}

func (y *Yubikey) gpgInputHandler(session *GPGSession, commands []string, pin string, errChan chan<- error) {
	defer close(session.SendInput)

	commandIndex := 0
	var lastFailure string

	for statusLine := range session.StatusMessages {
		slog.Debug("gpg status line", "status", statusLine)
		if !strings.HasPrefix(statusLine, "[GNUPG:]") {
			continue
		}

		parts := strings.Fields(statusLine)
		if len(parts) < 2 {
			continue
		}

		statusType := parts[1]

		switch statusType {
		case "GET_LINE":
			if commandIndex < len(commands) {
				session.SendInput <- commands[commandIndex]
				commandIndex++
			}
		case "GET_HIDDEN":
			session.SendInput <- pin
		case "SC_OP_FAILURE":
			slog.Debug("gpg operation failed", "status", statusLine)
			lastFailure = statusLine
		case "FAILURE":
			slog.Debug("gpg failure", "status", statusLine)
			lastFailure = statusLine
		}
	}

	if lastFailure != "" {
		errChan <- fmt.Errorf("gpg operation failed: %s", lastFailure)
	} else {
		errChan <- nil
	}
}
