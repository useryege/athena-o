package e2e

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	blocksnifferpb "github.com/useryege/athena/pkg/apiclient/blocksniffer"
	blockSnifferFixture "github.com/useryege/athena/test/e2e/fixture/blocksniffer"
)

func TestSetAndGetEvmNodeWsURL(t *testing.T) {
	const expectedURL = "https://rpc.ankr.com/eth"

	ctx := blockSnifferFixture.Given(t)
	ctx.
		When().
		SetEvmNodeWsURL(expectedURL).
		Then().
		SetEvmNodeWsURLResult(func(response *blocksnifferpb.SetEvmNodeWsURLResponse, err error) {
			require.NoError(t, err)
			require.NotNil(t, response)
			assert.Equal(t, expectedURL, response.GetEvmNodeWsURL())
		}).
		When().
		GetEvmNodeWsURL().
		Then().
		GetEvmNodeWsURLResult(func(response *blocksnifferpb.GetEvmNodeWsURLResponse, err error) {
			require.NoError(t, err)
			require.NotNil(t, response)
			assert.Equal(t, expectedURL, response.GetEvmNodeWsURL())
		})
}
