package executor

import (
	"strings"
	"testing"
	"time"
)

func TestNewShellAction(t *testing.T) {
	a := NewShellAction("test", "echo hello")
	if a.Name() != "test" {
		t.Errorf("expected name 'test', got %q", a.Name())
	}
	if a.Description() != "Run: echo hello" {
		t.Errorf("unexpected description: %s", a.Description())
	}
}

func TestShellActionExecute(t *testing.T) {
	a := NewShellAction("true", "true")
	if err := a.Execute(); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestShellActionExecuteFailure(t *testing.T) {
	a := NewShellAction("false", "false")
	if err := a.Execute(); err == nil {
		t.Error("expected error for failing command")
	}
}

func TestShellActionUndo(t *testing.T) {
	a := NewShellAction("test", "echo hello")
	if err := a.Undo(); err == nil {
		t.Error("expected undo to return error")
	}
}

func TestShellActionTimeout(t *testing.T) {
	// Sleep longer than the 15s timeout
	a := NewShellAction("sleep", "sleep 20")
	start := time.Now()
	err := a.Execute()
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed > 20*time.Second {
		t.Errorf("should have timed out before 20s, took %v", elapsed)
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout message, got: %v", err)
	}
}

func TestShellActionMissingCommand(t *testing.T) {
	// Use a binary name that is extremely unlikely to exist.
	a := NewShellAction("missing", "this_binary_definitely_does_not_exist_12345")
	err := a.Execute()
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if !strings.Contains(err.Error(), "command not found") {
		t.Errorf("expected 'command not found' in error, got: %v", err)
	}
}

func TestShellActionMissingCommandInPipeline(t *testing.T) {
	// The second command in a pipe is missing.
	a := NewShellAction("pipe-missing", "echo hello | this_binary_definitely_does_not_exist_12345")
	err := a.Execute()
	if err == nil {
		t.Fatal("expected error for missing command in pipeline")
	}
	if !strings.Contains(err.Error(), "command not found") {
		t.Errorf("expected 'command not found' in error, got: %v", err)
	}
	// Make sure the extracted command name is the missing one, not "echo".
	if strings.Contains(err.Error(), "command not found: echo") {
		t.Errorf("should not blame 'echo', got: %v", err)
	}
}

func TestShellActionValidateExisting(t *testing.T) {
	a := NewShellAction("true", "true")
	if err := a.Validate(); err != nil {
		t.Errorf("expected validation to pass for 'true', got %v", err)
	}
}

func TestShellActionValidateBuiltin(t *testing.T) {
	// Builtins like echo should pass validation (they never fail).
	a := NewShellAction("echo-test", "echo hello")
	if err := a.Validate(); err != nil {
		t.Errorf("expected validation to pass for builtin 'echo', got %v", err)
	}
}

func TestShellActionValidateMissing(t *testing.T) {
	a := NewShellAction("missing", "this_binary_definitely_does_not_exist_12345")
	err := a.Validate()
	if err == nil {
		t.Fatal("expected validation to fail for missing binary")
	}
	if !strings.Contains(err.Error(), "command not found") {
		t.Errorf("expected 'command not found' in validation error, got: %v", err)
	}
}

func TestShellActionValidateWithBackgroundAmpersand(t *testing.T) {
	a := NewShellAction("firefox-bg", "firefox &")
	// firefox may or may not be installed — test the structural logic.
	// We just ensure Validate doesn't panic and handles the '&' correctly.
	_ = a.Validate()
}

func TestShellActionValidateWithArgs(t *testing.T) {
	a := NewShellAction("ls-home", "ls -la ~")
	if err := a.Validate(); err != nil {
		t.Errorf("expected validation to pass for 'ls', got %v", err)
	}
}

func TestFirstExternalBinary(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"spotify", "spotify"},
		{"gnome-screenshot -a &", "gnome-screenshot"},
		{"pactl set-sink-volume @DEFAULT_SINK@ +10%", "pactl"},
		{"echo hello | xclip -selection clipboard", ""}, // echo is a builtin
		{"df -h / | tail -1 | awk '{print $1}'", "df"},
		{"", ""},
		{"   ", ""},
		{"&", ""},
		{"cd /tmp && ls", ""}, // cd is a builtin
		{"FOO=bar echo hi", ""}, // echo is a builtin
		{"echo hi; true", ""}, // echo is a builtin
	}

	for _, c := range cases {
		got := firstExternalBinary(c.input)
		if got != c.expected {
			t.Errorf("firstExternalBinary(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}
