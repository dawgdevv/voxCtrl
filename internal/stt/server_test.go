package stt

import (
	"os"
	"strings"
	"testing"
)

func TestParseServerResponse(t *testing.T) {
	jsonData := `{"task":"transcribe","language":"en","duration":1.5,"text":"Open Spotify"}`
	got, err := parseServerResponse([]byte(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "Open Spotify" {
		t.Errorf("expected 'Open Spotify', got %q", got)
	}
}

func TestParseServerResponseEmpty(t *testing.T) {
	jsonData := `{"task":"transcribe","language":"en","duration":1.5,"text":""}`
	got, err := parseServerResponse([]byte(jsonData))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestParseServerResponseInvalidJSON(t *testing.T) {
	_, err := parseServerResponse([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestBuildMultipartBody(t *testing.T) {
	// Create a temp wav file — buildMultipartBody needs a real file on disk.
	f, err := os.CreateTemp("", "test_*.wav")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.Write([]byte("RIFF....WAVE"))
	f.Close()

	body, ct, err := buildMultipartBody(f.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	if !strings.Contains(ct, "multipart/form-data") {
		t.Errorf("expected multipart content-type, got %q", ct)
	}
}
