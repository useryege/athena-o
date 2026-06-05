package model

const MinWethValue = 100000000000000000                           // 0.1 WETH
const MagicAddress = "0xF80891f66da152Dd69dA0905546691f99B5b86bD" // magic address
const ZeroAddress = "0x0000000000000000000000000000000000000000"  // zero address
const DeadAddress = "0x000000000000000000000000000000000000dEaD"  // dead address

const (
	ChainIDEthereumMainnet int64 = 1
	ChainIDBSCMainnet      int64 = 56

	EthereumMainnetConfirmationDepth uint64 = 3
	BSCMainnetConfirmationDepth      uint64 = 10
)
