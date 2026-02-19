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
package get

import (
	"context"

	"github.com/gonzaloalvarez/kepr/internal/workflow"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

func NewWorkflow(key, outputPath, repoPath string, gh github.Client, sh shell.Executor, ui cout.IO) *workflow.Workflow {
	c := &Context{
		Shell:      sh,
		UI:         ui,
		GitHub:     gh,
		RepoPath:   repoPath,
		Key:        key,
		OutputPath: outputPath,
	}

	w := workflow.New(StateStart)

	w.Configure(StateStart).
		Permit(TriggerValidateToken, StateTokenValidated)

	w.Configure(StateTokenValidated).
		OnEntryFrom(TriggerValidateToken, entryWithRetry(c.stepValidateToken())).
		Permit(TriggerValidateConfig, StateConfigValidated)

	w.Configure(StateConfigValidated).
		OnEntryFrom(TriggerValidateConfig, entryWithRetry(c.stepValidateConfig())).
		Permit(TriggerValidateIdentity, StateIdentityValidated)

	w.Configure(StateIdentityValidated).
		OnEntryFrom(TriggerValidateIdentity, entryWithRetry(c.stepValidateIdentity())).
		Permit(TriggerValidateGitHub, StateGitHubValidated)

	w.Configure(StateGitHubValidated).
		OnEntryFrom(TriggerValidateGitHub, entryWithRetry(c.stepValidateGitHub())).
		Permit(TriggerGetSecretsPath, StateSecretsPathReady)

	w.Configure(StateSecretsPathReady).
		OnEntryFrom(TriggerGetSecretsPath, entryWithRetry(c.stepGetSecretsPath())).
		Permit(TriggerPull, StatePulled)

	w.Configure(StatePulled).
		OnEntryFrom(TriggerPull, entryWithRetry(c.stepPull())).
		Permit(TriggerValidateGPG, StateGPGValidated)

	w.Configure(StateGPGValidated).
		OnEntryFrom(TriggerValidateGPG, entryWithRetry(c.stepValidateGPG())).
		Permit(TriggerCheckYubikey, StateYubikeyReady)

	w.Configure(StateYubikeyReady).
		OnEntryFrom(TriggerCheckYubikey, entryWithRetry(c.stepCheckYubikey())).
		Permit(TriggerValidateKey, StateKeyValidated)

	w.Configure(StateKeyValidated).
		OnEntryFrom(TriggerValidateKey, entryWithRetry(c.stepValidateKey())).
		Permit(TriggerInitStore, StateStoreReady)

	w.Configure(StateStoreReady).
		OnEntryFrom(TriggerInitStore, entryWithRetry(c.stepInitStore())).
		Permit(TriggerGetSecret, StateSecretRetrieved)

	w.Configure(StateSecretRetrieved).
		OnEntryFrom(TriggerGetSecret, entryWithRetry(c.stepGetSecret())).
		Permit(TriggerComplete, StateComplete)

	w.Configure(StateComplete)

	w.AddTrigger(TriggerValidateToken)
	w.AddTrigger(TriggerValidateConfig)
	w.AddTrigger(TriggerValidateIdentity)
	w.AddTrigger(TriggerValidateGitHub)
	w.AddTrigger(TriggerGetSecretsPath)
	w.AddTrigger(TriggerPull)
	w.AddTrigger(TriggerValidateGPG)
	w.AddTrigger(TriggerCheckYubikey)
	w.AddTrigger(TriggerValidateKey)
	w.AddTrigger(TriggerInitStore)
	w.AddTrigger(TriggerGetSecret)
	w.AddTrigger(TriggerComplete)

	return w
}

func entryWithRetry(cfg workflow.StepConfig) func(ctx context.Context, args ...any) error {
	return func(ctx context.Context, args ...any) error {
		return workflow.ExecuteWithRetry(ctx, cfg)
	}
}
