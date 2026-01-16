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
	"path/filepath"

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/git"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/pass"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/gonzaloalvarez/kepr/pkg/store"
)

func Execute(key string, githubClient github.Client, executor shell.Executor, io cout.IO) error {
	token := config.GetToken()
	if token == "" {
		return fmt.Errorf("not authenticated: run 'kepr init' first")
	}
	githubClient.SetToken(token)

	if err := IsInitialized(githubClient, executor, io); err != nil {
		return err
	}

	configDir, err := config.Dir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	secretsPath := filepath.Join(configDir, "secrets")
	fingerprint := config.GetUserFingerprint()

	if fingerprint == "" {
		return fmt.Errorf("fingerprint not found: run 'kepr init' first")
	}

	g, err := gpg.New(configDir, executor, io)
	if err != nil {
		return fmt.Errorf("failed to initialize gpg: %w", err)
	}

	st, err := store.New(secretsPath, fingerprint, g)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	gitClient, err := git.New(executor)
	if err != nil {
		return fmt.Errorf("failed to initialize git client: %w", err)
	}

	p := pass.New(configDir, g, gitClient, io, executor, st)

	if err := p.Add(key); err != nil {
		return err
	}

	io.Successfln("Secret added: %s", key)

	if err := gitClient.Push(secretsPath, "origin", "main"); err != nil {
		return fmt.Errorf("failed to push to remote: %w", err)
	}

	io.Successfln("Pushed to remote repository")

	return nil
}
