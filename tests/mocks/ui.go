package mocks

import (
	"bytes"
	"fmt"

	"github.com/gonzaloalvarez/kepr/pkg/cout"
)

type MockUI struct {
	Output        bytes.Buffer
	ConfirmInputs []bool
	TextInputs    []string
	confirmIndex  int
	textIndex     int
}

func NewMockUI() *MockUI {
	return &MockUI{
		ConfirmInputs: []bool{},
		TextInputs:    []string{},
	}
}

func (m *MockUI) Confirm(prompt string) (bool, error) {
	m.Output.WriteString(fmt.Sprintf("Confirm: %s\n", prompt))
	if m.confirmIndex >= len(m.ConfirmInputs) {
		return false, fmt.Errorf("mock: no more confirm inputs available")
	}
	result := m.ConfirmInputs[m.confirmIndex]
	m.confirmIndex++
	return result, nil
}

func (m *MockUI) Input(prompt string, defaultValue string) (string, error) {
	m.Output.WriteString(fmt.Sprintf("Input: %s (default: %s)\n", prompt, defaultValue))
	if m.textIndex >= len(m.TextInputs) {
		return "", fmt.Errorf("mock: no more text inputs available")
	}
	result := m.TextInputs[m.textIndex]
	m.textIndex++
	return result, nil
}

func (m *MockUI) InputPassword(prompt string) (string, error) {
	m.Output.WriteString(fmt.Sprintf("InputPassword: %s\n", prompt))
	if m.textIndex >= len(m.TextInputs) {
		return "", fmt.Errorf("mock: no more text inputs available")
	}
	result := m.TextInputs[m.textIndex]
	m.textIndex++
	return result, nil
}

func (m *MockUI) Info(a ...interface{}) {
	m.Output.WriteString(fmt.Sprint(a...))
}

func (m *MockUI) Infoln(a ...interface{}) {
	m.Output.WriteString(fmt.Sprint(a...))
	m.Output.WriteString("\n")
}

func (m *MockUI) Infof(format string, a ...interface{}) {
	m.Output.WriteString(fmt.Sprintf(format, a...))
}

func (m *MockUI) Infofln(format string, a ...interface{}) {
	m.Output.WriteString(fmt.Sprintf(format, a...))
	m.Output.WriteString("\n")
}

func (m *MockUI) Success(a ...interface{}) {
	m.Output.WriteString(fmt.Sprint(a...))
}

func (m *MockUI) Successln(a ...interface{}) {
	m.Output.WriteString(fmt.Sprint(a...))
	m.Output.WriteString("\n")
}

func (m *MockUI) Successf(format string, a ...interface{}) {
	m.Output.WriteString(fmt.Sprintf(format, a...))
}

func (m *MockUI) Successfln(format string, a ...interface{}) {
	m.Output.WriteString(fmt.Sprintf(format, a...))
	m.Output.WriteString("\n")
}

func (m *MockUI) Warning(message string) {
	m.Output.WriteString(fmt.Sprintf("Warning: %s\n", message))
}

func (m *MockUI) HasOutput(substring string) bool {
	return bytes.Contains(m.Output.Bytes(), []byte(substring))
}

func (m *MockUI) GetOutput() string {
	return m.Output.String()
}

var _ cout.IO = (*MockUI)(nil)
