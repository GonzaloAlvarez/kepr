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
package list

import (
	"github.com/gonzaloalvarez/kepr/internal/workflow"
)

const (
	StateStart             workflow.State = "start"
	StateTokenValidated    workflow.State = "token_validated"
	StateConfigValidated   workflow.State = "config_validated"
	StateIdentityValidated workflow.State = "identity_validated"
	StateGitHubValidated   workflow.State = "github_validated"
	StateSecretsPathReady  workflow.State = "secrets_path_ready"
	StatePulled            workflow.State = "pulled"
	StateGPGValidated      workflow.State = "gpg_validated"
	StateYubikeyReady      workflow.State = "yubikey_ready"
	StateKeyValidated      workflow.State = "key_validated"
	StateStoreReady        workflow.State = "store_ready"
	StateListed            workflow.State = "listed"
	StateComplete          workflow.State = "complete"
)

const (
	TriggerValidateToken    workflow.Trigger = "validate_token"
	TriggerValidateConfig   workflow.Trigger = "validate_config"
	TriggerValidateIdentity workflow.Trigger = "validate_identity"
	TriggerValidateGitHub   workflow.Trigger = "validate_github"
	TriggerGetSecretsPath   workflow.Trigger = "get_secrets_path"
	TriggerPull             workflow.Trigger = "pull"
	TriggerValidateGPG      workflow.Trigger = "validate_gpg"
	TriggerCheckYubikey     workflow.Trigger = "check_yubikey"
	TriggerValidateKey      workflow.Trigger = "validate_key"
	TriggerInitStore        workflow.Trigger = "init_store"
	TriggerList             workflow.Trigger = "list"
	TriggerComplete         workflow.Trigger = "complete"
)
