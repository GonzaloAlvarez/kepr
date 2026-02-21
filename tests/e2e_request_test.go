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

func TestList_HappyPath(t *testing.T) {
	tempDir := t.TempDir()

	keprHome := filepath.Join(tempDir, "kepr")
	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	t.Setenv("KEPR_CI", "true")
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
	if err := os.WriteFile(filepath.Join(secretsPath, ".gitignore"), []byte("*\n!.gitignore\n!.gpg.id\n!*.gpg\n!keys/\n!keys/*.key\n!requests/\n"), 0600); err != nil {
		t.Fatalf("failed to create .gitignore: %v", err)
	}

	requestUUID := "04cd4f01-ecc5-4662-b0ec-66d62e14ba82"
	requestJSON := `{"fingerprint":"REQ01234REQ01234REQ01234REQ01234REQ01234","path":"prod/db","public_key":"-----BEGIN PGP PUBLIC KEY BLOCK-----\nfake-key\n-----END PGP PUBLIC KEY BLOCK-----\n","timestamp":"2026-02-08T00:00:00Z"}`

	requestsDir := filepath.Join(secretsPath, "requests")
	if err := os.MkdirAll(requestsDir, 0700); err != nil {
		t.Fatalf("failed to create requests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(requestsDir, requestUUID+".json.gpg"), []byte("encrypted-request"), 0600); err != nil {
		t.Fatalf("failed to write request file: %v", err)
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

	if err := config.SaveToken("mock-token"); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}
	if err := config.SaveUserIdentity("Owner User", "owner@example.com"); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}
	if err := config.SaveFingerprint(rootFingerprint); err != nil {
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
	mockGitHub := mocks.NewMockGitHub("Owner User", "owner@example.com")

	mockShell.AddQueuedResponse("/usr/bin/gpg", []string{"--decrypt", "--batch", "--pinentry-mode", "loopback", "--passphrase", ""},
		requestJSON, "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--import"}, "", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--list-keys", "--with-colons"},
		"fpr:::::::::REQ01234REQ01234REQ01234REQ01234REQ01234:\nuid:-::::::::Requester User <requester@example.com>:\n", "", nil)

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"request", "-r", "testuser/testrepo"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := mockUI.GetOutput()
	if !strings.Contains(output, requestUUID) {
		t.Errorf("expected output to contain request UUID, got: %s", output)
	}
	if !strings.Contains(output, "Requester User") {
		t.Errorf("expected output to contain requester name, got: %s", output)
	}
	if !strings.Contains(output, "requester@example.com") {
		t.Errorf("expected output to contain requester email, got: %s", output)
	}
	if !strings.Contains(output, "prod/db") {
		t.Errorf("expected output to contain requested path, got: %s", output)
	}
}

