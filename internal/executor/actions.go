package executor

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

const cmdTimeout = 15 * time.Second

type Action interface {
	Name() string
	Description() string
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
