package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDir_ReturnsKeprSubdirectory(t *testing.T) {
	oldKeprHome := os.Getenv("KEPR_HOME")
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()
	os.Unsetenv("KEPR_HOME")

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

func TestDir_UsesKeprHomeEnv(t *testing.T) {
	tempDir := t.TempDir()
	keprHome := filepath.Join(tempDir, "custom-kepr")

	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()

	dir, err := Dir()
	if err != nil {
		t.Fatalf("Dir() failed: %v", err)
	}

	if dir != keprHome {
		t.Errorf("expected Dir() to return KEPR_HOME value %q, got %q", keprHome, dir)
	}
}

func TestEnsureConfigDir_CreatesDirectory(t *testing.T) {
	tempDir := t.TempDir()
	keprHome := filepath.Join(tempDir, "kepr")

	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()

	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("EnsureConfigDir() failed: %v", err)
	}

	info, err := os.Stat(keprHome)
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
	tempDir := t.TempDir()
	keprHome := filepath.Join(tempDir, "kepr")

	oldKeprHome := os.Getenv("KEPR_HOME")
	os.Setenv("KEPR_HOME", keprHome)
	defer func() {
		if oldKeprHome != "" {
			os.Setenv("KEPR_HOME", oldKeprHome)
		} else {
			os.Unsetenv("KEPR_HOME")
		}
	}()

	err := EnsureConfigDir()
	if err != nil {
		t.Fatalf("first EnsureConfigDir() failed: %v", err)
	}

	err = EnsureConfigDir()
	if err != nil {
		t.Fatalf("second EnsureConfigDir() failed: %v", err)
	}
}

func TestCheckDependencies(t *testing.T) {
	err := CheckDependencies()
	if err != nil {
		t.Logf("CheckDependencies() returned error (may be expected if gpg or git not installed): %v", err)
	}
}
