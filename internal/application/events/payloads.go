package events

type ContractCreatedPayload struct {
	Contract    string `json:"contract"`
	Creator     string `json:"creator"`
	TxHash      string `json:"tx_hash"`
	CodeHash    string `json:"code_hash"`
	WethPair    string `json:"weth_pair,omitempty"`
	UsdtPair    string `json:"usdt_pair,omitempty"`
	BlockNumber int64  `json:"block_number"`
	BlockTime   int64  `json:"block_time"`
	TxIndex     int64  `json:"tx_index"`
}

type DexSwapPayload struct {
	Pair        string `json:"pair"`
	Token0      string `json:"token0"`
	Token1      string `json:"token1"`
	TxHash      string `json:"tx_hash"`
	BlockNumber int64  `json:"block_number"`
}
