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
	"os/exec"
	"path/filepath"
)

type GitHubRepo struct {
	Name    string `json:"name"`
	Default bool   `json:"default,omitempty"`
}

type GitHub struct {
	Token string       `json:"token,omitempty"`
	Owner string       `json:"owner,omitempty"`
	Repos []GitHubRepo `json:"repos,omitempty"`
}

type Config struct {
	GitHub          GitHub `json:"github"`
	UserName        string `json:"user_name,omitempty"`
	UserEmail       string `json:"user_email,omitempty"`
	UserFingerprint string `json:"user_fingerprint,omitempty"`
	YubikeyAdminPin string `json:"yubikey_admin_pin,omitempty"`
	YubikeyUserPin  string `json:"yubikey_user_pin,omitempty"`
	YubikeySerial   string `json:"yubikey_serial,omitempty"`
}

var cfg *Config

func Dir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "kepr"), nil
}

func Init() error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}
	if err := loadConfig(); err != nil {
		return err
	}
	return CheckDependencies()
}

func EnsureConfigDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

func loadConfig() error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = &Config{}
			return nil
		}
		return err
	}

	cfg = &Config{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	return nil
}

func saveConfig() error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}

	dir, err := Dir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return err
	}

	return nil
}

func CheckDependencies() error {
	dependencies := []string{"gpg"}
	for _, tool := range dependencies {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("missing dependency: %s is not installed or in PATH", tool)
		}
		slog.Debug("dependency check passed", "tool", tool)
	}

	pinentryVariants := []string{"pinentry-mac", "pinentry-gnome3", "pinentry", "pinentry-curses"}
	found := false
	for _, variant := range pinentryVariants {
		if _, err := exec.LookPath(variant); err == nil {
			slog.Debug("dependency check passed", "tool", variant)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("missing dependency: no pinentry program found (tried: %v)", pinentryVariants)
	}

	return nil
}

func GetToken() string {
	if cfg == nil {
		return ""
	}
	return cfg.GitHub.Token
}

func SaveToken(token string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.GitHub.Token = token
	return saveConfig()
}

func GetGitHubOwner() string {
	if cfg == nil {
		return ""
	}
	return cfg.GitHub.Owner
}

func SaveGitHubOwner(owner string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.GitHub.Owner = owner
	return saveConfig()
}

func AddRepo(name string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}

	for _, r := range cfg.GitHub.Repos {
		if r.Name == name {
			return nil
		}
	}

	isDefault := len(cfg.GitHub.Repos) == 0
	cfg.GitHub.Repos = append(cfg.GitHub.Repos, GitHubRepo{
		Name:    name,
		Default: isDefault,
	})
	return saveConfig()
}

func GetDefaultRepo() string {
	if cfg == nil || cfg.GitHub.Owner == "" {
		return ""
	}

	for _, r := range cfg.GitHub.Repos {
		if r.Default {
			return cfg.GitHub.Owner + "/" + r.Name
		}
	}

	if len(cfg.GitHub.Repos) > 0 {
		return cfg.GitHub.Owner + "/" + cfg.GitHub.Repos[0].Name
	}

	return ""
}

func GetRepoNames() []string {
	if cfg == nil {
		return nil
	}
	names := make([]string, len(cfg.GitHub.Repos))
	for i, r := range cfg.GitHub.Repos {
		names[i] = r.Name
	}
	return names
}

func SecretsPathForRepo(repoPath string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, repoPath), nil
}

func GetUserName() string {
	if cfg == nil {
		return ""
	}
	return cfg.UserName
}

func GetUserEmail() string {
	if cfg == nil {
		return ""
	}
	return cfg.UserEmail
}

func SaveUserIdentity(name, email string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.UserName = name
	cfg.UserEmail = email
	return saveConfig()
}

func GetUserFingerprint() string {
	if cfg == nil {
		return ""
	}
	return cfg.UserFingerprint
}

func SaveFingerprint(fingerprint string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.UserFingerprint = fingerprint
	return saveConfig()
}

func GetYubikeyAdminPin() string {
	if cfg == nil {
		return ""
	}
	return cfg.YubikeyAdminPin
}

func SaveYubikeyAdminPin(pin string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.YubikeyAdminPin = pin
	return saveConfig()
}

func GetYubikeyUserPin() string {
	if cfg == nil {
		return ""
	}
	return cfg.YubikeyUserPin
}

func SaveYubikeyUserPin(pin string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.YubikeyUserPin = pin
	return saveConfig()
}

func GetYubikeySerial() string {
	if cfg == nil {
		return ""
	}
	return cfg.YubikeySerial
}

func SaveYubikeySerial(serial string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.YubikeySerial = serial
	return saveConfig()
}

func GetGitHubRepo() string {
	return GetDefaultRepo()
}

func SaveGitHubRepo(repoPath string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}

	owner, name := splitRepoPath(repoPath)
	if owner == "" || name == "" {
		return fmt.Errorf("invalid repo path: %s", repoPath)
	}

	if cfg.GitHub.Owner == "" {
		cfg.GitHub.Owner = owner
	}

	return AddRepo(name)
}

func splitRepoPath(repoPath string) (owner, name string) {
	for i := 0; i < len(repoPath); i++ {
		if repoPath[i] == '/' {
			return repoPath[:i], repoPath[i+1:]
		}
	}
	return "", repoPath
}
