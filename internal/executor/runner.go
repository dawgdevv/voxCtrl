package executor

import (
	"log"

	"github.com/dawgdevv/voxctrl/internal/tray"
)

type Runner struct {
	tray *tray.Tray
}

func NewRunner(t *tray.Tray) *Runner {
	return &Runner{tray: t}
}

func (r *Runner) Run(action Action) error {
	// Preflight: validate the action before executing.
	if err := action.Validate(); err != nil {
		log.Printf("[Runner] Preflight failed for %q: %v", action.Name(), err)
		r.tray.Error(err.Error())
		return err
	}

	err := action.Execute()
	if err != nil {
		log.Printf("[Runner] Error executing action %q: %v", action.Name(), err)
		r.tray.Error(action.Name() + ": " + err.Error())
		return err
	}

	log.Printf("[Runner] Successfully executed action %q", action.Name())
	r.tray.Success(action.Name())
	return nil
}
