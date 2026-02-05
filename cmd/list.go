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
	"github.com/gonzaloalvarez/kepr/internal/list"
	"github.com/spf13/cobra"
)

func NewListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:     "list [path]",
		Aliases: []string{"ls"},
		Short:   "List secrets and directories at a path",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoPath, err := RequireRepo()
			if err != nil {
				return err
			}
			path := ""
			if len(args) > 0 {
				path = args[0]
			}
			w := list.NewWorkflow(path, repoPath, app.GitHub, app.Shell, app.UI)
			return w.Run(cmd.Context())
		},
	}
}
