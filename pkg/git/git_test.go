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

func TestPush_WithLocalFileRemote(t *testing.T) {
	tempDir := t.TempDir()

	bareRepoPath := filepath.Join(tempDir, "bare.git")
	_, err := gogit.PlainInit(bareRepoPath, true)
	if err != nil {
		t.Fatalf("Failed to create bare repo: %v", err)
	}

	workRepoPath := filepath.Join(tempDir, "work")
	g := New()
	err = g.Init(workRepoPath)
	if err != nil {
		t.Fatalf("Init() returned error: %v", err)
	}

	testFile := filepath.Join(workRepoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = g.Commit(workRepoPath, "initial commit", "Test", "test@test.com")
	if err != nil {
		t.Fatalf("Commit() returned error: %v", err)
	}

	err = g.ConfigureRemote(workRepoPath, "origin", "file://"+bareRepoPath)
	if err != nil {
		t.Fatalf("ConfigureRemote() returned error: %v", err)
	}

	err = g.Push(workRepoPath, "origin", "main")
	if err != nil {
		t.Fatalf("Push() returned error: %v", err)
	}
}
