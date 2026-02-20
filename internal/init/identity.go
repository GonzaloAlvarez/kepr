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
	"os"

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
)

var (
	githubClientID     = "Ov23liaarzPv4HBvyPtW"
	githubClientSecret = ""
)

func AuthGithub(client github.Client, io cout.IO, headless bool) (string, error) {
	token := config.GetToken()
	if token == "" {
		slog.Debug("no token found locally, starting authentication")
		var err error

		ciMode := os.Getenv("KEPR_CI") == "true"

		if ciMode || headless || githubClientSecret == "" {
			if ciMode {
				slog.Debug("CI mode: forcing device code flow")
			} else if headless {
				slog.Debug("headless mode: forcing device code flow")
			} else {
				slog.Debug("client secret not available, using device code flow")
			}
			token, err = client.CodeBasedAuthentication(githubClientID, io)
		} else {
			slog.Debug("client secret available, using PKCE flow")
			token, err = client.PKCEAuthentication(githubClientID, githubClientSecret, io)
		}

		if err != nil {
			return "", fmt.Errorf("authentication failed: %w", err)
		}

		if err := config.SaveToken(token); err != nil {
			return "", fmt.Errorf("failed to save token: %w", err)
		}

		io.Successln("Authentication successful.")
	} else {
		slog.Debug("token found locally, skipping authentication")
		io.Infoln("Already authenticated.")
	}

	return token, nil
}

func UserInfo(client github.Client, io cout.IO) error {
	userName := config.GetUserName()
	userEmail := config.GetUserEmail()
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
