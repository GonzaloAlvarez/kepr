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

func Print(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.DefaultBasicText.Print(a...)
}

func Println(a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprint(a...))
	pterm.DefaultBasicText.Print(a...)
	pterm.DefaultBasicText.Print("\n")
}

func Printf(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.DefaultBasicText.Printf(format, a...)
}

func Printfln(format string, a ...interface{}) {
	slog.Debug("cout: " + fmt.Sprintf(format, a...))
	pterm.DefaultBasicText.Printf(format, a...)
	pterm.DefaultBasicText.Print("\n")
}
