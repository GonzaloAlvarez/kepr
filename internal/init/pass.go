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

	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/pass"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

func SetupPasswordStore(configDir, gpgHome, fingerprint string, executor shell.Executor, io cout.IO) error {
	slog.Debug("initializing password store")

	p := pass.New(configDir, gpgHome, executor)

	if err := p.Init(fingerprint); err != nil {
		return fmt.Errorf("failed to initialize password store: %w", err)
	}

	io.Successfln("Initialized local secret store at %s", p.SecretsPath)
	return nil
}
