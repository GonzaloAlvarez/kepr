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
package config

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/viper"
)

func Dir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "kepr"), nil
}

func Init() error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}
	if err := InitViper(); err != nil {
		return err
	}
	return CheckDependencies()
}

func EnsureConfigDir() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	return os.MkdirAll(dir, 0700)
}

func InitViper() error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	viper.SetConfigName("config")
	viper.SetConfigType("json")
	viper.AddConfigPath(dir)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return nil
}

func CheckDependencies() error {
	dependencies := []string{"gpg", "git", "gopass"}
	for _, tool := range dependencies {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("missing dependency: %s is not installed or in PATH", tool)
		}
		slog.Debug("dependency check passed", "tool", tool)
	}

	pinentryVariants := []string{"pinentry-mac", "pinentry-gnome3", "pinentry", "pinentry-curses"}
	found := false
	for _, variant := range pinentryVariants {
		if _, err := exec.LookPath(variant); err == nil {
			slog.Debug("dependency check passed", "tool", variant)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("missing dependency: no pinentry program found (tried: %v)", pinentryVariants)
	}

	return nil
}

func SaveToken(token string) error {
	viper.Set("github_token", token)
	return writeConfig()
}

func GetToken() string {
	return viper.GetString("github_token")
}

func SaveUserIdentity(name, email string) error {
	viper.Set("user_name", name)
	viper.Set("user_email", email)
	return writeConfig()
}

func SaveFingerprint(fingerprint string) error {
	viper.Set("user_fingerprint", fingerprint)
	return writeConfig()
}

func GetUserName() string {
	return viper.GetString("user_name")
}

func GetUserEmail() string {
	return viper.GetString("user_email")
}

func GetUserFingerprint() string {
	return viper.GetString("user_fingerprint")
}

func SaveYubikeyAdminPin(pin string) error {
	viper.Set("yubikey_admin_pin", pin)
	return writeConfig()
}

func GetYubikeyAdminPin() string {
	return viper.GetString("yubikey_admin_pin")
}

func SaveYubikeyUserPin(pin string) error {
	viper.Set("yubikey_user_pin", pin)
	return writeConfig()
}

func GetYubikeyUserPin() string {
	return viper.GetString("yubikey_user_pin")
}

func writeConfig() error {
	dir, err := Dir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(dir, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := viper.SafeWriteConfig(); err != nil {
			return err
		}
	} else {
		if err := viper.WriteConfig(); err != nil {
			return err
		}
	}

	return os.Chmod(configPath, 0600)
}
