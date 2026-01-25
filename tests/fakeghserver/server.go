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
package fakeghserver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	Port      int
	ReposDir  string
	ReadyFile string
	UserName  string
	UserEmail string
	UserLogin string
	Token     string
}

func DefaultConfig() Config {
	return Config{
		Port:      0,
		UserName:  "Test User",
		UserEmail: "test@example.com",
		UserLogin: "testuser",
		Token:     "fake-github-token-12345",
	}
}

type Server struct {
	config     Config
	server     *http.Server
	listener   net.Listener
	repos      map[string]string
	reposMutex sync.RWMutex
	baseURL    string
}

func New(cfg Config) *Server {
	s := &Server{
		config: cfg,
		repos:  make(map[string]string),
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/login/device/code", s.handleDeviceCode)
	mux.HandleFunc("/login/oauth/access_token", s.handleAccessToken)
	mux.HandleFunc("/login/device", s.handleDevicePage)

	mux.HandleFunc("/user", s.handleUser)
	mux.HandleFunc("/user/emails", s.handleUserEmails)
	mux.HandleFunc("/user/repos", s.handleCreateRepo)
	mux.HandleFunc("/repos/", s.handleRepos)

	s.server = &http.Server{
		Handler: mux,
	}

	return s
}

func (s *Server) Start() error {
	var err error

	addr := fmt.Sprintf("127.0.0.1:%d", s.config.Port)
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	port := s.listener.Addr().(*net.TCPAddr).Port
	s.baseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	slog.Info("fake GitHub server started", "url", s.baseURL)

	if s.config.ReposDir != "" {
		if err := os.MkdirAll(s.config.ReposDir, 0755); err != nil {
			return fmt.Errorf("failed to create repos directory: %w", err)
		}
	}

	if s.config.ReadyFile != "" {
		if err := os.WriteFile(s.config.ReadyFile, []byte(s.baseURL), 0644); err != nil {
			slog.Warn("failed to write ready file", "error", err)
		}
	}

	return s.server.Serve(s.listener)
}

func (s *Server) StartBackground() (string, error) {
	var err error

	addr := fmt.Sprintf("127.0.0.1:%d", s.config.Port)
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("failed to listen: %w", err)
	}

	port := s.listener.Addr().(*net.TCPAddr).Port
	s.baseURL = fmt.Sprintf("http://127.0.0.1:%d", port)
	slog.Info("fake GitHub server started", "url", s.baseURL)

	if s.config.ReposDir == "" {
		s.config.ReposDir, err = os.MkdirTemp("", "fakegh-repos-")
		if err != nil {
			return "", fmt.Errorf("failed to create temp repos directory: %w", err)
		}
	}
	if err := os.MkdirAll(s.config.ReposDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create repos directory: %w", err)
	}

	go func() {
		if err := s.server.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
		}
	}()

	return s.baseURL, nil
}

func (s *Server) URL() string {
	return s.baseURL
}

func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		s.server.Close()
		return err
	}

	if s.config.ReposDir != "" && strings.HasPrefix(s.config.ReposDir, os.TempDir()) {
		os.RemoveAll(s.config.ReposDir)
	}

	return nil
}

func (s *Server) RunWithSignalHandler() error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.Start()
	}()

	select {
	case sig := <-sigChan:
		slog.Info("received signal, shutting down", "signal", sig)
		return s.Close()
	case err := <-errChan:
		return err
	}
}

func (s *Server) handleDeviceCode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slog.Debug("handling device code request")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"device_code":      "fake-device-code-12345",
		"user_code":        "FAKE-1234",
		"verification_uri": s.baseURL + "/login/device",
		"expires_in":       900,
		"interval":         0,
	})
}

func (s *Server) handleAccessToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slog.Debug("handling access token request")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"access_token": s.config.Token,
		"token_type":   "bearer",
		"scope":        "repo,read:org,user:email,delete_repo",
	})
}

func (s *Server) handleDevicePage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<html><body><h1>Fake GitHub Device Authorization</h1>
<p>This is a fake GitHub server for testing.</p>
<p>Authorization is automatic - no action needed.</p>
</body></html>`))
}

func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slog.Debug("handling user request")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"login": s.config.UserLogin,
		"name":  s.config.UserName,
		"email": s.config.UserEmail,
	})
}

func (s *Server) handleUserEmails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slog.Debug("handling user emails request")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode([]map[string]interface{}{
		{
			"email":    s.config.UserEmail,
			"primary":  true,
			"verified": true,
		},
	})
}

func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name    string `json:"name"`
		Private bool   `json:"private"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	slog.Debug("handling create repo request", "name", req.Name)

	repoPath := filepath.Join(s.config.ReposDir, s.config.UserLogin, req.Name+".git")
	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		slog.Error("failed to create repo directory", "error", err)
		http.Error(w, "failed to create repository", http.StatusInternalServerError)
		return
	}

	cmd := exec.Command("git", "init", "--bare", repoPath)
	if err := cmd.Run(); err != nil {
		slog.Error("failed to init bare repo", "error", err)
		http.Error(w, "failed to create repository", http.StatusInternalServerError)
		return
	}

	s.reposMutex.Lock()
	s.repos[s.config.UserLogin+"/"+req.Name] = repoPath
	s.reposMutex.Unlock()

	cloneURL := "file://" + repoPath

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":      req.Name,
		"full_name": s.config.UserLogin + "/" + req.Name,
		"private":   req.Private,
		"clone_url": cloneURL,
		"html_url":  s.baseURL + "/" + s.config.UserLogin + "/" + req.Name,
	})
}

func (s *Server) handleRepos(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/repos/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	owner := parts[0]
	repo := parts[1]
	repoKey := owner + "/" + repo

	if len(parts) >= 3 && parts[2] == "clone-url" {
		s.handleCloneURL(w, r, repoKey)
		return
	}

	if r.Method == http.MethodGet {
		s.reposMutex.RLock()
		repoPath, exists := s.repos[repoKey]
		s.reposMutex.RUnlock()

		if !exists {
			http.Error(w, "repository not found", http.StatusNotFound)
			return
		}

		cloneURL := "file://" + repoPath

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":      repo,
			"full_name": repoKey,
			"clone_url": cloneURL,
		})
		return
	}

	if r.Method == http.MethodDelete {
		s.reposMutex.Lock()
		repoPath, exists := s.repos[repoKey]
		if exists {
			delete(s.repos, repoKey)
			os.RemoveAll(repoPath)
		}
		s.reposMutex.Unlock()

		if !exists {
			http.Error(w, "repository not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleCloneURL(w http.ResponseWriter, r *http.Request, repoKey string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.reposMutex.RLock()
	repoPath, exists := s.repos[repoKey]
	s.reposMutex.RUnlock()

	if !exists {
		http.Error(w, "repository not found", http.StatusNotFound)
		return
	}

	cloneURL := "file://" + repoPath

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"clone_url": cloneURL,
	})
}
