package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/shared"
)

type AveMarketDataRequest struct {
	ChainID  int64
	Contract shared.Address
	WethPair shared.Address
	UsdtPair shared.Address
}

type MarketDataProvider interface {
	GetMarketData(context.Context, AveMarketDataRequest) (collection.AveResultV1, error)
}

type AveProcessor struct{ Provider MarketDataProvider }

func (AveProcessor) DataType() collection.DataType { return collection.DataTypeAve }

func (processor AveProcessor) Process(ctx context.Context, project collection.ProjectContext) (CollectionOutput, error) {
	value, err := processor.Provider.GetMarketData(ctx, AveMarketDataRequest{
		ChainID:  project.ChainID,
		Contract: project.Contract,
		WethPair: project.WethPair,
		UsdtPair: project.UsdtPair,
	})
	return CollectionOutput{Result: value}, err
}

type ProjectStateReader interface {
	ReadChainState(context.Context, collection.ProjectContext) (collection.ChainStateResultV1, uint64, error)
	ReadWalletAssetState(context.Context, collection.ProjectContext) (collection.WalletAssetResultV1, uint64, error)
	ReadSimulationResult(context.Context, collection.ProjectContext) (collection.SimulationResultV1, uint64, error)
}

type ChainStateProcessor struct{ Reader ProjectStateReader }

func (ChainStateProcessor) DataType() collection.DataType { return collection.DataTypeChainState }

func (processor ChainStateProcessor) Process(ctx context.Context, project collection.ProjectContext) (CollectionOutput, error) {
	value, block, err := processor.Reader.ReadChainState(ctx, project)
	return CollectionOutput{Result: value, BlockNumber: &block}, err
}

type WalletAssetStateProcessor struct{ Reader ProjectStateReader }

func (WalletAssetStateProcessor) DataType() collection.DataType {
	return collection.DataTypeWalletAssetState
}

func (processor WalletAssetStateProcessor) Process(ctx context.Context, project collection.ProjectContext) (CollectionOutput, error) {
	value, block, err := processor.Reader.ReadWalletAssetState(ctx, project)
	return CollectionOutput{Result: value, BlockNumber: &block}, err
}

type SimulationResultProcessor struct{ Reader ProjectStateReader }

func (SimulationResultProcessor) DataType() collection.DataType {
	return collection.DataTypeSimulationResult
}

func (processor SimulationResultProcessor) Process(ctx context.Context, project collection.ProjectContext) (CollectionOutput, error) {
	value, block, err := processor.Reader.ReadSimulationResult(ctx, project)
	return CollectionOutput{Result: value, BlockNumber: &block}, err
}

type ContractCodeSnapshot struct {
	SourceCode string
	Fetched    bool
}

type ContractCodeReader interface {
	GetContractCodeSnapshot(context.Context, shared.Hash) (*ContractCodeSnapshot, error)
}

type SourceCodeProvider interface {
	GetSourceCode(context.Context, int64, shared.Address) (string, error)
}

type ContractSourceProcessor struct {
	Codes    ContractCodeReader
	Provider SourceCodeProvider
}

func (ContractSourceProcessor) DataType() collection.DataType {
	return collection.DataTypeContractCodeSource
}

func (processor ContractSourceProcessor) Process(ctx context.Context, project collection.ProjectContext) (CollectionOutput, error) {
	code, err := processor.Codes.GetContractCodeSnapshot(ctx, project.CodeHash)
	if err != nil {
		return CollectionOutput{}, fmt.Errorf("%w: read contract source cache: %w", ErrCollectionInfrastructure, err)
	}
	if code != nil && code.Fetched {
		output := contractSourceOutput(project.CodeHash, code.SourceCode)
		// A cache hit did not perform a provider fetch, so preserve the original
		// source_code_fetched_at timestamp instead of writing collection time.
		output.CodeSource = nil
		return output, nil
	}
	if processor.Provider == nil {
		return CollectionOutput{}, fmt.Errorf("contract source provider is not configured")
	}
	source, err := processor.Provider.GetSourceCode(ctx, project.ChainID, project.Contract)
	if err != nil {
		return CollectionOutput{}, err
	}
	return contractSourceOutput(project.CodeHash, source), nil
}

func contractSourceOutput(codeHash shared.Hash, source string) CollectionOutput {
	result := collection.ContractSourceResultV1{
		CodeHash:           codeHash,
		VerificationStatus: collection.SourceVerificationStatusUnverified,
	}
	if strings.TrimSpace(source) == "" {
		return CollectionOutput{Result: result, CodeSource: &CodeSourceUpdate{CodeHash: codeHash}}
	}
	result.VerificationStatus = collection.SourceVerificationStatusVerified
	result.ArtifactReference = codeHash.Hex()
	return CollectionOutput{
		Result:     result,
		CodeSource: &CodeSourceUpdate{CodeHash: codeHash, SourceCode: source},
	}
}

type WalletNormalTransactionProvider interface {
	ListNormalTransactions(context.Context, int64, shared.Address, uint64) ([]collection.WalletNormalTransaction, error)
}

type WalletNormalTransactionsProcessor struct {
	Provider WalletNormalTransactionProvider
}

func (WalletNormalTransactionsProcessor) DataType() collection.DataType {
	return collection.DataTypeWalletNormalTransactions
}

func (processor WalletNormalTransactionsProcessor) Process(ctx context.Context, project collection.ProjectContext) (CollectionOutput, error) {
	wallets := uniqueWalletsInOrder(project.RelatedWallets)
	transactions := make([]collection.WalletNormalTransaction, 0)
	cappedWallets := make([]shared.Address, 0)
	var snapshotBlock *uint64
	if project.DeploymentBlockNumber > 0 {
		endBlock := project.DeploymentBlockNumber - 1
		snapshotBlock = &endBlock
		for _, wallet := range wallets {
			items, err := processor.Provider.ListNormalTransactions(ctx, project.ChainID, wallet, endBlock)
			if err != nil {
				return CollectionOutput{}, err
			}
			if len(items) == normalTransactionPageSize {
				cappedWallets = append(cappedWallets, wallet)
			}
			transactions = append(transactions, items...)
		}
	}
	transactions, err := uniqueNormalTransactionAssociationsInOrder(transactions)
	if err != nil {
		return CollectionOutput{}, err
	}
	return CollectionOutput{
		Result:             summarizeNormalTransactions(wallets, transactions, cappedWallets),
		BlockNumber:        snapshotBlock,
		NormalTransactions: transactions,
	}, nil
}
