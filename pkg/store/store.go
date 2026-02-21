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
	"strings"

	"github.com/gonzaloalvarez/kepr/pkg/gpg"
)

const MaxFileSize = 1 << 20

var (
	ErrAlreadyInitialized  = errors.New("store already initialized")
	ErrInvalidGPGClient    = errors.New("gpg client cannot be nil")
	ErrSecretAlreadyExists = errors.New("secret already exists")
	ErrStoreNotInitialized = errors.New("store not initialized")
	ErrSecretNotFound      = errors.New("secret not found")
	ErrFileTooLarge        = errors.New("file exceeds maximum size of 1MB")
)

type Store struct {
	SecretsPath string
	Fingerprint string
	gpg         *gpg.GPG
}

func New(secretsPath string, gpgClient *gpg.GPG, fingerprint string) (*Store, error) {
	if gpgClient == nil {
		return nil, ErrInvalidGPGClient
	}

	return &Store{
		SecretsPath: secretsPath,
		Fingerprint: fingerprint,
		gpg:         gpgClient,
	}, nil
}

func (s *Store) hasAccess(dirPath string) bool {
	if s.Fingerprint == "" {
		return true
	}
	fingerprints, err := ReadGpgID(dirPath)
	if err != nil {
		return false
	}
	for _, fp := range fingerprints {
		if fp == s.Fingerprint {
			return true
		}
	}
	return false
}

func ReadGpgID(dirPath string) ([]string, error) {
	gpgIDPath := filepath.Join(dirPath, ".gpg.id")
	data, err := os.ReadFile(gpgIDPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .gpg.id: %w", err)
	}
	var fingerprints []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			fingerprints = append(fingerprints, line)
		}
	}
	if len(fingerprints) == 0 {
		return nil, fmt.Errorf("no fingerprints found in .gpg.id")
	}
	return fingerprints, nil
}

func WriteGpgID(dirPath string, fingerprints []string) error {
	if len(fingerprints) == 0 {
		return fmt.Errorf("at least one fingerprint is required")
	}
	content := strings.Join(fingerprints, "\n") + "\n"
	gpgIDPath := filepath.Join(dirPath, ".gpg.id")
	return os.WriteFile(gpgIDPath, []byte(content), 0600)
}

func (s *Store) Init(fingerprints []string) error {
	slog.Debug("initializing store", "path", s.SecretsPath)

	if len(fingerprints) == 0 {
		return fmt.Errorf("at least one fingerprint is required")
	}

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

	slog.Debug("creating .gpg.id file", "fingerprints", fingerprints)
	if err := WriteGpgID(s.SecretsPath, fingerprints); err != nil {
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

func (s *Store) findOrCreateDirectory(parentDirPath string, dirName string, fullPath string) (string, error) {
	slog.Debug("finding or creating directory", "parent", parentDirPath, "name", dirName, "fullPath", fullPath)

	uuid, err := s.findDirectory(parentDirPath, dirName)
	if err == nil {
		return uuid, nil
	}

	slog.Debug("directory not found, creating new", "name", dirName)

	parentFingerprints, err := ReadGpgID(parentDirPath)
	if err != nil {
		return "", fmt.Errorf("failed to read parent .gpg.id: %w", err)
	}

	uuid, err = GenerateUUID()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}

	dirPath := filepath.Join(parentDirPath, uuid)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	if err := WriteGpgID(dirPath, parentFingerprints); err != nil {
		return "", fmt.Errorf("failed to write .gpg.id: %w", err)
	}

	metadata := &Metadata{
		Path: fullPath,
		Type: TypeDir,
	}

	metadataJSON, err := SerializeMetadata(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to serialize metadata: %w", err)
	}

	metadataEncrypted, err := s.gpg.Encrypt(metadataJSON, parentFingerprints...)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt metadata: %w", err)
	}

	metadataPath := filepath.Join(dirPath, uuid+"_md.gpg")
	if err := os.WriteFile(metadataPath, metadataEncrypted, 0600); err != nil {
		return "", fmt.Errorf("failed to write metadata file: %w", err)
	}

	slog.Debug("created new directory", "uuid", uuid, "name", dirName, "fullPath", fullPath)
	return uuid, nil
}
