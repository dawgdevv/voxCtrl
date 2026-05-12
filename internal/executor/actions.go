package executor

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const cmdTimeout = 15 * time.Second

// commandNotFoundRe matches bash's "command not found" stderr output.
// It handles both simple and "line N:" prefixed variants.
var commandNotFoundRe = regexp.MustCompile(`(?:bash:\s*(?:line\s*\d+:\s*)?)?([^:]+):\s*command not found`)

func extractMissingCommand(stderr string) string {
	matches := commandNotFoundRe.FindStringSubmatch(stderr)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

type Action interface {
	Name() string
	Description() string
	// Validate performs a best-effort preflight check.
	// For shell actions it verifies the first binary token exists in PATH.
	Validate() error
	Execute() error
	Undo() error
}

type ShellAction struct {
	name    string
	command string
}

func NewShellAction(name, command string) *ShellAction {
	return &ShellAction{name: name, command: command}
}

func (a *ShellAction) Name() string        { return a.name }
func (a *ShellAction) Description() string { return fmt.Sprintf("Run: %s", a.command) }
func (a *ShellAction) Undo() error         { return fmt.Errorf("undo not supported for it") }

// Validate checks whether the first external binary referenced by the shell
// command is available in PATH. It is intentionally best-effort: pipes,
// subshells and builtins are ignored, but app-launcher style commands like
// "spotify" or "gnome-screenshot -a &" are caught early.
func (a *ShellAction) Validate() error {
	binary := firstExternalBinary(a.command)
	if binary == "" {
		return nil
	}

	check := exec.Command("bash", "-c", "command -v "+binary)
	if err := check.Run(); err != nil {
		return fmt.Errorf("command not found: %s", binary)
	}
	return nil
}

// firstExternalBinary extracts the first non-builtin, non-assignment command
// token from a shell command string. It returns "" if none is found.
func firstExternalBinary(cmd string) string {
	// Trim background operator and whitespace.
	s := strings.TrimSpace(cmd)
	s = strings.TrimSuffix(s, "&")
	s = strings.TrimSpace(s)

	if s == "" {
		return ""
	}

	// Find the first simple command before any shell metacharacter.
	end := len(s)
	for i, ch := range s {
		if ch == '|' || ch == ';' || ch == '<' || ch == '>' {
			end = i
			break
		}
		// Stop at '&&' or '||' — peek ahead for the second char.
		if ch == '&' && i+1 < len(s) && s[i+1] == '&' {
			end = i
			break
		}
	}
	first := strings.TrimSpace(s[:end])

	// Walk tokens, skipping variable assignments (FOO=bar) and builtins.
	fields := strings.Fields(first)
	for _, tok := range fields {
		if isVarAssignment(tok) {
			continue
		}
		if isBuiltin(tok) {
			return ""
		}
		return tok
	}
	return ""
}

func isVarAssignment(tok string) bool {
	if !strings.Contains(tok, "=") {
		return false
	}
	// Must not contain spaces and the '=' must not be the first char.
	if strings.Contains(tok, " ") {
		return false
	}
	idx := strings.Index(tok, "=")
	return idx > 0
}

func isBuiltin(tok string) bool {
	switch tok {
	case "cd", "echo", "printf", "true", "false", "exit", "source", ".",
		"shift", "unset", "export", "alias", "eval", "exec":
		return true
	}
	return false
}

func (a *ShellAction) Execute() error {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", a.command)
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("command timed out after %s (stdout: %q, stderr: %q)", cmdTimeout, stdout.String(), stderr.String())
	}
	if err != nil {
		// If bash reports "command not found", surface a clean message.
		if missing := extractMissingCommand(stderr.String()); missing != "" {
			return fmt.Errorf("command not found: %s", missing)
		}
		return fmt.Errorf("%w (stdout: %q, stderr: %q)", err, stdout.String(), stderr.String())
	}
	if out := strings.TrimSpace(stdout.String()); out != "" {
		log.Printf("[exec] %s stdout: %s", a.name, out)
	}
	if out := strings.TrimSpace(stderr.String()); out != "" {
		log.Printf("[exec] %s stderr: %s", a.name, out)
	}
	return nil
}
