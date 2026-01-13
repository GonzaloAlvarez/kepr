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
package gpg

import (
	"bufio"
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gonzaloalvarez/kepr/pkg/cout"
	"github.com/gonzaloalvarez/kepr/pkg/shell"
)

type GPG struct {
	BinaryPath         string
	PinentryPath       string
	GPGConfPath        string
	HomeDir            string
	AgentConfigPath    string
	ConfigPath         string
	SCDaemonConfigPath string
	executor           shell.Executor
	io                 cout.IO
}

type GpGRequestLineType int

const (
	line GpGRequestLineType = iota
	hidden
	failure
)

type GPGSession struct {
	StatusMessages <-chan string
	SendInput      chan<- string
	LastInput      GpGRequestLineType
	Done           <-chan error
}

func findPinentry(executor shell.Executor) (string, error) {
	candidates := []string{"pinentry-mac", "pinentry-gnome3", "pinentry", "pinentry-curses"}

	for _, name := range candidates {
		path, err := executor.LookPath(name)
		if err == nil {
			slog.Debug("found pinentry", "program", name, "path", path)
			return path, nil
		}
	}

	return "", fmt.Errorf("no pinentry program found (tried: %v)", candidates)
}

func findGPGConf(executor shell.Executor) string {
	path, err := executor.LookPath("gpgconf")
	if err == nil {
		slog.Debug("found gpgconf", "path", path)
		return path
	}
	slog.Warn("gpgconf not found, some YubiKey operations may fail")
	return ""
}

func New(configBaseDir string, executor shell.Executor, io cout.IO) (*GPG, error) {
	gpgBinary, err := executor.LookPath("gpg")
	if err != nil {
		return nil, fmt.Errorf("gpg binary not found: %w", err)
	}
	slog.Debug("found gpg binary", "path", gpgBinary)

	pinentryBinary, err := findPinentry(executor)
	if err != nil {
		return nil, err
	}

	gpgconfBinary := findGPGConf(executor)

	homeDir := filepath.Join(configBaseDir, "gpg")
	slog.Debug("creating gpg home directory", "path", homeDir)
	if err := os.MkdirAll(homeDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create gpg home directory: %w", err)
	}

	gpg := &GPG{
		BinaryPath:         gpgBinary,
		PinentryPath:       pinentryBinary,
		GPGConfPath:        gpgconfBinary,
		HomeDir:            homeDir,
		AgentConfigPath:    filepath.Join(homeDir, "gpg-agent.conf"),
		ConfigPath:         filepath.Join(homeDir, "gpg.conf"),
		SCDaemonConfigPath: filepath.Join(homeDir, "scdaemon.conf"),
		executor:           executor,
		io:                 io,
	}

	return gpg, nil
}

