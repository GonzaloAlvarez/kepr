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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir_ReturnsKeprSubdirectory(t *testing.T) {
	oldKeprHome := os.Getenv("KEPR_HOME")
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()
	os.Unsetenv("KEPR_HOME")

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() failed: %v", err)
	}

	if !strings.HasSuffix(dir, "kepr") {
		t.Errorf("expected Dir() to end with 'kepr', got %q", dir)
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("expected Dir() to return absolute path, got %q", dir)
	}
}

func TestDir_UsesKeprHomeEnv(t *testing.T) {
	tempDir := t.TempDir()
	keprHome := filepath.Join(tempDir, "custom-kepr")

	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() failed: %v", err)
	}

	if dir != keprHome {
		t.Errorf("expected Dir() to return KEPR_HOME value %q, got %q", keprHome, dir)
	}
}

func TestEnsureConfigDir_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	keprHome := filepath.Join(tempDir, "kepr")

	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()

	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() failed: %v", err)
	}

	info, err := os.Stat(keprHome)
	if err != nil {
		t.Fatalf("config directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("expected kepr path to be a directory")
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0700)
	if mode != expectedMode {
		t.Errorf("expected directory permissions %o, got %o", expectedMode, mode)
	}
}

func TestEnsureConfigDir_Idempotent(t *testing.T) {
	tempDir := t.TempDir()
	keprHome := filepath.Join(tempDir, "kepr")

	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()

	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("first EnsureConfigDir() failed: %v", err)
	}

	err = EnsureConfigDir()
	if err != nil {
		t.Fatalf("second EnsureConfigDir() failed: %v", err)
	}
}

func TestCheckDependencies(t *testing.T) {
	err := CheckDependencies()
	if err != nil {
		t.Logf("CheckDependencies() returned error (may be expected if gpg or git not installed): %v", err)
	}
}

func TestSplitRepoPath(t *testing.T) {
	tests := []struct {
		input         string
		expectedOwner string
		expectedName  string
	}{
		{"owner/repo", "owner", "repo"},
		{"myuser/myrepo", "myuser", "myrepo"},
		{"org/project-name", "org", "project-name"},
		{"repo-only", "", "repo-only"},
		{"", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, name := splitRepoPath(tt.input)
			if owner != tt.expectedOwner {
				t.Errorf("splitRepoPath(%q) owner = %q, want %q", tt.input, owner, tt.expectedOwner)
			}
			if name != tt.expectedName {
				t.Errorf("splitRepoPath(%q) name = %q, want %q", tt.input, name, tt.expectedName)
			}
		})
	}
}

func TestGetToken_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	token := GetToken()
	if token != "" {
		t.Errorf("GetToken() with nil config = %q, want empty", token)
	}
}

func TestGetGitHubOwner_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	owner := GetGitHubOwner()
	if owner != "" {
		t.Errorf("GetGitHubOwner() with nil config = %q, want empty", owner)
	}
}

func TestGetUserName_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	name := GetUserName()
	if name != "" {
		t.Errorf("GetUserName() with nil config = %q, want empty", name)
	}
}

func TestGetUserEmail_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	email := GetUserEmail()
	if email != "" {
		t.Errorf("GetUserEmail() with nil config = %q, want empty", email)
	}
}

func TestGetUserFingerprint_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	fp := GetUserFingerprint()
	if fp != "" {
		t.Errorf("GetUserFingerprint() with nil config = %q, want empty", fp)
	}
}

func TestGetYubikeyAdminPin_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	pin := GetYubikeyAdminPin()
	if pin != "" {
		t.Errorf("GetYubikeyAdminPin() with nil config = %q, want empty", pin)
	}
}

func TestGetYubikeyUserPin_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	pin := GetYubikeyUserPin()
	if pin != "" {
		t.Errorf("GetYubikeyUserPin() with nil config = %q, want empty", pin)
	}
}

func TestGetYubikeySerial_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	serial := GetYubikeySerial()
	if serial != "" {
		t.Errorf("GetYubikeySerial() with nil config = %q, want empty", serial)
	}
}

