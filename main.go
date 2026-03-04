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
package main

import (
	"log/slog"
	"os"

	"github.com/gonzaloalvarez/kepr/cmd"
	"github.com/gonzaloalvarez/kepr/internal/buildflags"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/github"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

func main() {
	slog.Debug("kepr starting", "version", buildflags.Version, "env", buildflags.Env)

	app := &cmd.App{
		Shell:  &shell.SystemExecutor{},
		UI:     cout.NewTerminal(),
		GitHub: github.NewGitHubClient(),
	}

	rootCmd := cmd.NewRootCmd(app)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
