package mocks

import (
	"fmt"

	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
)

type MockGitHub struct {
	Token      string
	UserName   string
	UserEmail  string
	UserLogin  string
	Repos      map[string]bool
	CloneURLs  map[string]string
	AuthCalled bool
}

func NewMockGitHub(userName, userEmail string) *MockGitHub {
	return &MockGitHub{
		UserName:  userName,
		UserEmail: userEmail,
		UserLogin: "testuser",
		Repos:     make(map[string]bool),
		CloneURLs: make(map[string]string),
	}
}

func (m *MockGitHub) CodeBasedAuthentication(clientID string, io cout.IO) (string, error) {
	m.AuthCalled = true
	io.Infofln("Please visit: %s", "https://github.com/login/device")
	io.Infofln("Enter code: %s", "ABCD-1234")
	m.Token = "mock-github-token-12345"
	return m.Token, nil
}

func (m *MockGitHub) PKCEAuthentication(clientID, clientSecret string, io cout.IO) (string, error) {
	m.AuthCalled = true
	io.Infofln("Please visit: %s", "https://github.com/login/oauth/authorize")
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

func (m *MockGitHub) CheckRepoExists(name string) (bool, error) {
	_, exists := m.Repos[name]
	return exists, nil
}

func (m *MockGitHub) CreateRepo(name string) error {
	m.Repos[name] = true
	return nil
}

func (m *MockGitHub) GetCloneURL(name string) (string, error) {
	if url, exists := m.CloneURLs[name]; exists {
		return url, nil
	}
	return fmt.Sprintf("file:///tmp/mock-repos/%s/%s.git", m.UserLogin, name), nil
}

func (m *MockGitHub) WasRepoCalled(name string) bool {
	_, exists := m.Repos[name]
	return exists
}

var _ github.Client = (*MockGitHub)(nil)
