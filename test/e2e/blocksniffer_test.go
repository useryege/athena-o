package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	blocksnifferpb "github.com/useryege/athena/pkg/apiclient/blocksniffer"
	blockSnifferFixture "github.com/useryege/athena/test/e2e/fixture/blocksniffer"
)

func TestSetAndGetNodeGrpcURL(t *testing.T) {
	const expectedURL = "https://rpc.ankr.com/eth"

	ctx := blockSnifferFixture.Given(t)
	ctx.
		When().
		SetNodeGrpcURL(expectedURL).
		Then().
		SetNodeGrpcURLResult(func(response *blocksnifferpb.SetNodeGrpcURLResponse, err error) {
			require.NoError(t, err)
			require.NotNil(t, response)
			assert.Equal(t, expectedURL, response.GetNodeGrpcUrl())
		}).
		When().
		GetNodeGrpcURL().
		Then().
		GetNodeGrpcURLResult(func(response *blocksnifferpb.GetNodeGrpcURLResponse, err error) {
			require.NoError(t, err)
			require.NotNil(t, response)
			assert.Equal(t, expectedURL, response.GetNodeGrpcUrl())
		})
}
