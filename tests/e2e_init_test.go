//go:build e2e

package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gogitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"github.com/gonzaloalvarez/kepr/cmd"
	"github.com/gonzaloalvarez/kepr/tests/mocks"
)

func TestInit_HappyPath(t *testing.T) {
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

	mockShell := mocks.NewMockShell()
	mockUI := mocks.NewMockUI()
	mockGitHub := mocks.NewMockGitHub("Test User", "test@example.com")

	mockShell.AddResponse("gpg", []string{"--version"}, "gpg (GnuPG) 2.4.0", "", nil)
	mockShell.AddResponse("git", []string{"--version"}, "git version 2.39.0", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--batch", "--gen-key"}, "", "", nil)
	mockShell.AddResponse("/usr/bin/gpg", []string{"--list-keys", "--with-colons"},
		"fpr:::::::::ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234:\nuid:-::::::::Test User <test@example.com>:\n", "", nil)
	mockShell.AddResponse("/usr/bin/gpg", []string{"--batch", "--pinentry-mode", "loopback", "--passphrase", "",
		"--quick-add-key", "ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234", "cv25519", "encr", "0"}, "", "", nil)
	mockShell.AddResponse("/usr/bin/gpg", []string{"--armor", "--export-secret-key", "ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234"},
		"-----BEGIN PGP PRIVATE KEY BLOCK-----\nfake-key-content\n-----END PGP PRIVATE KEY BLOCK-----\n", "", nil)
	mockShell.AddResponse("/usr/bin/gpg", []string{"--batch", "--yes", "--delete-secret-keys", "ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234"},
		"", "", nil)

	mockUI.ConfirmInputs = []bool{true, true, true, true, true}

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"init", "testrepo"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !mockGitHub.AuthCalled {
		t.Error("expected GitHub authentication to be called")
	}

	if mockGitHub.Token != "mock-github-token-12345" {
		t.Errorf("expected token to be set, got: %s", mockGitHub.Token)
	}

	configPath := filepath.Join(keprHome, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("expected config file to exist at %s", configPath)
	}

	if !mockShell.WasCalled("/usr/bin/gpg", "--batch", "--gen-key") {
		t.Error("expected gpg --batch --gen-key to be called")
	}

	stdin := mockShell.GetStdinForCall("/usr/bin/gpg", "--batch", "--gen-key")
	if !strings.Contains(stdin, "Name-Real: Test User") {
		t.Errorf("expected stdin to contain user name, got: %s", stdin)
	}
	if !strings.Contains(stdin, "Name-Email: test@example.com") {
		t.Errorf("expected stdin to contain user email, got: %s", stdin)
	}
	if !strings.Contains(stdin, "Key-Type: EDDSA") {
		t.Errorf("expected stdin to contain key type, got: %s", stdin)
	}

	output := mockUI.GetOutput()
	if !strings.Contains(output, "WARNING") || !strings.Contains(output, "Master Key") {
		t.Errorf("expected warning message about master key, got: %s", output)
	}

	if !mockUI.HasOutput("GPG environment initialized") {
		t.Error("expected GPG initialization message in output")
	}

	if !mockUI.HasOutput("Test User") {
		t.Error("expected user identity in output")
	}

	if !mockUI.HasOutput("testuser/testrepo") {
		t.Error("expected repo name in output")
	}
}

func TestInit_ExistingRepo(t *testing.T) {
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

	bareRepoPath := filepath.Join(tempDir, "bare-repo.git")
	bareRepo, err := gogit.PlainInit(bareRepoPath, true)
	if err != nil {
		t.Fatalf("failed to create bare repo: %v", err)
	}
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := bareRepo.Storer.SetReference(headRef); err != nil {
		t.Fatalf("failed to set bare repo HEAD: %v", err)
	}

	srcPath := filepath.Join(tempDir, "src-repo")
	srcRepo, err := gogit.PlainInit(srcPath, false)
	if err != nil {
		t.Fatalf("failed to create source repo: %v", err)
	}
	srcHeadRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := srcRepo.Storer.SetReference(srcHeadRef); err != nil {
		t.Fatalf("failed to set source repo HEAD: %v", err)
	}

	if err := os.WriteFile(filepath.Join(srcPath, ".gpg.id"), []byte("ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234\n"), 0600); err != nil {
		t.Fatalf("failed to create .gpg.id: %v", err)
	}
	secretDir := filepath.Join(srcPath, "a1b2c3d4-uuid-dir")
	if err := os.MkdirAll(secretDir, 0700); err != nil {
		t.Fatalf("failed to create secret dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretDir, ".gpg.id"), []byte("ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234\n"), 0600); err != nil {
		t.Fatalf("failed to create sub .gpg.id: %v", err)
	}

	w, err := srcRepo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	if _, err := w.Add("."); err != nil {
		t.Fatalf("failed to stage: %v", err)
	}
	sig := &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()}
	if _, err := w.Commit("initial", &gogit.CommitOptions{Author: sig, Committer: sig}); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	if _, err := srcRepo.CreateRemote(&gogitconfig.RemoteConfig{Name: "origin", URLs: []string{"file://" + bareRepoPath}}); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}
	if err := srcRepo.Push(&gogit.PushOptions{RemoteName: "origin"}); err != nil {
		t.Fatalf("failed to push: %v", err)
	}

	mockShell := mocks.NewMockShell()
	mockUI := mocks.NewMockUI()
	mockGitHub := mocks.NewMockGitHub("Test User", "test@example.com")

	mockGitHub.Repos["testrepo"] = true
	mockGitHub.CloneURLs["testrepo"] = "file://" + bareRepoPath

	mockShell.AddResponse("gpg", []string{"--version"}, "gpg (GnuPG) 2.4.0", "", nil)
	mockShell.AddResponse("git", []string{"--version"}, "git version 2.39.0", "", nil)

	mockUI.ConfirmInputs = []bool{true, true, true}

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"init", "testrepo"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !mockGitHub.AuthCalled {
		t.Error("expected GitHub authentication to be called")
	}

	configPath := filepath.Join(keprHome, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("expected config file at %s", configPath)
	}

	secretsPath := filepath.Join(keprHome, "testuser", "testrepo")
	if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
		t.Fatalf("expected cloned secrets directory at %s", secretsPath)
	}

	gpgIDPath := filepath.Join(secretsPath, ".gpg.id")
	data, err := os.ReadFile(gpgIDPath)
	if err != nil {
		t.Fatalf("expected .gpg.id in cloned repo: %v", err)
	}
	if !strings.Contains(string(data), "ABCD1234") {
		t.Errorf("expected .gpg.id to contain fingerprint, got: %s", string(data))
	}

	clonedSecretDir := filepath.Join(secretsPath, "a1b2c3d4-uuid-dir")
	if _, err := os.Stat(clonedSecretDir); os.IsNotExist(err) {
		t.Errorf("expected cloned secret directory at %s", clonedSecretDir)
	}

	if mockShell.WasCalled("/usr/bin/gpg", "--batch", "--gen-key") {
		t.Error("GPG key generation should NOT be called when cloning existing repo")
	}

	output := mockUI.GetOutput()
	if !strings.Contains(output, "already exists") {
		t.Errorf("expected 'already exists' message, got: %s", output)
	}
	if !strings.Contains(output, "Joined existing repository") {
		t.Errorf("expected 'Joined existing repository' message, got: %s", output)
	}
	if !strings.Contains(output, "Cloned secret store") {
		t.Errorf("expected 'Cloned secret store' message, got: %s", output)
	}
}

