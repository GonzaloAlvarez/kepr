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
package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanFingerprint_Found(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".gpg.id"), []byte("AAAA\nBBBB\n"), 0600); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, ".gpg.id"), []byte("CCCC\nDDDD\n"), 0600); err != nil {
		t.Fatal(err)
	}

	found, err := ScanFingerprint(tempDir, "CCCC")
	if err != nil {
		t.Fatalf("ScanFingerprint() error: %v", err)
	}
	if !found {
		t.Error("expected fingerprint CCCC to be found")
	}
}

func TestScanFingerprint_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".gpg.id"), []byte("AAAA\nBBBB\n"), 0600); err != nil {
		t.Fatal(err)
	}

	found, err := ScanFingerprint(tempDir, "ZZZZ")
	if err != nil {
		t.Fatalf("ScanFingerprint() error: %v", err)
	}
	if found {
		t.Error("expected fingerprint ZZZZ to not be found")
	}
}

func TestScanFingerprint_RootOnly(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tempDir, ".gpg.id"), []byte("ROOT_FP\n"), 0600); err != nil {
		t.Fatal(err)
	}

	found, err := ScanFingerprint(tempDir, "ROOT_FP")
	if err != nil {
		t.Fatalf("ScanFingerprint() error: %v", err)
	}
	if !found {
		t.Error("expected fingerprint ROOT_FP to be found in root .gpg.id")
	}
}

func TestScanFingerprint_EmptyStore(t *testing.T) {
	tempDir := t.TempDir()

	found, err := ScanFingerprint(tempDir, "ANYTHING")
	if err != nil {
		t.Fatalf("ScanFingerprint() error: %v", err)
	}
	if found {
		t.Error("expected no fingerprints in empty store")
	}
}

func TestScanFingerprint_InvalidPath(t *testing.T) {
	_, err := ScanFingerprint("/nonexistent/path", "FINGERPRINT")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}
