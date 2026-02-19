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
	"os"

	"github.com/gonzaloalvarez/kepr/internal/add"
	"github.com/spf13/cobra"
)

func NewAddCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "add [key] [file]",
		Aliases: []string{"insert"},
		Short:   "Add a secret or file to the store",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath, err := RequireRepo()
			if err != nil {
				return err
			}

			var filePath string
			if len(args) == 2 {
				filePath = args[1]
				if _, err := os.Stat(filePath); err != nil {
					return fmt.Errorf("file not found: %s", filePath)
				}
			}

			w := add.NewWorkflow(args[0], filePath, repoPath, app.GitHub, app.Shell, app.UI)
			return w.Run(cmd.Context())
		},
	}
}
