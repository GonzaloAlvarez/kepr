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
	"log/slog"
	"net/http"

	"github.com/cli/oauth/device"
	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/google/go-github/v67/github"
	"golang.org/x/oauth2"
)

func Authenticate(clientID string) (string, error) {
	httpClient := &http.Client{}
	scopes := []string{"repo", "read:org", "user:email"}

	slog.Debug("requesting device code", "scopes", scopes)
	code, err := device.RequestCode(httpClient, "https://github.com/login/device/code", clientID, scopes)
	if err != nil {
		slog.Error("failed to request device code", "error", err)
		return "", err
	}
	cout.Infofln("Please visit: %s", code.VerificationURI)
	cout.Infofln("Enter code: %s", code.UserCode)

	slog.Debug("waiting for user authentication")
	accessToken, err := device.Wait(context.Background(), httpClient, "https://github.com/login/oauth/access_token", device.WaitOptions{
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

func NewClient(token string) *github.Client {
	slog.Debug("creating github client")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	return github.NewClient(tc)
}
