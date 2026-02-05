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
package pass

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/git"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/gonzaloalvarez/kepr/pkg/store"
)

type Pass struct {
	SecretsPath string
	gpg         *gpg.GPG
	store       *store.Store
	git         *git.Git
	io          cout.IO
	executor    shell.Executor
}

func New(secretsPath string, gpgClient *gpg.GPG, gitClient *git.Git, io cout.IO, executor shell.Executor, st *store.Store) *Pass {
	return &Pass{
		SecretsPath: secretsPath,
		gpg:         gpgClient,
		git:         gitClient,
		store:       st,
		io:          io,
		executor:    executor,
	}
}

func (p *Pass) Init(fingerprint string) error {
	slog.Debug("initializing password store", "path", p.SecretsPath, "fingerprint", fingerprint)

	if err := p.store.Init(); err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	slog.Debug("password store initialized successfully")
	return nil
}

func (p *Pass) Add(key string) error {
	slog.Debug("adding secret to password store", "key", key)

	uuid, err := p.store.Add(key, p.io)
	if err != nil {
		return fmt.Errorf("failed to add secret: %w", err)
	}

	userName := config.GetUserName()
	userEmail := config.GetUserEmail()

	if err := p.git.Commit(p.SecretsPath, "updated store with new UUID "+uuid, userName, userEmail); err != nil {
		return fmt.Errorf("failed to commit changes: %w", err)
	}

	slog.Debug("secret added successfully")
	return nil
}

func (p *Pass) Get(key string) error {
	slog.Debug("getting secret from password store", "key", key)

	secretBytes, err := p.store.Get(key)
	if err != nil {
		return fmt.Errorf("failed to get secret: %w", err)
	}

	if _, err := os.Stdout.Write(secretBytes); err != nil {
		return fmt.Errorf("failed to write secret to stdout: %w", err)
	}

	slog.Debug("secret retrieved successfully")
	return nil
}

func (p *Pass) List(path string) ([]store.Entry, error) {
	slog.Debug("listing entries from password store", "path", path)

	entries, err := p.store.List(path)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}

	slog.Debug("entries listed successfully", "count", len(entries))
	return entries, nil
}
