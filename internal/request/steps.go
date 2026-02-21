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
package request

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/gonzaloalvarez/kepr/internal/common"
	"github.com/gonzaloalvarez/kepr/internal/workflow"
	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/git"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/gonzaloalvarez/kepr/pkg/store"
)

type AccessRequest struct {
	Fingerprint string `json:"fingerprint"`
	Path        string `json:"path"`
	PublicKey   string `json:"public_key"`
	Timestamp   string `json:"timestamp"`
}

type Context struct {
	Shell       shell.Executor
	UI          cout.IO
	GitHub      github.Client
	RepoPath    string
	Path        string
	Token       string
	ConfigDir   string
	UserName    string
	UserEmail   string
	Fingerprint string
	SecretsPath string
	GPG         *gpg.GPG
	RequestUUID string
}

func (c *Context) stepValidate() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "validate",
		Execute: func(ctx context.Context) error {
			c.Token = config.GetToken()
			if err := common.ValidateToken(c.Token); err != nil {
				return err
			}
			c.GitHub.SetToken(c.Token)

			configDir, err := common.ValidateConfigDir()
			if err != nil {
				return err
			}
			c.ConfigDir = configDir

			userName, userEmail, err := common.ValidateUserIdentity()
			if err != nil {
				return err
			}
			c.UserName = userName
			c.UserEmail = userEmail

			g, err := common.ValidateGPGSetup(c.ConfigDir, c.Shell, c.UI)
			if err != nil {
				return err
			}
			c.GPG = g

			fingerprint, err := common.ValidateFingerprint()
			if err != nil {
				return err
			}
			c.Fingerprint = fingerprint

			secretsPath, err := common.GetSecretsPath(c.RepoPath)
			if err != nil {
				return err
			}
			c.SecretsPath = secretsPath

			return nil
		},
	}
}

func (c *Context) stepPull() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "pull",
		Execute: func(ctx context.Context) error {
			gitClient := git.NewWithAuth(c.Token)
			if err := gitClient.Pull(c.SecretsPath, "origin", "main", true); err != nil {
				return fmt.Errorf("failed to pull latest changes: %w", err)
			}
			c.UI.Successfln("Pulled latest changes from remote")
			return nil
		},
	}
}

func (c *Context) stepCheckAccess() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "check_access",
		Execute: func(ctx context.Context) error {
			found, err := store.ScanFingerprint(c.SecretsPath, c.Fingerprint)
			if err != nil {
				return fmt.Errorf("failed to scan fingerprints: %w", err)
			}
			if found {
				return fmt.Errorf("you already have access (fingerprint %s found in store)", c.Fingerprint)
			}
			return nil
		},
	}
}

func (c *Context) stepImportRootKey() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "import_root_key",
		Execute: func(ctx context.Context) error {
			rootFingerprints, err := store.ReadGpgID(c.SecretsPath)
			if err != nil {
				return fmt.Errorf("failed to read root .gpg.id: %w", err)
			}

			for _, fp := range rootFingerprints {
				keyPath := filepath.Join(c.SecretsPath, "keys", fp+".key")
				keyData, err := os.ReadFile(keyPath)
				if err != nil {
					return fmt.Errorf("failed to read root public key %s: %w", fp, err)
				}

				if err := c.GPG.ImportPublicKey(keyData); err != nil {
					return fmt.Errorf("failed to import root public key %s: %w", fp, err)
				}
			}

			c.UI.Successfln("Imported root identity public key(s)")
			return nil
		},
	}
}

func (c *Context) stepCreateBranch() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "create_branch",
		Execute: func(ctx context.Context) error {
			uuid, err := store.GenerateUUID()
			if err != nil {
				return fmt.Errorf("failed to generate request UUID: %w", err)
			}
			c.RequestUUID = uuid

			branchName := "access-request/" + c.RequestUUID
			gitClient := git.New()
			if err := gitClient.CreateBranch(c.SecretsPath, branchName); err != nil {
				return fmt.Errorf("failed to create branch: %w", err)
			}
			c.UI.Successfln("Created branch %s", branchName)
			return nil
		},
	}
}

func (c *Context) stepBuildRequest() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "build_request",
		Execute: func(ctx context.Context) error {
			pubKey, err := c.GPG.ExportPublicKey(c.Fingerprint)
			if err != nil {
				return fmt.Errorf("failed to export requester public key: %w", err)
			}

			req := AccessRequest{
				Fingerprint: c.Fingerprint,
				Path:        c.Path,
				PublicKey:   string(pubKey),
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
			}

			jsonData, err := json.MarshalIndent(req, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal request JSON: %w", err)
			}

			rootFingerprints, err := store.ReadGpgID(c.SecretsPath)
			if err != nil {
				return fmt.Errorf("failed to read root .gpg.id: %w", err)
			}

			encrypted, err := c.GPG.Encrypt(jsonData, rootFingerprints...)
			if err != nil {
				return fmt.Errorf("failed to encrypt request: %w", err)
			}

			requestsDir := filepath.Join(c.SecretsPath, "requests")
			if err := os.MkdirAll(requestsDir, 0700); err != nil {
				return fmt.Errorf("failed to create requests directory: %w", err)
			}

			requestPath := filepath.Join(requestsDir, c.RequestUUID+".json.gpg")
			if err := os.WriteFile(requestPath, encrypted, 0600); err != nil {
				return fmt.Errorf("failed to write encrypted request: %w", err)
			}

			c.UI.Successfln("Created access request %s", c.RequestUUID)
			return nil
		},
	}
}

func (c *Context) stepCommitAndPush() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "commit_and_push",
		Execute: func(ctx context.Context) error {
			gitClient := git.NewWithAuth(c.Token)

			message := fmt.Sprintf("New access request %s", c.RequestUUID)
			if err := gitClient.Commit(c.SecretsPath, message, c.UserName, c.UserEmail); err != nil {
				return fmt.Errorf("failed to commit request: %w", err)
			}

			branchName := "access-request/" + c.RequestUUID
			if err := gitClient.Push(c.SecretsPath, "origin", branchName); err != nil {
				return fmt.Errorf("failed to push request branch: %w", err)
			}

			c.UI.Successfln("Pushed access request to branch %s", branchName)
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
