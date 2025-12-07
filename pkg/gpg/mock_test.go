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
	"io"
	"strings"

	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

type MockExecutor struct {
	Calls     []MockCall
	Responses map[string]MockResponse
}

type MockCall struct {
	Name  string
	Args  []string
	Stdin string
}

type MockResponse struct {
	Stdout string
	Stderr string
	Err    error
}

func NewMockExecutor() *MockExecutor {
	return &MockExecutor{
		Calls:     []MockCall{},
		Responses: make(map[string]MockResponse),
	}
}

func (m *MockExecutor) LookPath(file string) (string, error) {
	return file, nil
}

func (m *MockExecutor) Command(name string, args ...string) shell.Cmd {
	return &MockCmd{
		executor: m,
		name:     name,
		args:     args,
	}
}

func (m *MockExecutor) executeMock(c *MockCmd) error {
	m.Calls = append(m.Calls, MockCall{
		Name:  c.name,
		Args:  c.args,
		Stdin: c.stdinContent,
	})

	key := m.makeKey(c.name, c.args)
	resp, ok := m.Responses[key]
	if !ok {
		return fmt.Errorf("mock: unexpected command: %s %v", c.name, c.args)
	}

	if c.stdout != nil {
		c.stdout.Write([]byte(resp.Stdout))
	}
	if c.stderr != nil {
		c.stderr.Write([]byte(resp.Stderr))
	}

	return resp.Err
}

func (m *MockExecutor) makeKey(name string, args []string) string {
	return name + " " + strings.Join(args, " ")
}

func (m *MockExecutor) AddResponse(name string, args []string, stdout, stderr string, err error) {
	key := m.makeKey(name, args)
	m.Responses[key] = MockResponse{
		Stdout: stdout,
		Stderr: stderr,
		Err:    err,
	}
}

func (m *MockExecutor) WasCalled(name string, args ...string) bool {
	for _, call := range m.Calls {
		if call.Name == name && len(call.Args) == len(args) {
			match := true
			for i, arg := range args {
				if call.Args[i] != arg {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

type MockCmd struct {
	executor     *MockExecutor
	name         string
	args         []string
	stdinContent string
	stdout       io.Writer
	stderr       io.Writer
}

func (c *MockCmd) SetDir(dir string)   {}
func (c *MockCmd) SetEnv(env []string) {}
func (c *MockCmd) SetStdin(r io.Reader) {
	if r != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r)
		c.stdinContent = buf.String()
	}
}
func (c *MockCmd) SetStdout(w io.Writer) {
	c.stdout = w
}
func (c *MockCmd) SetStderr(w io.Writer) {
	c.stderr = w
}
func (c *MockCmd) Run() error {
	return c.executor.executeMock(c)
}
func (c *MockCmd) Output() ([]byte, error) {
	var out bytes.Buffer
	c.SetStdout(&out)
	err := c.Run()
	return out.Bytes(), err
}
func (c *MockCmd) CombinedOutput() ([]byte, error) {
	var out bytes.Buffer
	c.SetStdout(&out)
	c.SetStderr(&out)
	err := c.Run()
	return out.Bytes(), err
}
