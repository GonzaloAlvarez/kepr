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
package store

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gonzaloalvarez/kepr/pkg/gpg"
)

var (
	ErrAlreadyInitialized = errors.New("store already initialized")
	ErrInvalidFingerprint = errors.New("fingerprint cannot be empty")
	ErrInvalidGPGClient   = errors.New("gpg client cannot be nil")
)

type Store struct {
	SecretsPath string
	Fingerprint string
	gpg         *gpg.GPG
}

func New(secretsPath string, fingerprint string, gpgClient *gpg.GPG) (*Store, error) {
	if fingerprint == "" {
		return nil, ErrInvalidFingerprint
	}
	if gpgClient == nil {
		return nil, ErrInvalidGPGClient
	}

	return &Store{
		SecretsPath: secretsPath,
		Fingerprint: fingerprint,
		gpg:         gpgClient,
	}, nil
}

func (s *Store) Init() error {
	slog.Debug("initializing store", "path", s.SecretsPath)

	gpgIDPath := filepath.Join(s.SecretsPath, ".gpg.id")

	if _, err := os.Stat(s.SecretsPath); err == nil {
		if _, err := os.Stat(gpgIDPath); err == nil {
			return ErrAlreadyInitialized
		}
	}

	slog.Debug("creating secrets directory", "path", s.SecretsPath)
	if err := os.MkdirAll(s.SecretsPath, 0700); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	slog.Debug("creating .gpg.id file", "fingerprint", s.Fingerprint)
	gpgIDContent := s.Fingerprint + "\n"
	if err := os.WriteFile(gpgIDPath, []byte(gpgIDContent), 0600); err != nil {
		return fmt.Errorf("failed to create .gpg.id file: %w", err)
	}

	gitignorePath := filepath.Join(s.SecretsPath, ".gitignore")
	slog.Debug("creating .gitignore file")
	if err := os.WriteFile(gitignorePath, []byte(GenerateGitignore()), 0600); err != nil {
		return fmt.Errorf("failed to create .gitignore file: %w", err)
	}

	slog.Debug("store initialized successfully")
	return nil
}
