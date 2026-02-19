//go:build e2e

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
package tests

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gonzaloalvarez/kepr/cmd"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/gonzaloalvarez/kepr/tests/fakeghserver"
)

func TestE2E_InitAddGet_WithFakeServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E test in short mode")
	}

	tempDir := t.TempDir()
	keprHome := filepath.Join(tempDir, "kepr")

	serverCfg := fakeghserver.DefaultConfig()
	serverCfg.ReposDir = filepath.Join(tempDir, "repos")
	server := fakeghserver.New(serverCfg)

	serverURL, err := server.StartBackground()
	if err != nil {
		t.Fatalf("failed to start fake server: %v", err)
	}
	defer server.Close()

	t.Setenv("KEPR_CI", "true")
	t.Setenv("GITHUB_HOST", serverURL)
	t.Setenv("KEPR_HOME", keprHome)

	app := &cmd.App{
		Shell:  &shell.SystemExecutor{},
		UI:     cout.NewTerminal(),
		GitHub: github.NewGitHubClient(),
	}

	t.Run("init", func(t *testing.T) {
		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"init", "test-repo"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("init failed: %v", err)
		}

		configPath := filepath.Join(keprHome, "config.json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("expected config file to exist at %s", configPath)
		}

		secretsPath := filepath.Join(keprHome, "testuser", "test-repo")
		if _, err := os.Stat(secretsPath); os.IsNotExist(err) {
			t.Errorf("expected secrets directory to exist at %s", secretsPath)
		}
	})

	t.Run("add", func(t *testing.T) {
		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()

		r, w, _ := os.Pipe()
		os.Stdin = r
		go func() {
			w.WriteString("my-test-secret\n")
			w.Close()
		}()

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"add", "aws/main/keys"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("add failed: %v", err)
		}
	})

	t.Run("get", func(t *testing.T) {
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()

		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"get", "aws/main/keys"})

		err := rootCmd.Execute()

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		if err != nil {
			t.Fatalf("get failed: %v", err)
		}

		expectedSecret := "my-test-secret"
		if !strings.Contains(output, expectedSecret) {
			t.Errorf("expected output to contain %q, got %q", expectedSecret, output)
		}
	})

	t.Run("add_file", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "test_secret.pem")
		testContent := "-----BEGIN PRIVATE KEY-----\nfake-key-content\n-----END PRIVATE KEY-----\n"
		if err := os.WriteFile(testFile, []byte(testContent), 0600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"add", "ssh/gonzalo/main.ssh", testFile})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("add file failed: %v", err)
		}
	})

	t.Run("get_file_with_output", func(t *testing.T) {
		outputFile := filepath.Join(tempDir, "retrieved_key.pem")

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"get", "ssh/gonzalo/main.ssh", "-o", outputFile})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("get file failed: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		expectedContent := "-----BEGIN PRIVATE KEY-----\nfake-key-content\n-----END PRIVATE KEY-----\n"
		if string(data) != expectedContent {
			t.Errorf("expected file content %q, got %q", expectedContent, string(data))
		}
	})

	t.Run("get_file_default_name", func(t *testing.T) {
		oldWd, _ := os.Getwd()
		os.Chdir(tempDir)
		defer os.Chdir(oldWd)

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"get", "ssh/gonzalo/main.ssh"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("get file default name failed: %v", err)
		}

		defaultFile := filepath.Join(tempDir, "test_secret.pem")
		data, err := os.ReadFile(defaultFile)
		if err != nil {
			t.Fatalf("failed to read default output file: %v", err)
		}

		expectedContent := "-----BEGIN PRIVATE KEY-----\nfake-key-content\n-----END PRIVATE KEY-----\n"
		if string(data) != expectedContent {
			t.Errorf("expected file content %q, got %q", expectedContent, string(data))
		}
	})

	t.Run("list_root", func(t *testing.T) {
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()

		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"list"})

		err := rootCmd.Execute()

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		if err != nil {
			t.Fatalf("list failed: %v", err)
		}

		if !strings.Contains(output, "aws/") {
			t.Errorf("expected output to contain 'aws/', got %q", output)
		}
	})

	t.Run("list_aws", func(t *testing.T) {
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()

		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"list", "aws"})

		err := rootCmd.Execute()

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		if err != nil {
			t.Fatalf("list aws failed: %v", err)
		}

		if !strings.Contains(output, "main/") {
			t.Errorf("expected output to contain 'main/', got %q", output)
		}
	})

	t.Run("list_aws_main", func(t *testing.T) {
		oldStdout := os.Stdout
		defer func() { os.Stdout = oldStdout }()

		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"list", "aws/main"})

		err := rootCmd.Execute()

		w.Close()
		var buf bytes.Buffer
		buf.ReadFrom(r)
		output := buf.String()

		if err != nil {
			t.Fatalf("list aws/main failed: %v", err)
		}

		if !strings.Contains(output, "keys") {
			t.Errorf("expected output to contain 'keys', got %q", output)
		}
	})
}

