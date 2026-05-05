package intent

import (
	"testing"

	"github.com/dawgdevv/voxctrl/internal/executor"
)

func TestParserResolveExactMatch(t *testing.T) {
	reg := &Registry{
		Commands: []CommandDef{
			{Name: "open vscode", Aliases: []string{"launch vscode"}, Exec: "code ."},
			{Name: "git status", Aliases: []string{}, Exec: "git status"},
		},
	}
	p := NewParser(reg)

	action, confidence, err := p.Resolve("open vscode")
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if action == nil {
		t.Fatal("expected action, got nil")
	}
	if action.Name() != "open vscode" {
		t.Errorf("expected 'open vscode', got %q", action.Name())
	}
	if confidence < 0.75 {
		t.Errorf("expected confidence >= 0.75, got %.2f", confidence)
	}
}

func TestParserResolveAlias(t *testing.T) {
	reg := &Registry{
		Commands: []CommandDef{
			{Name: "open vscode", Aliases: []string{"launch vscode"}, Exec: "code ."},
		},
	}
	p := NewParser(reg)

	action, confidence, err := p.Resolve("launch vscode")
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if action == nil || action.Name() != "open vscode" {
		t.Fatalf("expected 'open vscode' action, got %v", action)
	}
	if confidence < 0.75 {
		t.Errorf("expected confidence >= 0.75, got %.2f", confidence)
	}
}

func TestParserResolveNoMatch(t *testing.T) {
	reg := &Registry{
		Commands: []CommandDef{
			{Name: "open vscode", Aliases: []string{}, Exec: "code ."},
		},
	}
	p := NewParser(reg)

	action, confidence, err := p.Resolve("zzzzzzzzz")
	if err == nil {
		t.Fatal("expected no match error")
	}
	if action != nil {
		t.Errorf("expected nil action, got %v", action)
	}
	if confidence != 0 {
		t.Errorf("expected 0 confidence, got %.2f", confidence)
	}
}

func TestParserResolveCaseInsensitive(t *testing.T) {
	reg := &Registry{
		Commands: []CommandDef{
			{Name: "Open VSCode", Aliases: []string{}, Exec: "code ."},
		},
	}
	p := NewParser(reg)

	action, _, err := p.Resolve("OPEN VSCODE")
	if err != nil {
		t.Fatalf("expected match, got error: %v", err)
	}
	if action == nil || action.Name() != "Open VSCode" {
		t.Fatalf("expected match")
	}
}

func TestParserResolveEmptyInput(t *testing.T) {
	reg := &Registry{Commands: []CommandDef{}}
	p := NewParser(reg)

	_, _, err := p.Resolve("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestParserResolveReturnsShellAction(t *testing.T) {
	reg := &Registry{
		Commands: []CommandDef{
			{Name: "lock screen", Aliases: []string{}, Exec: "loginctl lock-session"},
		},
	}
	p := NewParser(reg)

	action, _, err := p.Resolve("lock screen")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sa, ok := action.(*executor.ShellAction)
	if !ok {
		t.Fatalf("expected *ShellAction, got %T", action)
	}
	if sa.Description() != "Run: loginctl lock-session" {
		t.Errorf("unexpected description: %s", sa.Description())
	}
}
