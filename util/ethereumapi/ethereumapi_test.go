package ethereumapi

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEthereumAPI(t *testing.T) {
	api := NewEthereumAPI("https://api.etherscan.io/v2/api", "454GKQPGFRIYNEKESBHS2E8PVJESNN9B5I")
	sourceCode, err := api.GetSourceCode(context.Background(), 1, "0x9e68096675578cccf6eb7ad01350f731dde633ed")
	require.NoError(t, err)
	fmt.Println(sourceCode)
}

func TestEthereumAPI_GetABI(t *testing.T) {
	api := NewEthereumAPI("https://api.etherscan.io/v2/api", "454GKQPGFRIYNEKESBHS2E8PVJESNN9B5I")
	abi, err := api.GetABI(context.Background(), 1, "0x9e68096675578cccf6eb7ad01350f731dde633ed")
	require.NoError(t, err)
	fmt.Println(abi)
}
