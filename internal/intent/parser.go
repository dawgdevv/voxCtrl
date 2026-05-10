package intent

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/dawgdevv/voxctrl/internal/executor"

	"github.com/hbollon/go-edlib"
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
	return strings.Join(strings.Fields(b.String()), " ")
}

func runeLen(s string) int {
	return len([]rune(s))
}

func tooFar(a, b string) bool {
	dist := edlib.DamerauLevenshteinDistance(a, b)
	maxLen := runeLen(a)

	if runeLen(b) > maxLen {
		maxLen = runeLen(b)
	}

	if maxLen == 0 {
		return false
	}
	return dist > maxLen/2
}

func (p *Parser) Resolve(transcription string) (executor.Action, float64, error) {
	input := normalize(transcription)
	if input == "" {
		return nil, 0, fmt.Errorf("empty transcript")
	}

	var bestAction executor.Action
	bestScore := 0.0

	for _, cmd := range p.registry.Commands {
		targets := make([]string, 0, len(cmd.Aliases)+1)
		targets = append(targets, normalize(cmd.Name))
		for _, alias := range cmd.Aliases {
			if n := normalize(alias); n != "" {
				targets = append(targets, n)
			}
		}

		for _, target := range targets {
			if tooFar(input, target) {
				continue
			}
			score := matchScore(input, target)
			if score > bestScore {
				bestScore = score
				bestAction = executor.NewShellAction(cmd.Name, cmd.Exec)
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
	return float64(edlib.JaroWinklerSimilarity(a, b))
}
