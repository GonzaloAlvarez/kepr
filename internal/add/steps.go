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
package add

import (
	"context"
	"fmt"

	"github.com/gonzaloalvarez/kepr/internal/common"
	"github.com/gonzaloalvarez/kepr/internal/workflow"
	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/git"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/pass"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/gonzaloalvarez/kepr/pkg/store"
)

type Context struct {
	Shell       shell.Executor
	UI          cout.IO
	GitHub      github.Client
	RepoPath    string
	Key         string
	Token       string
	ConfigDir   string
	UserName    string
	UserEmail   string
	Fingerprint string
	SecretsPath string
	GPG         *gpg.GPG
	Store       *store.Store
	Pass        *pass.Pass
}

func (c *Context) stepValidateToken() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "validate_token",
		Execute: func(ctx context.Context) error {
			c.Token = config.GetToken()
			if err := common.ValidateToken(c.Token); err != nil {
				return err
			}
			c.GitHub.SetToken(c.Token)
			return nil
		},
	}
}

func (c *Context) stepValidateConfig() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "validate_config",
		Execute: func(ctx context.Context) error {
			configDir, err := common.ValidateConfigDir()
			if err != nil {
				return err
			}
			c.ConfigDir = configDir
			return nil
		},
	}
}

func (c *Context) stepValidateIdentity() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "validate_identity",
		Execute: func(ctx context.Context) error {
			userName, userEmail, err := common.ValidateUserIdentity()
			if err != nil {
				return err
			}
			c.UserName = userName
			c.UserEmail = userEmail
			return nil
		},
	}
}

func (c *Context) stepValidateGitHub() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "validate_github",
		Execute: func(ctx context.Context) error {
			return common.ValidateGitHubIdentity(c.GitHub, c.UserEmail)
		},
	}
}

func (c *Context) stepValidateGPG() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "validate_gpg",
		Execute: func(ctx context.Context) error {
			g, err := common.ValidateGPGSetup(c.ConfigDir, c.Shell, c.UI)
			if err != nil {
				return err
			}
			c.GPG = g
			return common.ValidateGPGKey(c.GPG, c.UserEmail)
		},
	}
}

func (c *Context) stepValidateKey() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "validate_key",
		Execute: func(ctx context.Context) error {
			fingerprint, err := common.ValidateFingerprint()
			if err != nil {
				return err
			}
			c.Fingerprint = fingerprint
			return nil
		},
	}
}

func (c *Context) stepGetSecretsPath() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "get_secrets_path",
		Execute: func(ctx context.Context) error {
			secretsPath, err := common.GetSecretsPath(c.RepoPath)
			if err != nil {
				return err
			}
			c.SecretsPath = secretsPath
			return nil
		},
	}
}

func (c *Context) stepInitStore() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "init_store",
		Execute: func(ctx context.Context) error {
			st, err := store.New(c.SecretsPath, c.Fingerprint, c.GPG)
			if err != nil {
				return fmt.Errorf("failed to create store: %w", err)
			}
			c.Store = st

			gitClient := git.NewWithAuth(c.Token)
			c.Pass = pass.New(c.SecretsPath, c.GPG, gitClient, c.UI, c.Shell, c.Store)
			return nil
		},
	}
}

func (c *Context) stepAddSecret() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "add_secret",
		Execute: func(ctx context.Context) error {
			if err := c.Pass.Add(c.Key); err != nil {
				return err
			}
			c.UI.Successfln("Secret added: %s", c.Key)
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
			c.UI.Successfln("Pushed to remote repository")
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
