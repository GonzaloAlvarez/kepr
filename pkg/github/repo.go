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
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/go-github/v67/github"
)

func ExtractRepoName(fullPath string) string {
	parts := strings.Split(fullPath, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return fullPath
}

func ExtractRepoOwner(fullPath string) string {
	parts := strings.Split(fullPath, "/")
	if len(parts) >= 1 {
		return parts[0]
	}
	return ""
}

func NormalizeRepoPath(input string) string {
	parts := strings.Split(input, "/")
	if len(parts) >= 2 {
		return input
	}
	return input + "/kepr-store"
}

func (c *GitHubClient) CheckRepoExists(name string) (bool, error) {
	slog.Debug("checking if repository exists", "name", name)

	owner, _, err := c.client.Users.Get(c.ctx, "")
	if err != nil {
		slog.Error("failed to get current user", "error", err)
		return false, fmt.Errorf("failed to get current user: %w", err)
	}

	ownerLogin := ""
	if owner.Login != nil {
		ownerLogin = *owner.Login
	} else {
		return false, fmt.Errorf("user login not found")
	}

	_, resp, err := c.client.Repositories.Get(c.ctx, ownerLogin, name)
	if err == nil {
		slog.Debug("repository exists", "name", name)
		return true, nil
	}

	if resp == nil || resp.StatusCode != 404 {
		slog.Error("failed to check repository", "error", err)
		return false, fmt.Errorf("failed to check repository: %w", err)
	}

	slog.Debug("repository does not exist", "name", name)
	return false, nil
}

func (c *GitHubClient) CreateRepo(name string) error {
	slog.Debug("creating repository", "name", name)

	repo := &github.Repository{
		Name:    github.String(name),
		Private: github.Bool(true),
	}

	_, _, err := c.client.Repositories.Create(c.ctx, "", repo)
	if err != nil {
		slog.Error("failed to create repository", "error", err)
		return fmt.Errorf("failed to create repository: %w", err)
	}

	slog.Debug("repository created successfully", "name", name, "private", true)
	return nil
}
