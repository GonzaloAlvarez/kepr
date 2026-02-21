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
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type Git struct {
	AuthToken string
}

func New() *Git {
	return &Git{}
}

func NewWithAuth(token string) *Git {
	return &Git{AuthToken: token}
}

func (g *Git) SetAuth(token string) {
	g.AuthToken = token
}

func (g *Git) getAuth() *http.BasicAuth {
	if g.AuthToken == "" {
		return nil
	}
	return &http.BasicAuth{
		Username: "x-access-token",
		Password: g.AuthToken,
	}
}

func (g *Git) getAuthForRemote(repo *git.Repository, remoteName string) transport.AuthMethod {
	remote, err := repo.Remote(remoteName)
	if err != nil {
		return g.getAuth()
	}

	urls := remote.Config().URLs
	if len(urls) > 0 && strings.HasPrefix(urls[0], "file://") {
		slog.Debug("file:// URL detected, skipping auth", "url", urls[0])
		return nil
	}

	return g.getAuth()
}

func (g *Git) Clone(url, destPath string) error {
	slog.Debug("cloning repository", "url", url, "dest", destPath)

	auth := g.getAuth()
	if strings.HasPrefix(url, "file://") {
		auth = nil
	}

	_, err := git.PlainClone(destPath, false, &git.CloneOptions{
		URL:  url,
		Auth: auth,
	})
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	slog.Debug("repository cloned successfully")
	return nil
}

func (g *Git) Init(repoPath string) error {
	slog.Debug("initializing git repository", "path", repoPath)

	repo, err := git.PlainInit(repoPath, false)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := repo.Storer.SetReference(headRef); err != nil {
		return fmt.Errorf("failed to set HEAD to main: %w", err)
	}

	slog.Debug("git repository initialized")
	return nil
}

func (g *Git) Commit(repoPath, message, authorName, authorEmail string) error {
	slog.Debug("committing changes", "path", repoPath, "message", message)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to check status: %w", err)
	}

	if status.IsClean() {
		slog.Debug("no changes to commit")
		return nil
	}

	if err := w.AddWithOptions(&git.AddOptions{All: true}); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	sig := &object.Signature{
		Name:  authorName,
		Email: authorEmail,
		When:  time.Now(),
	}

	_, err = w.Commit(message, &git.CommitOptions{
		Author:    sig,
		Committer: sig,
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	slog.Debug("changes committed successfully")
	return nil
}

func (g *Git) ConfigureRemote(repoPath, remoteName, remoteURL string) error {
	slog.Debug("configuring git remote", "path", repoPath, "remote", remoteName)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: remoteName,
		URLs: []string{remoteURL},
	})
	if err != nil {
		if err == git.ErrRemoteExists {
			return fmt.Errorf("remote '%s' already exists", remoteName)
		}
		return fmt.Errorf("failed to add remote: %w", err)
	}

	slog.Debug("git remote configured", "remote", remoteName)
	return nil
}

func (g *Git) Push(repoPath, remoteName, branch string) error {
	slog.Debug("pushing to remote", "path", repoPath, "remote", remoteName, "branch", branch)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: remoteName,
		Auth:       g.getAuthForRemote(repo, remoteName),
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to push: %w", err)
	}

	slog.Debug("successfully pushed to remote")
	return nil
}

func (g *Git) CreateBranch(repoPath, branchName string) error {
	slog.Debug("creating branch", "path", repoPath, "branch", branchName)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branchName),
		Create: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	slog.Debug("branch created and checked out", "branch", branchName)
	return nil
}

func (g *Git) Pull(repoPath, remoteName, branch string, silent bool) error {
	slog.Debug("pulling from remote", "path", repoPath, "remote", remoteName, "branch", branch)

	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	refSpec := config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", branch, remoteName, branch))
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: remoteName,
		RefSpecs:   []config.RefSpec{refSpec},
		Auth:       g.getAuthForRemote(repo, remoteName),
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return fmt.Errorf("failed to fetch: %w", err)
	}

	remoteRef, err := repo.Reference(plumbing.NewRemoteReferenceName(remoteName, branch), true)
	if err != nil {
		return fmt.Errorf("failed to get remote reference: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	err = w.Reset(&git.ResetOptions{
		Commit: remoteRef.Hash(),
		Mode:   git.HardReset,
	})
	if err != nil {
		return fmt.Errorf("failed to reset to remote: %w", err)
	}

	localRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branch), remoteRef.Hash())
	if err := repo.Storer.SetReference(localRef); err != nil {
		return fmt.Errorf("failed to update local branch: %w", err)
	}

	slog.Debug("successfully pulled from remote")
	return nil
}
