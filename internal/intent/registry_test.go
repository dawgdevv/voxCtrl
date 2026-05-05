package intent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "commands.yaml")
	data := `
- name: open vscode
  aliases: ["launch vscode"]
  exec: "code ."
- name: git status
  aliases: []
  exec: "git status"
`
	if err := os.WriteFile(tmp, []byte(data), 0644); err != nil {
		t.Fatalf("write test yaml: %v", err)
	}

	reg, err := NewRegistry(tmp)
	if err != nil {
		t.Fatalf("NewRegistry failed: %v", err)
	}
	if len(reg.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(reg.Commands))
	}
	if reg.Commands[0].Name != "open vscode" {
		t.Errorf("expected first command 'open vscode', got %q", reg.Commands[0].Name)
	}
}

func TestNewRegistryMissingFile(t *testing.T) {
	_, err := NewRegistry("/nonexistent/path/commands.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
