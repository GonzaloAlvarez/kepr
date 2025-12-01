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
	"fmt"
	"strings"
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

func (m *MockExecutor) Execute(name string, args ...string) (string, string, error) {
	m.Calls = append(m.Calls, MockCall{
		Name: name,
		Args: args,
	})

	key := m.makeKey(name, args)
	resp, ok := m.Responses[key]
	if !ok {
		return "", "", fmt.Errorf("mock: unexpected command: %s %v", name, args)
	}

	return resp.Stdout, resp.Stderr, resp.Err
}

func (m *MockExecutor) ExecuteWithStdin(stdin string, name string, args ...string) (string, string, error) {
	m.Calls = append(m.Calls, MockCall{
		Name:  name,
		Args:  args,
		Stdin: stdin,
	})

	key := m.makeKey(name, args)
	resp, ok := m.Responses[key]
	if !ok {
		return "", "", fmt.Errorf("mock: unexpected command: %s %v", name, args)
	}

	return resp.Stdout, resp.Stderr, resp.Err
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
