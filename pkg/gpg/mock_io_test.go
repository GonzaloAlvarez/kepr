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

	"github.com/gonzaloalvarez/kepr/pkg/cout"
)

type MockIO struct {
	ConfirmResult bool
	ConfirmError  error
	InputResult   string
	InputError    error
	Messages      []string
}

func NewMockIO() *MockIO {
	return &MockIO{
		Messages: []string{},
	}
}

func (m *MockIO) Confirm(prompt string) (bool, error) {
	m.Messages = append(m.Messages, fmt.Sprintf("Confirm: %s", prompt))
	return m.ConfirmResult, m.ConfirmError
}

func (m *MockIO) Input(prompt string, defaultValue string) (string, error) {
	m.Messages = append(m.Messages, fmt.Sprintf("Input: %s (default: %s)", prompt, defaultValue))
	return m.InputResult, m.InputError
}

func (m *MockIO) Info(a ...interface{}) {
	m.Messages = append(m.Messages, fmt.Sprint(a...))
}

func (m *MockIO) Infoln(a ...interface{}) {
	m.Messages = append(m.Messages, fmt.Sprint(a...))
}

func (m *MockIO) Infof(format string, a ...interface{}) {
	m.Messages = append(m.Messages, fmt.Sprintf(format, a...))
}

func (m *MockIO) Infofln(format string, a ...interface{}) {
	m.Messages = append(m.Messages, fmt.Sprintf(format, a...))
}

func (m *MockIO) Success(a ...interface{}) {
	m.Messages = append(m.Messages, fmt.Sprint(a...))
}

func (m *MockIO) Successln(a ...interface{}) {
	m.Messages = append(m.Messages, fmt.Sprint(a...))
}

func (m *MockIO) Successf(format string, a ...interface{}) {
	m.Messages = append(m.Messages, fmt.Sprintf(format, a...))
}

func (m *MockIO) Successfln(format string, a ...interface{}) {
	m.Messages = append(m.Messages, fmt.Sprintf(format, a...))
}

func (m *MockIO) Warning(message string) {
	m.Messages = append(m.Messages, fmt.Sprintf("Warning: %s", message))
}

func (m *MockIO) HasMessage(substring string) bool {
	for _, msg := range m.Messages {
		if strings.Contains(msg, substring) {
			return true
		}
	}
	return false
}

var _ cout.IO = (*MockIO)(nil)
