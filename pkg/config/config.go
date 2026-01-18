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
	"sort"
)

const CurrentConfigVersion = 2

type Identity struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	YubikeySerial  string `json:"yubikey_serial,omitempty"`
	YubikeyAdminPin string `json:"yubikey_admin_pin,omitempty"`
	YubikeyUserPin  string `json:"yubikey_user_pin,omitempty"`
}

type RepoConfig struct {
	Fingerprint string `json:"fingerprint"`
}

type Config struct {
	Version     int                    `json:"version"`
	GitHubToken string                 `json:"github_token,omitempty"`
	DefaultRepo string                 `json:"default_repo,omitempty"`
	Identities  map[string]*Identity   `json:"identities,omitempty"`
	Repos       map[string]*RepoConfig `json:"repos,omitempty"`
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
			cfg = &Config{
				Version:    CurrentConfigVersion,
				Identities: make(map[string]*Identity),
				Repos:      make(map[string]*RepoConfig),
			}
			return nil
		}
		return err
	}

	cfg = &Config{
		Identities: make(map[string]*Identity),
		Repos:      make(map[string]*RepoConfig),
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.Identities == nil {
		cfg.Identities = make(map[string]*Identity)
	}
	if cfg.Repos == nil {
		cfg.Repos = make(map[string]*RepoConfig)
	}

	if cfg.Version < CurrentConfigVersion {
		if err := migrateConfig(); err != nil {
			return fmt.Errorf("failed to migrate config: %w", err)
		}
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

func GetDefaultRepo() string {
	if cfg == nil {
		return ""
	}
	return cfg.DefaultRepo
}

func SetDefaultRepo(repoPath string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	if _, exists := cfg.Repos[repoPath]; !exists {
		return fmt.Errorf("repository '%s' not found in config", repoPath)
	}
	cfg.DefaultRepo = repoPath
	return saveConfig()
}

func GetRepoConfig(repoPath string) (*RepoConfig, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config not initialized")
	}
	repo, exists := cfg.Repos[repoPath]
	if !exists {
		return nil, fmt.Errorf("repository '%s' not found", repoPath)
	}
	return repo, nil
}

func AddRepo(repoPath, fingerprint string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.Repos[repoPath] = &RepoConfig{
		Fingerprint: fingerprint,
	}
	if cfg.DefaultRepo == "" {
		cfg.DefaultRepo = repoPath
	}
	return saveConfig()
}

func ListRepos() []string {
	if cfg == nil {
		return nil
	}
	repos := make([]string, 0, len(cfg.Repos))
	for repo := range cfg.Repos {
		repos = append(repos, repo)
	}
	sort.Strings(repos)
	return repos
}

func GetIdentity(fingerprint string) (*Identity, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config not initialized")
	}
	identity, exists := cfg.Identities[fingerprint]
	if !exists {
		return nil, fmt.Errorf("identity with fingerprint '%s' not found", fingerprint)
	}
	return identity, nil
}

func GetIdentityForRepo(repoPath string) (*Identity, error) {
	repo, err := GetRepoConfig(repoPath)
	if err != nil {
		return nil, err
	}
	return GetIdentity(repo.Fingerprint)
}

func AddIdentity(fingerprint string, identity *Identity) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.Identities[fingerprint] = identity
	return saveConfig()
}

func ListIdentities() map[string]*Identity {
	if cfg == nil {
		return nil
	}
	return cfg.Identities
}

func SecretsPathForRepo(repoPath string) (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, repoPath), nil
}

func SaveToken(token string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	cfg.GitHubToken = token
	return saveConfig()
}

func GetToken() string {
	if cfg == nil {
		return ""
	}
	return cfg.GitHubToken
}

func SaveUserIdentity(name, email string) error {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return fmt.Errorf("no default repository set")
	}
	repo, err := GetRepoConfig(repoPath)
	if err != nil {
		return err
	}
	identity, err := GetIdentity(repo.Fingerprint)
	if err != nil {
		return err
	}
	identity.Name = name
	identity.Email = email
	return saveConfig()
}

func SaveFingerprint(fingerprint string) error {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return fmt.Errorf("no default repository set")
	}
	repo, err := GetRepoConfig(repoPath)
	if err != nil {
		return err
	}
	repo.Fingerprint = fingerprint
	return saveConfig()
}

func GetUserName() string {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return ""
	}
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return ""
	}
	return identity.Name
}

func GetUserEmail() string {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return ""
	}
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return ""
	}
	return identity.Email
}

func GetUserFingerprint() string {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return ""
	}
	repo, err := GetRepoConfig(repoPath)
	if err != nil {
		return ""
	}
	return repo.Fingerprint
}

func SaveYubikeyAdminPin(pin string) error {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return fmt.Errorf("no default repository set")
	}
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return err
	}
	identity.YubikeyAdminPin = pin
	return saveConfig()
}

func GetYubikeyAdminPin() string {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return ""
	}
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return ""
	}
	return identity.YubikeyAdminPin
}

func SaveYubikeyUserPin(pin string) error {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return fmt.Errorf("no default repository set")
	}
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return err
	}
	identity.YubikeyUserPin = pin
	return saveConfig()
}

func GetYubikeyUserPin() string {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return ""
	}
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return ""
	}
	return identity.YubikeyUserPin
}

func SaveYubikeySerial(serial string) error {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return fmt.Errorf("no default repository set")
	}
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return err
	}
	identity.YubikeySerial = serial
	return saveConfig()
}

func GetYubikeySerial() string {
	repoPath := GetDefaultRepo()
	if repoPath == "" {
		return ""
	}
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return ""
	}
	return identity.YubikeySerial
}

func SaveGitHubRepo(repo string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	if _, exists := cfg.Repos[repo]; !exists {
		cfg.Repos[repo] = &RepoConfig{}
	}
	if cfg.DefaultRepo == "" {
		cfg.DefaultRepo = repo
	}
	return saveConfig()
}

func GetGitHubRepo() string {
	return GetDefaultRepo()
}

func GetUserNameForRepo(repoPath string) string {
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return ""
	}
	return identity.Name
}

func GetUserEmailForRepo(repoPath string) string {
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return ""
	}
	return identity.Email
}

func GetUserFingerprintForRepo(repoPath string) string {
	repo, err := GetRepoConfig(repoPath)
	if err != nil {
		return ""
	}
	return repo.Fingerprint
}

func GetYubikeyUserPinForRepo(repoPath string) string {
	identity, err := GetIdentityForRepo(repoPath)
	if err != nil {
		return ""
	}
	return identity.YubikeyUserPin
}

func SetRepoFingerprint(repoPath, fingerprint string) error {
	if cfg == nil {
		return fmt.Errorf("config not initialized")
	}
	repo, exists := cfg.Repos[repoPath]
	if !exists {
		return fmt.Errorf("repository '%s' not found", repoPath)
	}
	repo.Fingerprint = fingerprint
	return saveConfig()
}
