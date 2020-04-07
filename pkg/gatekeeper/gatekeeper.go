package gatekeeper

import "time"

type Gatekeeper struct {
	notBefore time.Time
}

func (g *Gatekeeper) NotBefore(t time.Time) {
	g.notBefore = t
}

func (g *Gatekeeper) CanProceed() bool {
	return time.Now().After(g.notBefore)
}

func (g *Gatekeeper) RetryAfter() time.Time {
	return g.notBefore
}
