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
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gonzaloalvarez/kepr/pkg/cout"
)

func (s *Store) Add(path string, io cout.IO) (string, error) {
	slog.Debug("adding secret", "path", path)

	gpgIDPath := filepath.Join(s.SecretsPath, ".gpg.id")
	if _, err := os.Stat(gpgIDPath); err != nil {
		return "", ErrStoreNotInitialized
	}

	normalizedPath, err := NormalizePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	segments := SplitPath(normalizedPath)
	if len(segments) == 0 {
		return "", fmt.Errorf("path cannot be empty")
	}

	dirSegments := segments[:len(segments)-1]
	secretName := segments[len(segments)-1]

	slog.Debug("path segments", "dirs", dirSegments, "secret", secretName)

	currentPath := s.SecretsPath
	for _, segment := range dirSegments {
		uuid, err := s.findOrCreateDirectory(currentPath, segment)
		if err != nil {
			return "", fmt.Errorf("failed to create directory structure: %w", err)
		}
		currentPath = filepath.Join(currentPath, uuid)
	}

	slog.Debug("checking if secret already exists", "path", currentPath, "name", secretName)

	_, err = s.findSecret(currentPath, secretName)
	if err == nil {
		return "", ErrSecretAlreadyExists
	}

	slog.Debug("reading secret value from user")
	secretValue, err := io.InputPassword("Enter secret for " + normalizedPath)
	if err != nil {
		return "", fmt.Errorf("failed to read secret value: %w", err)
	}

	uuid, err := GenerateUUID()
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}

	slog.Debug("encrypting secret", "uuid", uuid)

	secretEncrypted, err := s.gpg.Encrypt([]byte(secretValue), s.Fingerprint)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt secret: %w", err)
	}

	secretPath := filepath.Join(currentPath, uuid+".gpg")
	if err := os.WriteFile(secretPath, secretEncrypted, 0600); err != nil {
		return "", fmt.Errorf("failed to write secret file: %w", err)
	}

	slog.Debug("encrypting metadata")

	metadata := &Metadata{
		Path: secretName,
		Type: TypePassword,
	}

	metadataJSON, err := SerializeMetadata(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to serialize metadata: %w", err)
	}

	metadataEncrypted, err := s.gpg.Encrypt(metadataJSON, s.Fingerprint)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt metadata: %w", err)
	}

	metadataPath := filepath.Join(currentPath, uuid+"_md.gpg")
	if err := os.WriteFile(metadataPath, metadataEncrypted, 0600); err != nil {
		return "", fmt.Errorf("failed to write metadata file: %w", err)
	}

	slog.Debug("secret added successfully", "path", normalizedPath, "uuid", uuid)
	return uuid, nil
}
