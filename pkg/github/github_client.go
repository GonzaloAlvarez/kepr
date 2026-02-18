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
package github

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	stdio "io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cli/oauth/device"
	"github.com/gonzaloalvarez/kepr/internal/buildflags"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"
)

const defaultGitHubHost = "https://github.com"

func getGitHubHost() string {
	if host := os.Getenv("GITHUB_HOST"); host != "" {
		slog.Debug("using custom GITHUB_HOST", "host", host)
		return strings.TrimSuffix(host, "/")
	}
	return defaultGitHubHost
}

type GitHubClient struct {
	client *github.Client
	ctx    context.Context
}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		ctx: context.Background(),
	}
}

func (c *GitHubClient) CodeBasedAuthentication(clientID string, io cout.IO) (string, error) {
	httpClient := &http.Client{}
	scopes := []string{"repo", "read:org", "user:email"}

	if buildflags.IsDev() {
		scopes = append(scopes, "delete_repo")
	}

	host := getGitHubHost()
	deviceCodeURL := host + "/login/device/code"
	tokenURL := host + "/login/oauth/access_token"

	slog.Debug("requesting device code", "scopes", scopes, "url", deviceCodeURL)
	code, err := device.RequestCode(httpClient, deviceCodeURL, clientID, scopes)
	if err != nil {
		slog.Error("failed to request device code", "error", err)
		return "", err
	}
	io.Infofln("Please visit: %s", code.VerificationURI)
	io.Infofln("Enter code: %s", code.UserCode)

	slog.Debug("waiting for user authentication", "tokenURL", tokenURL)
	accessToken, err := device.Wait(c.ctx, httpClient, tokenURL, device.WaitOptions{
		ClientID:   clientID,
		DeviceCode: code,
	})
	if err != nil {
		slog.Error("authentication failed", "error", err)
		return "", err
	}

	slog.Debug("authentication successful")
	return accessToken.Token, nil
}

func generateCodeVerifier() (string, error) {
	b := make([]byte, 64)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func generateCodeChallenge(verifier string) string {
	h := sha256.New()
	h.Write([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

type callbackServer struct {
	server    *http.Server
	port      int
	codeChan  chan string
	stateChan chan string
	errChan   chan error
}

func startCallbackServer() (*callbackServer, error) {
	cs := &callbackServer{
		codeChan:  make(chan string, 1),
		stateChan: make(chan string, 1),
		errChan:   make(chan error, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" {
			cs.errChan <- fmt.Errorf("authorization failed: no code received")
			w.WriteHeader(http.StatusBadRequest)
			stdio.WriteString(w, "<html><body><h1>Authorization Failed</h1><p>No authorization code received.</p></body></html>")
			return
		}

		if state == "" {
			cs.errChan <- fmt.Errorf("authorization failed: no state received")
			w.WriteHeader(http.StatusBadRequest)
			stdio.WriteString(w, "<html><body><h1>Authorization Failed</h1><p>No state parameter received.</p></body></html>")
			return
		}

		cs.codeChan <- code
		cs.stateChan <- state

		w.WriteHeader(http.StatusOK)
		stdio.WriteString(w, "<html><body><h1>Authorization Successful</h1><p>You can close this window and return to your terminal.</p></body></html>")
	})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}

	cs.port = listener.Addr().(*net.TCPAddr).Port
	cs.server = &http.Server{Handler: mux}

	go func() {
		if err := cs.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("callback server error", "error", err)
		}
	}()

	return cs, nil
}

func (cs *callbackServer) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cs.server.Shutdown(ctx); err != nil {
		cs.server.Close()
		return err
	}

	close(cs.codeChan)
	close(cs.stateChan)
	close(cs.errChan)

	return nil
}

