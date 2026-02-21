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
	"sort"
	"strings"
)

func (s *Store) findDirectory(parentPath string, dirName string) (string, error) {
	slog.Debug("finding directory", "parent", parentPath, "name", dirName)

	entries, err := os.ReadDir(parentPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || !isStoreDir(entry.Name()) {
			continue
		}

		uuid := entry.Name()
		dirPath := filepath.Join(parentPath, uuid)

		if !s.hasAccess(dirPath) {
			slog.Debug("no access to directory, skipping", "uuid", uuid)
			continue
		}

		metadataPath := filepath.Join(dirPath, uuid+"_md.gpg")

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

		if metadata.Type == TypeDir && (metadata.Path == dirName || pathSegment(metadata.Path) == dirName) {
			slog.Debug("found directory", "uuid", uuid, "name", dirName)
			return uuid, nil
		}
	}

	return "", fmt.Errorf("directory not found: %s", dirName)
}

func pathSegment(p string) string {
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		return p
	}
	return p[idx+1:]
}

func (s *Store) resolveAccessiblePath(segments []string) (string, error) {
	return s.resolveAccessiblePathRecursive(s.SecretsPath, segments, "")
}

func isStoreDir(name string) bool {
	return len(name) > 0 && name[0] != '.'
}

func (s *Store) resolveAccessiblePathRecursive(currentDir string, remainingSegments []string, pathSoFar string) (string, error) {
	if len(remainingSegments) == 0 {
		return currentDir, nil
	}

	targetSegment := remainingSegments[0]
	rest := remainingSegments[1:]

	var expectedFullPath string
	if pathSoFar == "" {
		expectedFullPath = targetSegment
	} else {
		expectedFullPath = pathSoFar + "/" + targetSegment
	}

	entries, err := os.ReadDir(currentDir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	var opaqueCandidates []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isStoreDir(name) {
			continue
		}

		dirPath := filepath.Join(currentDir, name)

		if s.hasAccess(dirPath) {
			metadataPath := filepath.Join(dirPath, name+"_md.gpg")

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

			if metadata.Type != TypeDir {
				continue
			}

			if metadata.Path == expectedFullPath || metadata.Path == targetSegment {
				slog.Debug("found accessible directory", "uuid", name, "path", metadata.Path)
				return s.resolveAccessiblePathRecursive(dirPath, rest, expectedFullPath)
			}
		} else if len(rest) > 0 {
			opaqueCandidates = append(opaqueCandidates, dirPath)
		}
	}

	for _, candidatePath := range opaqueCandidates {
		result, err := s.resolveAccessiblePathRecursive(candidatePath, rest, expectedFullPath)
		if err == nil {
			slog.Debug("found path through opaque directory", "candidate", candidatePath, "target", expectedFullPath)
			return result, nil
		}
	}

	return "", fmt.Errorf("directory not found: %s", expectedFullPath)
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

		if metadata.Path == secretName && (metadata.Type == TypePassword || metadata.Type == TypeFile) {
			slog.Debug("found secret", "uuid", baseFileName, "name", secretName)
			return baseFileName, nil
		}
	}

	return "", fmt.Errorf("secret not found: %s", secretName)
}

func (s *Store) List(path string) ([]Entry, error) {
	slog.Debug("listing entries", "path", path)

	targetPath := s.SecretsPath

	if path != "" {
		normalizedPath, err := NormalizePath(path)
		if err != nil {
			return nil, fmt.Errorf("invalid path: %w", err)
		}

		segments := SplitPath(normalizedPath)
		resolved, err := s.resolveAccessiblePath(segments)
		if err != nil {
			return []Entry{}, nil
		}
		targetPath = resolved
	}

	entries, err := os.ReadDir(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Entry{}, nil
		}
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var result []Entry

	for _, entry := range entries {
		if entry.IsDir() {
			uuid := entry.Name()
			if !isStoreDir(uuid) {
				continue
			}
			dirPath := filepath.Join(targetPath, uuid)

			if !s.hasAccess(dirPath) {
				slog.Debug("no access to subdirectory, skipping", "uuid", uuid)
				continue
			}

			metadataPath := filepath.Join(dirPath, uuid+"_md.gpg")

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

			if metadata.Type == TypeDir {
				displayName := pathSegment(metadata.Path)
				result = append(result, Entry{Name: displayName, Type: TypeDir})
			}
		} else {
			fileName := entry.Name()
			if filepath.Ext(fileName) != ".gpg" {
				continue
			}

			if strings.HasSuffix(fileName, "_md.gpg") {
				continue
			}

			baseFileName := fileName[:len(fileName)-len(".gpg")]
			if len(baseFileName) == 0 {
				continue
			}

			metadataFileName := baseFileName + "_md.gpg"
			metadataPath := filepath.Join(targetPath, metadataFileName)

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

			if metadata.Type == TypePassword || metadata.Type == TypeFile {
				result = append(result, Entry{Name: metadata.Path, Type: metadata.Type})
			}
		}
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Type == TypeDir && result[j].Type != TypeDir {
			return true
		}
		if result[i].Type != TypeDir && result[j].Type == TypeDir {
			return false
		}
		return result[i].Name < result[j].Name
	})

	return result, nil
}
