package chaos

import (
	"kube-monkey/internal/pkg/victims"
)

// Result is the outcome of one attempted termination
type Result struct {
	chaos *Chaos
	err   error
}

func (r *Result) Victim() *victims.Victim {
	return r.chaos.Victim()
}

func (r *Result) Error() error {
	return r.err
}
