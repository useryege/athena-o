package reconcile

const (
	persistenceEventVersion            = 1
	PersistenceOpProjectGenesisReplace = "project_genesis_wallet_replace"
)

type projectGenesisWalletReplacePayload struct {
	Contract          string                            `json:"contract"`
	SourceTxHash      string                            `json:"source_tx_hash"`
	SourceBlockNumber uint64                            `json:"source_block_number"`
	TotalSupply       string                            `json:"total_supply"`
	Items             []projectGenesisWalletItemPayload `json:"items"`
}

type projectGenesisWalletItemPayload struct {
	Wallet    string `json:"wallet"`
	NetAmount string `json:"net_amount"`
	RatioBPS  int64  `json:"ratio_bps"`
	RankIndex int32  `json:"rank_index"`
}
