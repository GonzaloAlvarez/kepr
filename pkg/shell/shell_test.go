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
package shell

import (
	"bytes"
	"strings"
	"testing"
)

func TestSystemExecutor_LookPath(t *testing.T) {
	executor := &SystemExecutor{}

	path, err := executor.LookPath("ls")
	if err != nil {
		t.Skipf("ls not found in PATH: %v", err)
	}
	if path == "" {
		t.Error("LookPath(\"ls\") returned empty path")
	}
}

func TestSystemExecutor_LookPath_NotFound(t *testing.T) {
	executor := &SystemExecutor{}

	_, err := executor.LookPath("nonexistent-command-12345")
	if err == nil {
		t.Error("LookPath() with nonexistent command should return error")
	}
}

func TestSystemExecutor_Command(t *testing.T) {
	executor := &SystemExecutor{}

	cmd := executor.Command("echo", "hello")
	if cmd == nil {
		t.Fatal("Command() returned nil")
	}
}

func TestSystemCmd_SetDir(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("pwd")

	cmd.SetDir("/tmp")

	sysCmd, ok := cmd.(*SystemCmd)
	if !ok {
		t.Fatal("Command() did not return *SystemCmd")
	}
	if sysCmd.cmd.Dir != "/tmp" {
		t.Errorf("SetDir() Dir = %q, want \"/tmp\"", sysCmd.cmd.Dir)
	}
}

func TestSystemCmd_SetEnv(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("env")

	env := []string{"FOO=bar", "BAZ=qux"}
	cmd.SetEnv(env)

	sysCmd := cmd.(*SystemCmd)
	if len(sysCmd.cmd.Env) != 2 {
		t.Errorf("SetEnv() Env length = %d, want 2", len(sysCmd.cmd.Env))
	}
}

func TestSystemCmd_GetEnv(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("env")

	cmd.SetEnv([]string{"FOO=bar", "BAZ=qux", "EMPTY="})

	if val := cmd.GetEnv("FOO"); val != "bar" {
		t.Errorf("GetEnv(\"FOO\") = %q, want \"bar\"", val)
	}
	if val := cmd.GetEnv("BAZ"); val != "qux" {
		t.Errorf("GetEnv(\"BAZ\") = %q, want \"qux\"", val)
	}
	if val := cmd.GetEnv("EMPTY"); val != "" {
		t.Errorf("GetEnv(\"EMPTY\") = %q, want empty", val)
	}
	if val := cmd.GetEnv("NOTSET"); val != "" {
		t.Errorf("GetEnv(\"NOTSET\") = %q, want empty", val)
	}
}

func TestSystemCmd_SetStdin(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("cat")

	input := bytes.NewBufferString("test input")
	cmd.SetStdin(input)

	sysCmd := cmd.(*SystemCmd)
	if sysCmd.cmd.Stdin == nil {
		t.Error("SetStdin() Stdin is nil")
	}
}

func TestSystemCmd_SetStdout(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("echo", "test")

	var buf bytes.Buffer
	cmd.SetStdout(&buf)

	sysCmd := cmd.(*SystemCmd)
	if sysCmd.cmd.Stdout == nil {
		t.Error("SetStdout() Stdout is nil")
	}
}

func TestSystemCmd_SetStderr(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("ls")

	var buf bytes.Buffer
	cmd.SetStderr(&buf)

	sysCmd := cmd.(*SystemCmd)
	if sysCmd.cmd.Stderr == nil {
		t.Error("SetStderr() Stderr is nil")
	}
}

func TestSystemCmd_Output(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("echo", "hello")

	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Output() returned error: %v", err)
	}

	if !strings.Contains(string(output), "hello") {
		t.Errorf("Output() = %q, want to contain \"hello\"", string(output))
	}
}

func TestSystemCmd_Run(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("true")

	err := cmd.Run()
	if err != nil {
		t.Errorf("Run() returned error: %v", err)
	}
}

func TestSystemCmd_Run_Failure(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("false")

	err := cmd.Run()
	if err == nil {
		t.Error("Run() with failing command should return error")
	}
}

func TestSystemCmd_CombinedOutput(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("echo", "combined")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("CombinedOutput() returned error: %v", err)
	}

	if !strings.Contains(string(output), "combined") {
		t.Errorf("CombinedOutput() = %q, want to contain \"combined\"", string(output))
	}
}

func TestSystemCmd_StartWait(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("echo", "async")

	err := cmd.Start()
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	err = cmd.Wait()
	if err != nil {
		t.Fatalf("Wait() returned error: %v", err)
	}
}

func TestSystemCmd_SetExtraFiles(t *testing.T) {
	executor := &SystemExecutor{}
	cmd := executor.Command("ls")

	cmd.SetExtraFiles(nil)

	sysCmd := cmd.(*SystemCmd)
	if sysCmd.cmd.ExtraFiles != nil {
		t.Error("SetExtraFiles(nil) should set ExtraFiles to nil")
	}
}