func TestInit_Headless(t *testing.T) {
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

	mockShell := mocks.NewMockShell()
	mockUI := mocks.NewMockUI()
	mockGitHub := mocks.NewMockGitHub("Test User", "test@example.com")

	mockShell.AddResponse("gpg", []string{"--version"}, "gpg (GnuPG) 2.4.0", "", nil)
	mockShell.AddResponse("git", []string{"--version"}, "git version 2.39.0", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--batch", "--gen-key"}, "", "", nil)
	mockShell.AddResponse("/usr/bin/gpg", []string{"--list-keys", "--with-colons"},
		"fpr:::::::::ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234:\nuid:-::::::::Test User <test@example.com>:\n", "", nil)
	mockShell.AddResponse("/usr/bin/gpg", []string{"--batch", "--pinentry-mode", "loopback", "--passphrase", "",
		"--quick-add-key", "ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234", "cv25519", "encr", "0"}, "", "", nil)

	mockUI.ConfirmInputs = []bool{true, true, true, true, true}

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"init", "--headless", "testrepo"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !mockGitHub.AuthCalled {
		t.Error("expected GitHub authentication to be called")
	}

	output := mockUI.GetOutput()
	if !strings.Contains(output, "Headless mode") {
		t.Errorf("expected headless mode message, got: %s", output)
	}

	if strings.Contains(output, "WARNING") && strings.Contains(output, "DELETED") {
		t.Error("headless mode should not show master key deletion warning")
	}

	if !mockShell.WasCalled("/usr/bin/gpg", "--batch", "--gen-key") {
		t.Error("expected gpg key generation in headless mode")
	}

	configPath := filepath.Join(keprHome, "config.json")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	if !strings.Contains(string(configData), `"headless":true`) && !strings.Contains(string(configData), `"headless": true`) {
		t.Errorf("expected config to contain headless:true, got: %s", string(configData))
	}
}
