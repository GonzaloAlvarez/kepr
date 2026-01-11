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

func (s *Store) findDirectory(parentPath string, dirName string) (string, error) {
	slog.Debug("finding directory", "parent", parentPath, "name", dirName)

	entries, err := os.ReadDir(parentPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		uuid := entry.Name()
		metadataPath := filepath.Join(parentPath, uuid, uuid+"_md.gpg")

		metadataEncrypted, err := os.ReadFile(metadataPath)
		if err != nil {
			slog.Debug("failed to read metadata file, skipping", "path", metadataPath, "error", err)
			continue
		}

		metadataDecrypted, err := s.gpg.Decrypt(metadataEncrypted)
		if err != nil {
			slog.Debug("failed to decrypt metadata, skipping", "path", metadataPath, "error", err)
			continue
		}

		metadata, err := DeserializeMetadata(metadataDecrypted)
		if err != nil {
			slog.Debug("failed to deserialize metadata, skipping", "path", metadataPath, "error", err)
			continue
		}

		if metadata.Path == dirName && metadata.Type == TypeDir {
			slog.Debug("found directory", "uuid", uuid, "name", dirName)
			return uuid, nil
		}
	}

	return "", fmt.Errorf("directory not found: %s", dirName)
}

func (s *Store) findSecret(parentPath string, secretName string) (string, error) {
	slog.Debug("finding secret", "parent", parentPath, "name", secretName)

	entries, err := os.ReadDir(parentPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if filepath.Ext(fileName) != ".gpg" {
			continue
		}

		baseFileName := fileName[:len(fileName)-len(".gpg")]
		if len(baseFileName) == 0 {
			continue
		}

		metadataFileName := baseFileName + "_md.gpg"
		metadataPath := filepath.Join(parentPath, metadataFileName)

		if _, err := os.Stat(metadataPath); err != nil {
			continue
		}

		metadataEncrypted, err := os.ReadFile(metadataPath)
		if err != nil {
			slog.Debug("failed to read metadata file, skipping", "path", metadataPath, "error", err)
			continue
		}

		metadataDecrypted, err := s.gpg.Decrypt(metadataEncrypted)
		if err != nil {
			slog.Debug("failed to decrypt metadata, skipping", "path", metadataPath, "error", err)
			continue
		}

		metadata, err := DeserializeMetadata(metadataDecrypted)
		if err != nil {
			slog.Debug("failed to deserialize metadata, skipping", "path", metadataPath, "error", err)
			continue
		}

		if metadata.Path == secretName && metadata.Type == TypePassword {
			slog.Debug("found secret", "uuid", baseFileName, "name", secretName)
			return baseFileName, nil
		}
	}

	return "", fmt.Errorf("secret not found: %s", secretName)
}
