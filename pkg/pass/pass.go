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
package pass

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

type Pass struct {
	SecretsPath string
	GpgHome     string
	executor    shell.Executor
}

func New(configDir, gpgHome string, executor shell.Executor) *Pass {
	return &Pass{
		SecretsPath: filepath.Join(configDir, "secrets"),
		GpgHome:     gpgHome,
		executor:    executor,
	}
}

func (p *Pass) Init(fingerprint string) error {
	slog.Debug("initializing password store", "path", p.SecretsPath, "fingerprint", fingerprint)

	if err := os.MkdirAll(p.SecretsPath, 0700); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	gopassPath, err := p.executor.LookPath("gopass")
	if err != nil {
		return fmt.Errorf("gopass binary not found: %w", err)
	}

	slog.Debug("found gopass binary", "path", gopassPath)

	cmd := p.executor.Command(gopassPath, "init", "--path", p.SecretsPath, "--crypto", "gpg", fingerprint)
	cmd.SetEnv(append(os.Environ(), fmt.Sprintf("GNUPGHOME=%s", p.GpgHome)))

	var stdout, stderr bytes.Buffer
	cmd.SetStdout(&stdout)
	cmd.SetStderr(&stderr)

	slog.Debug("executing gopass init")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gopass init failed: %w, stdout: %s, stderr: %s", err, stdout.String(), stderr.String())
	}

	slog.Debug("gopass init successful", "stdout", stdout.String())

	if err := p.writeGitignore(); err != nil {
		return fmt.Errorf("failed to write .gitignore: %w", err)
	}

	slog.Debug("password store initialized successfully")
	return nil
}

func (p *Pass) writeGitignore() error {
	gitignorePath := filepath.Join(p.SecretsPath, ".gitignore")
	content := `*
!*/
!**/*.gpg
!.gpg-id
`

	slog.Debug("writing .gitignore", "path", gitignorePath)
	if err := os.WriteFile(gitignorePath, []byte(content), 0600); err != nil {
		return err
	}

	return nil
}
