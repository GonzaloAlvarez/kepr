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
	"strings"

	initialize "github.com/gonzaloalvarez/kepr/internal/init"
	"github.com/spf13/cobra"
)

const defaultInitRepoName = "kepr-store"

func NewInitCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [repo-name]",
		Short: "Initialize a new kepr repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoName := defaultInitRepoName
			if len(args) > 0 {
				repoName = args[0]
				if strings.Contains(repoName, "/") {
					return fmt.Errorf("repo name must not contain '/'")
				}
			}
			headless, _ := cmd.Flags().GetBool("headless")
			w := initialize.NewWorkflow(repoName, headless, app.GitHub, app.Shell, app.UI)
			return w.Run(cmd.Context())
		},
	}
	cmd.Flags().Bool("headless", false, "initialize without YubiKey or browser (for remote/VM environments)")
	return cmd
}