func (c *GitHubClient) PKCEAuthentication(clientID, clientSecret string, io cout.IO) (string, error) {
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		slog.Error("failed to generate code verifier", "error", err)
		return "", fmt.Errorf("failed to generate code verifier: %w", err)
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	state, err := generateState()
	if err != nil {
		slog.Error("failed to generate state", "error", err)
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	scopes := []string{"repo", "read:org", "user:email"}
	if buildflags.IsDev() {
		scopes = append(scopes, "delete_repo")
	}
	slog.Debug("requesting authorization", "scopes", scopes)

	cs, err := startCallbackServer()
	if err != nil {
		slog.Error("failed to start callback server", "error", err)
		return "", err
	}
	defer cs.shutdown()

	slog.Debug("callback server started", "port", cs.port)

	params := url.Values{}
	params.Add("client_id", clientID)
	params.Add("redirect_uri", fmt.Sprintf("http://127.0.0.1:%d/callback", cs.port))
	params.Add("response_type", "code")
	params.Add("scope", strings.Join(scopes, " "))
	params.Add("code_challenge", codeChallenge)
	params.Add("code_challenge_method", "S256")
	params.Add("state", state)

	host := getGitHubHost()
	authURL := fmt.Sprintf("%s/login/oauth/authorize?%s", host, params.Encode())

	io.Infofln("Please visit the following URL to authorize:")
	io.Infofln("%s", authURL)

	slog.Debug("waiting for user authorization")

	ctx, cancel := context.WithTimeout(c.ctx, 2*time.Minute)
	defer cancel()

	var code, receivedState string

	select {
	case code = <-cs.codeChan:
		receivedState = <-cs.stateChan
	case err := <-cs.errChan:
		slog.Error("authorization failed", "error", err)
		return "", err
	case <-ctx.Done():
		slog.Error("authentication timed out")
		return "", fmt.Errorf("authentication timed out after 2 minutes")
	}

	if receivedState != state {
		slog.Error("state mismatch", "expected", state, "received", receivedState)
		return "", fmt.Errorf("state mismatch: possible CSRF attack")
	}

	slog.Debug("state validated successfully")

	tokenParams := url.Values{}
	tokenParams.Add("client_id", clientID)
	tokenParams.Add("client_secret", clientSecret)
	tokenParams.Add("code", code)
	tokenParams.Add("redirect_uri", fmt.Sprintf("http://127.0.0.1:%d/callback", cs.port))
	tokenParams.Add("code_verifier", codeVerifier)

	tokenURL := host + "/login/oauth/access_token"
	req, err := http.NewRequestWithContext(c.ctx, "POST", tokenURL, strings.NewReader(tokenParams.Encode()))
	if err != nil {
		slog.Error("failed to create token request", "error", err)
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		slog.Error("failed to exchange code for token", "error", err)
		return "", fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := stdio.ReadAll(resp.Body)
		slog.Error("token exchange failed", "status", resp.StatusCode, "body", string(body))
		return "", fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		slog.Error("failed to decode token response", "error", err)
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		slog.Error("token exchange error", "error", tokenResp.Error, "description", tokenResp.ErrorDesc)
		return "", fmt.Errorf("token exchange error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	if tokenResp.AccessToken == "" {
		slog.Error("no access token in response")
		return "", fmt.Errorf("no access token in response")
	}

	slog.Debug("authentication successful")
	return tokenResp.AccessToken, nil
}

func (c *GitHubClient) SetToken(token string) {
	slog.Debug("creating github client")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(c.ctx, ts)
	c.client = github.NewClient(tc)

	if host := os.Getenv("GITHUB_HOST"); host != "" {
		baseURL, err := url.Parse(strings.TrimSuffix(host, "/") + "/")
		if err == nil {
			c.client.BaseURL = baseURL
			slog.Debug("configured custom GitHub API base URL", "url", baseURL.String())
		} else {
			slog.Warn("failed to parse GITHUB_HOST as URL", "host", host, "error", err)
		}
	}
}

func (c *GitHubClient) GetUserIdentity() (string, string, error) {
	slog.Debug("fetching user profile from GitHub")
	user, _, err := c.client.Users.Get(c.ctx, "")
	if err != nil {
		slog.Error("failed to fetch user profile", "error", err)
		return "", "", err
	}

	name := ""
	if user.Name != nil {
		name = *user.Name
	}

	email := ""
	if user.Email != nil && *user.Email != "" {
		email = *user.Email
	} else {
		slog.Debug("email not found in profile, fetching from email list")
		emails, _, err := c.client.Users.ListEmails(c.ctx, nil)
		if err != nil {
			slog.Error("failed to fetch user emails", "error", err)
			return name, "", err
		}

		for _, e := range emails {
			if e.Primary != nil && *e.Primary && e.Verified != nil && *e.Verified {
				email = *e.Email
				slog.Debug("found primary verified email", "email", email)
				break
			}
		}

		if email == "" && len(emails) > 0 {
			for _, e := range emails {
				if e.Verified != nil && *e.Verified {
					email = *e.Email
					slog.Debug("found verified email", "email", email)
					break
				}
			}
		}

		if email == "" {
			return name, "", fmt.Errorf("no verified email found in GitHub account")
		}
	}

	slog.Debug("user identity fetched", "name", name, "email", email)
	return name, email, nil
}

func (c *GitHubClient) GetCurrentUserLogin() (string, error) {
	slog.Debug("fetching current user login from GitHub")
	user, _, err := c.client.Users.Get(c.ctx, "")
	if err != nil {
		slog.Error("failed to fetch current user", "error", err)
		return "", fmt.Errorf("failed to fetch current user: %w", err)
	}
	if user.Login == nil || *user.Login == "" {
		return "", fmt.Errorf("user login not found")
	}
	slog.Debug("current user login fetched", "login", *user.Login)
	return *user.Login, nil
}
