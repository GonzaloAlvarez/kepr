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
package git

import (
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func TestNew(t *testing.T) {
	g := New()
	if g == nil {
		t.Fatal("New() returned nil")
	}
	if g.AuthToken != "" {
		t.Errorf("New() AuthToken = %q, want empty", g.AuthToken)
	}
}

func TestNewWithAuth(t *testing.T) {
	g := NewWithAuth("test-token")
	if g == nil {
		t.Fatal("NewWithAuth() returned nil")
	}
	if g.AuthToken != "test-token" {
		t.Errorf("NewWithAuth() AuthToken = %q, want \"test-token\"", g.AuthToken)
	}
}

func TestSetAuth(t *testing.T) {
	g := New()
	g.SetAuth("new-token")
	if g.AuthToken != "new-token" {
		t.Errorf("SetAuth() AuthToken = %q, want \"new-token\"", g.AuthToken)
	}
}

func TestGetAuth_WithToken(t *testing.T) {
	g := NewWithAuth("test-token")
	auth := g.getAuth()
	if auth == nil {
		t.Fatal("getAuth() returned nil with token set")
	}
	if auth.Username != "x-access-token" {
		t.Errorf("getAuth() Username = %q, want \"x-access-token\"", auth.Username)
	}
	if auth.Password != "test-token" {
		t.Errorf("getAuth() Password = %q, want \"test-token\"", auth.Password)
	}
}

func TestGetAuth_WithoutToken(t *testing.T) {
	g := New()
	auth := g.getAuth()
	if auth != nil {
		t.Errorf("getAuth() without token = %v, want nil", auth)
	}
}

func TestInit(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	g := New()
	err := g.Init(repoPath)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("Failed to open initialized repo: %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		if err == plumbing.ErrReferenceNotFound {
			ref, err := repo.Reference(plumbing.HEAD, false)
			if err != nil {
				t.Fatalf("Failed to get HEAD reference: %v", err)
			}
			if ref.Target().Short() != "main" {
				t.Errorf("HEAD target = %q, want \"main\"", ref.Target().Short())
			}
		} else {
			t.Fatalf("Failed to get HEAD: %v", err)
		}
	} else {
		if head.Name().Short() != "main" {
			t.Errorf("HEAD branch = %q, want \"main\"", head.Name().Short())
		}
	}
}

func TestInit_AlreadyExists(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	g := New()
	err := g.Init(repoPath)
	if err != nil {
		t.Fatalf("First Init() returned error: %v", err)
	}

	err = g.Init(repoPath)
	if err == nil {
		t.Error("Second Init() should return error")
	}
}

func TestCommit_CleanWorktree(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	g := New()
	err := g.Init(repoPath)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	err = g.Commit(repoPath, "test commit", "Test User", "test@example.com")
	if err != nil {
		t.Fatalf("Commit() with clean worktree returned error: %v", err)
	}
}

func TestCommit_WithChanges(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	g := New()
	err := g.Init(repoPath)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = g.Commit(repoPath, "add test file", "Test User", "test@example.com")
	if err != nil {
		t.Fatalf("Commit() returned error: %v", err)
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	head, err := repo.Head()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		t.Fatalf("Failed to get commit: %v", err)
	}

	if commit.Message != "add test file" {
		t.Errorf("Commit message = %q, want \"add test file\"", commit.Message)
	}

	if commit.Author.Name != "Test User" {
		t.Errorf("Author name = %q, want \"Test User\"", commit.Author.Name)
	}

	if commit.Author.Email != "test@example.com" {
		t.Errorf("Author email = %q, want \"test@example.com\"", commit.Author.Email)
	}
}

func TestCommit_InvalidRepo(t *testing.T) {
	g := New()
	err := g.Commit("/nonexistent/path", "test", "User", "user@example.com")
	if err == nil {
		t.Error("Commit() with invalid repo should return error")
	}
}

func TestConfigureRemote(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	g := New()
	err := g.Init(repoPath)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	err = g.ConfigureRemote(repoPath, "origin", "https://github.com/test/repo.git")
	if err != nil {
		t.Fatalf("ConfigureRemote() returned error: %v", err)
	}

	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		t.Fatalf("Failed to open repo: %v", err)
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		t.Fatalf("Failed to get remote: %v", err)
	}

	urls := remote.Config().URLs
	if len(urls) != 1 || urls[0] != "https://github.com/test/repo.git" {
		t.Errorf("Remote URLs = %v, want [\"https://github.com/test/repo.git\"]", urls)
	}
}

func TestConfigureRemote_AlreadyExists(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "test-repo")

	g := New()
	err := g.Init(repoPath)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	err = g.ConfigureRemote(repoPath, "origin", "https://github.com/test/repo.git")
	if err != nil {
		t.Fatalf("First ConfigureRemote() returned error: %v", err)
	}

	err = g.ConfigureRemote(repoPath, "origin", "https://github.com/test/repo2.git")
	if err == nil {
		t.Error("Second ConfigureRemote() with same name should return error")
	}
}

