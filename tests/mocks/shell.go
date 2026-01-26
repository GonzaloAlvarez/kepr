package mocks

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

type MockShell struct {
	Calls     []ShellCall
	Responses map[string]ShellResponse
}

type ShellCall struct {
	Name  string
	Args  []string
	Stdin string
}

type ShellResponse struct {
	Stdout string
	Stderr string
	Err    error
}

func NewMockShell() *MockShell {
	return &MockShell{
		Calls:     []ShellCall{},
		Responses: make(map[string]ShellResponse),
	}
}

func (m *MockShell) LookPath(file string) (string, error) {
	resp, ok := m.Responses["lookpath:"+file]
	if !ok {
		return "/usr/bin/" + file, nil
	}
	if resp.Err != nil {
		return "", resp.Err
	}
	return resp.Stdout, nil
}

func (m *MockShell) Command(name string, args ...string) shell.Cmd {
	return &MockCmd{
		shell: m,
		name:  name,
		args:  args,
	}
}

func (m *MockShell) AddResponse(name string, args []string, stdout, stderr string, err error) {
	key := m.makeKey(name, args)
	m.Responses[key] = ShellResponse{
		Stdout: stdout,
		Stderr: stderr,
		Err:    err,
	}
}

func (m *MockShell) makeKey(name string, args []string) string {
	return name + " " + strings.Join(args, " ")
}

func (m *MockShell) WasCalled(name string, args ...string) bool {
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

func (m *MockShell) GetStdinForCall(name string, args ...string) string {
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
				return call.Stdin
			}
		}
	}
	return ""
}

type MockCmd struct {
	shell        *MockShell
	name         string
	args         []string
	dir          string
	env          []string
	stdin        io.Reader
	stdout       io.Writer
	stderr       io.Writer
	stdinContent string
	extraFiles   []*os.File
}

func (c *MockCmd) SetDir(dir string) {
	c.dir = dir
}

func (c *MockCmd) SetEnv(env []string) {
	c.env = env
}

func (c *MockCmd) SetStdin(r io.Reader) {
	c.stdin = r
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

func (c *MockCmd) GetEnv(key string) string {
	for _, env := range c.env {
		if len(env) > len(key) && env[:len(key)+1] == key+"=" {
			return env[len(key)+1:]
		}
	}
	return ""
}

func (c *MockCmd) SetExtraFiles(files []*os.File) {
	c.extraFiles = files
}

func (c *MockCmd) Start() error {
	return nil
}

func (c *MockCmd) Wait() error {
	return c.Run()
}

func (c *MockCmd) Run() error {
	c.shell.Calls = append(c.shell.Calls, ShellCall{
		Name:  c.name,
		Args:  c.args,
		Stdin: c.stdinContent,
	})

	key := c.shell.makeKey(c.name, c.args)
	resp, ok := c.shell.Responses[key]
	if !ok {
		return fmt.Errorf("mock: unexpected command: %s %v", c.name, c.args)
	}

	if c.stdout != nil && resp.Stdout != "" {
		c.stdout.Write([]byte(resp.Stdout))
	}
	if c.stderr != nil && resp.Stderr != "" {
		c.stderr.Write([]byte(resp.Stderr))
	}

	return resp.Err
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

var _ shell.Executor = (*MockShell)(nil)
var _ shell.Cmd = (*MockCmd)(nil)
