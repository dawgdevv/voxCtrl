package stt

import (
	"fmt"
	"os/exec"
	"strings"
)

type Whisper struct {
	modelPath string
	threads   string
	beamSize  string
}

func NewWhisper(modelPath string, threads, beamSize int) *Whisper {
	return &Whisper{
		modelPath: modelPath,
		threads:   fmt.Sprintf("%d", threads),
		beamSize:  fmt.Sprintf("%d", beamSize),
	}
}

func (w *Whisper) Transcribe(wavPath string) (string, error) {
	// Greedy + max threads = fastest. Use --single-segment for short utterances.
	out, err := exec.Command("whisper-cli",
		"-m", w.modelPath,
		"-f", wavPath,
		"-t", w.threads,
		"-bs", w.beamSize,
		"--no-timestamps",
		"--language", "en",
		"-sns",
	).Output()

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
	return strings.TrimSpace(strings.Join(lines, " "))
}
