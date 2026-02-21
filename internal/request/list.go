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

	"github.com/gonzaloalvarez/kepr/internal/workflow"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/gonzaloalvarez/kepr/pkg/store"
)

const (
	ListStateStart     workflow.State = "list_start"
	ListStateValidated workflow.State = "list_validated"
	ListStatePulled    workflow.State = "list_pulled"
	ListStateFetched   workflow.State = "list_fetched"
	ListStateDisplayed workflow.State = "list_displayed"
	ListStateComplete  workflow.State = "list_complete"

	ListTriggerValidate workflow.Trigger = "list_validate"
	ListTriggerPull     workflow.Trigger = "list_pull"
	ListTriggerFetch    workflow.Trigger = "list_fetch"
	ListTriggerDisplay  workflow.Trigger = "list_display"
	ListTriggerComplete workflow.Trigger = "list_complete"
)

type ListContext struct {
	Context
}

func (c *ListContext) stepListDisplay() workflow.StepConfig {
	return workflow.StepConfig{
		Name: "display",
		Execute: func(ctx context.Context) error {
			requests, err := store.ListRequests(c.SecretsPath, c.GPG)
			if err != nil {
				return fmt.Errorf("failed to list requests: %w", err)
			}

			if len(requests) == 0 {
				c.UI.Infoln("No pending access requests")
				return nil
			}

			for _, req := range requests {
				name, email := resolveIdentity(c.GPG, req)
				c.UI.Infofln("%s - %s - %s - %s", req.UUID, name, email, req.Path)
			}

			return nil
		},
	}
}

func resolveIdentity(g *gpg.GPG, req store.PendingRequest) (string, string) {
	_ = g.ImportPublicKey([]byte(req.PublicKey))

	keys, err := g.ListPublicKeys()
	if err != nil {
		return req.Fingerprint[:16], ""
	}

	for _, k := range keys {
		if k.Fingerprint == req.Fingerprint {
			return k.Name, k.Email
		}
	}

	return req.Fingerprint[:16], ""
}

func NewListWorkflow(repoPath string, gh github.Client, sh shell.Executor, ui cout.IO) *workflow.Workflow {
	c := &ListContext{
		Context: Context{
			Shell:    sh,
			UI:       ui,
			GitHub:   gh,
			RepoPath: repoPath,
		},
	}

	w := workflow.New(ListStateStart)

	w.Configure(ListStateStart).
		Permit(ListTriggerValidate, ListStateValidated)

	w.Configure(ListStateValidated).
		OnEntryFrom(ListTriggerValidate, entryWithRetry(c.stepValidate())).
		Permit(ListTriggerPull, ListStatePulled)

	w.Configure(ListStatePulled).
		OnEntryFrom(ListTriggerPull, entryWithRetry(c.stepPull())).
		Permit(ListTriggerFetch, ListStateFetched)

	w.Configure(ListStateFetched).
		OnEntryFrom(ListTriggerFetch, entryWithRetry(c.stepFetchRequests())).
		Permit(ListTriggerDisplay, ListStateDisplayed)

	w.Configure(ListStateDisplayed).
		OnEntryFrom(ListTriggerDisplay, entryWithRetry(c.stepListDisplay())).
		Permit(ListTriggerComplete, ListStateComplete)

	w.Configure(ListStateComplete)

	w.AddTrigger(ListTriggerValidate)
	w.AddTrigger(ListTriggerPull)
	w.AddTrigger(ListTriggerFetch)
	w.AddTrigger(ListTriggerDisplay)
	w.AddTrigger(ListTriggerComplete)

	return w
}
