package stt

import (
	"fmt"
	"os/exec"
	"strings"
)

type Whisper struct {
	modelPath string
}

func NewWhisper(modelPath string) *Whisper {
	return &Whisper{modelPath: modelPath}
}

func (w *Whisper) Transcribe(wavPath string) (string, error) {

	out, err := exec.Command("whisper-cli", "-m", w.modelPath, "-f", wavPath, "--no-timestamps", "--language", "en").Output()

	if err != nil {
		return "", fmt.Errorf("whisper-cli: %w", err)
	}

	return cleanTranscript(string(out)), nil
}

func cleanTranscript(raw string) string {

	var lines []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "whisper_") || strings.HasPrefix(line, "system_info") || strings.HasPrefix(line, "main:") {
			continue
		}

		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, ""))
}
