package notify

import "os/exec"

type Notifier struct {
}

func New() *Notifier { return &Notifier{} }

func (n *Notifier) Success(msg string) {

	n.send("VoxCtrl ✓", msg, "normal")

}

func (n *Notifier) Error(msg string) {

	n.send("VoxCtrl ✗", msg, "critical")

}

func (n *Notifier) Info(msg string) {

	n.send("VoxCtrl", msg, "low")

}

func (n *Notifier) send(title, msg, urgency string) {
	// Implementation for sending a notification

	exec.Command("notify-send", "-u", urgency, title, msg).Start()
}
