package chaos

import (
	"github.com/asobti/kube-monkey/victims"
)

type Result struct {
	chaos *Chaos
	err   error
}

func (r *Result) Victim() victims.Victim {
	return r.chaos.Victim()
}

func (r *Result) Error() error {
	return r.err
}

// NewResult creates a new Result instance
func NewResult(chaos *Chaos, err error) *Result {
	return &Result{
		chaos: chaos,
		err:   err,
	}
}
