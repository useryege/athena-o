package v1alpha1

type FIFAMarketDashboardEventConfig struct {
	WormEventID string `protobuf:"bytes,1,opt,name=wormEventId" json:"wormEventId"`
	EventRef    string `protobuf:"bytes,2,opt,name=eventRef" json:"eventRef"`
}

type FIFAMarketDashboard struct {
	Config                  *FIFAMarketDashboardEventConfig         `protobuf:"bytes,1,opt,name=config" json:"config"`
	WormEvent               *WormMarketsEventItem                   `protobuf:"bytes,2,opt,name=wormEvent" json:"wormEvent"`
	WormFetchedAt           int64                                   `protobuf:"varint,3,opt,name=wormFetchedAt" json:"wormFetchedAt"`
	WormError               string                                  `protobuf:"bytes,4,opt,name=wormError" json:"wormError"`
	PolymarketEvent         *PolymarketFIFAMoneylineEventItem       `protobuf:"bytes,5,opt,name=polymarketEvent" json:"polymarketEvent"`
	PolymarketFetchedAt     int64                                   `protobuf:"varint,6,opt,name=polymarketFetchedAt" json:"polymarketFetchedAt"`
	PolymarketError         string                                  `protobuf:"bytes,7,opt,name=polymarketError" json:"polymarketError"`
	WalletBalances          []*FIFAMarketDashboardWalletBalanceItem `protobuf:"bytes,8,rep,name=walletBalances" json:"walletBalances"`
	WalletBalancesFetchedAt int64                                   `protobuf:"varint,9,opt,name=walletBalancesFetchedAt" json:"walletBalancesFetchedAt"`
	WalletBalancesError     string                                  `protobuf:"bytes,10,opt,name=walletBalancesError" json:"walletBalancesError"`
	WalletHoldings          []*FIFAMarketDashboardWalletHoldingItem `protobuf:"bytes,11,rep,name=walletHoldings" json:"walletHoldings"`
	WalletHoldingsFetchedAt int64                                   `protobuf:"varint,12,opt,name=walletHoldingsFetchedAt" json:"walletHoldingsFetchedAt"`
	WalletHoldingsError     string                                  `protobuf:"bytes,13,opt,name=walletHoldingsError" json:"walletHoldingsError"`
	FetchedAt               int64                                   `protobuf:"varint,14,opt,name=fetchedAt" json:"fetchedAt"`
}