func TestE2E_FakeServer_OAuthFlow(t *testing.T) {
	server := fakeghserver.New(fakeghserver.DefaultConfig())

	serverURL, err := server.StartBackground()
	if err != nil {
		t.Fatalf("failed to start fake server: %v", err)
	}
	defer server.Close()

	t.Setenv("GITHUB_HOST", serverURL)

	client := github.NewGitHubClient()

	mockUI := &captureUI{}

	token, err := client.CodeBasedAuthentication("test-client-id", mockUI)
	if err != nil {
		t.Fatalf("OAuth failed: %v", err)
	}

	expectedToken := "fake-github-token-12345"
	if token != expectedToken {
		t.Errorf("expected token %q, got %q", expectedToken, token)
	}
}

func TestE2E_FakeServer_RepoOperations(t *testing.T) {
	tempDir := t.TempDir()

	serverCfg := fakeghserver.DefaultConfig()
	serverCfg.ReposDir = filepath.Join(tempDir, "repos")
	server := fakeghserver.New(serverCfg)

	serverURL, err := server.StartBackground()
	if err != nil {
		t.Fatalf("failed to start fake server: %v", err)
	}
	defer server.Close()

	t.Setenv("GITHUB_HOST", serverURL)

	client := github.NewGitHubClient()
	client.SetToken("test-token")

	exists, err := client.CheckRepoExists("my-new-repo")
	if err != nil {
		t.Fatalf("CheckRepoExists failed: %v", err)
	}
	if exists {
		t.Error("expected repo to not exist initially")
	}

	err = client.CreateRepo("my-new-repo")
	if err != nil {
		t.Fatalf("CreateRepo failed: %v", err)
	}

	exists, err = client.CheckRepoExists("my-new-repo")
	if err != nil {
		t.Fatalf("CheckRepoExists failed: %v", err)
	}
	if !exists {
		t.Error("expected repo to exist after creation")
	}

	cloneURL, err := client.GetCloneURL("my-new-repo")
	if err != nil {
		t.Fatalf("GetCloneURL failed: %v", err)
	}
	if !strings.HasPrefix(cloneURL, "file://") {
		t.Errorf("expected clone URL to be file://, got %s", cloneURL)
	}

	expectedPath := filepath.Join(tempDir, "repos", "testuser", "my-new-repo.git")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("expected bare repo at %s", expectedPath)
	}
}

type captureUI struct {
	output bytes.Buffer
}

func (u *captureUI) Confirm(prompt string) (bool, error)                  { return true, nil }
func (u *captureUI) Input(prompt string, defaultValue string) (string, error) { return defaultValue, nil }
func (u *captureUI) InputPassword(prompt string) (string, error)          { return "password", nil }
func (u *captureUI) Info(a ...interface{})                                { u.output.WriteString(strings.Join(toStrings(a), " ")) }
func (u *captureUI) Infoln(a ...interface{})                              { u.Info(a...); u.output.WriteString("\n") }
func (u *captureUI) Infof(format string, a ...interface{})                {}
func (u *captureUI) Infofln(format string, a ...interface{})              {}
func (u *captureUI) Success(a ...interface{})                             {}
func (u *captureUI) Successln(a ...interface{})                           {}
func (u *captureUI) Successf(format string, a ...interface{})             {}
func (u *captureUI) Successfln(format string, a ...interface{})           {}
func (u *captureUI) Warning(message string)                               {}

func toStrings(a []interface{}) []string {
	s := make([]string, len(a))
	for i, v := range a {
		s[i] = v.(string)
	}
	return s
}
