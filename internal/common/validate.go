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
package common

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

func ValidateToken(token string) error {
	slog.Debug("validating token")
	if token == "" {
		return fmt.Errorf("not authenticated: run 'kepr init' first")
	}
	return nil
}

func ValidateConfigDir() (string, error) {
	slog.Debug("validating config directory")
	configDir, err := config.Dir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return "", fmt.Errorf("kepr is not initialized: run 'kepr init' first")
	}
	return configDir, nil
}

func ValidateUserIdentity() (string, string, error) {
	slog.Debug("validating user identity")
	userName := config.GetUserName()
	userEmail := config.GetUserEmail()

	if userName == "" || userEmail == "" {
		return "", "", fmt.Errorf("user identity not configured: run 'kepr init' first")
	}
	return userName, userEmail, nil
}

func ValidateGitHubIdentity(gh github.Client, expectedEmail string) error {
	slog.Debug("validating GitHub identity")
	_, email, err := gh.GetUserIdentity()
	if err != nil {
		return fmt.Errorf("failed to validate GitHub token: %w", err)
	}

	if email != expectedEmail {
		return fmt.Errorf("email mismatch: GitHub (%s) != config (%s)", email, expectedEmail)
	}
	return nil
}

func ValidateGPGSetup(configDir string, executor shell.Executor, io cout.IO) (*gpg.GPG, error) {
	slog.Debug("validating GPG setup")
	gpgHome := configDir + "/gpg"
	if _, err := os.Stat(gpgHome); os.IsNotExist(err) {
		return nil, fmt.Errorf("GPG directory does not exist: run 'kepr init' first")
	}

	g, err := gpg.New(configDir, executor, io)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GPG: %w", err)
	}
	return g, nil
}

func ValidateGPGKey(g *gpg.GPG, expectedEmail string) error {
	slog.Debug("validating GPG key")
	keys, err := g.ListPublicKeys()
	if err != nil {
		return fmt.Errorf("failed to list GPG keys: %w", err)
	}

	for _, key := range keys {
		if key.Email == expectedEmail {
			slog.Debug("GPG key validation passed")
			return nil
		}
	}

	return fmt.Errorf("no GPG key found with email %s", expectedEmail)
}

func ValidateFingerprint() (string, error) {
	slog.Debug("validating fingerprint")
	fingerprint := config.GetUserFingerprint()
	if fingerprint == "" {
		return "", fmt.Errorf("fingerprint not found: run 'kepr init' first")
	}
	return fingerprint, nil
}

func GetSecretsPath(repoPath string) (string, error) {
	slog.Debug("getting secrets path", "repo", repoPath)
	secretsPath, err := config.SecretsPathForRepo(repoPath)
	if err != nil {
		return "", fmt.Errorf("failed to get secrets path: %w", err)
	}
	return secretsPath, nil
}