func TestApprove_HappyPath(t *testing.T) {
	tempDir := t.TempDir()

	keprHome := filepath.Join(tempDir, "kepr")
	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	t.Setenv("KEPR_CI", "true")
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
	requesterFingerprint := "REQ01234REQ01234REQ01234REQ01234REQ01234"

	if err := os.WriteFile(filepath.Join(secretsPath, ".gpg.id"), []byte(rootFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create .gpg.id: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretsPath, ".gitignore"), []byte("*\n!.gitignore\n!.gpg.id\n!*.gpg\n!keys/\n!keys/*.key\n!requests/\n"), 0600); err != nil {
		t.Fatalf("failed to create .gitignore: %v", err)
	}

	keysDir := filepath.Join(secretsPath, "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		t.Fatalf("failed to create keys dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, rootFingerprint+".key"), []byte("root-pub-key"), 0644); err != nil {
		t.Fatalf("failed to write root key: %v", err)
	}

	dirUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dirPath := filepath.Join(secretsPath, dirUUID)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, ".gpg.id"), []byte(rootFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create dir .gpg.id: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, dirUUID+"_md.gpg"), []byte("encrypted-dir-metadata"), 0600); err != nil {
		t.Fatalf("failed to write dir metadata: %v", err)
	}

	subDirUUID := "11111111-2222-3333-4444-555555555555"
	subDirPath := filepath.Join(dirPath, subDirUUID)
	if err := os.MkdirAll(subDirPath, 0700); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirPath, ".gpg.id"), []byte(rootFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create sub dir .gpg.id: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirPath, subDirUUID+"_md.gpg"), []byte("encrypted-subdir-metadata"), 0600); err != nil {
		t.Fatalf("failed to write sub dir metadata: %v", err)
	}

	secretUUID := "66666666-7777-8888-9999-aaaaaaaaaaaa"
	if err := os.WriteFile(filepath.Join(subDirPath, secretUUID+".gpg"), []byte("encrypted-secret"), 0600); err != nil {
		t.Fatalf("failed to write secret: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirPath, secretUUID+"_md.gpg"), []byte("encrypted-secret-metadata"), 0600); err != nil {
		t.Fatalf("failed to write secret metadata: %v", err)
	}

	requestUUID := "04cd4f01-ecc5-4662-b0ec-66d62e14ba82"
	requestJSON := `{"fingerprint":"` + requesterFingerprint + `","path":"prod/db","public_key":"-----BEGIN PGP PUBLIC KEY BLOCK-----\nfake-key\n-----END PGP PUBLIC KEY BLOCK-----\n","timestamp":"2026-02-08T00:00:00Z"}`

	requestsDir := filepath.Join(secretsPath, "requests")
	if err := os.MkdirAll(requestsDir, 0700); err != nil {
		t.Fatalf("failed to create requests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(requestsDir, requestUUID+".json.gpg"), []byte("encrypted-request"), 0600); err != nil {
		t.Fatalf("failed to write request file: %v", err)
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
	if err := config.SaveUserIdentity("Owner User", "owner@example.com"); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}
	if err := config.SaveFingerprint(rootFingerprint); err != nil {
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
	mockGitHub := mocks.NewMockGitHub("Owner User", "owner@example.com")

	decryptArgs := []string{"--decrypt", "--batch", "--pinentry-mode", "loopback", "--passphrase", ""}

	dirMetadataJSON := `{"path":"prod","type":"dir"}`
	subDirMetadataJSON := `{"path":"db","type":"dir"}`
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, requestJSON, "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, dirMetadataJSON, "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, subDirMetadataJSON, "", nil)

	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, "encrypted-dir-metadata-decrypted", "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, "encrypted-subdir-metadata-decrypted", "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, "encrypted-secret-decrypted", "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, "encrypted-secret-metadata-decrypted", "", nil)

	encryptArgs := []string{"--encrypt", "--armor", "--batch", "--trust-model", "always", "-r", rootFingerprint, "-r", requesterFingerprint}
	mockShell.AddResponse("/usr/bin/gpg", encryptArgs, "re-encrypted-data", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--import"}, "", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--armor", "--export", requesterFingerprint},
		"-----BEGIN PGP PUBLIC KEY BLOCK-----\nrequester-exported-key\n-----END PGP PUBLIC KEY BLOCK-----\n", "", nil)

	mockUI.ConfirmInputs = []bool{true, true, true}

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"request", "-r", "testuser/testrepo", "--approve", "04cd4f01"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := mockUI.GetOutput()
	if !strings.Contains(output, "Found request "+requestUUID) {
		t.Errorf("expected 'Found request' in output, got: %s", output)
	}
	if !strings.Contains(output, "Rekeying") {
		t.Errorf("expected 'Rekeying' in output, got: %s", output)
	}
	if !strings.Contains(output, "Committed and pushed to main") {
		t.Errorf("expected 'Committed and pushed' in output, got: %s", output)
	}

	if _, err := os.Stat(filepath.Join(requestsDir, requestUUID+".json.gpg")); !os.IsNotExist(err) {
		t.Errorf("expected request file to be removed, but it still exists")
	}

	gpgIDContent, err := os.ReadFile(filepath.Join(subDirPath, ".gpg.id"))
	if err != nil {
		t.Fatalf("failed to read updated .gpg.id: %v", err)
	}
	if !strings.Contains(string(gpgIDContent), requesterFingerprint) {
		t.Errorf("expected .gpg.id to contain requester fingerprint, got: %s", string(gpgIDContent))
	}
	if !strings.Contains(string(gpgIDContent), rootFingerprint) {
		t.Errorf("expected .gpg.id to still contain root fingerprint, got: %s", string(gpgIDContent))
	}

	requesterKeyPath := filepath.Join(keysDir, requesterFingerprint+".key")
	if _, err := os.Stat(requesterKeyPath); os.IsNotExist(err) {
		t.Errorf("expected requester key file to exist at %s", requesterKeyPath)
	}
}

