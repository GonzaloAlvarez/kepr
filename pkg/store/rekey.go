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
	"strings"
)

func (s *Store) ResolvePath(path string) (string, error) {
	normalizedPath, err := NormalizePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	segments := SplitPath(normalizedPath)
	currentPath := s.SecretsPath
	for _, segment := range segments {
		uuid, err := s.findDirectory(currentPath, segment)
		if err != nil {
			return "", fmt.Errorf("failed to resolve path segment %q: %w", segment, err)
		}
		currentPath = filepath.Join(currentPath, uuid)
	}

	return currentPath, nil
}

func (s *Store) Rekey(dirPath string, updatedFingerprints []string, logicalPath string) error {
	slog.Debug("rekeying directory", "path", dirPath, "logicalPath", logicalPath, "recipients", updatedFingerprints)

	if err := WriteGpgID(dirPath, updatedFingerprints); err != nil {
		return fmt.Errorf("failed to write .gpg.id: %w", err)
	}

	dirName := filepath.Base(dirPath)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(dirPath, entry.Name())
			gpgIDPath := filepath.Join(subDir, ".gpg.id")
			if _, err := os.Stat(gpgIDPath); err != nil {
				continue
			}

			subLogicalPath := s.resolveSubdirLogicalPath(subDir, entry.Name(), logicalPath)
			if err := s.Rekey(subDir, updatedFingerprints, subLogicalPath); err != nil {
				return fmt.Errorf("failed to rekey subdirectory %s: %w", entry.Name(), err)
			}
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".gpg") {
			continue
		}

		filePath := filepath.Join(dirPath, name)
		slog.Debug("rekeying file", "path", filePath)

		encrypted, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", name, err)
		}

		decrypted, err := s.gpg.Decrypt(encrypted)
		if err != nil {
			return fmt.Errorf("failed to decrypt %s: %w", name, err)
		}

		isDirMetadata := strings.HasSuffix(name, "_md.gpg") && strings.TrimSuffix(name, "_md.gpg") == dirName
		if isDirMetadata && logicalPath != "" {
			metadata, mErr := DeserializeMetadata(decrypted)
			if mErr == nil && metadata.Type == TypeDir {
				metadata.Path = logicalPath
				updated, sErr := SerializeMetadata(metadata)
				if sErr == nil {
					decrypted = updated
				}
			}
		}

		reencrypted, err := s.gpg.Encrypt(decrypted, updatedFingerprints...)
		if err != nil {
			return fmt.Errorf("failed to re-encrypt %s: %w", name, err)
		}

		if err := os.WriteFile(filePath, reencrypted, 0600); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}

	slog.Debug("rekeying complete", "path", dirPath)
	return nil
}

func (s *Store) resolveSubdirLogicalPath(subDir string, uuid string, parentLogicalPath string) string {
	metadataPath := filepath.Join(subDir, uuid+"_md.gpg")
	encrypted, err := os.ReadFile(metadataPath)
	if err != nil {
		return parentLogicalPath + "/" + uuid
	}
	decrypted, err := s.gpg.Decrypt(encrypted)
	if err != nil {
		return parentLogicalPath + "/" + uuid
	}
	metadata, err := DeserializeMetadata(decrypted)
	if err != nil || metadata.Type != TypeDir {
		return parentLogicalPath + "/" + uuid
	}
	segmentName := pathSegment(metadata.Path)
	return parentLogicalPath + "/" + segmentName
}
