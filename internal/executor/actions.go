package executor

import (
	"fmt"
	"os/exec"
)

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
	cmd := exec.Command("sh", "-c", a.command)
	return cmd.Run()
}