func TestList_FetchFromBranch(t *testing.T) {
	tempDir := t.TempDir()

	keprHome := filepath.Join(tempDir, "kepr")
	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	t.Setenv("KEPR_CI", "true")
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

	requestUUID := "abc12345-ecc5-4662-b0ec-66d62e14ba82"
	branchRef := plumbing.NewBranchReferenceName("access-request/" + requestUUID)
	if err := w.Checkout(&gogit.CheckoutOptions{Branch: branchRef, Create: true}); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	requestsDir := filepath.Join(secretsPath, "requests")
	if err := os.MkdirAll(requestsDir, 0700); err != nil {
		t.Fatalf("failed to create requests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(requestsDir, requestUUID+".json.gpg"), []byte("encrypted-request"), 0600); err != nil {
		t.Fatalf("failed to write request file: %v", err)
	}

	if _, err := w.Add("."); err != nil {
		t.Fatalf("failed to stage request: %v", err)
	}
	sig2 := &object.Signature{Name: "Requester", Email: "req@test.com", When: time.Now()}
	if _, err := w.Commit("add request", &gogit.CommitOptions{Author: sig2, Committer: sig2}); err != nil {
		t.Fatalf("failed to commit request: %v", err)
	}
	if err := srcRepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []gogitconfig.RefSpec{gogitconfig.RefSpec("refs/heads/access-request/" + requestUUID + ":refs/heads/access-request/" + requestUUID)},
	}); err != nil {
		t.Fatalf("failed to push branch: %v", err)
	}

	mainRef := plumbing.NewBranchReferenceName("main")
	if err := w.Checkout(&gogit.CheckoutOptions{Branch: mainRef}); err != nil {
		t.Fatalf("failed to checkout main: %v", err)
	}

	if _, err := os.Stat(filepath.Join(secretsPath, "requests")); !os.IsNotExist(err) {
		os.RemoveAll(filepath.Join(secretsPath, "requests"))
	}

	gpgHome := filepath.Join(keprHome, "gpg")
	if err := os.MkdirAll(gpgHome, 0700); err != nil {
		t.Fatalf("failed to create gpg home: %v", err)
	}

	if err := config.SaveToken("mock-token"); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}
	if err := config.SaveUserIdentity("Owner User", "owner@example.com"); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}
	if err := config.SaveFingerprint(rootFingerprint); err != nil {
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

	requestJSON := `{"fingerprint":"REQ01234REQ01234REQ01234REQ01234REQ01234","path":"prod/db","public_key":"-----BEGIN PGP PUBLIC KEY BLOCK-----\nfake-key\n-----END PGP PUBLIC KEY BLOCK-----\n","timestamp":"2026-02-08T00:00:00Z"}`

	mockShell := mocks.NewMockShell()
	mockUI := mocks.NewMockUI()
	mockGitHub := mocks.NewMockGitHub("Owner User", "owner@example.com")

	mockShell.AddQueuedResponse("/usr/bin/gpg", []string{"--decrypt", "--batch", "--pinentry-mode", "loopback", "--passphrase", ""},
		requestJSON, "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--import"}, "", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--list-keys", "--with-colons"},
		"fpr:::::::::REQ01234REQ01234REQ01234REQ01234REQ01234:\nuid:-::::::::Requester User <requester@example.com>:\n", "", nil)

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"request", "-r", "testuser/testrepo"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := mockUI.GetOutput()
	if !strings.Contains(output, requestUUID) {
		t.Errorf("expected output to contain request UUID %s, got: %s", requestUUID, output)
	}
	if !strings.Contains(output, "Requester User") {
		t.Errorf("expected output to contain requester name, got: %s", output)
	}
	if !strings.Contains(output, "requester@example.com") {
		t.Errorf("expected output to contain requester email, got: %s", output)
	}
	if !strings.Contains(output, "prod/db") {
		t.Errorf("expected output to contain requested path, got: %s", output)
	}

	materializedFile := filepath.Join(secretsPath, "requests", requestUUID+".json.gpg")
	if _, err := os.Stat(materializedFile); os.IsNotExist(err) {
		t.Errorf("expected request file to be materialized at %s", materializedFile)
	}
}

