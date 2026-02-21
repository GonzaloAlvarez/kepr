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
	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/tests/mocks"
)

func TestRequest_HappyPath(t *testing.T) {
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

	if err := os.MkdirAll(keprHome, 0700); err != nil {
		t.Fatalf("failed to create kepr home: %v", err)
	}
	if err := config.Init(); err != nil {
		t.Fatalf("failed to init config: %v", err)
	}
	if err := config.EnsureConfigDir(); err != nil {
		t.Fatalf("failed to ensure config dir: %v", err)
	}

	bareRepoPath := filepath.Join(tempDir, "bare-repo.git")
	bareRepo, err := gogit.PlainInit(bareRepoPath, true)
	if err != nil {
		t.Fatalf("failed to create bare repo: %v", err)
	}
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := bareRepo.Storer.SetReference(headRef); err != nil {
		t.Fatalf("failed to set bare repo HEAD: %v", err)
	}

	secretsPath := filepath.Join(keprHome, "testuser", "testrepo")
	if err := os.MkdirAll(secretsPath, 0700); err != nil {
		t.Fatalf("failed to create secrets path: %v", err)
	}

	rootFingerprint := "ROOT1234ROOT1234ROOT1234ROOT1234ROOT1234"
	if err := os.WriteFile(filepath.Join(secretsPath, ".gpg.id"), []byte(rootFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create .gpg.id: %v", err)
	}

	keysDir := filepath.Join(secretsPath, "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		t.Fatalf("failed to create keys dir: %v", err)
	}
	rootPubKey := "-----BEGIN PGP PUBLIC KEY BLOCK-----\nfake-root-public-key\n-----END PGP PUBLIC KEY BLOCK-----\n"
	if err := os.WriteFile(filepath.Join(keysDir, rootFingerprint+".key"), []byte(rootPubKey), 0644); err != nil {
		t.Fatalf("failed to write root public key: %v", err)
	}

	if err := os.WriteFile(filepath.Join(secretsPath, ".gitignore"), []byte("*\n!.gitignore\n!.gpg.id\n!*.gpg\n!keys/\n!keys/*.key\n!requests/\n"), 0600); err != nil {
		t.Fatalf("failed to create .gitignore: %v", err)
	}

	srcRepo, err := gogit.PlainInit(secretsPath, false)
	if err != nil {
		t.Fatalf("failed to init local repo: %v", err)
	}
	srcHeadRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := srcRepo.Storer.SetReference(srcHeadRef); err != nil {
		t.Fatalf("failed to set HEAD: %v", err)
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

	gpgHome := filepath.Join(keprHome, "gpg")
	if err := os.MkdirAll(gpgHome, 0700); err != nil {
		t.Fatalf("failed to create gpg home: %v", err)
	}

	requesterFingerprint := "REQ01234REQ01234REQ01234REQ01234REQ01234"

	if err := config.SaveToken("mock-token"); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}
	if err := config.SaveUserIdentity("Requester User", "requester@example.com"); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}
	if err := config.SaveFingerprint(requesterFingerprint); err != nil {
		t.Fatalf("failed to save fingerprint: %v", err)
	}
	if err := config.SaveGitHubOwner("testuser"); err != nil {
		t.Fatalf("failed to save owner: %v", err)
	}
	if err := config.AddRepo("testrepo"); err != nil {
		t.Fatalf("failed to add repo: %v", err)
	}
	if err := config.SaveGitHubRepo("testuser/testrepo"); err != nil {
		t.Fatalf("failed to save github repo: %v", err)
	}

	mockShell := mocks.NewMockShell()
	mockUI := mocks.NewMockUI()
	mockGitHub := mocks.NewMockGitHub("Requester User", "requester@example.com")
	mockGitHub.Repos["testrepo"] = true

	mockShell.AddResponse("/usr/bin/gpg", []string{"--import"}, "", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--armor", "--export", requesterFingerprint},
		"-----BEGIN PGP PUBLIC KEY BLOCK-----\nfake-requester-public-key\n-----END PGP PUBLIC KEY BLOCK-----\n", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--encrypt", "--armor", "--batch", "--trust-model", "always", "-r", rootFingerprint},
		"-----BEGIN PGP MESSAGE-----\nfake-encrypted-request\n-----END PGP MESSAGE-----\n", "", nil)

	mockUI.ConfirmInputs = []bool{true, true, true}

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"request", "-r", "testuser/testrepo", "prod/db"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	requestsDir := filepath.Join(secretsPath, "requests")
	entries, err := os.ReadDir(requestsDir)
	if err != nil {
		t.Fatalf("failed to read requests dir: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 request file, got %d", len(entries))
	}

	requestFile := entries[0].Name()
	if !strings.HasSuffix(requestFile, ".json.gpg") {
		t.Errorf("expected request file ending in .json.gpg, got: %s", requestFile)
	}

	output := mockUI.GetOutput()
	if !strings.Contains(output, "Created access request") {
		t.Errorf("expected 'Created access request' message, got: %s", output)
	}
	if !strings.Contains(output, "access-request/") {
		t.Errorf("expected branch name in output, got: %s", output)
	}
	if !strings.Contains(output, "Pushed access request") {
		t.Errorf("expected 'Pushed access request' message, got: %s", output)
	}

	repo, err := gogit.PlainOpen(secretsPath)
	if err != nil {
		t.Fatalf("failed to open repo: %v", err)
	}
	head, err := repo.Head()
	if err != nil {
		t.Fatalf("failed to get HEAD: %v", err)
	}
	if !strings.HasPrefix(head.Name().Short(), "access-request/") {
		t.Errorf("expected HEAD on access-request/ branch, got: %s", head.Name().Short())
	}
}

