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
	lastSetResult *blocksnifferpb.SetNodeGrpcURLResponse
	lastGetResult *blocksnifferpb.GetNodeGrpcURLResponse
	lastError     error
}

func (a *Actions) SetNodeGrpcURL(nodeGrpcURL string) *Actions {
	a.context.T().Helper()

	closer, client, err := fixture.AthenaClientset.NewBlockSnifferClient()
	require.NoError(a.context.T(), err)
	defer utilio.Close(closer)

	resp, err := client.SetNodeGrpcURL(context.Background(), &blocksnifferpb.SetNodeGrpcURLRequest{
		NodeGrpcUrl: nodeGrpcURL,
	})
	a.lastSetResult = resp
	a.lastError = err
	return a
}

func (a *Actions) GetNodeGrpcURL() *Actions {
	a.context.T().Helper()

	closer, client, err := fixture.AthenaClientset.NewBlockSnifferClient()
	require.NoError(a.context.T(), err)
	defer utilio.Close(closer)

	resp, err := client.GetNodeGrpcURL(context.Background(), &blocksnifferpb.GetNodeGrpcURLRequest{})
	a.lastGetResult = resp
	a.lastError = err
	return a
}

func (a *Actions) Then() *Consequences {
	a.context.T().Helper()
	time.Sleep(fixture.WhenThenSleepInterval)
	return &Consequences{context: a.context, actions: a}
}
