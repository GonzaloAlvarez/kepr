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

func TestListRequests_NoRequestsDir(t *testing.T) {
	tempDir := t.TempDir()

	requests, err := ListRequests(tempDir, nil)
	if err != nil {
		t.Fatalf("ListRequests() error: %v", err)
	}
	if requests != nil {
		t.Errorf("expected nil, got %v", requests)
	}
}

func TestListRequests_EmptyRequestsDir(t *testing.T) {
	tempDir := t.TempDir()
	requestsDir := filepath.Join(tempDir, "requests")
	if err := os.MkdirAll(requestsDir, 0700); err != nil {
		t.Fatal(err)
	}

	requests, err := ListRequests(tempDir, nil)
	if err != nil {
		t.Fatalf("ListRequests() error: %v", err)
	}
	if len(requests) != 0 {
		t.Errorf("expected 0 requests, got %d", len(requests))
	}
}

func TestListRequests_IgnoresNonGpgFiles(t *testing.T) {
	tempDir := t.TempDir()
	requestsDir := filepath.Join(tempDir, "requests")
	if err := os.MkdirAll(requestsDir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(requestsDir, "readme.txt"), []byte("not a request"), 0600); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(requestsDir, "subdir")
	if err := os.MkdirAll(subDir, 0700); err != nil {
		t.Fatal(err)
	}

	requests, err := ListRequests(tempDir, nil)
	if err != nil {
		t.Fatalf("ListRequests() error: %v", err)
	}
	if len(requests) != 0 {
		t.Errorf("expected 0 requests, got %d", len(requests))
	}
}

func TestFindRequestByPrefix_NoMatch(t *testing.T) {
	tempDir := t.TempDir()
	requestsDir := filepath.Join(tempDir, "requests")
	if err := os.MkdirAll(requestsDir, 0700); err != nil {
		t.Fatal(err)
	}

	_, err := FindRequestByPrefix(tempDir, nil, "nonexistent")
	if err == nil {
		t.Fatal("expected error for no matching request")
	}
}
