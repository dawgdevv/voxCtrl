package intent

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/dawgdevv/voxctrl/internal/executor"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

type Parser struct {
	registry *Registry
}

func NewParser(r *Registry) *Parser {
	return &Parser{registry: r}
}

// normalize strips punctuation and lowercases a string so that
// spoken transcripts like "Open Spotify." match "open spotify".
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (p *Parser) Resolve(transcription string) (executor.Action, float64, error) {
	input := normalize(transcription)
	if input == "" {
		return nil, 0, fmt.Errorf("empty transcript")
	}

	var bestAction executor.Action
	bestScore := 0.0

	for _, cmd := range p.registry.Commands {
		targets := append([]string{normalize(cmd.Name)}, cmd.Aliases...)
		for _, target := range targets {
			target = normalize(target)
			if target == "" {
				continue
			}

			if fuzzy.Match(input, target) || fuzzy.Match(target, input) {
				score := matchScore(input, target)
				if score > bestScore {
					bestScore = score
					bestAction = executor.NewShellAction(cmd.Name, cmd.Exec)
				}
			}
		}
	}

	if bestAction == nil {
		return nil, 0, fmt.Errorf("no match")
	}
	return bestAction, bestScore, nil
}

// matchScore returns a confidence between 0 and 1.
// Exact match after normalization = 1.0.
func matchScore(a, b string) float64 {
	if a == b {
		return 1.0
	}
	// Use the longer string as the denominator so the score
	// reflects how much of the longer text is covered.
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	if maxLen == 0 {
		return 0
	}
	// Simple overlap: number of matching runes in order.
	// This is a rough proxy; fuzzysearch doesn't expose a ratio,
	// so we approximate with length ratio for substring matches.
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}
	return float64(minLen) / float64(maxLen)
}
