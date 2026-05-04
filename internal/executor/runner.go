package executor

import (
	"log"

	"github.com/dawgdevv/voxctrl/internal/notify"
)

type Runner struct {
	notifier *notify.Notifier
}

func NewRunner(n *notify.Notifier) *Runner {
	return &Runner{notifier: n}
}

func (r *Runner) Run(action Action) error {

	err := action.Execute()

	if err != nil {
		log.Printf("[Runner] Error executing action %q: %v", action.Name(), err)
		r.notifier.Error("Failed to execute action: " + action.Name())
		return err
	}

	log.Printf("[Runner] Successfully executed action %q", action.Name())
	r.notifier.Success("Executed action: " + action.Name())
	return nil
}
