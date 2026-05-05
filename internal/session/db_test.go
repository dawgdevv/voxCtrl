package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "voxctrl")
	t.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	dbPath := filepath.Join(tmpDir, ".local", "share", "voxctrl", "sessions.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("expected db file to be created at %s", dbPath)
	}
}

func TestEntryFinishSuccess(t *testing.T) {
	e := &Entry{Intent: "test", start: time.Now()}
	time.Sleep(10 * time.Millisecond)
	e.Finish(nil)

	if e.Result != "success" {
		t.Errorf("expected result 'success', got %q", e.Result)
	}
	if e.DurationMs < 1 {
		t.Errorf("expected positive duration, got %d", e.DurationMs)
	}
}

func TestEntryFinishError(t *testing.T) {
	e := &Entry{Intent: "test", start: time.Now()}
	e.Finish(os.ErrNotExist)

	if e.Result != "error: file does not exist" {
		t.Errorf("expected error result, got %q", e.Result)
	}
}

func TestStoreSaveAndRetrieve(t *testing.T) {
	tmpDir := filepath.Join(t.TempDir(), "voxctrl")
	t.Setenv("HOME", tmpDir)

	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	e := store.NewEntry("open vscode")
	e.Confidence = 0.95
	e.Executed = "code ."
	e.ActiveWindow = "Terminal"
	e.ContextJson = "{}"
	e.Finish(nil)

	if err := store.Save(e); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	var count int
	err = store.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE intent = ?", "open vscode").Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}
