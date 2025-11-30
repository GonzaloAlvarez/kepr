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
	initialize "github.com/gonzaloalvarez/kepr/internal/init"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [username/repo]",
	Short: "Initialize a new kepr repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repo := args[0]

		token, err := initialize.AuthGithub()
		if err != nil {
			return err
		}

		if err := initialize.UserInfo(token); err != nil {
			return err
		}

		cout.Infofln("Initializing kepr for repo: %s", repo)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
