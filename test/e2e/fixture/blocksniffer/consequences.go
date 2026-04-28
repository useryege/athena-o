package blocksniffer

import (
	blocksnifferpb "github.com/useryege/athena/pkg/apiclient/blocksniffer"
)

// Consequences implements the "then" part of given/when/then.
type Consequences struct {
	context *Context
	actions *Actions
}

func (c *Consequences) SetNodeGrpcURLResult(block func(response *blocksnifferpb.SetNodeGrpcURLResponse, err error)) *Consequences {
	c.context.T().Helper()
	block(c.actions.lastSetResult, c.actions.lastError)
	return c
}

func (c *Consequences) GetNodeGrpcURLResult(block func(response *blocksnifferpb.GetNodeGrpcURLResponse, err error)) *Consequences {
	c.context.T().Helper()
	block(c.actions.lastGetResult, c.actions.lastError)
	return c
}

func (c *Consequences) Given() *Context {
	return c.context
}

func (c *Consequences) When() *Actions {
	return c.actions
}
