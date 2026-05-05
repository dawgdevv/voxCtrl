package executor

import (
	"errors"
	"testing"

	"github.com/dawgdevv/voxctrl/internal/notify"
)

type mockAction struct {
	name    string
	execute func() error
}

func (m *mockAction) Name() string        { return m.name }
func (m *mockAction) Description() string { return "mock" }
func (m *mockAction) Execute() error      { return m.execute() }
func (m *mockAction) Undo() error         { return nil }

func TestRunnerRunSuccess(t *testing.T) {
	r := NewRunner(notify.New())
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

func TestRunnerRunFailure(t *testing.T) {
	r := NewRunner(notify.New())
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
