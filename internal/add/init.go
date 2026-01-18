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
package add

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

func IsInitialized(repoPath string, githubClient github.Client, executor shell.Executor, io cout.IO) error {
	slog.Debug("checking if kepr is initialized", "repo", repoPath)

	configDir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("kepr is not initialized: run 'kepr init' first")
	}

	configUserName := config.GetUserName()
	configUserEmail := config.GetUserEmail()

	if configUserName == "" || configUserEmail == "" {
		return fmt.Errorf("user identity not configured: run 'kepr init' first")
	}

	_, email, err := githubClient.GetUserIdentity()
	if err != nil {
		return fmt.Errorf("failed to validate GitHub token: %w", err)
	}

	if email != configUserEmail {
		return fmt.Errorf("email mismatch: GitHub (%s) != config (%s)", email, configUserEmail)
	}

	gpgHome := configDir + "/gpg"
	if _, err := os.Stat(gpgHome); os.IsNotExist(err) {
		return fmt.Errorf("GPG directory does not exist: run 'kepr init' first")
	}

	g, err := gpg.New(configDir, executor, io)
	if err != nil {
		return fmt.Errorf("failed to initialize GPG: %w", err)
	}

	keys, err := g.ListPublicKeys()
	if err != nil {
		return fmt.Errorf("failed to list GPG keys: %w", err)
	}

	found := false
	for _, key := range keys {
		if key.Email == configUserEmail {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("no GPG key found with email %s", configUserEmail)
	}

	slog.Debug("initialization check passed")
	return nil
}
