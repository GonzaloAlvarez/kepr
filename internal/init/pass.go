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
	"path/filepath"

	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/git"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/pass"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/gonzaloalvarez/kepr/pkg/store"
)

func SetupPasswordStore(configDir, repoPath string, g *gpg.GPG, fingerprint string, executor shell.Executor, io cout.IO) error {
	slog.Debug("initializing password store", "repo", repoPath)

	secretsPath := filepath.Join(configDir, repoPath)

	st, err := store.New(secretsPath, g)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	gitClient := git.New()

	p := pass.New(secretsPath, g, gitClient, io, executor, st)

	if err := p.Init([]string{fingerprint}); err != nil {
		return fmt.Errorf("failed to initialize password store: %w", err)
	}

	if err := gitClient.Init(secretsPath); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	io.Successfln("Initialized local secret store at %s", p.SecretsPath)
	return nil
}
