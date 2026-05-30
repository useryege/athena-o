package components

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	appcache "github.com/useryege/athena/internal/application/cache"
	appstore "github.com/useryege/athena/internal/application/store"
	erc20contract "github.com/useryege/athena/pkg/abi/ERC20"
)

var (
	errGenesisReceiptNil   = errors.New("project transaction receipt is nil")
	erc20TransferTopicHash = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
)

type GenesisWalletNodeClient interface {
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
}

type GenesisWalletShare struct {
	Wallet   common.Address
	Amount   *big.Int
	RatioBPS int64
}

type GenesisWalletComponent struct {
	store      appstore.Store
	cache      appcache.ProjectComponentCache
	nodeClient GenesisWalletNodeClient
	bus        EventBus
	consumer   *Consumer
}

func NewGenesisWalletComponent(store appstore.Store, cache appcache.ProjectComponentCache, nodeClient GenesisWalletNodeClient, bus EventBus) *GenesisWalletComponent {
	if store == nil || nodeClient == nil {
		return nil
	}
	c := &GenesisWalletComponent{store: store, cache: cache, nodeClient: nodeClient, bus: bus}
	c.consumer = NewConsumer(appstore.ProjectComponentGenesisWallet, bus, c.handleEvent)
	return c
}

func (c *GenesisWalletComponent) Start(ctx context.Context) error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Start(ctx)
}

func (c *GenesisWalletComponent) Stop() error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *GenesisWalletComponent) handleEvent(ctx context.Context, event Event) error {
	if event.Type == EventComponentCompleted && event.Component != appstore.ProjectComponentChainState {
		return nil
	}
	if event.Type != EventComponentCompleted && event.Type != EventProjectInitialized && event.Type != EventProjectRefresh {
		return nil
	}
	contract := event.ProjectContract()
	if contract == (common.Address{}) {
		return nil
	}
	if event.Type != EventProjectRefresh {
		done, err := ComponentSucceeded(ctx, c.store, contract, appstore.ProjectComponentGenesisWallet)
		if err != nil || done {
			return nil
		}
	}
	if err := c.refresh(ctx, contract); err != nil {
		_ = MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentGenesisWallet, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, ComponentFailedEvent(contract, appstore.ProjectComponentGenesisWallet, err))
		}
		return nil
	}
	return nil
}

func (c *GenesisWalletComponent) refresh(ctx context.Context, contract common.Address) error {
	base, err := LoadProjectBase(ctx, c.cache, c.store, contract)
	if err != nil || base == nil {
		return err
	}
	chainState, err := LoadProjectChainState(ctx, c.cache, c.store, contract)
	if err != nil || chainState == nil {
		return err
	}
	if err := MarkComponentRunning(ctx, c.store, contract, appstore.ProjectComponentGenesisWallet, nowUTC()); err != nil {
		return err
	}
	shares, err := c.fetchGenesisWallets(ctx, *base, chainState.ChainState.Token.TotalSupply)
	if err != nil {
		return err
	}
	items := genesisWalletsToStore(*base, chainState.ChainState.Token.TotalSupply, shares)
	if err := c.store.ReplaceProjectGenesisWallets(ctx, contract, items); err != nil {
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetGenesisWallets(ctx, contract, items); err != nil {
			return err
		}
	}
	at := nowUTC()
	if err := MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentGenesisWallet, at); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, ComponentCompletedEvent(contract, appstore.ProjectComponentGenesisWallet))
	}
	return nil
}

func (c *GenesisWalletComponent) fetchGenesisWallets(ctx context.Context, base appstore.ProjectBase, totalSupply *big.Int) ([]GenesisWalletShare, error) {
	logs, err := c.fetchGenesisWalletsFromReceipt(ctx, base)
	if err == nil {
		return extractGenesisWalletShares(logs, base.Contract, totalSupply), nil
	}
	if !shouldFallbackToLogs(err) {
		return nil, err
	}
	fallbackLogs, fallbackErr := c.fetchGenesisWalletsFromLogsFallback(ctx, base)
	if fallbackErr != nil {
		return nil, fmt.Errorf("fetch genesis wallets by logs fallback: %w", fallbackErr)
	}
	return extractGenesisWalletShares(fallbackLogs, base.Contract, totalSupply), nil
}

func (c *GenesisWalletComponent) fetchGenesisWalletsFromReceipt(ctx context.Context, base appstore.ProjectBase) ([]*types.Log, error) {
	txHash := projectTxHash(base)
	if txHash == (common.Hash{}) {
		return nil, errors.New("project tx hash is empty")
	}
	receipt, err := c.nodeClient.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch transaction receipt %s: %w", txHash.Hex(), err)
	}
	if receipt == nil {
		return nil, fmt.Errorf("%w: %s", errGenesisReceiptNil, txHash.Hex())
	}
	return receipt.Logs, nil
}

