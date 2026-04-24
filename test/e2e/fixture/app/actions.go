package app

import (
	"time"

	"github.com/useryege/athena/test/e2e/fixture"
)

// this implements the "when" part of given/when/then
//
// none of the func implement error checks, and that is complete intended, you should check for errors
// using the Then()
type Actions struct {
	context    *Context
	lastOutput string
	lastError  error
}

func (a *Actions) And(block func()) *Actions {
	a.context.T().Helper()
	block()
	return a
}

func (a *Actions) Then() *Consequences {
	a.context.T().Helper()
	time.Sleep(fixture.WhenThenSleepInterval)
	return &Consequences{a.context, a}
}