func TestConfigureRemote_InvalidRepo(t *testing.T) {
	g := New()
	err := g.ConfigureRemote("/nonexistent/path", "origin", "https://example.com")
	if err == nil {
		t.Error("ConfigureRemote() with invalid repo should return error")
	}
}

func TestPush_InvalidRepo(t *testing.T) {
	g := New()
	err := g.Push("/nonexistent/path", "origin", "main")
	if err == nil {
		t.Error("Push() with invalid repo should return error")
	}
}

func TestPull_InvalidRepo(t *testing.T) {
	g := New()
	err := g.Pull("/nonexistent/path", "origin", "main", true)
	if err == nil {
		t.Error("Pull() with invalid repo should return error")
	}
}

func createBareRepo(t *testing.T, path string) {
	t.Helper()
	repo, err := gogit.PlainInit(path, true)
	if err != nil {
		t.Fatalf("Failed to create bare repo: %v", err)
	}
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := repo.Storer.SetReference(headRef); err != nil {
		t.Fatalf("Failed to set bare repo HEAD to main: %v", err)
	}
}

func TestClone(t *testing.T) {
	tempDir := t.TempDir()

	bareRepoPath := filepath.Join(tempDir, "bare.git")
	createBareRepo(t, bareRepoPath)

	srcPath := filepath.Join(tempDir, "src")
	g := New()
	if err := g.Init(srcPath); err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	testFile := filepath.Join(srcPath, "secret.txt")
	if err := os.WriteFile(testFile, []byte("secret-data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := g.Commit(srcPath, "initial commit", "Test", "test@test.com"); err != nil {
		t.Fatalf("Commit() returned error: %v", err)
	}

	if err := g.ConfigureRemote(srcPath, "origin", "file://"+bareRepoPath); err != nil {
		t.Fatalf("ConfigureRemote() returned error: %v", err)
	}

	if err := g.Push(srcPath, "origin", "main"); err != nil {
		t.Fatalf("Push() returned error: %v", err)
	}

	clonePath := filepath.Join(tempDir, "clone")
	if err := g.Clone("file://"+bareRepoPath, clonePath); err != nil {
		t.Fatalf("Clone() returned error: %v", err)
	}

	clonedFile := filepath.Join(clonePath, "secret.txt")
	data, err := os.ReadFile(clonedFile)
	if err != nil {
		t.Fatalf("Failed to read cloned file: %v", err)
	}
	if string(data) != "secret-data" {
		t.Errorf("Cloned file content = %q, want %q", string(data), "secret-data")
	}

	repo, err := gogit.PlainOpen(clonePath)
	if err != nil {
		t.Fatalf("Failed to open cloned repo: %v", err)
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		t.Fatalf("Failed to get remote from cloned repo: %v", err)
	}

	urls := remote.Config().URLs
	if len(urls) != 1 || urls[0] != "file://"+bareRepoPath {
		t.Errorf("Clone remote URLs = %v, want [%q]", urls, "file://"+bareRepoPath)
	}
}

func TestClone_InvalidURL(t *testing.T) {
	tempDir := t.TempDir()
	clonePath := filepath.Join(tempDir, "clone")

	g := New()
	err := g.Clone("file:///nonexistent/repo.git", clonePath)
	if err == nil {
		t.Error("Clone() with invalid URL should return error")
	}
}

func TestClone_DestinationExists(t *testing.T) {
	tempDir := t.TempDir()

	bareRepoPath := filepath.Join(tempDir, "bare.git")
	createBareRepo(t, bareRepoPath)

	clonePath := filepath.Join(tempDir, "clone")
	if err := os.MkdirAll(clonePath, 0755); err != nil {
		t.Fatalf("Failed to create destination: %v", err)
	}
	if err := os.WriteFile(filepath.Join(clonePath, ".git"), []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create .git file: %v", err)
	}

	g := New()
	if err := g.Clone("file://"+bareRepoPath, clonePath); err == nil {
		t.Error("Clone() to existing repo should return error")
	}
}

func TestPush_WithLocalFileRemote(t *testing.T) {
	tempDir := t.TempDir()

	bareRepoPath := filepath.Join(tempDir, "bare.git")
	createBareRepo(t, bareRepoPath)

	workRepoPath := filepath.Join(tempDir, "work")
	g := New()
	if err := g.Init(workRepoPath); err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	testFile := filepath.Join(workRepoPath, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := g.Commit(workRepoPath, "initial commit", "Test", "test@test.com"); err != nil {
		t.Fatalf("Commit() returned error: %v", err)
	}

	if err := g.ConfigureRemote(workRepoPath, "origin", "file://"+bareRepoPath); err != nil {
		t.Fatalf("ConfigureRemote() returned error: %v", err)
	}

	if err := g.Push(workRepoPath, "origin", "main"); err != nil {
		t.Fatalf("Push() returned error: %v", err)
	}
}
