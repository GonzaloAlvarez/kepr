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
	"fmt"
	"os"
	"path/filepath"

	"github.com/gonzaloalvarez/kepr/internal/workflow"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/git"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/gonzaloalvarez/kepr/pkg/store"
)

const (
	ApproveStateStart         workflow.State = "approve_start"
	ApproveStateValidated     workflow.State = "approve_validated"
	ApproveStatePulled        workflow.State = "approve_pulled"
	ApproveStateFetched       workflow.State = "approve_fetched"
	ApproveStateRequestFound  workflow.State = "approve_request_found"
	ApproveStateKeyImported   workflow.State = "approve_key_imported"
	ApproveStateRekeyed       workflow.State = "approve_rekeyed"
	ApproveStateKeyExported   workflow.State = "approve_key_exported"
	ApproveStateCleaned       workflow.State = "approve_cleaned"
	ApproveStatePushed        workflow.State = "approve_pushed"
	ApproveStateBranchDeleted workflow.State = "approve_branch_deleted"
	ApproveStateComplete      workflow.State = "approve_complete"

	ApproveTriggerValidate     workflow.Trigger = "approve_validate"
	ApproveTriggerPull         workflow.Trigger = "approve_pull"
	ApproveTriggerFetch        workflow.Trigger = "approve_fetch"
	ApproveTriggerFindRequest  workflow.Trigger = "approve_find_request"
	ApproveTriggerImportKey    workflow.Trigger = "approve_import_key"
	ApproveTriggerRekey        workflow.Trigger = "approve_rekey"
	ApproveTriggerExportKey    workflow.Trigger = "approve_export_key"
	ApproveTriggerCleanup      workflow.Trigger = "approve_cleanup"
	ApproveTriggerCommitPush   workflow.Trigger = "approve_commit_push"
	ApproveTriggerDeleteBranch workflow.Trigger = "approve_delete_branch"
	ApproveTriggerComplete     workflow.Trigger = "approve_complete"
)

type ApproveContext struct {
	Context
	UUIDPrefix string
	Request    *store.PendingRequest
}

func (c *ApproveContext) stepFindRequest() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "find_request",
		Execute: func(ctx context.Context) error {
			req, err := store.FindRequestByPrefix(c.SecretsPath, c.GPG, c.UUIDPrefix)
			if err != nil {
				return fmt.Errorf("failed to find request: %w", err)
			}
			c.Request = req
			c.UI.Successfln("Found request %s", req.UUID)
			return nil
		},
	}
}

func (c *ApproveContext) stepImportRequesterKey() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "import_requester_key",
		Execute: func(ctx context.Context) error {
			if err := c.GPG.ImportPublicKey([]byte(c.Request.PublicKey)); err != nil {
				return fmt.Errorf("failed to import requester public key: %w", err)
			}
			c.UI.Successfln("Imported requester public key %s", c.Request.Fingerprint)
			return nil
		},
	}
}

func (c *ApproveContext) stepRekey() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "rekey",
		Execute: func(ctx context.Context) error {
			s, err := store.New(c.SecretsPath, c.GPG, c.Fingerprint)
			if err != nil {
				return fmt.Errorf("failed to create store: %w", err)
			}

			targetDir, err := s.ResolvePath(c.Request.Path)
			if err != nil {
				return fmt.Errorf("failed to resolve path %q: %w", c.Request.Path, err)
			}

			existingFingerprints, err := store.ReadGpgID(targetDir)
			if err != nil {
				return fmt.Errorf("failed to read existing fingerprints: %w", err)
			}

			updatedFingerprints := append(existingFingerprints, c.Request.Fingerprint)

			c.UI.Infofln("Rekeying %s and subfolders", c.Request.Path)
			if err := s.Rekey(targetDir, updatedFingerprints, c.Request.Path); err != nil {
				return fmt.Errorf("failed to rekey: %w", err)
			}

			c.UI.Successfln("Rekeying complete")
			return nil
		},
	}
}

func (c *ApproveContext) stepExportRequesterKey() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "export_key",
		Execute: func(ctx context.Context) error {
			keysDir := filepath.Join(c.SecretsPath, "keys")
			if err := os.MkdirAll(keysDir, 0700); err != nil {
				return fmt.Errorf("failed to create keys directory: %w", err)
			}

			pubKey, err := c.GPG.ExportPublicKey(c.Request.Fingerprint)
			if err != nil {
				return fmt.Errorf("failed to export requester public key: %w", err)
			}

			keyPath := filepath.Join(keysDir, c.Request.Fingerprint+".key")
			if err := os.WriteFile(keyPath, pubKey, 0644); err != nil {
				return fmt.Errorf("failed to write requester public key: %w", err)
			}

			c.UI.Successfln("Exported requester public key to keys/")
			return nil
		},
	}
}

func (c *ApproveContext) stepCleanupRequest() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "cleanup",
		Execute: func(ctx context.Context) error {
			requestPath := filepath.Join(c.SecretsPath, "requests", c.Request.UUID+".json.gpg")
			if err := os.Remove(requestPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove request file: %w", err)
			}
			c.UI.Successfln("Removed request file")
			return nil
		},
	}
}

