package evmtool

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestGetToken0(t *testing.T) {
	token0 := GetToken0(common.HexToAddress("0x123"), common.HexToAddress("0x456"))
	require.Equal(t, common.HexToAddress("0x123"), token0)
}

func TestGetToken1(t *testing.T) {
	token1 := GetToken1(common.HexToAddress("0x123"), common.HexToAddress("0x456"))
	require.Equal(t, common.HexToAddress("0x456"), token1)
}
