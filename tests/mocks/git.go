package mocks

type MockGit struct {
	InitCalled   bool
	InitRepoPath string
	InitErr      error

	CommitCalled      bool
	CommitRepoPath    string
	CommitMessage     string
	CommitAuthorName  string
	CommitAuthorEmail string
	CommitErr         error

	ConfigureRemoteCalled     bool
	ConfigureRemoteRepoPath   string
	ConfigureRemoteRemoteName string
	ConfigureRemoteRemoteURL  string
	ConfigureRemoteErr        error

	PushCalled     bool
	PushRepoPath   string
	PushRemoteName string
	PushBranch     string
	PushErr        error

	PullCalled     bool
	PullRepoPath   string
	PullRemoteName string
	PullBranch     string
	PullSilent     bool
	PullErr        error
}

func NewMockGit() *MockGit {
	return &MockGit{}
}

func (m *MockGit) Init(repoPath string) error {
	m.InitCalled = true
	m.InitRepoPath = repoPath
	return m.InitErr
}

func (m *MockGit) Commit(repoPath, message, authorName, authorEmail string) error {
	m.CommitCalled = true
	m.CommitRepoPath = repoPath
	m.CommitMessage = message
	m.CommitAuthorName = authorName
	m.CommitAuthorEmail = authorEmail
	return m.CommitErr
}

func (m *MockGit) ConfigureRemote(repoPath, remoteName, remoteURL string) error {
	m.ConfigureRemoteCalled = true
	m.ConfigureRemoteRepoPath = repoPath
	m.ConfigureRemoteRemoteName = remoteName
	m.ConfigureRemoteRemoteURL = remoteURL
	return m.ConfigureRemoteErr
}

func (m *MockGit) Push(repoPath, remoteName, branch string) error {
	m.PushCalled = true
	m.PushRepoPath = repoPath
	m.PushRemoteName = remoteName
	m.PushBranch = branch
	return m.PushErr
}

func (m *MockGit) Pull(repoPath, remoteName, branch string, silent bool) error {
	m.PullCalled = true
	m.PullRepoPath = repoPath
	m.PullRemoteName = remoteName
	m.PullBranch = branch
	m.PullSilent = silent
	return m.PullErr
}

func (m *MockGit) SetAuth(token string) {}
