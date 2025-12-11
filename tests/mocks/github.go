package mocks

import (
	"fmt"

	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
)

type MockGitHub struct {
	Token         string
	UserName      string
	UserEmail     string
	Repos         map[string]bool
	UploadedFiles map[string]map[string][]byte
	AuthCalled    bool
}

func NewMockGitHub(userName, userEmail string) *MockGitHub {
	return &MockGitHub{
		UserName:      userName,
		UserEmail:     userEmail,
		Repos:         make(map[string]bool),
		UploadedFiles: make(map[string]map[string][]byte),
	}
}

func (m *MockGitHub) Authenticate(clientID string, io cout.IO) (string, error) {
	m.AuthCalled = true
	io.Infofln("Please visit: %s", "https://github.com/login/device")
	io.Infofln("Enter code: %s", "ABCD-1234")
	m.Token = "mock-github-token-12345"
	return m.Token, nil
}

func (m *MockGitHub) SetToken(token string) {
	m.Token = token
}

func (m *MockGitHub) GetUserIdentity() (string, string, error) {
	if m.UserName == "" || m.UserEmail == "" {
		return "", "", fmt.Errorf("mock: user identity not configured")
	}
	return m.UserName, m.UserEmail, nil
}

func (m *MockGitHub) EnsureRepo(name string, private bool) error {
	m.Repos[name] = private
	return nil
}

func (m *MockGitHub) UploadFile(repo string, filePath string, content []byte) error {
	if m.UploadedFiles[repo] == nil {
		m.UploadedFiles[repo] = make(map[string][]byte)
	}
	m.UploadedFiles[repo][filePath] = content
	return nil
}

func (m *MockGitHub) WasRepoCalled(name string) bool {
	_, exists := m.Repos[name]
	return exists
}

func (m *MockGitHub) WasFileUploaded(repo string, filePath string) bool {
	if m.UploadedFiles[repo] == nil {
		return false
	}
	_, exists := m.UploadedFiles[repo][filePath]
	return exists
}

var _ github.Client = (*MockGitHub)(nil)
