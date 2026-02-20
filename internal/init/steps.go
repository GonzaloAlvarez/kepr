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
package initialize

import (
	"context"
	"fmt"

	"github.com/gonzaloalvarez/kepr/internal/workflow"
	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/git"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

type Context struct {
	Shell       shell.Executor
	UI          cout.IO
	GitHub      github.Client
	RepoPath    string
	Headless    bool
	Token       string
	Fingerprint string
	GPG         *gpg.GPG
	SecretsPath string
	UserName    string
	UserEmail   string
}

func (c *Context) stepAuthenticate() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "authenticate",
		Execute: func(ctx context.Context) error {
			if err := config.EnsureConfigDir(); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}
			token, err := AuthGithub(c.GitHub, c.UI, c.Headless)
			if err != nil {
				return err
			}
			c.Token = token
			c.GitHub.SetToken(token)
			owner, err := c.GitHub.GetCurrentUserLogin()
			if err != nil {
				return fmt.Errorf("failed to get current user: %w", err)
			}
			c.RepoPath = owner + "/" + c.RepoPath
			return nil
		},
	}
}

func (c *Context) stepCheckRepo() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "check_repo",
		Execute: func(ctx context.Context) error {
			repoName := github.ExtractRepoName(c.RepoPath)
			exists, err := c.GitHub.CheckRepoExists(repoName)
			if err != nil {
				return fmt.Errorf("failed to check repository: %w", err)
			}
			if exists {
				return fmt.Errorf("repository '%s' already exists", repoName)
			}
			return nil
		},
	}
}

func (c *Context) stepCreateRepo() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "create_repo",
		Execute: func(ctx context.Context) error {
			repoName := github.ExtractRepoName(c.RepoPath)
			if err := c.GitHub.CreateRepo(repoName); err != nil {
				return fmt.Errorf("failed to create remote repository: %w", err)
			}
			c.UI.Successfln("Created private remote repository: github.com/%s", c.RepoPath)
			return nil
		},
		Retry: &workflow.RetryConfig{
			MaxAttempts: 3,
			PromptRetry: func(err error, attempt int) (bool, error) {
				return c.UI.Confirm(fmt.Sprintf("Failed to create repo: %v. Retry?", err))
			},
		},
	}
}

func (c *Context) stepSaveConfig() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "save_config",
		Execute: func(ctx context.Context) error {
			if err := config.SaveGitHubRepo(c.RepoPath); err != nil {
				return err
			}
			if c.Headless {
				return config.SaveHeadless(true)
			}
			return nil
		},
	}
}

func (c *Context) stepFetchUserInfo() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "fetch_user_info",
		Execute: func(ctx context.Context) error {
			return UserInfo(c.GitHub, c.UI)
		},
	}
}

func (c *Context) stepSetupGPG() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "setup_gpg",
		Execute: func(ctx context.Context) error {
			g, err := SetupGPG(c.Shell, c.UI, c.Headless)
			if err != nil {
				return err
			}
			c.GPG = g
			c.Fingerprint = config.GetUserFingerprint()
			return nil
		},
	}
}

func (c *Context) stepInitStore() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "init_store",
		Execute: func(ctx context.Context) error {
			configDir, err := config.Dir()
			if err != nil {
				return err
			}
			return SetupPasswordStore(configDir, c.RepoPath, c.GPG, c.Fingerprint, c.Shell, c.UI)
		},
	}
}

func (c *Context) stepCommit() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "commit",
		Execute: func(ctx context.Context) error {
			var err error
			c.SecretsPath, err = config.SecretsPathForRepo(c.RepoPath)
			if err != nil {
				return err
			}
			c.UserName = config.GetUserName()
			c.UserEmail = config.GetUserEmail()
			gitClient := git.NewWithAuth(c.Token)
			return gitClient.Commit(c.SecretsPath, "initialized secret store", c.UserName, c.UserEmail)
		},
	}
}

func (c *Context) stepConfigureRemote() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "configure_remote",
		Execute: func(ctx context.Context) error {
			gitClient := git.NewWithAuth(c.Token)
			repoName := github.ExtractRepoName(c.RepoPath)
			remoteURL, err := c.GitHub.GetCloneURL(repoName)
			if err != nil {
				return fmt.Errorf("failed to get clone URL: %w", err)
			}
			if err := gitClient.ConfigureRemote(c.SecretsPath, "origin", remoteURL); err != nil {
				return fmt.Errorf("failed to configure git remote: %w", err)
			}
			c.UI.Successfln("Configured git remote for repository")
			return nil
		},
	}
}

func (c *Context) stepPush() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "push",
		Execute: func(ctx context.Context) error {
			gitClient := git.NewWithAuth(c.Token)
			if err := gitClient.Push(c.SecretsPath, "origin", "main"); err != nil {
				return fmt.Errorf("failed to push to remote: %w", err)
			}
			c.UI.Successfln("Successfully pushed local secrets to remote repository")
			return nil
		},
		Retry: &workflow.RetryConfig{
			MaxAttempts: 3,
			PromptRetry: func(err error, attempt int) (bool, error) {
				return c.UI.Confirm(fmt.Sprintf("Push failed: %v. Retry?", err))
			},
		},
	}
}
