package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestBuildRootCommand(t *testing.T) {
	cmd, err := BuildRootCommand()
	if err != nil {
		t.Fatalf("BuildRootCommand failed: %v", err)
	}

	if cmd.Use != "posture-guard" {
		t.Fatalf("expected Use 'posture-guard', got %q", cmd.Use)
	}
	if cmd.Version != Version {
		t.Fatalf("expected Version %q, got %q", Version, cmd.Version)
	}
}

func TestConfigCommands(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cmd, err := BuildRootCommand()
	if err != nil {
		t.Fatalf("BuildRootCommand failed: %v", err)
	}

	// 1. config set
	cmd.SetArgs([]string{"config", "set", "sensitivity", "high"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	// 2. config get
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetArgs([]string{"config", "get", "sensitivity"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config get failed: %v", err)
	}

	// 3. config reset
	cmd.SetArgs([]string{"config", "reset"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("config reset failed: %v", err)
	}
}

func TestVersionFlagContract(t *testing.T) {
	cmd, err := BuildRootCommand()
	if err != nil {
		t.Fatalf("BuildRootCommand failed: %v", err)
	}

	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("version flag execution failed: %v", err)
	}

	out := strings.TrimSpace(buf.String())
	if out != Version {
		t.Fatalf("expected output exactly %q, got %q", Version, out)
	}
}

func TestCalibrateCommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AGENT", "1")

	cmd, err := BuildRootCommand()
	if err != nil {
		t.Fatalf("BuildRootCommand failed: %v", err)
	}

	cmd.SetArgs([]string{"calibrate", "--frames", "5", "--simulated"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("calibrate command failed: %v", err)
	}
}
