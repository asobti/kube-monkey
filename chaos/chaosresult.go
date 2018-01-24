package chaos

import "github.com/asobti/kube-monkey/victims"

type ChaosResult struct {
	chaos *Chaos
	err   error
}

func (r *ChaosResult) Victim() victims.Victim {
	return r.chaos.Victim()
}

func (r *ChaosResult) Error() error {
	return r.err
}
