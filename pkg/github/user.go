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
package github

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/go-github/v67/github"
)

func FetchUserIdentity(client *github.Client) (string, string, error) {
	ctx := context.Background()

	slog.Debug("fetching user profile from GitHub")
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		slog.Error("failed to fetch user profile", "error", err)
		return "", "", err
	}

	name := ""
	if user.Name != nil {
		name = *user.Name
	}

	email := ""
	if user.Email != nil && *user.Email != "" {
		email = *user.Email
	} else {
		slog.Debug("email not found in profile, fetching from email list")
		emails, _, err := client.Users.ListEmails(ctx, nil)
		if err != nil {
			slog.Error("failed to fetch user emails", "error", err)
			return name, "", err
		}

		for _, e := range emails {
			if e.Primary != nil && *e.Primary && e.Verified != nil && *e.Verified {
				email = *e.Email
				slog.Debug("found primary verified email", "email", email)
				break
			}
		}

		if email == "" && len(emails) > 0 {
			for _, e := range emails {
				if e.Verified != nil && *e.Verified {
					email = *e.Email
					slog.Debug("found verified email", "email", email)
					break
				}
			}
		}

		if email == "" {
			return name, "", fmt.Errorf("no verified email found in GitHub account")
		}
	}

	slog.Debug("user identity fetched", "name", name, "email", email)
	return name, email, nil
}
