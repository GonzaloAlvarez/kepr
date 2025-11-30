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
package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/spf13/cobra"
)

var debugMode bool

var rootCmd = &cobra.Command{
	Use:          "kepr",
	Short:        "Encrypted distributed key-value store backed by Git and YubiKey",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		initLogging()

		slog.Debug("starting kepr")

		if err := config.EnsureConfigDir(); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		slog.Debug("config directory ensured")

		if err := config.InitViper(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}
		slog.Debug("viper initialized")

		dependencies := []string{"gpg", "git", "gopass"}
		for _, tool := range dependencies {
			if _, err := exec.LookPath(tool); err != nil {
				return fmt.Errorf("missing dependency: %s is not installed or in PATH", tool)
			}
			slog.Debug("dependency check passed", "tool", tool)
		}

		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "enable debug logging")
}

func initLogging() {
	level := slog.LevelWarn
	if debugMode {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler))
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
