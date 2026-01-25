package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		"fpr:::::::::ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234:\n", "", nil)
	mockShell.AddResponse("/usr/bin/gpg", []string{"--batch", "--pinentry-mode", "loopback", "--passphrase", "",
		"--quick-add-key", "ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234", "cv25519", "encr", "0"}, "", "", nil)
	mockShell.AddResponse("/usr/bin/gpg", []string{"--armor", "--export-secret-key", "ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234"},
		"-----BEGIN PGP PRIVATE KEY BLOCK-----\nfake-key-content\n-----END PGP PRIVATE KEY BLOCK-----\n", "", nil)
	mockShell.AddResponse("/usr/bin/gpg", []string{"--batch", "--yes", "--delete-secret-keys", "ABCD1234ABCD1234ABCD1234ABCD1234ABCD1234"},
		"", "", nil)

	mockUI.ConfirmInputs = []bool{true, true}

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"init", "testuser/testrepo"})

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
