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
)

func (s *Store) Get(path string) ([]byte, *Metadata, error) {
	slog.Debug("getting secret", "path", path)

	gpgIDPath := filepath.Join(s.SecretsPath, ".gpg.id")
	if _, err := os.Stat(gpgIDPath); err != nil {
		return nil, nil, ErrStoreNotInitialized
	}

	normalizedPath, err := NormalizePath(path)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid path: %w", err)
	}

	segments := SplitPath(normalizedPath)
	if len(segments) == 0 {
		return nil, nil, fmt.Errorf("path cannot be empty")
	}

	dirSegments := segments[:len(segments)-1]
	secretName := segments[len(segments)-1]

	slog.Debug("path segments", "dirs", dirSegments, "secret", secretName)

	currentPath := s.SecretsPath
	for _, segment := range dirSegments {
		uuid, err := s.findDirectory(currentPath, segment)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find directory %s: %w", segment, err)
		}
		currentPath = filepath.Join(currentPath, uuid)
	}

	slog.Debug("looking for secret in directory", "path", currentPath, "name", secretName)

	uuid, err := s.findSecret(currentPath, secretName)
	if err != nil {
		return nil, nil, ErrSecretNotFound
	}

	secretPath := filepath.Join(currentPath, uuid+".gpg")
	slog.Debug("reading secret file", "path", secretPath)

	secretEncrypted, err := os.ReadFile(secretPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read secret file: %w", err)
	}

	secretDecrypted, err := s.gpg.Decrypt(secretEncrypted)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	metadataPath := filepath.Join(currentPath, uuid+"_md.gpg")
	metadataEncrypted, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	metadataDecrypted, err := s.gpg.Decrypt(metadataEncrypted)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt metadata: %w", err)
	}

	metadata, err := DeserializeMetadata(metadataDecrypted)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to deserialize metadata: %w", err)
	}

	slog.Debug("secret retrieved successfully", "path", normalizedPath, "type", metadata.Type)
	return secretDecrypted, metadata, nil
}
