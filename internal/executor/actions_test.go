package executor

import (
	"testing"
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
