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
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewAddCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "add [key] [value]",
		Short: "Add a secret to the store",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			var value string

			if len(args) == 2 {
				value = args[1]
			} else {
				return fmt.Errorf("interactive prompt not implemented yet, please provide value argument")
			}

			app.UI.Infofln("Adding secret: %s", key)
			_ = value
			return nil
		},
	}
}
