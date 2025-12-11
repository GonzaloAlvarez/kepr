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
	"fmt"
	"log/slog"
	"net/http"

	"github.com/cli/oauth/device"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"
)

type GitHubClient struct {
	client *github.Client
	ctx    context.Context
}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		ctx: context.Background(),
	}
}

func (c *GitHubClient) Authenticate(clientID string, io cout.IO) (string, error) {
	httpClient := &http.Client{}
	scopes := []string{"repo", "read:org", "user:email"}

	slog.Debug("requesting device code", "scopes", scopes)
	code, err := device.RequestCode(httpClient, "https://github.com/login/device/code", clientID, scopes)
	if err != nil {
		slog.Error("failed to request device code", "error", err)
		return "", err
	}
	io.Infofln("Please visit: %s", code.VerificationURI)
	io.Infofln("Enter code: %s", code.UserCode)

	slog.Debug("waiting for user authentication")
	accessToken, err := device.Wait(c.ctx, httpClient, "https://github.com/login/oauth/access_token", device.WaitOptions{
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

func (c *GitHubClient) SetToken(token string) {
	slog.Debug("creating github client")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(c.ctx, ts)
	c.client = github.NewClient(tc)
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

func (c *GitHubClient) EnsureRepo(name string, private bool) error {
	slog.Debug("checking if repository exists", "name", name)

	owner, _, err := c.client.Users.Get(c.ctx, "")
	if err != nil {
		slog.Error("failed to get current user", "error", err)
		return fmt.Errorf("failed to get current user: %w", err)
	}

	ownerLogin := ""
	if owner.Login != nil {
		ownerLogin = *owner.Login
	} else {
		return fmt.Errorf("user login not found")
	}

	_, resp, err := c.client.Repositories.Get(c.ctx, ownerLogin, name)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			slog.Debug("repository does not exist, creating", "name", name)
			repo := &github.Repository{
				Name:    github.String(name),
				Private: github.Bool(private),
			}
			_, _, err := c.client.Repositories.Create(c.ctx, "", repo)
			if err != nil {
				slog.Error("failed to create repository", "error", err)
				return fmt.Errorf("failed to create repository: %w", err)
			}
			slog.Debug("repository created successfully", "name", name)
			return nil
		}
		slog.Error("failed to check repository", "error", err)
		return fmt.Errorf("failed to check repository: %w", err)
	}

	slog.Debug("repository already exists", "name", name)
	return nil
}

func (c *GitHubClient) UploadFile(repo string, filePath string, content []byte) error {
	slog.Debug("uploading file to repository", "repo", repo, "file", filePath)

	owner, _, err := c.client.Users.Get(c.ctx, "")
	if err != nil {
		slog.Error("failed to get current user", "error", err)
		return fmt.Errorf("failed to get current user: %w", err)
	}

	ownerLogin := ""
	if owner.Login != nil {
		ownerLogin = *owner.Login
	} else {
		return fmt.Errorf("user login not found")
	}

	fileContent, _, _, err := c.client.Repositories.GetContents(c.ctx, ownerLogin, repo, filePath, nil)

	opts := &github.RepositoryContentFileOptions{
		Message: github.String("Update " + filePath),
		Content: content,
	}

	if err == nil && fileContent != nil {
		opts.SHA = fileContent.SHA
		slog.Debug("file exists, updating", "file", filePath)
	} else {
		slog.Debug("file does not exist, creating", "file", filePath)
	}

	_, _, err = c.client.Repositories.CreateFile(c.ctx, ownerLogin, repo, filePath, opts)
	if err != nil {
		slog.Error("failed to upload file", "error", err)
		return fmt.Errorf("failed to upload file: %w", err)
	}

	slog.Debug("file uploaded successfully", "file", filePath)
	return nil
}