func (g *GPG) execute(stdin string, args ...string) (string, string, error) {
	cmd := g.executor.Command(g.BinaryPath, args...)
	cmd.SetEnv(append(os.Environ(), fmt.Sprintf("GNUPGHOME=%s", g.HomeDir)))

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SetStdout(&stdoutBuf)
	cmd.SetStderr(&stderrBuf)

	if stdin != "" {
		cmd.SetStdin(bytes.NewBufferString(stdin))
	}

	err := cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

func (g *GPG) executeBytes(stdin []byte, args ...string) ([]byte, string, error) {
	cmd := g.executor.Command(g.BinaryPath, args...)
	cmd.SetEnv(append(os.Environ(), fmt.Sprintf("GNUPGHOME=%s", g.HomeDir)))

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SetStdout(&stdoutBuf)
	cmd.SetStderr(&stderrBuf)

	if len(stdin) > 0 {
		cmd.SetStdin(bytes.NewBuffer(stdin))
	}

	err := cmd.Run()
	return stdoutBuf.Bytes(), stderrBuf.String(), err
}

func (g *GPG) executeWithPinentry(stdin string, args ...string) (string, string, error) {
	cmd := g.executor.Command(g.BinaryPath, args...)
	cmd.SetEnv(append(os.Environ(), fmt.Sprintf("GNUPGHOME=%s", g.HomeDir)))

	pr, pw, err := os.Pipe()
	if err != nil {
		return "", "", fmt.Errorf("failed to create pipe: %w", err)
	}
	defer pr.Close()

	tty := cmd.GetEnv("GPG_TTY")
	if tty == "" {
		if out, err := exec.Command("tty").Output(); err == nil {
			tty = strings.TrimSpace(string(out))
		}
	}

	if tty != "" {
		cmd.SetEnv(append(os.Environ(), fmt.Sprintf("GPG_TTY=%s", tty)))
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SetStdout(&stdoutBuf)
	cmd.SetStderr(&stderrBuf)

	cmd.SetExtraFiles([]*os.File{pr})
	cmd.SetStdin(os.Stdin)

	err = cmd.Start()
	if err != nil {
		return "", "", fmt.Errorf("failed to start command: %w", err)
	}

	go func() {
		pw.Write([]byte(stdin))
		pw.Close()
	}()

	err = cmd.Wait()
	if err != nil {
		return "", "", fmt.Errorf("failed to wait for command: %w", err)
	}

	return stdoutBuf.String(), stderrBuf.String(), err
}

func (g *GPG) executeBytesWithPinentry(stdin []byte, args ...string) ([]byte, string, error) {
	cmd := g.executor.Command(g.BinaryPath, args...)
	cmd.SetEnv(append(os.Environ(), fmt.Sprintf("GNUPGHOME=%s", g.HomeDir)))

	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, "", fmt.Errorf("failed to create pipe: %w", err)
	}
	defer pr.Close()

	tty := cmd.GetEnv("GPG_TTY")
	if tty == "" {
		if out, err := exec.Command("tty").Output(); err == nil {
			tty = strings.TrimSpace(string(out))
		}
	}

	if tty != "" {
		cmd.SetEnv(append(os.Environ(), fmt.Sprintf("GPG_TTY=%s", tty)))
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SetStdout(&stdoutBuf)
	cmd.SetStderr(&stderrBuf)

	cmd.SetExtraFiles([]*os.File{pr})
	cmd.SetStdin(os.Stdin)

	err = cmd.Start()
	if err != nil {
		return nil, "", fmt.Errorf("failed to start command: %w", err)
	}

	go func() {
		pw.Write(stdin)
		pw.Close()
	}()

	err = cmd.Wait()
	if err != nil {
		return nil, "", fmt.Errorf("failed to wait for command: %w", err)
	}

	return stdoutBuf.Bytes(), stderrBuf.String(), err
}

func (g *GPG) ExecuteInteractive(args ...string) (*GPGSession, error) {
	cmd := g.executor.Command(g.BinaryPath, args...)
	cmd.SetEnv(append(os.Environ(), fmt.Sprintf("GNUPGHOME=%s", g.HomeDir)))

	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	statusReader, statusWriter, err := os.Pipe()
	if err != nil {
		stdinReader.Close()
		stdinWriter.Close()
		return nil, fmt.Errorf("failed to create status pipe: %w", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.SetStdin(stdinReader)
	cmd.SetStdout(&stdoutBuf)
	cmd.SetStderr(&stderrBuf)
	cmd.SetExtraFiles([]*os.File{statusWriter})

	statusChan := make(chan string)
	inputChan := make(chan string)
	doneChan := make(chan error, 1)

	if err := cmd.Start(); err != nil {
		stdinReader.Close()
		stdinWriter.Close()
		statusReader.Close()
		statusWriter.Close()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	statusWriter.Close()

	go func() {
		scanner := bufio.NewScanner(statusReader)
		for scanner.Scan() {
			line := scanner.Text()
			slog.Debug("gpg status", "line", line)
			statusChan <- line
		}
		close(statusChan)
		statusReader.Close()
	}()

	go func() {
		writer := bufio.NewWriter(stdinWriter)
		for input := range inputChan {
			slog.Debug("sending to gpg", "input", input)
			writer.WriteString(input + "\n")
			writer.Flush()
		}
		stdinWriter.Close()
		stdinReader.Close()
	}()

	go func() {
		err := cmd.Wait()
		stderr := stderrBuf.String()
		stdout := stdoutBuf.String()

		slog.Debug("gpg command completed", "error", err, "stderr", stderr, "stdout", stdout)

		if err != nil {
			if strings.Contains(stderr, "Bad PIN") {
				doneChan <- ErrBadPIN
			} else {
				doneChan <- fmt.Errorf("gpg command failed: %w", err)
			}
		} else {
			doneChan <- nil
		}
		close(doneChan)
	}()

	return &GPGSession{
		StatusMessages: statusChan,
		SendInput:      inputChan,
		LastInput:      line,
		Done:           doneChan,
	}, nil
}
