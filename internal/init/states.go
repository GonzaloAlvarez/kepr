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
	"github.com/gonzaloalvarez/kepr/internal/workflow"
)

const (
	StateStart            workflow.State = "start"
	StateAuthenticated    workflow.State = "authenticated"
	StateRepoChecked      workflow.State = "repo_checked"
	StateRepoCreated      workflow.State = "repo_created"
	StateConfigSaved      workflow.State = "config_saved"
	StateUserInfoFetched  workflow.State = "user_info_fetched"
	StateGPGReady         workflow.State = "gpg_ready"
	StateStoreInitialized workflow.State = "store_initialized"
	StateGitCommitted     workflow.State = "git_committed"
	StateRemoteConfigured workflow.State = "remote_configured"
	StatePushed           workflow.State = "pushed"
	StateComplete         workflow.State = "complete"
)

const (
	TriggerAuthenticate    workflow.Trigger = "authenticate"
	TriggerCheckRepo       workflow.Trigger = "check_repo"
	TriggerCreateRepo      workflow.Trigger = "create_repo"
	TriggerSaveConfig      workflow.Trigger = "save_config"
	TriggerFetchUserInfo   workflow.Trigger = "fetch_user_info"
	TriggerSetupGPG        workflow.Trigger = "setup_gpg"
	TriggerInitStore       workflow.Trigger = "init_store"
	TriggerCommit          workflow.Trigger = "commit"
	TriggerConfigureRemote workflow.Trigger = "configure_remote"
	TriggerPush            workflow.Trigger = "push"
	TriggerComplete        workflow.Trigger = "complete"
)
