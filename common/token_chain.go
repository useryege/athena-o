package common

import "fmt"

const (
	ChainIDEthereumMainnet int64 = 1
	ChainIDBSCMainnet      int64 = 56

	ChainNameEthereumMainnet = "Ethereum Mainnet"
	ChainNameBSCMainnet      = "BSC Mainnet"
)

func ChainName(chainID int64) string {
	switch chainID {
	case ChainIDEthereumMainnet:
		return ChainNameEthereumMainnet
	case ChainIDBSCMainnet:
		return ChainNameBSCMainnet
	default:
		return fmt.Sprintf("chain %d", chainID)
	}
}
