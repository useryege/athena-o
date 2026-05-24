package store

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ProjectMeta struct {
	BlockTime                          uint64
	BlockNumber                        uint64
	Contract                           common.Address
	Creator                            common.Address
	WethPair                           common.Address
	UsdtPair                           common.Address
	FetchAt                            time.Time
	Tx                                 *types.Transaction
	TxHash                             common.Hash
	TxIndex                            uint64
	SourceCode                         string
	SourceCodeHash                     common.Hash
	SourceCodeFetchedAt                time.Time
	SourceCodeOrigin                   string
	CodeBinHash                        common.Hash
	CodeBinHashFetchedAt               time.Time
	SourceQualityReport                string
	SourceQualityReportFetchedAt       time.Time
	SourceQualityReportOrigin          string
	CreatorResult                      SimulateResult
	Report                             ProjectReport
	GenesisWalletsFetchedAt            time.Time
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

type SimulateResult struct {
	CanMintFromDeadViaTransferFrom     bool
	CanMintFromZeroViaTransferFrom     bool
	CanMintFromWethPairViaTransferFrom bool
	CanMintFromUsdtPairViaTransferFrom bool
	CanMintViaTransferToWethPair       bool
	CanMintViaTransferToUsdtPair       bool
}

type ProjectReport struct {
	IsPolicyEvaluated          bool
	IsBlacklistedCreatorWallet bool
	IsBlacklistedGenesisWallet bool
	IsBlacklistedBytecode      bool
	IsBlacklistedSourceCode    bool
	HasMintRisk                bool
}

type ProjectEventLog struct {
	ID             int64
	Contract       common.Address
	EventType      int16
	OccurredAt     time.Time
	Message        string
	Payload        string
	IdempotencyKey string
	CreatedAt      time.Time
}

type ProjectComment struct {
	ID        int64
	Contract  common.Address
	Username  string
	Content   string
	CreatedAt time.Time
}

type ProjectGenesisWallet struct {
	ID                int64
	ProjectContract   common.Address
	Wallet            common.Address
	NetAmount         *big.Int
	RatioBPS          int64
	RankIndex         int32
	TotalSupply       *big.Int
	SourceTxHash      common.Hash
	SourceBlockNumber uint64
	CreatedAt         time.Time
}

type ProjectCreatorHistoricalProject struct {
	ID                        int64
	ProjectContract           common.Address
	HistoricalProjectContract common.Address
	RankIndex                 int32
	CreatedAt                 time.Time
}

type ProjectStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
	GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error)
	ListProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	ListAllProjectMetas(ctx context.Context) ([]ProjectMeta, error)
	ListProjectMetasByPairAddresses(ctx context.Context, pairs []common.Address) ([]ProjectMeta, error)
	ListProjectMetasByCodeBinHash(ctx context.Context, codeBinHash common.Hash) ([]ProjectMeta, error)
	UpdateProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string, origin string) error
	UpdateProjectCodeBinHash(ctx context.Context, contract common.Address, codeBinHash common.Hash) error
	UpdateProjectSourceQualityReport(ctx context.Context, contract common.Address, report string, origin string) error
	UpsertProjectAveDetail(ctx context.Context, contract common.Address, detail ProjectAveDetail) error
	UpdateProjectCreatorResult(ctx context.Context, contract common.Address, result SimulateResult) error
	UpdateProjectReport(ctx context.Context, contract common.Address, report ProjectReport) error
	ListProjectMetasByCreator(ctx context.Context, creator common.Address) ([]ProjectMeta, error)
	ListProjectMetasByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectMeta, error)
	GetProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error)
}

type ProjectAveDetailStore interface {
	UpsertProjectAveDetail(ctx context.Context, contract common.Address, detail ProjectAveDetail) error
	GetProjectAveDetail(ctx context.Context, contract common.Address) (*ProjectAveDetail, error)
	ListProjectAveDetailsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]ProjectAveDetail, error)
}

type ProjectEventLogStore interface {
	AddProjectEventLog(ctx context.Context, item ProjectEventLog) error
	ListProjectEventLogsByContract(ctx context.Context, contract common.Address) ([]ProjectEventLog, error)
}

type ProjectCommentStore interface {
	AddProjectComment(ctx context.Context, item ProjectComment) (ProjectComment, error)
	ListProjectCommentsByContract(ctx context.Context, contract common.Address, page int32, pageSize int32) ([]ProjectComment, int64, int32, int32, error)
}

type ProjectGenesisWalletStore interface {
	ReplaceProjectGenesisWallets(ctx context.Context, contract common.Address, items []ProjectGenesisWallet) error
	ListProjectGenesisWalletsByContract(ctx context.Context, contract common.Address) ([]ProjectGenesisWallet, error)
	ListProjectGenesisWalletsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address][]ProjectGenesisWallet, error)
	ListProjectGenesisWalletsByWallet(ctx context.Context, wallet common.Address) ([]ProjectGenesisWallet, error)
}

type ProjectCreatorHistoricalProjectStore interface {
	ReplaceProjectCreatorHistoricalProjects(ctx context.Context, contract common.Address, items []ProjectCreatorHistoricalProject) error
	ListProjectCreatorHistoricalProjectsByContract(ctx context.Context, contract common.Address) ([]ProjectCreatorHistoricalProject, error)
	ListProjectCreatorHistoricalProjectsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address][]ProjectCreatorHistoricalProject, error)
}

type Store interface {
	ProjectStore
}