func (c *ApproveContext) stepApproveCommitAndPush() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "commit_and_push",
		Execute: func(ctx context.Context) error {
			gitClient := git.NewWithAuth(c.Token)

			message := fmt.Sprintf("Approve access request %s", c.Request.UUID)
			if err := gitClient.Commit(c.SecretsPath, message, c.UserName, c.UserEmail); err != nil {
				return fmt.Errorf("failed to commit: %w", err)
			}

			if err := gitClient.Push(c.SecretsPath, "origin", "main"); err != nil {
				return fmt.Errorf("failed to push: %w", err)
			}

			c.UI.Successfln("Committed and pushed to main")
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

func (c *ApproveContext) stepDeleteBranch() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "delete_branch",
		Execute: func(ctx context.Context) error {
			branchName := "access-request/" + c.Request.UUID
			gitClient := git.NewWithAuth(c.Token)

			if err := gitClient.DeleteRemoteBranch(c.SecretsPath, "origin", branchName); err != nil {
				c.UI.Warning(fmt.Sprintf("Failed to delete remote branch %s: %v", branchName, err))
				return nil
			}

			c.UI.Successfln("Deleted remote branch %s", branchName)
			return nil
		},
	}
}

func NewApproveWorkflow(uuidPrefix, repoPath string, gh github.Client, sh shell.Executor, ui cout.IO) *workflow.Workflow {
	c := &ApproveContext{
		Context: Context{
			Shell:    sh,
			UI:       ui,
			GitHub:   gh,
			RepoPath: repoPath,
		},
		UUIDPrefix: uuidPrefix,
	}

	w := workflow.New(ApproveStateStart)

	w.Configure(ApproveStateStart).
		Permit(ApproveTriggerValidate, ApproveStateValidated)

	w.Configure(ApproveStateValidated).
		OnEntryFrom(ApproveTriggerValidate, entryWithRetry(c.stepValidate())).
		Permit(ApproveTriggerPull, ApproveStatePulled)

	w.Configure(ApproveStatePulled).
		OnEntryFrom(ApproveTriggerPull, entryWithRetry(c.stepPull())).
		Permit(ApproveTriggerFetch, ApproveStateFetched)

	w.Configure(ApproveStateFetched).
		OnEntryFrom(ApproveTriggerFetch, entryWithRetry(c.stepFetchRequests())).
		Permit(ApproveTriggerFindRequest, ApproveStateRequestFound)

	w.Configure(ApproveStateRequestFound).
		OnEntryFrom(ApproveTriggerFindRequest, entryWithRetry(c.stepFindRequest())).
		Permit(ApproveTriggerImportKey, ApproveStateKeyImported)

	w.Configure(ApproveStateKeyImported).
		OnEntryFrom(ApproveTriggerImportKey, entryWithRetry(c.stepImportRequesterKey())).
		Permit(ApproveTriggerRekey, ApproveStateRekeyed)

	w.Configure(ApproveStateRekeyed).
		OnEntryFrom(ApproveTriggerRekey, entryWithRetry(c.stepRekey())).
		Permit(ApproveTriggerExportKey, ApproveStateKeyExported)

	w.Configure(ApproveStateKeyExported).
		OnEntryFrom(ApproveTriggerExportKey, entryWithRetry(c.stepExportRequesterKey())).
		Permit(ApproveTriggerCleanup, ApproveStateCleaned)

	w.Configure(ApproveStateCleaned).
		OnEntryFrom(ApproveTriggerCleanup, entryWithRetry(c.stepCleanupRequest())).
		Permit(ApproveTriggerCommitPush, ApproveStatePushed)

	w.Configure(ApproveStatePushed).
		OnEntryFrom(ApproveTriggerCommitPush, entryWithRetry(c.stepApproveCommitAndPush())).
		Permit(ApproveTriggerDeleteBranch, ApproveStateBranchDeleted)

	w.Configure(ApproveStateBranchDeleted).
		OnEntryFrom(ApproveTriggerDeleteBranch, entryWithRetry(c.stepDeleteBranch())).
		Permit(ApproveTriggerComplete, ApproveStateComplete)

	w.Configure(ApproveStateComplete)

	w.AddTrigger(ApproveTriggerValidate)
	w.AddTrigger(ApproveTriggerPull)
	w.AddTrigger(ApproveTriggerFetch)
	w.AddTrigger(ApproveTriggerFindRequest)
	w.AddTrigger(ApproveTriggerImportKey)
	w.AddTrigger(ApproveTriggerRekey)
	w.AddTrigger(ApproveTriggerExportKey)
	w.AddTrigger(ApproveTriggerCleanup)
	w.AddTrigger(ApproveTriggerCommitPush)
	w.AddTrigger(ApproveTriggerDeleteBranch)
	w.AddTrigger(ApproveTriggerComplete)

	return w
}