func TestApprove_FetchFromBranch(t *testing.T) {
	tempDir := t.TempDir()

	keprHome := filepath.Join(tempDir, "kepr")
	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	t.Setenv("KEPR_CI", "true")
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
	requesterFingerprint := "REQ01234REQ01234REQ01234REQ01234REQ01234"

	if err := os.WriteFile(filepath.Join(secretsPath, ".gpg.id"), []byte(rootFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create .gpg.id: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretsPath, ".gitignore"), []byte("*\n!.gitignore\n!.gpg.id\n!*.gpg\n!keys/\n!keys/*.key\n!requests/\n"), 0600); err != nil {
		t.Fatalf("failed to create .gitignore: %v", err)
	}

	keysDir := filepath.Join(secretsPath, "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		t.Fatalf("failed to create keys dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(keysDir, rootFingerprint+".key"), []byte("root-pub-key"), 0644); err != nil {
		t.Fatalf("failed to write root key: %v", err)
	}

	dirUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dirPath := filepath.Join(secretsPath, dirUUID)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, ".gpg.id"), []byte(rootFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create dir .gpg.id: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, dirUUID+"_md.gpg"), []byte("encrypted-dir-metadata"), 0600); err != nil {
		t.Fatalf("failed to write dir metadata: %v", err)
	}

	subDirUUID := "11111111-2222-3333-4444-555555555555"
	subDirPath := filepath.Join(dirPath, subDirUUID)
	if err := os.MkdirAll(subDirPath, 0700); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirPath, ".gpg.id"), []byte(rootFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create sub dir .gpg.id: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirPath, subDirUUID+"_md.gpg"), []byte("encrypted-subdir-metadata"), 0600); err != nil {
		t.Fatalf("failed to write sub dir metadata: %v", err)
	}

	secretUUID := "66666666-7777-8888-9999-aaaaaaaaaaaa"
	if err := os.WriteFile(filepath.Join(subDirPath, secretUUID+".gpg"), []byte("encrypted-secret"), 0600); err != nil {
		t.Fatalf("failed to write secret: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirPath, secretUUID+"_md.gpg"), []byte("encrypted-secret-metadata"), 0600); err != nil {
		t.Fatalf("failed to write secret metadata: %v", err)
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

	requestUUID := "04cd4f01-ecc5-4662-b0ec-66d62e14ba82"
	branchRef := plumbing.NewBranchReferenceName("access-request/" + requestUUID)
	if err := wt.Checkout(&gogit.CheckoutOptions{Branch: branchRef, Create: true}); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	requestsDir := filepath.Join(secretsPath, "requests")
	if err := os.MkdirAll(requestsDir, 0700); err != nil {
		t.Fatalf("failed to create requests dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(requestsDir, requestUUID+".json.gpg"), []byte("encrypted-request"), 0600); err != nil {
		t.Fatalf("failed to write request file: %v", err)
	}

	if _, err := wt.Add("."); err != nil {
		t.Fatalf("failed to stage request: %v", err)
	}
	sig2 := &object.Signature{Name: "Requester", Email: "req@test.com", When: time.Now()}
	if _, err := wt.Commit("add request", &gogit.CommitOptions{Author: sig2, Committer: sig2}); err != nil {
		t.Fatalf("failed to commit request: %v", err)
	}
	if err := srcRepo.Push(&gogit.PushOptions{
		RemoteName: "origin",
		RefSpecs:   []gogitconfig.RefSpec{gogitconfig.RefSpec("refs/heads/access-request/" + requestUUID + ":refs/heads/access-request/" + requestUUID)},
	}); err != nil {
		t.Fatalf("failed to push branch: %v", err)
	}

	mainBranchRef := plumbing.NewBranchReferenceName("main")
	if err := wt.Checkout(&gogit.CheckoutOptions{Branch: mainBranchRef}); err != nil {
		t.Fatalf("failed to checkout main: %v", err)
	}

	if _, err := os.Stat(filepath.Join(secretsPath, "requests")); !os.IsNotExist(err) {
		os.RemoveAll(filepath.Join(secretsPath, "requests"))
	}

	gpgHome := filepath.Join(keprHome, "gpg")
	if err := os.MkdirAll(gpgHome, 0700); err != nil {
		t.Fatalf("failed to create gpg home: %v", err)
	}

	if err := config.SaveToken("mock-token"); err != nil {
		t.Fatalf("failed to save token: %v", err)
	}
	if err := config.SaveUserIdentity("Owner User", "owner@example.com"); err != nil {
		t.Fatalf("failed to save identity: %v", err)
	}
	if err := config.SaveFingerprint(rootFingerprint); err != nil {
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

	requestJSON := `{"fingerprint":"` + requesterFingerprint + `","path":"prod/db","public_key":"-----BEGIN PGP PUBLIC KEY BLOCK-----\nfake-key\n-----END PGP PUBLIC KEY BLOCK-----\n","timestamp":"2026-02-08T00:00:00Z"}`

	mockShell := mocks.NewMockShell()
	mockUI := mocks.NewMockUI()
	mockGitHub := mocks.NewMockGitHub("Owner User", "owner@example.com")

	decryptArgs := []string{"--decrypt", "--batch", "--pinentry-mode", "loopback", "--passphrase", ""}

	dirMetadataJSON := `{"path":"prod","type":"dir"}`
	subDirMetadataJSON := `{"path":"db","type":"dir"}`
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, requestJSON, "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, dirMetadataJSON, "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, subDirMetadataJSON, "", nil)

	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, "encrypted-dir-metadata-decrypted", "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, "encrypted-subdir-metadata-decrypted", "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, "encrypted-secret-decrypted", "", nil)
	mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, "encrypted-secret-metadata-decrypted", "", nil)

	encryptArgs := []string{"--encrypt", "--armor", "--batch", "--trust-model", "always", "-r", rootFingerprint, "-r", requesterFingerprint}
	mockShell.AddResponse("/usr/bin/gpg", encryptArgs, "re-encrypted-data", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--import"}, "", "", nil)

	mockShell.AddResponse("/usr/bin/gpg", []string{"--armor", "--export", requesterFingerprint},
		"-----BEGIN PGP PUBLIC KEY BLOCK-----\nrequester-exported-key\n-----END PGP PUBLIC KEY BLOCK-----\n", "", nil)

	mockUI.ConfirmInputs = []bool{true, true, true}

	app := &cmd.App{
		Shell:  mockShell,
		UI:     mockUI,
		GitHub: mockGitHub,
	}

	rootCmd := cmd.NewRootCmd(app)
	rootCmd.SetArgs([]string{"request", "-r", "testuser/testrepo", "--approve", "04cd4f01"})

	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	output := mockUI.GetOutput()
	if !strings.Contains(output, "Found request "+requestUUID) {
		t.Errorf("expected 'Found request' in output, got: %s", output)
	}
	if !strings.Contains(output, "Rekeying") {
		t.Errorf("expected 'Rekeying' in output, got: %s", output)
	}
	if !strings.Contains(output, "Committed and pushed to main") {
		t.Errorf("expected 'Committed and pushed' in output, got: %s", output)
	}

	gpgIDContent, err := os.ReadFile(filepath.Join(subDirPath, ".gpg.id"))
	if err != nil {
		t.Fatalf("failed to read updated .gpg.id: %v", err)
	}
	if !strings.Contains(string(gpgIDContent), requesterFingerprint) {
		t.Errorf("expected .gpg.id to contain requester fingerprint, got: %s", string(gpgIDContent))
	}
	if !strings.Contains(string(gpgIDContent), rootFingerprint) {
		t.Errorf("expected .gpg.id to still contain root fingerprint, got: %s", string(gpgIDContent))
	}

	requesterKeyPath := filepath.Join(keysDir, requesterFingerprint+".key")
	if _, err := os.Stat(requesterKeyPath); os.IsNotExist(err) {
		t.Errorf("expected requester key file to exist at %s", requesterKeyPath)
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

func TestPartialAccess_ListAndGet(t *testing.T) {
	tempDir := t.TempDir()

	keprHome := filepath.Join(tempDir, "kepr")
	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	t.Setenv("KEPR_CI", "true")
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
	requesterFingerprint := "REQ01234REQ01234REQ01234REQ01234REQ01234"

	if err := os.WriteFile(filepath.Join(secretsPath, ".gpg.id"), []byte(rootFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create .gpg.id: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secretsPath, ".gitignore"), []byte("*\n!.gitignore\n!.gpg.id\n!*.gpg\n!keys/\n!keys/*.key\n!requests/\n"), 0600); err != nil {
		t.Fatalf("failed to create .gitignore: %v", err)
	}

	dirUUID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	dirPath := filepath.Join(secretsPath, dirUUID)
	if err := os.MkdirAll(dirPath, 0700); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, ".gpg.id"), []byte(rootFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create dir .gpg.id: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, dirUUID+"_md.gpg"), []byte("encrypted-dir-metadata"), 0600); err != nil {
		t.Fatalf("failed to write dir metadata: %v", err)
	}

	subDirUUID := "11111111-2222-3333-4444-555555555555"
	subDirPath := filepath.Join(dirPath, subDirUUID)
	if err := os.MkdirAll(subDirPath, 0700); err != nil {
		t.Fatalf("failed to create sub dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirPath, ".gpg.id"), []byte(rootFingerprint+"\n"+requesterFingerprint+"\n"), 0600); err != nil {
		t.Fatalf("failed to create sub dir .gpg.id: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirPath, subDirUUID+"_md.gpg"), []byte("encrypted-subdir-metadata"), 0600); err != nil {
		t.Fatalf("failed to write sub dir metadata: %v", err)
	}

	secretUUID := "66666666-7777-8888-9999-aaaaaaaaaaaa"
	if err := os.WriteFile(filepath.Join(subDirPath, secretUUID+".gpg"), []byte("encrypted-secret"), 0600); err != nil {
		t.Fatalf("failed to write secret: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDirPath, secretUUID+"_md.gpg"), []byte("encrypted-secret-metadata"), 0600); err != nil {
		t.Fatalf("failed to write secret metadata: %v", err)
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

	decryptArgs := []string{"--decrypt", "--batch", "--pinentry-mode", "loopback", "--passphrase", ""}

	t.Run("list_subfolder", func(t *testing.T) {
		mockShell := mocks.NewMockShell()
		mockUI := mocks.NewMockUI()
		mockGitHub := mocks.NewMockGitHub("Requester User", "requester@example.com")

		subDirMetadataJSON := `{"path":"prod/db","type":"dir"}`
		secretMetadataJSON := `{"path":"password","type":"password"}`

		mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, subDirMetadataJSON, "", nil)
		mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, secretMetadataJSON, "", nil)

		mockUI.ConfirmInputs = []bool{true}

		app := &cmd.App{
			Shell:  mockShell,
			UI:     mockUI,
			GitHub: mockGitHub,
		}

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"ls", "-r", "testuser/testrepo", "prod/db"})

		execErr := rootCmd.Execute()

		w.Close()
		os.Stdout = oldStdout

		var buf strings.Builder
		b := make([]byte, 1024)
		for {
			n, readErr := r.Read(b)
			if n > 0 {
				buf.Write(b[:n])
			}
			if readErr != nil {
				break
			}
		}
		output := buf.String()

		if execErr != nil {
			t.Fatalf("expected no error, got: %v", execErr)
		}

		if !strings.Contains(output, "password") {
			t.Errorf("expected 'password' in list output, got: %s", output)
		}
	})

	t.Run("get_secret_through_opaque_parent", func(t *testing.T) {
		mockShell := mocks.NewMockShell()
		mockUI := mocks.NewMockUI()
		mockGitHub := mocks.NewMockGitHub("Requester User", "requester@example.com")

		subDirMetadataJSON := `{"path":"prod/db","type":"dir"}`
		secretMetadataJSON := `{"path":"password","type":"password"}`

		mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, subDirMetadataJSON, "", nil)
		mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, secretMetadataJSON, "", nil)
		mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, "my-secret-value", "", nil)
		mockShell.AddQueuedResponse("/usr/bin/gpg", decryptArgs, secretMetadataJSON, "", nil)

		mockUI.ConfirmInputs = []bool{true}

		app := &cmd.App{
			Shell:  mockShell,
			UI:     mockUI,
			GitHub: mockGitHub,
		}

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		rootCmd := cmd.NewRootCmd(app)
		rootCmd.SetArgs([]string{"get", "-r", "testuser/testrepo", "prod/db/password"})

		execErr := rootCmd.Execute()

		w.Close()
		os.Stdout = oldStdout

		var buf strings.Builder
		b := make([]byte, 1024)
		for {
			n, readErr := r.Read(b)
			if n > 0 {
				buf.Write(b[:n])
			}
			if readErr != nil {
				break
			}
		}
		output := buf.String()

		if execErr != nil {
			t.Fatalf("expected no error, got: %v", execErr)
		}

		if !strings.Contains(output, "my-secret-value") {
			t.Errorf("expected 'my-secret-value' in output, got: %s", output)
		}
	})
}
