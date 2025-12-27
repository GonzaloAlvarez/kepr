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
package initialize

import (
	"fmt"
	"log/slog"

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/spf13/viper"
)

const githubClientID = "Ov23liaarzPv4HBvyPtW"

func AuthGithub(client github.Client, io cout.IO) (string, error) {
	token := viper.GetString("github_token")
	if token == "" {
		slog.Debug("no token found locally, starting authentication")
		var err error
		token, err = client.Authenticate(githubClientID, io)
		if err != nil {
			return "", fmt.Errorf("authentication failed: %w", err)
		}

		if err := config.SaveToken(token); err != nil {
			return "", fmt.Errorf("failed to save token: %w", err)
		}

		io.Successln("Authentication successful.")
	} else {
		slog.Debug("token foundlocally , skipping authentication")
		io.Infoln("Already authenticated.")
	}

	return token, nil
}

func UserInfo(client github.Client, io cout.IO) error {
	userName := viper.GetString("user_name")
	userEmail := viper.GetString("user_email")
	if userName == "" || userEmail == "" {
		slog.Debug("user identity not found locally, fetching from GitHub")

		name, email, err := client.GetUserIdentity()
		if err != nil {
			return fmt.Errorf("failed to fetch user identity: %w", err)
		}

		io.Infofln("Detected identity: %s <%s>", name, email)

		confirmed, err := io.Confirm(fmt.Sprintf("Is this identity correct? [%s <%s>]", name, email))
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}

		if !confirmed {
			slog.Debug("user rejected identity, requesting correction")
			name, err = io.Input("Correct Name:", name)
			if err != nil {
				return fmt.Errorf("failed to get name: %w", err)
			}

			email, err = io.Input("Correct Email:", email)
			if err != nil {
				return fmt.Errorf("failed to get email: %w", err)
			}
		}

		if err := config.SaveUserIdentity(name, email); err != nil {
			return fmt.Errorf("failed to save user identity: %w", err)
		}

		io.Successfln("User identity saved: %s <%s>", name, email)
	} else {
		io.Successfln("Welcome back, %s!", userName)
		slog.Debug("user identity already configured", "name", userName, "email", userEmail)
	}

	return nil
}
