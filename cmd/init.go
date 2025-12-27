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

	initialize "github.com/gonzaloalvarez/kepr/internal/init"
	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/spf13/cobra"
)

func NewInitCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "init [username/repo]",
		Short: "Initialize a new kepr repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo := args[0]

			token, err := initialize.AuthGithub(app.GitHub, app.UI)
			if err != nil {
				return err
			}

			app.GitHub.SetToken(token)

			repoName := github.ExtractRepoName(repo)
			exists, err := app.GitHub.CheckRepoExists(repoName)
			if err != nil {
				return fmt.Errorf("failed to check repository: %w", err)
			}

			if exists {
				return fmt.Errorf("repository '%s' already exists", repoName)
			}

			if err := app.GitHub.CreateRepo(repoName); err != nil {
				return fmt.Errorf("failed to create remote repository: %w", err)
			}

			app.UI.Successfln("Created private remote repository: github.com/%s", repo)

			if err := initialize.UserInfo(app.GitHub, app.UI); err != nil {
				return err
			}

			if err := initialize.SetupGPG(app.Shell, app.UI); err != nil {
				return err
			}

			configDir, err := config.Dir()
			if err != nil {
				return err
			}

			gpgHome := configDir + "/gpg"
			fingerprint := config.GetUserFingerprint()

			if err := initialize.SetupPasswordStore(configDir, gpgHome, fingerprint, app.Shell, app.UI); err != nil {
				return err
			}

			return nil
		},
	}
}
