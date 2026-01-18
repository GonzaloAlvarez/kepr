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

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/spf13/cobra"
)

func NewListCmd(app *App) *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured repositories or identities",
	}

	listCmd.AddCommand(newListReposCmd(app))
	listCmd.AddCommand(newListIdentitiesCmd(app))

	return listCmd
}

func newListReposCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "repos",
		Short: "List configured repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			repos := config.ListRepos()
			defaultRepo := config.GetDefaultRepo()

			if len(repos) == 0 {
				app.UI.Warning("No repositories configured")
				return nil
			}

			for _, repo := range repos {
				if repo == defaultRepo {
					fmt.Printf("* %s (default)\n", repo)
				} else {
					fmt.Printf("  %s\n", repo)
				}
			}
			return nil
		},
	}
}

func newListIdentitiesCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "identities",
		Short: "List configured GPG identities",
		RunE: func(cmd *cobra.Command, args []string) error {
			identities := config.ListIdentities()

			if len(identities) == 0 {
				app.UI.Warning("No identities configured")
				return nil
			}

			for fingerprint, identity := range identities {
				fmt.Printf("%s\n", fingerprint)
				fmt.Printf("  Name:  %s\n", identity.Name)
				fmt.Printf("  Email: %s\n", identity.Email)
				if identity.YubikeySerial != "" {
					fmt.Printf("  YubiKey: %s\n", identity.YubikeySerial)
				}
				fmt.Println()
			}
			return nil
		},
	}
}
