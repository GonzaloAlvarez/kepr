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
package cout

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/pterm/pterm"
)

type IO interface {
	Confirm(prompt string) (bool, error)
	Input(prompt string, defaultValue string) (string, error)
	InputPassword(prompt string) (string, error)
	Info(a ...interface{})
	Infoln(a ...interface{})
	Infof(format string, a ...interface{})
	Infofln(format string, a ...interface{})
	Success(a ...interface{})
	Successln(a ...interface{})
	Successf(format string, a ...interface{})
	Successfln(format string, a ...interface{})
	Warning(message string)
}

type Terminal struct {
	ciMode bool
}

func NewTerminal() *Terminal {
	ciMode := os.Getenv("KEPR_CI") == "true"
	if ciMode {
		slog.Debug("CI mode enabled: prompts will be auto-confirmed")
	}
	return &Terminal{
		ciMode: ciMode,
	}
}

func (t *Terminal) Confirm(prompt string) (bool, error) {
	slog.Debug("cout: confirm prompt", "message", prompt)
	if t.ciMode {
		slog.Debug("CI mode: auto-confirming", "prompt", prompt)
		return true, nil
	}
	result, err := pterm.DefaultInteractiveConfirm.Show(prompt)
	if err != nil {
		slog.Error("confirm prompt failed", "error", err)
		return false, err
	}
	slog.Debug("cout: confirm result", "result", result)
	return result, nil
}

func (t *Terminal) Input(prompt string, defaultValue string) (string, error) {
	slog.Debug("cout: input prompt", "message", prompt, "default", defaultValue)
	if t.ciMode {
		slog.Debug("CI mode: using default value", "prompt", prompt, "value", defaultValue)
		return defaultValue, nil
	}
	result, err := pterm.DefaultInteractiveTextInput.WithDefaultValue(defaultValue).Show(prompt)
	if err != nil {
		slog.Error("input prompt failed", "error", err)
		return "", err
	}
	slog.Debug("cout: input result", "result", result)
	return result, nil
}

func (t *Terminal) InputPassword(prompt string) (string, error) {
	slog.Debug("cout: password input prompt", "message", prompt)

	if t.ciMode {
		slog.Debug("CI mode: reading password from stdin")
		var password string
		_, err := fmt.Scanln(&password)
		if err != nil {
			slog.Error("failed to read password from stdin", "error", err)
			return "", fmt.Errorf("CI mode: failed to read password from stdin: %w", err)
		}
		if password == "" {
			return "", fmt.Errorf("CI mode: password cannot be empty")
		}
		return password, nil
	}

	maxAttempts := 5
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		password1, err := pterm.DefaultInteractiveTextInput.WithMask("*").Show(prompt)
		if err != nil {
			slog.Error("password input failed", "error", err)
			return "", err
		}

		if password1 == "" {
			return "", fmt.Errorf("password cannot be empty")
		}

		password2, err := pterm.DefaultInteractiveTextInput.WithMask("*").Show("Confirm " + prompt)
		if err != nil {
			slog.Error("password confirmation failed", "error", err)
			return "", err
		}

		if password1 == password2 {
			slog.Debug("cout: password input successful")
			return password1, nil
		}

		if attempt < maxAttempts {
			pterm.Warning.Println("Passwords do not match. Please try again.")
		}
	}

	return "", fmt.Errorf("passwords do not match after %d attempts", maxAttempts)
}

func (t *Terminal) Info(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.DefaultBasicText.Print(a...)
}

func (t *Terminal) Infoln(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.DefaultBasicText.Print(a...)
	pterm.DefaultBasicText.Print("\n")
}

func (t *Terminal) Infof(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.DefaultBasicText.Printf(format, a...)
}

func (t *Terminal) Infofln(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.DefaultBasicText.Printf(format, a...)
	pterm.DefaultBasicText.Print("\n")
}

func (t *Terminal) Success(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.FgGreen.Print(a...)
}

func (t *Terminal) Successln(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.FgGreen.Print(a...)
	pterm.FgGreen.Print("\n")
}

func (t *Terminal) Successf(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.FgGreen.Printf(format, a...)
}

func (t *Terminal) Successfln(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.FgGreen.Printf(format, a...)
	pterm.FgGreen.Print("\n")
}

func (t *Terminal) Warning(message string) {
	slog.Debug("cout: warning - " + message)
	pterm.FgRed.Println(message)
}
