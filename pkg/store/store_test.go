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
	"strings"
	"testing"
)

func TestNormalizePath_ValidPaths(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"foo", "foo"},
		{"foo/bar", "foo/bar"},
		{"foo/bar/baz", "foo/bar/baz"},
		{"a/b/c/d/e", "a/b/c/d/e"},
		{"my-secret", "my-secret"},
		{"prod/db/password", "prod/db/password"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := NormalizePath(tt.input)
			if err != nil {
				t.Errorf("NormalizePath(%q) returned error: %v", tt.input, err)
			}
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizePath_EmptyPath(t *testing.T) {
	_, err := NormalizePath("")
	if err != ErrEmptyPath {
		t.Errorf("NormalizePath(\"\") = %v, want ErrEmptyPath", err)
	}
}

func TestNormalizePath_AbsolutePath(t *testing.T) {
	_, err := NormalizePath("/foo/bar")
	if err != ErrAbsolutePath {
		t.Errorf("NormalizePath(\"/foo/bar\") = %v, want ErrAbsolutePath", err)
	}
}

func TestNormalizePath_TrailingSlash(t *testing.T) {
	_, err := NormalizePath("foo/bar/")
	if err != ErrTrailingSlash {
		t.Errorf("NormalizePath(\"foo/bar/\") = %v, want ErrTrailingSlash", err)
	}
}

func TestNormalizePath_RelativePath(t *testing.T) {
	tests := []string{
		"..",
		"foo/..",
		"foo/../bar",
		"../foo",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			_, err := NormalizePath(path)
			if err != ErrRelativePath {
				t.Errorf("NormalizePath(%q) = %v, want ErrRelativePath", path, err)
			}
		})
	}
}

func TestNormalizePath_InvalidChars(t *testing.T) {
	_, err := NormalizePath("foo\x00bar")
	if err != ErrInvalidPath {
		t.Errorf("NormalizePath with null char = %v, want ErrInvalidPath", err)
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"foo", []string{"foo"}},
		{"foo/bar", []string{"foo", "bar"}},
		{"foo/bar/baz", []string{"foo", "bar", "baz"}},
		{"a/b/c/d", []string{"a", "b", "c", "d"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := SplitPath(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("SplitPath(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("SplitPath(%q)[%d] = %q, want %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGenerateUUID(t *testing.T) {
	uuid1, err := GenerateUUID()
	if err != nil {
		t.Fatalf("GenerateUUID() returned error: %v", err)
	}

	if len(uuid1) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid1))
	}

	parts := strings.Split(uuid1, "-")
	if len(parts) != 5 {
		t.Errorf("UUID has %d parts, want 5", len(parts))
	}

	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			t.Errorf("UUID part %d length = %d, want %d", i, len(part), expectedLengths[i])
		}
	}

	uuid2, err := GenerateUUID()
	if err != nil {
		t.Fatalf("GenerateUUID() returned error: %v", err)
	}

	if uuid1 == uuid2 {
		t.Error("Two UUIDs should be different")
	}
}

func TestGenerateGitignore(t *testing.T) {
	content := GenerateGitignore()

	if !strings.Contains(content, "!.gpg.id") {
		t.Error("Gitignore should include .gpg.id")
	}

	if !strings.Contains(content, "!*.gpg") {
		t.Error("Gitignore should include *.gpg")
	}

	if !strings.Contains(content, "*") {
		t.Error("Gitignore should ignore all by default")
	}
}

func TestSerializeDeserializeMetadata(t *testing.T) {
	original := &Metadata{
		Path: "test/secret",
		Type: TypePassword,
	}

	data, err := SerializeMetadata(original)
	if err != nil {
		t.Fatalf("SerializeMetadata() returned error: %v", err)
	}

	restored, err := DeserializeMetadata(data)
	if err != nil {
		t.Fatalf("DeserializeMetadata() returned error: %v", err)
	}

	if restored.Path != original.Path {
		t.Errorf("Path = %q, want %q", restored.Path, original.Path)
	}

	if restored.Type != original.Type {
		t.Errorf("Type = %q, want %q", restored.Type, original.Type)
	}
}

