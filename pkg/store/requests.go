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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gonzaloalvarez/kepr/pkg/gpg"
)

type PendingRequest struct {
	UUID        string
	Fingerprint string
	Path        string
	PublicKey   string
	Timestamp   string
}

func ListRequests(secretsPath string, g *gpg.GPG) ([]PendingRequest, error) {
	requestsDir := filepath.Join(secretsPath, "requests")

	entries, err := os.ReadDir(requestsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read requests directory: %w", err)
	}

	var requests []PendingRequest
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json.gpg") {
			continue
		}

		uuid := strings.TrimSuffix(entry.Name(), ".json.gpg")

		encryptedData, err := os.ReadFile(filepath.Join(requestsDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("failed to read request %s: %w", uuid, err)
		}

		decrypted, err := g.Decrypt(encryptedData)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt request %s: %w", uuid, err)
		}

		var req struct {
			Fingerprint string `json:"fingerprint"`
			Path        string `json:"path"`
			PublicKey   string `json:"public_key"`
			Timestamp   string `json:"timestamp"`
		}
		if err := json.Unmarshal(decrypted, &req); err != nil {
			return nil, fmt.Errorf("failed to parse request %s: %w", uuid, err)
		}

		requests = append(requests, PendingRequest{
			UUID:        uuid,
			Fingerprint: req.Fingerprint,
			Path:        req.Path,
			PublicKey:   req.PublicKey,
			Timestamp:   req.Timestamp,
		})
	}

	return requests, nil
}

func FindRequestByPrefix(secretsPath string, g *gpg.GPG, prefix string) (*PendingRequest, error) {
	all, err := ListRequests(secretsPath, g)
	if err != nil {
		return nil, err
	}

	var matches []PendingRequest
	for _, r := range all {
		if strings.HasPrefix(r.UUID, prefix) {
			matches = append(matches, r)
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no request found matching prefix %q", prefix)
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("ambiguous prefix %q matches %d requests", prefix, len(matches))
	}

	return &matches[0], nil
}
