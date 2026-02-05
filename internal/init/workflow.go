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

	"github.com/gonzaloalvarez/kepr/internal/workflow"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

func NewWorkflow(repoPath string, gh github.Client, sh shell.Executor, ui cout.IO) *workflow.Workflow {
	c := &Context{
		Shell:    sh,
		UI:       ui,
		GitHub:   gh,
		RepoPath: repoPath,
	}

	w := workflow.New(StateStart)

	w.Configure(StateStart).
		Permit(TriggerAuthenticate, StateAuthenticated)

	w.Configure(StateAuthenticated).
		OnEntryFrom(TriggerAuthenticate, entryWithRetry(c.stepAuthenticate())).
		Permit(TriggerCheckRepo, StateRepoChecked)

	w.Configure(StateRepoChecked).
		OnEntryFrom(TriggerCheckRepo, entryWithRetry(c.stepCheckRepo())).
		Permit(TriggerCreateRepo, StateRepoCreated)

	w.Configure(StateRepoCreated).
		OnEntryFrom(TriggerCreateRepo, entryWithRetry(c.stepCreateRepo())).
		Permit(TriggerSaveConfig, StateConfigSaved)

	w.Configure(StateConfigSaved).
		OnEntryFrom(TriggerSaveConfig, entryWithRetry(c.stepSaveConfig())).
		Permit(TriggerFetchUserInfo, StateUserInfoFetched)

	w.Configure(StateUserInfoFetched).
		OnEntryFrom(TriggerFetchUserInfo, entryWithRetry(c.stepFetchUserInfo())).
		Permit(TriggerSetupGPG, StateGPGReady)

	w.Configure(StateGPGReady).
		OnEntryFrom(TriggerSetupGPG, entryWithRetry(c.stepSetupGPG())).
		Permit(TriggerInitStore, StateStoreInitialized)

	w.Configure(StateStoreInitialized).
		OnEntryFrom(TriggerInitStore, entryWithRetry(c.stepInitStore())).
		Permit(TriggerCommit, StateGitCommitted)

	w.Configure(StateGitCommitted).
		OnEntryFrom(TriggerCommit, entryWithRetry(c.stepCommit())).
		Permit(TriggerConfigureRemote, StateRemoteConfigured)

	w.Configure(StateRemoteConfigured).
		OnEntryFrom(TriggerConfigureRemote, entryWithRetry(c.stepConfigureRemote())).
		Permit(TriggerPush, StatePushed)

	w.Configure(StatePushed).
		OnEntryFrom(TriggerPush, entryWithRetry(c.stepPush())).
		Permit(TriggerComplete, StateComplete)

	w.Configure(StateComplete)

	w.AddTrigger(TriggerAuthenticate)
	w.AddTrigger(TriggerCheckRepo)
	w.AddTrigger(TriggerCreateRepo)
	w.AddTrigger(TriggerSaveConfig)
	w.AddTrigger(TriggerFetchUserInfo)
	w.AddTrigger(TriggerSetupGPG)
	w.AddTrigger(TriggerInitStore)
	w.AddTrigger(TriggerCommit)
	w.AddTrigger(TriggerConfigureRemote)
	w.AddTrigger(TriggerPush)
	w.AddTrigger(TriggerComplete)

	return w
}

func entryWithRetry(cfg workflow.StepConfig) func(ctx context.Context, args ...any) error {
	return func(ctx context.Context, args ...any) error {
		return workflow.ExecuteWithRetry(ctx, cfg)
	}
}
