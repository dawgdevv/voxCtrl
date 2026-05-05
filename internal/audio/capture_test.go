package audio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateWavMissing(t *testing.T) {
	err := validateWav("/nonexistent/file.wav")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestValidateWavTooSmall(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "small.wav")
	if err := os.WriteFile(tmp, []byte("RIFF"), 0644); err != nil {
		t.Fatal(err)
	}
	err := validateWav(tmp)
	if err == nil {
		t.Error("expected error for file too small")
	}
}

func TestValidateWavValid(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "valid.wav")
	// Write 44 bytes (minimum WAV header size) + some data
	data := make([]byte, 100)
	copy(data, []byte("RIFF"))
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatal(err)
	}
	err := validateWav(tmp)
	if err != nil {
		t.Errorf("expected valid wav, got %v", err)
	}
}

func TestNewRecorderDefaults(t *testing.T) {
	r := NewRecorder("")
	if r.device != "default" {
		t.Errorf("expected default device, got %q", r.device)
	}
}

func TestNewRecorderCustom(t *testing.T) {
	r := NewRecorder("hw:1,0")
	if r.device != "hw:1,0" {
		t.Errorf("expected hw:1,0, got %q", r.device)
	}
}
