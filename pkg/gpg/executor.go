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
package gpg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

type CommandExecutor interface {
	Execute(name string, args ...string) (stdout, stderr string, err error)
	ExecuteWithStdin(stdin string, name string, args ...string) (stdout, stderr string, err error)
}

type RealExecutor struct {
	HomeDir string
}

func (e *RealExecutor) Execute(name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("GNUPGHOME=%s", e.HomeDir))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (e *RealExecutor) ExecuteWithStdin(stdin string, name string, args ...string) (string, string, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("GNUPGHOME=%s", e.HomeDir))

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = bytes.NewBufferString(stdin)

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}


