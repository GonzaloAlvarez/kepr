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
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/gpg"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

func SetupGPG(executor shell.Executor, io cout.IO) error {
	configDir, err := config.Dir()
	if err != nil {
		return err
	}

	g, err := gpg.New(configDir, executor, io)
	if err != nil {
		return err
	}

	if err := g.WriteConfigs(); err != nil {
		return fmt.Errorf("failed to write GPG configs: %w", err)
	}

	io.Successfln("GPG environment initialized at %s", g.HomeDir)

	fingerprint := config.GetUserFingerprint()
	if fingerprint == "" {
		slog.Debug("no fingerprint found, generating keys")
		userName := config.GetUserName()
		userEmail := config.GetUserEmail()

		if userName == "" || userEmail == "" {
			return fmt.Errorf("user identity not configured")
		}

		fingerprint, err = g.GenerateKeys(userName, userEmail)
		if err != nil {
			return fmt.Errorf("failed to generate keys: %w", err)
		}

		io.Successfln("Generated Identity: %s", fingerprint)

		if err := config.SaveFingerprint(fingerprint); err != nil {
			return fmt.Errorf("failed to save fingerprint: %w", err)
		}

		if err := g.BackupMasterKey(fingerprint); err != nil {
			return fmt.Errorf("failed to process master key: %w", err)
		}

		if err := initYubikey(g, io); err != nil {
			return err
		}

	} else {
		slog.Debug("fingerprint already exists", "fingerprint", fingerprint)
		io.Infofln("Using existing GPG key: %s", fingerprint)
	}

	return nil
}

func initYubikey(g *gpg.GPG, io cout.IO) error {
	slog.Debug("checking for YubiKey")

	if err := g.ReplaceSCDaemonConf(); err != nil {
		slog.Debug("failed to replace scdaemon.conf", "error", err)
	}
	defer func() {
		if err := g.RevertSCDaemonConf(); err != nil {
			slog.Debug("failed to revert scdaemon.conf", "error", err)
		}
	}()

	y := gpg.NewYubikey(g)
	y.KillSCDaemon()

	err := y.CheckCardPresent()
	if err != nil {
		slog.Debug("no YubiKey detected, prompting user")
		retry, confirmErr := io.Confirm("No YubiKey detected. Please insert your YubiKey and retry?")
		if confirmErr != nil {
			return fmt.Errorf("failed to get user confirmation: %w", confirmErr)
		}

		if !retry {
			io.Warning("YubiKey is required to continue. Exiting.")
			return fmt.Errorf("YubiKey not detected and user chose not to retry")
		}

		err = y.CheckCardPresent()
		if err != nil {
			io.Warning("YubiKey still not detected. Exiting.")
			return fmt.Errorf("YubiKey not detected after retry")
		}
	}

	userName := config.GetUserName()
	userEmail := config.GetUserEmail()
	fingerprint := config.GetUserFingerprint()

	if userName == "" || userEmail == "" || fingerprint == "" {
		return fmt.Errorf("user identity or fingerprint not configured")
	}

	err = y.Init(userName, userEmail, fingerprint)
	if err != nil {
		if strings.Contains(err.Error(), "yubikey slots are occupied") {
			io.Warning("WARNING: YubiKey slots are already occupied with existing keys.")
			io.Infoln("This may overwrite existing keys on the YubiKey.")
			proceed, confirmErr := io.Confirm("Do you want to proceed?")
			if confirmErr != nil {
				return fmt.Errorf("failed to get user confirmation: %w", confirmErr)
			}

			if !proceed {
				return fmt.Errorf("user chose not to proceed with occupied YubiKey")
			}
			io.Successfln("YubiKey detected and ready for configuration (Serial: %s).", y.SerialNumber)
		} else {
			return fmt.Errorf("failed to initialize YubiKey: %w", err)
		}
	} else {
		io.Successfln("YubiKey provisioned successfully (Serial: %s).", y.SerialNumber)
	}

	y.KillSCDaemon()
	adminPin := config.GetYubikeyAdminPin()
	if adminPin != "manual" {
		slog.Debug("verifying yubikey user pin")
		err := y.VerifyUserPin()
		if errors.Is(err, gpg.ErrBadPIN) {
			slog.Debug("default user pin failed, setting to manual")
			config.SaveYubikeyUserPin("manual")
		} else if err == nil {
			slog.Debug("default user pin verified, saving to config")
			config.SaveYubikeyUserPin("123456")
		} else {
			slog.Debug("user pin verification failed with unexpected error", "error", err)
		}
	}

	return nil
}
