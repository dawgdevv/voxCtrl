package stt

import "testing"

func TestCleanTranscript(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello world", "Hello world"},
		{"  Hello   world  ", "Hello   world"},
		{"whisper_init_from_file_with_params: loading model...\nHello world\n", "Hello world"},
		{"line one\nline two", "line one line two"},
		{"system_info: n_threads = 4\nmain: processing...\nResult here", "Result here"},
		{"", ""},
		{"whisper_log_only\n", ""},
	}

	for _, tt := range tests {
		got := cleanTranscript(tt.input)
		if got != tt.expected {
			t.Errorf("cleanTranscript(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
