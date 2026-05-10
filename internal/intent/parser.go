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

const confidenceThreshold = 0.82

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

	var threshold int

	switch {
	case maxLen <= 4:
		threshold = 1
	case maxLen <= 8:
		threshold = 2
	case maxLen <= 12:
		threshold = 3
	default:
		threshold = maxLen / 3
	}

	return dist > threshold
}

func tokenMatch(input, target string) float64 {
	inputWords := strings.Fields(input)
	targetWords := strings.Fields(target)
	targetLen := len(targetWords)

	if len(inputWords) > targetLen {
		return 0
	}

	best := 0.0

	for windowSize := targetLen - 1; windowSize <= targetLen+1; windowSize-- {
		if windowSize < 0 {
			continue
		}

		for start := 0; start <= len(inputWords)-windowSize; start++ {
			window := strings.Join(inputWords[start:start+windowSize], " ")
			score := float64(edlib.JaroWinklerSimilarity(window, target))
			if score > best {
				best = score
			}

		}
	}

	full := float64(edlib.JaroWinklerSimilarity(input, target))
	if full > best {
		return full
	}
	return best
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

	if bestAction == nil || bestScore < confidenceThreshold {
		return nil, 0, fmt.Errorf("no match for query: %q", transcription)
	}
	return bestAction, bestScore, nil
}

// matchScore returns a confidence between 0 and 1.
// Exact match after normalization = 1.0.
func matchScore(a, b string) float64 {
	return tokenMatch(a, b)
}
