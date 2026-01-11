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
	"path/filepath"

	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/gonzaloalvarez/kepr/pkg/store"
)

type Pass struct {
	SecretsPath string
	gpg         *gpg.GPG
	store       *store.Store
	io          cout.IO
	executor    shell.Executor
}

func New(configDir string, gpgClient *gpg.GPG, io cout.IO, executor shell.Executor) *Pass {
	return &Pass{
		SecretsPath: filepath.Join(configDir, "secrets"),
		gpg:         gpgClient,
		io:          io,
		executor:    executor,
	}
}

func (p *Pass) Init(fingerprint string) error {
	slog.Debug("initializing password store", "path", p.SecretsPath, "fingerprint", fingerprint)

	st, err := store.New(p.SecretsPath, fingerprint, p.gpg)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	p.store = st

	if err := p.store.Init(); err != nil {
		return fmt.Errorf("failed to initialize store: %w", err)
	}

	slog.Debug("password store initialized successfully")
	return nil
}

func (p *Pass) Add(key string) error {
	slog.Debug("adding secret to password store", "key", key)

	if err := p.store.Add(key, p.io); err != nil {
		return fmt.Errorf("failed to add secret: %w", err)
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
