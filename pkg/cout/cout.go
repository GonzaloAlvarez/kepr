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

	"github.com/pterm/pterm"
)

func Info(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.DefaultBasicText.Print(a...)
}

func Infoln(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.DefaultBasicText.Print(a...)
	pterm.DefaultBasicText.Print("\n")
}

func Infof(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.DefaultBasicText.Printf(format, a...)
}

func Infofln(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.DefaultBasicText.Printf(format, a...)
	pterm.DefaultBasicText.Print("\n")
}

func Success(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.FgGreen.Print(a...)
}

func Successln(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.FgGreen.Print(a...)
	pterm.FgGreen.Print("\n")
}

func Successf(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.FgGreen.Printf(format, a...)
}

func Successfln(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.FgGreen.Printf(format, a...)
	pterm.FgGreen.Print("\n")
}

func Confirm(message string) (bool, error) {
	slog.Debug("cout: confirm prompt", "message", message)
	result, err := pterm.DefaultInteractiveConfirm.Show(message)
	if err != nil {
		slog.Error("confirm prompt failed", "error", err)
		return false, err
	}
	slog.Debug("cout: confirm result", "result", result)
	return result, nil
}

func Input(message string, defaultValue string) (string, error) {
	slog.Debug("cout: input prompt", "message", message, "default", defaultValue)
	result, err := pterm.DefaultInteractiveTextInput.WithDefaultValue(defaultValue).Show(message)
	if err != nil {
		slog.Error("input prompt failed", "error", err)
		return "", err
	}
	slog.Debug("cout: input result", "result", result)
	return result, nil
}

func WarningMessage(message string) {
	slog.Debug("cout: warning - " + message)
	pterm.FgRed.Println(message)
}
