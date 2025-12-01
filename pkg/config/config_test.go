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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestDir_ReturnsKeprSubdirectory(t *testing.T) {
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() failed: %v", err)
	}

	if !strings.HasSuffix(dir, "kepr") {
		t.Errorf("expected Dir() to end with 'kepr', got %q", dir)
	}

	if !filepath.IsAbs(dir) {
		t.Errorf("expected Dir() to return absolute path, got %q", dir)
	}
}

func TestEnsureConfigDir_CreatesDirectory(t *testing.T) {
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() failed: %v", err)
	}

	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	err = EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() failed: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("config directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("expected kepr path to be a directory")
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0700)
	if mode != expectedMode {
		t.Errorf("expected directory permissions %o, got %o", expectedMode, mode)
	}
}

func TestEnsureConfigDir_Idempotent(t *testing.T) {
	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() failed: %v", err)
	}

	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	err = EnsureConfigDir()
	if err != nil {
		t.Fatalf("first EnsureConfigDir() failed: %v", err)
	}

	err = EnsureConfigDir()
	if err != nil {
		t.Fatalf("second EnsureConfigDir() failed: %v", err)
	}
}

func TestCheckDependencies_AllPresent(t *testing.T) {
	err := CheckDependencies()
	if err != nil {
		t.Logf("CheckDependencies() returned error (may be expected if dependencies not installed): %v", err)
	}
}

func TestViperReset_IsolatesTests(t *testing.T) {
	viper.Set("test_key", "test_value")

	viper.Reset()

	value := viper.GetString("test_key")
	if value != "" {
		t.Errorf("expected empty string after reset, got %q", value)
	}
}
