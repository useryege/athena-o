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
	context       *Context
	lastSetResult *blocksnifferpb.SetEvmNodeWsURLResponse
	lastGetResult *blocksnifferpb.GetEvmNodeWsURLResponse
	lastError     error
}

func (a *Actions) SetEvmNodeWsURL(evmNodeWsURL string) *Actions {
	a.context.T().Helper()

	closer, client, err := fixture.AthenaClientset.NewBlockSnifferClient()
	require.NoError(a.context.T(), err)
	defer utilio.Close(closer)

	resp, err := client.SetEvmNodeWsURL(context.Background(), &blocksnifferpb.SetEvmNodeWsURLRequest{
		EvmNodeWsURL: evmNodeWsURL,
	})
	a.lastSetResult = resp
	a.lastError = err
	return a
}

func (a *Actions) GetEvmNodeWsURL() *Actions {
	a.context.T().Helper()

	closer, client, err := fixture.AthenaClientset.NewBlockSnifferClient()
	require.NoError(a.context.T(), err)
	defer utilio.Close(closer)

	resp, err := client.GetEvmNodeWsURL(context.Background(), &blocksnifferpb.GetEvmNodeWsURLRequest{})
	a.lastGetResult = resp
	a.lastError = err
	return a
}

func (a *Actions) Then() *Consequences {
	a.context.T().Helper()
	time.Sleep(fixture.WhenThenSleepInterval)
	return &Consequences{context: a.context, actions: a}
}
