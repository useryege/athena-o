package blocksniffer

import (
	"context"
	"time"

	"github.com/stretchr/testify/require"
	blocksnifferpb "github.com/useryege/athena/pkg/apiclient/blocksniffer"
	"github.com/useryege/athena/test/e2e/fixture"
	utilio "github.com/useryege/athena/util/io"
)

// Actions implements the "when" part of given/when/then.
type Actions struct {
	context *Context
}

type SetEvmNodeWsURLResult struct {
	context  *Context
	response *blocksnifferpb.SetEvmNodeWsURLResponse
	err      error
}

func (a *Actions) SetEvmNodeWsURL(evmNodeWsURL string) *SetEvmNodeWsURLResult {
	a.context.T().Helper()

	closer, client, err := fixture.AthenaClientset.NewBlockSnifferClient()
	require.NoError(a.context.T(), err)
	defer utilio.Close(closer)

	resp, err := client.SetEvmNodeWsURL(context.Background(), &blocksnifferpb.SetEvmNodeWsURLRequest{
		EvmNodeWsURL: evmNodeWsURL,
	})
	return &SetEvmNodeWsURLResult{
		context:  a.context,
		response: resp,
		err:      err,
	}
}

func (r *SetEvmNodeWsURLResult) Then(block func(response *blocksnifferpb.SetEvmNodeWsURLResponse, err error)) *Context {
	r.context.T().Helper()
	time.Sleep(fixture.WhenThenSleepInterval)
	block(r.response, r.err)
	return r.context
}

type GetEvmNodeWsURLResult struct {
	context  *Context
	response *blocksnifferpb.GetEvmNodeWsURLResponse
	err      error
}

func (a *Actions) GetEvmNodeWsURL() *GetEvmNodeWsURLResult {
	a.context.T().Helper()

	closer, client, err := fixture.AthenaClientset.NewBlockSnifferClient()
	require.NoError(a.context.T(), err)
	defer utilio.Close(closer)

	resp, err := client.GetEvmNodeWsURL(context.Background(), &blocksnifferpb.GetEvmNodeWsURLRequest{})
	return &GetEvmNodeWsURLResult{
		context:  a.context,
		response: resp,
		err:      err,
	}
}

func (r *GetEvmNodeWsURLResult) Then(block func(response *blocksnifferpb.GetEvmNodeWsURLResponse, err error)) *Context {
	r.context.T().Helper()
	time.Sleep(fixture.WhenThenSleepInterval)
	block(r.response, r.err)
	return r.context
}