func (c *GenesisWalletComponent) fetchGenesisWalletsFromLogsFallback(ctx context.Context, base appstore.ProjectBase) ([]*types.Log, error) {
	txHash := projectTxHash(base)
	if txHash == (common.Hash{}) {
		return nil, errors.New("project tx hash is empty")
	}
	blockNumber := new(big.Int).SetUint64(base.BlockNumber)
	logs, err := c.nodeClient.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: blockNumber,
		ToBlock:   blockNumber,
		Addresses: []common.Address{base.Contract},
		Topics: [][]common.Hash{
			{erc20TransferTopicHash},
		},
	})
	if err != nil {
		return nil, err
	}
	return filterLogsByTxHash(logs, txHash), nil
}

func shouldFallbackToLogs(err error) bool {
	return errors.Is(err, ethereum.NotFound) || errors.Is(err, errGenesisReceiptNil)
}

func projectTxHash(base appstore.ProjectBase) common.Hash {
	if base.TxHash != (common.Hash{}) {
		return base.TxHash
	}
	if base.Tx != nil {
		return base.Tx.Hash()
	}
	return common.Hash{}
}

func filterLogsByTxHash(logs []types.Log, txHash common.Hash) []*types.Log {
	filtered := make([]*types.Log, 0, len(logs))
	for i := range logs {
		if logs[i].TxHash != txHash {
			continue
		}
		entry := logs[i]
		filtered = append(filtered, &entry)
	}
	return filtered
}

func extractGenesisWalletShares(logs []*types.Log, tokenContract common.Address, totalSupply *big.Int) []GenesisWalletShare {
	filterer, err := erc20contract.NewERC20Filterer(tokenContract, nil)
	if err != nil {
		return nil
	}
	candidates := make(map[common.Address]struct{})
	netBalance := make(map[common.Address]*big.Int)
	for _, entry := range logs {
		if entry == nil || entry.Address != tokenContract {
			continue
		}
		transferEvent, err := filterer.ParseTransfer(*entry)
		if err != nil || transferEvent.Tokens == nil || transferEvent.Tokens.Sign() <= 0 {
			continue
		}
		if transferEvent.To != (common.Address{}) {
			candidates[transferEvent.To] = struct{}{}
			if _, exists := netBalance[transferEvent.To]; !exists {
				netBalance[transferEvent.To] = new(big.Int)
			}
			netBalance[transferEvent.To].Add(netBalance[transferEvent.To], transferEvent.Tokens)
		}
		if transferEvent.From != (common.Address{}) {
			if _, exists := netBalance[transferEvent.From]; !exists {
				netBalance[transferEvent.From] = new(big.Int)
			}
			netBalance[transferEvent.From].Sub(netBalance[transferEvent.From], transferEvent.Tokens)
		}
	}
	shares := make([]GenesisWalletShare, 0, len(candidates))
	for wallet := range candidates {
		amount, exists := netBalance[wallet]
		if !exists || amount.Sign() <= 0 {
			continue
		}
		shares = append(shares, GenesisWalletShare{
			Wallet:   wallet,
			Amount:   new(big.Int).Set(amount),
			RatioBPS: ratioBPS(amount, totalSupply),
		})
	}
	sort.Slice(shares, func(i, j int) bool {
		amountCmp := shares[i].Amount.Cmp(shares[j].Amount)
		if amountCmp != 0 {
			return amountCmp > 0
		}
		return bytes.Compare(shares[i].Wallet.Bytes(), shares[j].Wallet.Bytes()) < 0
	})
	return shares
}

func ratioBPS(amount *big.Int, totalSupply *big.Int) int64 {
	if amount == nil || amount.Sign() <= 0 || totalSupply == nil || totalSupply.Sign() <= 0 {
		return 0
	}
	numerator := new(big.Int).Mul(amount, big.NewInt(10000))
	halfDenominator := new(big.Int).Div(new(big.Int).Set(totalSupply), big.NewInt(2))
	numerator.Add(numerator, halfDenominator)
	result := new(big.Int).Div(numerator, totalSupply)
	if !result.IsInt64() {
		return math.MaxInt64
	}
	return result.Int64()
}

func genesisWalletsToStore(base appstore.ProjectBase, totalSupply *big.Int, shares []GenesisWalletShare) []appstore.ProjectGenesisWallet {
	txHash := projectTxHash(base)
	result := make([]appstore.ProjectGenesisWallet, 0, len(shares))
	for i, share := range shares {
		if share.Wallet == (common.Address{}) || share.Amount == nil || share.Amount.Sign() <= 0 {
			continue
		}
		result = append(result, appstore.ProjectGenesisWallet{
			ProjectContract:   base.Contract,
			Wallet:            share.Wallet,
			NetAmount:         new(big.Int).Set(share.Amount),
			RatioBPS:          share.RatioBPS,
			RankIndex:         int32(i),
			TotalSupply:       BigIntOrZero(totalSupply),
			SourceTxHash:      txHash,
			SourceBlockNumber: base.BlockNumber,
		})
	}
	return result
}
