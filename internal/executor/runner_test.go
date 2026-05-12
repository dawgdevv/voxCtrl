package executor

import (
	"errors"
	"testing"

	"github.com/dawgdevv/voxctrl/internal/tray"
)

type mockAction struct {
	name    string
	validate func() error
	execute func() error
}

func (m *mockAction) Name() string        { return m.name }
func (m *mockAction) Description() string { return "mock" }
func (m *mockAction) Validate() error     {
	if m.validate != nil {
		return m.validate()
	}
	return nil
}
func (m *mockAction) Execute() error      { return m.execute() }
func (m *mockAction) Undo() error         { return nil }

func TestRunnerRunSuccess(t *testing.T) {
	r := NewRunner(tray.New())
	called := false
	a := &mockAction{
		name: "success-action",
		execute: func() error {
			called = true
			return nil
		},
	}

	err := r.Run(a)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !called {
		t.Error("expected Execute to be called")
	}
}

func TestRunnerRunPreflightFailure(t *testing.T) {
	r := NewRunner(tray.New())
	called := false
	a := &mockAction{
		name: "preflight-fail",
		validate: func() error {
			return errors.New("command not found: spotify")
		},
		execute: func() error {
			called = true
			return nil
		},
	}

	err := r.Run(a)
	if err == nil {
		t.Fatal("expected preflight error")
	}
	if err.Error() != "command not found: spotify" {
		t.Errorf("unexpected error message: %v", err)
	}
	if called {
		t.Error("expected Execute to NOT be called after preflight failure")
	}
}

func TestRunnerRunFailure(t *testing.T) {
	r := NewRunner(tray.New())
	a := &mockAction{
		name: "fail-action",
		execute: func() error {
			return errors.New("boom")
		},
	}

	err := r.Run(a)
	if err == nil {
		t.Error("expected error")
	}
}
