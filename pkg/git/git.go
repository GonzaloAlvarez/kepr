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
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

type Git struct {
	BinaryPath string
	executor   shell.Executor
}

func New(executor shell.Executor) (*Git, error) {
	gitBinary, err := executor.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git binary not found: %w", err)
	}

	return &Git{
		BinaryPath: gitBinary,
		executor:   executor,
	}, nil
}

func (g *Git) ConfigureRemote(repoPath, remoteName, remoteURL string) error {
	slog.Debug("configuring git remote", "path", repoPath, "remote", remoteName)

	cmd := g.executor.Command(g.BinaryPath, "remote", "add", remoteName, remoteURL)
	cmd.SetDir(repoPath)

	var stderrBuf bytes.Buffer
	cmd.SetStderr(&stderrBuf)

	err := cmd.Run()
	if err != nil {
		stderr := stderrBuf.String()
		if strings.Contains(stderr, "already exists") {
			return fmt.Errorf("remote '%s' already exists", remoteName)
		}
		return fmt.Errorf("failed to add remote: %w, stderr: %s", err, stderr)
	}

	slog.Debug("git remote configured", "remote", remoteName)
	return nil
}

func (g *Git) Pull(repoPath, remoteName, branch string, silent bool) error {
	slog.Debug("pulling from remote", "path", repoPath, "remote", remoteName, "branch", branch)

	cmd := g.executor.Command(g.BinaryPath, "pull", "--squash", remoteName, branch)
	cmd.SetDir(repoPath)

	var stderrBuf bytes.Buffer
	if silent {
		var stdoutBuf bytes.Buffer
		cmd.SetStdout(&stdoutBuf)
	}
	cmd.SetStderr(&stderrBuf)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull: %w, stderr: %s", err, stderrBuf.String())
	}

	slog.Debug("successfully pulled from remote")
	return nil
}

func (g *Git) Push(repoPath, remoteName, branch string) error {
	slog.Debug("pushing to remote", "path", repoPath, "remote", remoteName, "branch", branch)

	cmd := g.executor.Command(g.BinaryPath, "push", "-u", remoteName, branch)
	cmd.SetDir(repoPath)

	var stderrBuf bytes.Buffer
	cmd.SetStderr(&stderrBuf)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to push: %w, stderr: %s", err, stderrBuf.String())
	}

	slog.Debug("successfully pushed to remote")
	return nil
}

func (g *Git) Init(repoPath string) error {
	slog.Debug("initializing git repository", "path", repoPath)

	cmd := g.executor.Command(g.BinaryPath, "init", "-b", "main")
	cmd.SetDir(repoPath)

	var stderrBuf bytes.Buffer
	cmd.SetStderr(&stderrBuf)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to initialize git repository: %w, stderr: %s", err, stderrBuf.String())
	}

	slog.Debug("git repository initialized")
	return nil
}

func (g *Git) Commit(repoPath, message, authorName, authorEmail string) error {
	slog.Debug("committing changes", "path", repoPath, "message", message)

	statusCmd := g.executor.Command(g.BinaryPath, "status", "--porcelain")
	statusCmd.SetDir(repoPath)

	var statusOut bytes.Buffer
	statusCmd.SetStdout(&statusOut)

	if err := statusCmd.Run(); err != nil {
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if statusOut.Len() == 0 {
		slog.Debug("no changes to commit")
		return nil
	}

	addCmd := g.executor.Command(g.BinaryPath, "add", "-A", ".")
	addCmd.SetDir(repoPath)

	var addStderr bytes.Buffer
	addCmd.SetStderr(&addStderr)

	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("failed to stage changes: %w, stderr: %s", err, addStderr.String())
	}

	commitCmd := g.executor.Command(g.BinaryPath, "commit", "-m", message)
	commitCmd.SetDir(repoPath)
	commitCmd.SetEnv(append(
		os.Environ(),
		fmt.Sprintf("GIT_AUTHOR_NAME=%s", authorName),
		fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", authorEmail),
		fmt.Sprintf("GIT_COMMITTER_NAME=%s", authorName),
		fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", authorEmail),
	))

	var commitStderr bytes.Buffer
	commitCmd.SetStderr(&commitStderr)

	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("failed to commit: %w, stderr: %s", err, commitStderr.String())
	}

	slog.Debug("changes committed successfully")
	return nil
}
