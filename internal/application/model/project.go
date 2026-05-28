package model

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type Project struct {
	Meta      ProjectMeta
	Report    ProjectReport
	AveDetail *ProjectAveDetail
}

type ProjectReport struct {
	IsPolicyEvaluated          bool
	IsBlacklistedCreatorWallet bool
	IsBlacklistedGenesisWallet bool
	IsBlacklistedBytecode      bool
	HasMintRisk                bool
}

type ProjectMeta struct {
	BlockTime                          uint64
	BlockNumber                        uint64
	Contract                           common.Address
	Creator                            common.Address
	WethPair                           common.Address
	UsdtPair                           common.Address
	FetchAt                            time.Time
	TxHash                             common.Hash
	TxIndex                            uint64
	GenesisTx                          *types.Transaction
	ChainState                         athenacontract.AthenaProject
	IsBytecodeBlacklisted              bool
	GenesisWallets                     []GenesisWalletMeta
	GenesisWalletsFetchedAt            time.Time
	CreatorResult                      SimulateResult
	CreatorHistoricalProjects          []common.Address
	CreatorHistoricalProjectsFetchedAt time.Time
}

type ProjectAveDetail struct {
	Status    int
	Msg       string
	DataType  int
	IsAudited bool
	FetchedAt time.Time
	Token     ProjectAveTokenDetail
	Pairs     []ProjectAvePair
}

type ProjectAveTokenDetail struct {
	Total               string
	LaunchPrice         string
	CurrentPriceETH     string
	CurrentPriceUSD     string
	PriceChange1D       string
	PriceChange24H      string
	PriceChange1H       string
	LockAmount          string
	BurnAmount          string
	OtherAmount         string
	TxAmount24H         string
	TxVolumeU24H        string
	LockedPercent       string
	MarketCap           string
	FDV                 string
	TVL                 string
	MainPairTVL         string
	TokenPriceChange5M  string
	TokenPriceChange1H  string
	TokenPriceChange4H  string
	TokenPriceChange24H string
	TokenTxVolumeUSD5M  string
	TokenTxVolumeUSD1H  string
	TokenTxVolumeUSD4H  string
	TokenTxVolumeUSD24H string
	TokenBuyVolumeU5M   string
	TokenSellVolumeU5M  string
	Token               string
	Chain               string
	Decimal             int
	Name                string
	Symbol              string
	Holders             int
	Appendix            string
	RiskLevel           int
	LogoURL             string
	RiskInfo            string
	RiskScore           string
	LaunchAt            int64
	CreatedAt           int64
	TxCount24H          int
	LockPlatform        string
	IsMintable          string
	UpdatedAt           int64
	MainPair            string
	HasMintMethod       bool
	IsLPNotLocked       bool
	HasNotRenounced     bool
	HasNotAudited       bool
	HasNotOpenSource    bool
	IsInBlacklist       bool
	IsHoneypot          bool
	AveRiskLevel        int
}

type ProjectAvePair struct {
	Reserve0       string
	Reserve1       string
	Token0PriceETH string
	Token0PriceUSD string
	Token1PriceETH string
	Token1PriceUSD string
	PriceChange    string
	PriceChange24H string
	PriceChange1H  string
	VolumeU        string
	LowU           string
	HighU          string
	Fee            string
	TotalSupply    string
	TxAmount       string
	Pair           string
	Chain          string
	AMM            string
	Token0Address  string
	Token0Symbol   string
	Token0Decimal  int
	Token1Address  string
	Token1Symbol   string
	Token1Decimal  int
	TargetToken    string
	PriceChange1D  string
	CreatedAt      int64
	TxCount        int
	UpdatedAt      int64
	MarketCap      string
	FDV            string
	IsFake         bool
}

type GenesisWalletMeta struct {
	Wallet    common.Address
	NetAmount *big.Int
	RatioBPS  int64
	RankIndex int32
}

func FormatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
