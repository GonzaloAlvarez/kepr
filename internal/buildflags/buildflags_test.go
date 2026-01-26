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
package buildflags

import "testing"

func TestIsDev_WhenEnvIsDev(t *testing.T) {
	oldEnv := Env
	Env = "dev"
	defer func() { Env = oldEnv }()

	if !IsDev() {
		t.Error("IsDev() should return true when Env is 'dev'")
	}
}

func TestIsDev_WhenEnvIsProd(t *testing.T) {
	oldEnv := Env
	Env = "prod"
	defer func() { Env = oldEnv }()

	if IsDev() {
		t.Error("IsDev() should return false when Env is 'prod'")
	}
}

func TestIsProd_WhenEnvIsProd(t *testing.T) {
	oldEnv := Env
	Env = "prod"
	defer func() { Env = oldEnv }()

	if !IsProd() {
		t.Error("IsProd() should return true when Env is 'prod'")
	}
}

func TestIsProd_WhenEnvIsDev(t *testing.T) {
	oldEnv := Env
	Env = "dev"
	defer func() { Env = oldEnv }()

	if IsProd() {
		t.Error("IsProd() should return false when Env is 'dev'")
	}
}

func TestDefaultValues(t *testing.T) {
	if Version == "" {
		t.Error("Version should have a default value")
	}
	if Commit == "" {
		t.Error("Commit should have a default value")
	}
	if BuildTime == "" {
		t.Error("BuildTime should have a default value")
	}
}
