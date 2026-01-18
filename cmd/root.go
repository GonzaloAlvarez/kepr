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

	"github.com/gonzaloalvarez/kepr/pkg/config"
	"github.com/spf13/cobra"
)

var (
	debugMode    bool
	repoFlag     string
	resolvedRepo string
)

func NewRootCmd(app *App) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "kepr",
		Short:        "Encrypted distributed key-value store backed by Git and YubiKey",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			initLogging()

			slog.Debug("starting kepr")

			if err := config.Init(); err != nil {
				return fmt.Errorf("failed to initialize: %w", err)
			}
			slog.Debug("initialization complete")

			resolvedRepo = resolveRepo()
			slog.Debug("resolved repo", "repo", resolvedRepo)

			return nil
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&debugMode, "debug", "d", false, "enable debug logging")
	rootCmd.PersistentFlags().StringVarP(&repoFlag, "repo", "r", "", "repository to use (owner/repo)")

	rootCmd.AddCommand(NewInitCmd(app))
	rootCmd.AddCommand(NewAddCmd(app))
	rootCmd.AddCommand(NewGetCmd(app))

	return rootCmd
}

func resolveRepo() string {
	if repoFlag != "" {
		return repoFlag
	}
	return config.GetDefaultRepo()
}

func GetResolvedRepo() string {
	return resolvedRepo
}

func RequireRepo() (string, error) {
	if resolvedRepo == "" {
		return "", fmt.Errorf("no repository specified. Use -r flag or 'kepr use <repo>' to set default")
	}
	return resolvedRepo, nil
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
