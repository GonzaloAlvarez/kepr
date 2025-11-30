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
	"log/slog"

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const githubClientID = "Iv23li6c8kmcEmGBK3yC"

var initCmd = &cobra.Command{
	Use:   "init [username/repo]",
	Short: "Initialize a new kepr repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]

		token := viper.GetString("github_token")
		if token == "" {
			slog.Debug("no token found locally, starting authentication")
			var err error
			token, err = github.Authenticate(githubClientID)
			if err != nil {
				return fmt.Errorf("authentication failed: %w", err)
			}

			if err := config.SaveToken(token); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}

			cout.Successln("Authentication successful.")
		} else {
			slog.Debug("token foundlocally , skipping authentication")
			cout.Infoln("Already authenticated.")
		}

		userName := viper.GetString("user_name")
		userEmail := viper.GetString("user_email")
		if userName == "" || userEmail == "" {
			slog.Debug("user identity not found locally, fetching from GitHub")
			client := github.NewClient(token)

			name, email, err := github.FetchUserIdentity(client)
			if err != nil {
				return fmt.Errorf("failed to fetch user identity: %w", err)
			}

			cout.Infofln("Detected identity: %s <%s>", name, email)

			confirmed, err := cout.Confirm(fmt.Sprintf("Is this identity correct? [%s <%s>]", name, email))
			if err != nil {
				return fmt.Errorf("confirmation failed: %w", err)
			}

			if !confirmed {
				slog.Debug("user rejected identity, requesting correction")
				name, err = cout.Input("Correct Name:", name)
				if err != nil {
					return fmt.Errorf("failed to get name: %w", err)
				}

				email, err = cout.Input("Correct Email:", email)
				if err != nil {
					return fmt.Errorf("failed to get email: %w", err)
				}
			}

			if err := config.SaveUserIdentity(name, email); err != nil {
				return fmt.Errorf("failed to save user identity: %w", err)
			}

			cout.Successfln("User identity saved: %s <%s>", name, email)
		} else {
			cout.Successfln("Welcome back, %s!", userName)
			slog.Debug("user identity already configured", "name", userName, "email", userEmail)
		}

		cout.Infofln("Initializing kepr for repo: %s", repo)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
