package backpressure

import "time"

type Backpressure struct {
	notBefore time.Time
}

func (g *Backpressure) NotBefore(t time.Time) {
	g.notBefore = t
}

func (g *Backpressure) CanProceed() bool {
	return time.Now().After(g.notBefore)
}

func (g *Backpressure) RetryAfter() time.Time {
	return g.notBefore
}
