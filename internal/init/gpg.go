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
	"fmt"
	"log/slog"

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
	"github.com/spf13/viper"
)

func SetupGPG(executor shell.Executor) error {
	configDir, err := config.Dir()
	if err != nil {
		return err
	}

	g, err := gpg.New(configDir, executor)
	if err != nil {
		return err
	}

	cout.Successfln("GPG environment initialized at %s", g.HomeDir)

	fingerprint := viper.GetString("user_fingerprint")
	if fingerprint == "" {
		slog.Debug("no fingerprint found, generating keys")
		userName := viper.GetString("user_name")
		userEmail := viper.GetString("user_email")

		if userName == "" || userEmail == "" {
			return fmt.Errorf("user identity not configured")
		}

		fingerprint, err = g.GenerateKeys(userName, userEmail)
		if err != nil {
			return fmt.Errorf("failed to generate keys: %w", err)
		}

		cout.Successfln("Generated Identity: %s", fingerprint)

		if err := g.ProcessMasterKey(fingerprint); err != nil {
			return fmt.Errorf("failed to process master key: %w", err)
		}

		if err := config.SaveFingerprint(fingerprint); err != nil {
			return fmt.Errorf("failed to save fingerprint: %w", err)
		}
	} else {
		slog.Debug("fingerprint already exists", "fingerprint", fingerprint)
	}

	return nil
}
