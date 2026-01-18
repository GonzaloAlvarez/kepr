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
package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type v1Config struct {
	GitHubToken     string `json:"github_token"`
	GitHubRepo      string `json:"github_repo"`
	UserName        string `json:"user_name"`
	UserEmail       string `json:"user_email"`
	UserFingerprint string `json:"user_fingerprint"`
	YubikeyAdminPin string `json:"yubikey_admin_pin"`
	YubikeyUserPin  string `json:"yubikey_user_pin"`
	YubikeySerial   string `json:"yubikey_serial"`
}

func migrateConfig() error {
	slog.Info("migrating config from v1 to v2")

	dir, err := Dir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config for migration: %w", err)
	}

	var v1 v1Config
	if err := json.Unmarshal(data, &v1); err != nil {
		return fmt.Errorf("failed to parse v1 config: %w", err)
	}

	cfg.Version = CurrentConfigVersion
	cfg.GitHubToken = v1.GitHubToken

	if v1.UserFingerprint != "" {
		cfg.Identities[v1.UserFingerprint] = &Identity{
			Name:            v1.UserName,
			Email:           v1.UserEmail,
			YubikeySerial:   v1.YubikeySerial,
			YubikeyAdminPin: v1.YubikeyAdminPin,
			YubikeyUserPin:  v1.YubikeyUserPin,
		}
	}

	if v1.GitHubRepo != "" {
		cfg.Repos[v1.GitHubRepo] = &RepoConfig{
			Fingerprint: v1.UserFingerprint,
		}
		cfg.DefaultRepo = v1.GitHubRepo

		if err := migrateSecretsDirectory(v1.GitHubRepo); err != nil {
			slog.Warn("failed to migrate secrets directory", "error", err)
		}
	}

	if err := saveConfig(); err != nil {
		return fmt.Errorf("failed to save migrated config: %w", err)
	}

	slog.Info("config migration complete",
		"default_repo", cfg.DefaultRepo,
		"identities", len(cfg.Identities),
		"repos", len(cfg.Repos))

	return nil
}

func migrateSecretsDirectory(repoPath string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	oldPath := filepath.Join(dir, "secrets")
	if _, err := os.Stat(oldPath); os.IsNotExist(err) {
		return nil
	}

	newPath := filepath.Join(dir, repoPath)

	if err := os.MkdirAll(filepath.Dir(newPath), 0700); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to move secrets directory: %w", err)
	}

	slog.Info("migrated secrets directory", "from", oldPath, "to", newPath)
	return nil
}