func TestGetRepoNames_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	names := GetRepoNames()
	if names != nil {
		t.Errorf("GetRepoNames() with nil config = %v, want nil", names)
	}
}

func TestGetDefaultRepo_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	repo := GetDefaultRepo()
	if repo != "" {
		t.Errorf("GetDefaultRepo() with nil config = %q, want empty", repo)
	}
}

func TestGetDefaultRepo_EmptyOwner(t *testing.T) {
	oldCfg := cfg
	cfg = &Config{
		GitHub: GitHub{
			Owner: "",
			Repos: []GitHubRepo{{Name: "repo1", Default: true}},
		},
	}
	defer func() { cfg = oldCfg }()

	repo := GetDefaultRepo()
	if repo != "" {
		t.Errorf("GetDefaultRepo() with empty owner = %q, want empty", repo)
	}
}

func TestGetDefaultRepo_WithDefaultRepo(t *testing.T) {
	oldCfg := cfg
	cfg = &Config{
		GitHub: GitHub{
			Owner: "testowner",
			Repos: []GitHubRepo{
				{Name: "repo1", Default: false},
				{Name: "repo2", Default: true},
			},
		},
	}
	defer func() { cfg = oldCfg }()

	repo := GetDefaultRepo()
	if repo != "testowner/repo2" {
		t.Errorf("GetDefaultRepo() = %q, want \"testowner/repo2\"", repo)
	}
}

func TestGetDefaultRepo_NoDefaultFallsBackToFirst(t *testing.T) {
	oldCfg := cfg
	cfg = &Config{
		GitHub: GitHub{
			Owner: "testowner",
			Repos: []GitHubRepo{
				{Name: "repo1", Default: false},
				{Name: "repo2", Default: false},
			},
		},
	}
	defer func() { cfg = oldCfg }()

	repo := GetDefaultRepo()
	if repo != "testowner/repo1" {
		t.Errorf("GetDefaultRepo() = %q, want \"testowner/repo1\"", repo)
	}
}

func TestGetDefaultRepo_EmptyRepos(t *testing.T) {
	oldCfg := cfg
	cfg = &Config{
		GitHub: GitHub{
			Owner: "testowner",
			Repos: []GitHubRepo{},
		},
	}
	defer func() { cfg = oldCfg }()

	repo := GetDefaultRepo()
	if repo != "" {
		t.Errorf("GetDefaultRepo() with no repos = %q, want empty", repo)
	}
}

func TestGetRepoNames(t *testing.T) {
	oldCfg := cfg
	cfg = &Config{
		GitHub: GitHub{
			Repos: []GitHubRepo{
				{Name: "repo1"},
				{Name: "repo2"},
				{Name: "repo3"},
			},
		},
	}
	defer func() { cfg = oldCfg }()

	names := GetRepoNames()
	if len(names) != 3 {
		t.Fatalf("GetRepoNames() returned %d names, want 3", len(names))
	}
	if names[0] != "repo1" || names[1] != "repo2" || names[2] != "repo3" {
		t.Errorf("GetRepoNames() = %v, want [repo1 repo2 repo3]", names)
	}
}

func TestSecretsPathForRepo(t *testing.T) {
	tempDir := t.TempDir()
	keprHome := filepath.Join(tempDir, "kepr")

	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()

	path, err := SecretsPathForRepo("owner/repo")
	if err != nil {
		t.Fatalf("SecretsPathForRepo() returned error: %v", err)
	}

	expected := filepath.Join(keprHome, "owner/repo")
	if path != expected {
		t.Errorf("SecretsPathForRepo() = %q, want %q", path, expected)
	}
}

func TestSaveToken_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	err := SaveToken("token")
	if err == nil {
		t.Error("SaveToken() with nil config should return error")
	}
}

func TestSaveGitHubOwner_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	err := SaveGitHubOwner("owner")
	if err == nil {
		t.Error("SaveGitHubOwner() with nil config should return error")
	}
}

