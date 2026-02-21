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

	"github.com/gonzaloalvarez/kepr/internal/workflow"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

func NewWorkflow(path, repoPath string, gh github.Client, sh shell.Executor, ui cout.IO) *workflow.Workflow {
	c := &Context{
		Shell:    sh,
		UI:       ui,
		GitHub:   gh,
		RepoPath: repoPath,
		Path:     path,
	}

	w := workflow.New(StateStart)

	w.Configure(StateStart).
		Permit(TriggerValidate, StateValidated)

	w.Configure(StateValidated).
		OnEntryFrom(TriggerValidate, entryWithRetry(c.stepValidate())).
		Permit(TriggerPull, StatePulled)

	w.Configure(StatePulled).
		OnEntryFrom(TriggerPull, entryWithRetry(c.stepPull())).
		Permit(TriggerCheckAccess, StateAccessChecked)

	w.Configure(StateAccessChecked).
		OnEntryFrom(TriggerCheckAccess, entryWithRetry(c.stepCheckAccess())).
		Permit(TriggerImportRootKey, StateRootKeyImported)

	w.Configure(StateRootKeyImported).
		OnEntryFrom(TriggerImportRootKey, entryWithRetry(c.stepImportRootKey())).
		Permit(TriggerCreateBranch, StateBranchCreated)

	w.Configure(StateBranchCreated).
		OnEntryFrom(TriggerCreateBranch, entryWithRetry(c.stepCreateBranch())).
		Permit(TriggerBuildRequest, StateRequestBuilt)

	w.Configure(StateRequestBuilt).
		OnEntryFrom(TriggerBuildRequest, entryWithRetry(c.stepBuildRequest())).
		Permit(TriggerCommitAndPush, StatePushed)

	w.Configure(StatePushed).
		OnEntryFrom(TriggerCommitAndPush, entryWithRetry(c.stepCommitAndPush())).
		Permit(TriggerComplete, StateComplete)

	w.Configure(StateComplete)

	w.AddTrigger(TriggerValidate)
	w.AddTrigger(TriggerPull)
	w.AddTrigger(TriggerCheckAccess)
	w.AddTrigger(TriggerImportRootKey)
	w.AddTrigger(TriggerCreateBranch)
	w.AddTrigger(TriggerBuildRequest)
	w.AddTrigger(TriggerCommitAndPush)
	w.AddTrigger(TriggerComplete)

	return w
}

func entryWithRetry(cfg workflow.StepConfig) func(ctx context.Context, args ...any) error {
	return func(ctx context.Context, args ...any) error {
		return workflow.ExecuteWithRetry(ctx, cfg)
	}
}
