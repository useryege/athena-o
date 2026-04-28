package blocksniffer

import (
	"testing"
	"time"

	"github.com/useryege/athena/test/e2e/fixture"
)

// Context implements the "given" part of given/when/then.
type Context struct {
	*fixture.TestState
}

func Given(t *testing.T) *Context {
	t.Helper()
	state := fixture.EnsureCleanState(t)
	return &Context{TestState: state}
}

// GivenWithSameState creates a new Context sharing the same TestState.
func GivenWithSameState(ctx fixture.TestContext) *Context {
	ctx.T().Helper()
	return &Context{TestState: fixture.NewTestStateFromContext(ctx)}
}

func (c *Context) When() *Actions {
	time.Sleep(fixture.WhenThenSleepInterval)
	return &Actions{context: c}
}