func TestSaveUserIdentity_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	err := SaveUserIdentity("name", "email")
	if err == nil {
		t.Error("SaveUserIdentity() with nil config should return error")
	}
}

func TestSaveFingerprint_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	err := SaveFingerprint("fp")
	if err == nil {
		t.Error("SaveFingerprint() with nil config should return error")
	}
}

func TestSaveYubikeyAdminPin_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	err := SaveYubikeyAdminPin("pin")
	if err == nil {
		t.Error("SaveYubikeyAdminPin() with nil config should return error")
	}
}

func TestSaveYubikeyUserPin_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	err := SaveYubikeyUserPin("pin")
	if err == nil {
		t.Error("SaveYubikeyUserPin() with nil config should return error")
	}
}

func TestSaveYubikeySerial_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	err := SaveYubikeySerial("serial")
	if err == nil {
		t.Error("SaveYubikeySerial() with nil config should return error")
	}
}

func TestAddRepo_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	err := AddRepo("repo")
	if err == nil {
		t.Error("AddRepo() with nil config should return error")
	}
}

func TestSaveGitHubRepo_NilConfig(t *testing.T) {
	oldCfg := cfg
	cfg = nil
	defer func() { cfg = oldCfg }()

	err := SaveGitHubRepo("owner/repo")
	if err == nil {
		t.Error("SaveGitHubRepo() with nil config should return error")
	}
}

func TestSaveGitHubRepo_InvalidPath(t *testing.T) {
	tempDir := t.TempDir()
	keprHome := filepath.Join(tempDir, "kepr")

	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()

	oldCfg := cfg
	cfg = &Config{}
	defer func() { cfg = oldCfg }()

	err := SaveGitHubRepo("invalid-no-slash")
	if err == nil {
		t.Error("SaveGitHubRepo() with invalid path should return error")
	}
}

func TestGetGitHubRepo(t *testing.T) {
	oldCfg := cfg
	cfg = &Config{
		GitHub: GitHub{
			Owner: "testowner",
			Repos: []GitHubRepo{{Name: "testrepo", Default: true}},
		},
	}
	defer func() { cfg = oldCfg }()

	repo := GetGitHubRepo()
	if repo != "testowner/testrepo" {
		t.Errorf("GetGitHubRepo() = %q, want \"testowner/testrepo\"", repo)
	}
}

func TestGettersWithValues(t *testing.T) {
	oldCfg := cfg
	cfg = &Config{
		GitHub: GitHub{
			Token: "test-token",
			Owner: "test-owner",
		},
		UserName:        "Test User",
		UserEmail:       "test@example.com",
		UserFingerprint: "ABCD1234",
		YubikeyAdminPin: "12345678",
		YubikeyUserPin:  "123456",
		YubikeySerial:   "12345678",
	}
	defer func() { cfg = oldCfg }()

	if GetToken() != "test-token" {
		t.Errorf("GetToken() = %q, want \"test-token\"", GetToken())
	}
	if GetGitHubOwner() != "test-owner" {
		t.Errorf("GetGitHubOwner() = %q, want \"test-owner\"", GetGitHubOwner())
	}
	if GetUserName() != "Test User" {
		t.Errorf("GetUserName() = %q, want \"Test User\"", GetUserName())
	}
	if GetUserEmail() != "test@example.com" {
		t.Errorf("GetUserEmail() = %q, want \"test@example.com\"", GetUserEmail())
	}
	if GetUserFingerprint() != "ABCD1234" {
		t.Errorf("GetUserFingerprint() = %q, want \"ABCD1234\"", GetUserFingerprint())
	}
	if GetYubikeyAdminPin() != "12345678" {
		t.Errorf("GetYubikeyAdminPin() = %q, want \"12345678\"", GetYubikeyAdminPin())
	}
	if GetYubikeyUserPin() != "123456" {
		t.Errorf("GetYubikeyUserPin() = %q, want \"123456\"", GetYubikeyUserPin())
	}
	if GetYubikeySerial() != "12345678" {
		t.Errorf("GetYubikeySerial() = %q, want \"12345678\"", GetYubikeySerial())
	}
}
