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
package shell

import (
	"io"
	"os/exec"
)

// Cmd is an interface that mirrors exec.Cmd methods needed by the application.
type Cmd interface {
	SetDir(dir string)
	SetEnv(env []string)
	SetStdin(r io.Reader)
	SetStdout(w io.Writer)
	SetStderr(w io.Writer)
	Output() ([]byte, error)
	Run() error
	CombinedOutput() ([]byte, error)
}

// Executor is an interface for looking up paths and creating commands.
type Executor interface {
	LookPath(file string) (string, error)
	Command(name string, args ...string) Cmd
}

// SystemCmd wraps *exec.Cmd to satisfy the Cmd interface.
type SystemCmd struct {
	cmd *exec.Cmd
}

func (c *SystemCmd) SetDir(dir string) {
	c.cmd.Dir = dir
}

func (c *SystemCmd) SetEnv(env []string) {
	c.cmd.Env = env
}

func (c *SystemCmd) SetStdin(r io.Reader) {
	c.cmd.Stdin = r
}

func (c *SystemCmd) SetStdout(w io.Writer) {
	c.cmd.Stdout = w
}

func (c *SystemCmd) SetStderr(w io.Writer) {
	c.cmd.Stderr = w
}

func (c *SystemCmd) Output() ([]byte, error) {
	return c.cmd.Output()
}

func (c *SystemCmd) Run() error {
	return c.cmd.Run()
}

func (c *SystemCmd) CombinedOutput() ([]byte, error) {
	return c.cmd.CombinedOutput()
}

// SystemExecutor implements Executor using the standard os/exec package.
type SystemExecutor struct{}

func (e *SystemExecutor) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

func (e *SystemExecutor) Command(name string, args ...string) Cmd {
	return &SystemCmd{cmd: exec.Command(name, args...)}
}
