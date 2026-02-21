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
	"github.com/gonzaloalvarez/kepr/internal/workflow"
)

const (
	StateStart          workflow.State = "start"
	StateValidated      workflow.State = "validated"
	StatePulled         workflow.State = "pulled"
	StateAccessChecked  workflow.State = "access_checked"
	StateRootKeyImported workflow.State = "root_key_imported"
	StateRequestBuilt   workflow.State = "request_built"
	StateBranchCreated  workflow.State = "branch_created"
	StatePushed         workflow.State = "pushed"
	StateComplete       workflow.State = "complete"
)

const (
	TriggerValidate      workflow.Trigger = "validate"
	TriggerPull          workflow.Trigger = "pull"
	TriggerCheckAccess   workflow.Trigger = "check_access"
	TriggerImportRootKey workflow.Trigger = "import_root_key"
	TriggerBuildRequest  workflow.Trigger = "build_request"
	TriggerCreateBranch  workflow.Trigger = "create_branch"
	TriggerCommitAndPush workflow.Trigger = "commit_and_push"
	TriggerComplete      workflow.Trigger = "complete"
)
