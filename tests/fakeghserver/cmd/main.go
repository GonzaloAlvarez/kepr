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
package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/gonzaloalvarez/kepr/tests/fakeghserver"
)

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	reposDir := flag.String("repos-dir", "", "Directory for bare git repos (default: temp dir)")
	readyFile := flag.String("ready-file", "", "File to write when server is ready")
	userName := flag.String("user-name", "Test User", "User name for mock identity")
	userEmail := flag.String("user-email", "test@example.com", "User email for mock identity")
	userLogin := flag.String("user-login", "testuser", "User login for mock identity")
	token := flag.String("token", "fake-github-token-12345", "OAuth token to return")
	debug := flag.Bool("debug", false, "Enable debug logging")

	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))

	cfg := fakeghserver.Config{
		Port:      *port,
		ReposDir:  *reposDir,
		ReadyFile: *readyFile,
		UserName:  *userName,
		UserEmail: *userEmail,
		UserLogin: *userLogin,
		Token:     *token,
	}

	server := fakeghserver.New(cfg)

	slog.Info("starting fake GitHub server", "port", *port)

	if err := server.RunWithSignalHandler(); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
