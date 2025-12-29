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

func (g *Git) Pull(repoPath, remoteName, branch string) error {
	slog.Debug("pulling from remote", "path", repoPath, "remote", remoteName, "branch", branch)

	cmd := g.executor.Command(g.BinaryPath, "pull", "--squash", remoteName, branch)
	cmd.SetDir(repoPath)

	var stderrBuf bytes.Buffer
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
