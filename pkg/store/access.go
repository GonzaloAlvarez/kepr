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
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func ScanFingerprint(secretsPath, fingerprint string) (bool, error) {
	slog.Debug("scanning for fingerprint", "path", secretsPath, "fingerprint", fingerprint)

	found := false
	err := filepath.Walk(secretsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if info.Name() != ".gpg.id" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			slog.Debug("failed to read .gpg.id", "path", path, "error", err)
			return nil
		}

		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			if strings.TrimSpace(line) == fingerprint {
				slog.Debug("fingerprint found", "path", path)
				found = true
				return filepath.SkipAll
			}
		}

		return nil
	})

	return found, err
}
