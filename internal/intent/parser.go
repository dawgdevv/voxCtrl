package intent

import (
	"fmt"
	"strings"

	"github.com/dawgdevv/voxctrl/internal/executor"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

type Parser struct {
	registry *Registry
}

func NewParser(r *Registry) *Parser {
	return &Parser{registry: r}
}

func (p *Parser) Resolve(transcription string) (executor.Action, float64, error) {
	input := strings.ToLower(strings.TrimSpace(transcription))
	var bestAction executor.Action
	bestScore := 0.0

	for _, cmd := range p.registry.Commands {
		targets := append([]string{strings.ToLower(cmd.Name)}, cmd.Aliases...)
		for _, target := range targets {
			//Simple Scorer : ratio of matching
			if fuzzy.Match(input, target) {
				score := float64(len(target)) / float64(len(input)+1)
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