func TestRequest_AlreadyHasAccess(t *testing.T) {
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

	if err := os.MkdirAll(keprHome, 0700); err != nil {
		t.Fatalf("failed to create kepr home: %v", err)
	}
	if err := config.Init(); err != nil {
		t.Fatalf("failed to init config: %v", err)
	}
	if err := config.EnsureConfigDir(); err != nil {
		t.Fatalf("failed to ensure config dir: %v", err)
	}

	bareRepoPath := filepath.Join(tempDir, "bare-repo.git")
	bareRepo, err := gogit.PlainInit(bareRepoPath, true)
	if err != nil {
		t.Fatalf("failed to create bare repo: %v", err)
	}
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := bareRepo.Storer.SetReference(headRef); err != nil {
		t.Fatalf("failed to set bare repo HEAD: %v", err)
	}

	secretsPath := filepath.Join(keprHome, "testuser", "testrepo")
	if err := os.MkdirAll(secretsPath, 0700); err != nil {
		t.Fatalf("failed to create secrets path: %v", err)
	}

	requesterFingerprint := "REQ01234REQ01234REQ01234REQ01234REQ01234"
	if err := os.WriteFile(filepath.Join(secretsPath, ".gpg.id"), []byte(requesterFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create .gpg.id: %v", err)
	}

	srcRepo, err := gogit.PlainInit(secretsPath, false)
	if err != nil {
		t.Fatalf("failed to init local repo: %v", err)
	}
	srcHeadRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := srcRepo.Storer.SetReference(srcHeadRef); err != nil {
		t.Fatalf("failed to set HEAD: %v", err)
	}

	wt, err := srcRepo.Worktree()
	if err != nil {
		t.Fatalf("failed to get worktree: %v", err)
	}
	if _, err := wt.Add("."); err != nil {
		t.Fatalf("failed to stage: %v", err)
	}
	sig := &object.Signature{Name: "Test", Email: "test@test.com", When: time.Now()}
	if _, err := wt.Commit("initial", &gogit.CommitOptions{Author: sig, Committer: sig}); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	if _, err := srcRepo.CreateRemote(&gogitconfig.RemoteConfig{Name: "origin", URLs: []string{"file://" + bareRepoPath}}); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}
	if err := srcRepo.Push(&gogit.PushOptions{RemoteName: "origin"}); err != nil {
		t.Fatalf("failed to push: %v", err)
	}

	gpgHome := filepath.Join(keprHome, "gpg")
	if err := os.MkdirAll(gpgHome, 0700); err != nil {
		t.Fatalf("failed to create gpg home: %v", err)
	}

	if err := config.SaveToken("mock-token"); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}
	if err := config.SaveUserIdentity("Requester User", "requester@example.com"); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}
	if err := config.SaveFingerprint(requesterFingerprint); err != nil {
		t.Fatalf("failed to save fingerprint: %v", err)
	}
	if err := config.SaveGitHubOwner("testuser"); err != nil {
		t.Fatalf("failed to save owner: %v", err)
	}
	if err := config.AddRepo("testrepo"); err != nil {
		t.Fatalf("failed to add repo: %v", err)
	}
	if err := config.SaveGitHubRepo("testuser/testrepo"); err != nil {
		t.Fatalf("failed to save github repo: %v", err)
	}

	mockShell := mocks.NewMockShell()
	mockUI := mocks.NewMockUI()
	mockGitHub := mocks.NewMockGitHub("Requester User", "requester@example.com")

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"request", "-r", "testuser/testrepo", "prod/db"})

	err = rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when requester already has access")
	}

	if !strings.Contains(err.Error(), "already have access") {
		t.Errorf("expected 'already have access' error, got: %v", err)
	}
}