func TestDeserializeMetadata_InvalidJSON(t *testing.T) {
	_, err := DeserializeMetadata([]byte("not valid json"))
	if err == nil {
		t.Error("DeserializeMetadata with invalid JSON should return error")
	}
}

func TestMetadataTypes(t *testing.T) {
	if TypeDir != "dir" {
		t.Errorf("TypeDir = %q, want \"dir\"", TypeDir)
	}
	if TypePassword != "password" {
		t.Errorf("TypePassword = %q, want \"password\"", TypePassword)
	}
	if TypeFile != "file" {
		t.Errorf("TypeFile = %q, want \"file\"", TypeFile)
	}
}

func TestNew_NilGPGClient(t *testing.T) {
	_, err := New("/tmp/secrets", nil)
	if err != ErrInvalidGPGClient {
		t.Errorf("New() with nil GPG = %v, want ErrInvalidGPGClient", err)
	}
}

func TestStoreErrors(t *testing.T) {
	if ErrAlreadyInitialized.Error() != "store already initialized" {
		t.Errorf("ErrAlreadyInitialized message incorrect")
	}
	if ErrInvalidGPGClient.Error() != "gpg client cannot be nil" {
		t.Errorf("ErrInvalidGPGClient message incorrect")
	}
	if ErrSecretAlreadyExists.Error() != "secret already exists" {
		t.Errorf("ErrSecretAlreadyExists message incorrect")
	}
	if ErrStoreNotInitialized.Error() != "store not initialized" {
		t.Errorf("ErrStoreNotInitialized message incorrect")
	}
	if ErrSecretNotFound.Error() != "secret not found" {
		t.Errorf("ErrSecretNotFound message incorrect")
	}
}

func TestWriteGpgID(t *testing.T) {
	dir := t.TempDir()
	fingerprints := []string{"FP_AAA", "FP_BBB"}

	err := WriteGpgID(dir, fingerprints)
	if err != nil {
		t.Fatalf("WriteGpgID() returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".gpg.id"))
	if err != nil {
		t.Fatalf("failed to read .gpg.id: %v", err)
	}

	expected := "FP_AAA\nFP_BBB\n"
	if string(data) != expected {
		t.Errorf("WriteGpgID content = %q, want %q", string(data), expected)
	}
}

func TestWriteGpgID_Empty(t *testing.T) {
	dir := t.TempDir()
	err := WriteGpgID(dir, []string{})
	if err == nil {
		t.Error("WriteGpgID with empty fingerprints should return error")
	}
}

func TestReadGpgID(t *testing.T) {
	dir := t.TempDir()
	content := "FP_AAA\nFP_BBB\n"
	if err := os.WriteFile(filepath.Join(dir, ".gpg.id"), []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test .gpg.id: %v", err)
	}

	fps, err := ReadGpgID(dir)
	if err != nil {
		t.Fatalf("ReadGpgID() returned error: %v", err)
	}
	if len(fps) != 2 {
		t.Fatalf("ReadGpgID() returned %d fingerprints, want 2", len(fps))
	}
	if fps[0] != "FP_AAA" || fps[1] != "FP_BBB" {
		t.Errorf("ReadGpgID() = %v, want [FP_AAA FP_BBB]", fps)
	}
}

func TestReadGpgID_Missing(t *testing.T) {
	dir := t.TempDir()
	_, err := ReadGpgID(dir)
	if err == nil {
		t.Error("ReadGpgID on missing .gpg.id should return error")
	}
}

func TestReadGpgID_Empty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gpg.id"), []byte(""), 0600); err != nil {
		t.Fatalf("failed to write test .gpg.id: %v", err)
	}

	_, err := ReadGpgID(dir)
	if err == nil {
		t.Error("ReadGpgID on empty .gpg.id should return error")
	}
}

func TestReadGpgID_WhitespaceOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gpg.id"), []byte("  \n  \n"), 0600); err != nil {
		t.Fatalf("failed to write test .gpg.id: %v", err)
	}

	_, err := ReadGpgID(dir)
	if err == nil {
		t.Error("ReadGpgID on whitespace-only .gpg.id should return error")
	}
}
