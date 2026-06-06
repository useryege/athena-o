package components

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func ProjectFromCandidate(candidate model.DiscoveredProjectCandidate) appstore.Project {
	return appstore.Project{
		ChainID:     candidate.ChainID,
		BlockTime:   candidate.BlockTime,
		BlockNumber: candidate.BlockNumber,
		Contract:    candidate.Contract,
		Creator:     candidate.Creator,
		Tx:          candidate.Tx,
		TxHash:      candidate.TxHash,
		TxIndex:     candidate.TxIndex,
	}
}

func LoadProject(ctx context.Context, cache appcache.ProjectComponentCache, store appstore.ProjectStore, chainID int64, contract common.Address) (*appstore.Project, error) {
	if cache != nil {
		if base, ok, err := cache.GetProject(ctx, chainID, contract); err != nil {
			return nil, err
		} else if ok && base != nil {
			return base, nil
		}
	}
	if store == nil {
		return nil, nil
	}
	base, err := store.GetProjectByContract(ctx, chainID, contract)
	if err != nil || base == nil {
		return base, err
	}
	if cache != nil {
		if err := cache.SetProject(ctx, *base); err != nil {
			return nil, err
		}
	}
	return base, nil
}

func LoadProjectChainState(ctx context.Context, cache appcache.ProjectComponentCache, store appstore.ProjectChainStateStore, chainID int64, contract common.Address) (*appstore.ProjectChainState, error) {
	if cache != nil {
		if item, ok, err := cache.GetChainState(ctx, chainID, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	if store == nil {
		return nil, nil
	}
	item, err := store.GetProjectChainState(ctx, chainID, contract)
	if err != nil || item == nil {
		return item, err
	}
	if cache != nil {
		if err := cache.SetChainState(ctx, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func GenesisWalletAddresses(items []appstore.ProjectGenesisWallet) []common.Address {
	addresses := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item.Wallet != (common.Address{}) {
			addresses = append(addresses, item.Wallet)
		}
	}
	return addresses
}

func ProjectQuery(base appstore.Project, genesisWallets []appstore.ProjectGenesisWallet) athenacontract.AthenaProjectQuery {
	return athenacontract.AthenaProjectQuery{
		TokenContract:  base.Contract,
		MsgCaller:      base.Creator,
		GenesisWallets: GenesisWalletAddresses(genesisWallets),
	}
}

func ChainStateFromSnapshot(chainID int64, contract common.Address, snapshot athenacontract.AthenaProject, fetchedAt time.Time) (appstore.ProjectChainState, error) {
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return appstore.ProjectChainState{}, err
	}
	return appstore.ProjectChainState{
		ChainID:         chainID,
		ProjectContract: contract,
		ChainState:      snapshot,
		RawChainState:   raw,
		WethPair:        snapshot.WethPair.ContractAddress,
		UsdtPair:        snapshot.UsdtPair.ContractAddress,
		TokenName:       snapshot.Token.Name,
		TokenSymbol:     snapshot.Token.Symbol,
		FetchedAt:       fetchedAt,
	}, nil
}

func MarkComponentRunning(ctx context.Context, store appstore.ProjectComponentStateStore, chainID int64, contract common.Address, component string, at time.Time) error {
	if store == nil || contract == (common.Address{}) {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return store.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ChainID:         chainID,
		ProjectContract: contract,
		Component:       component,
		Status:          appstore.ProjectComponentStatusRunning,
		LastAttemptAt:   at,
	})
}

func MarkComponentSuccess(ctx context.Context, store appstore.ProjectComponentStateStore, chainID int64, contract common.Address, component string, at time.Time) error {
	if store == nil || contract == (common.Address{}) {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	return store.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ChainID:         chainID,
		ProjectContract: contract,
		Component:       component,
		Status:          appstore.ProjectComponentStatusSuccess,
		LastAttemptAt:   at,
		LastSuccessAt:   at,
	})
}

func MarkComponentFailed(ctx context.Context, store appstore.ProjectComponentStateStore, chainID int64, contract common.Address, component string, cause error, at time.Time) error {
	if store == nil || contract == (common.Address{}) {
		return nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	lastError := ""
	if cause != nil {
		lastError = cause.Error()
	}
	return store.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
		ChainID:         chainID,
		ProjectContract: contract,
		Component:       component,
		Status:          appstore.ProjectComponentStatusFailed,
		LastAttemptAt:   at,
		LastError:       lastError,
	})
}

func ComponentSucceeded(ctx context.Context, store appstore.ProjectComponentStateStore, chainID int64, contract common.Address, component string) (bool, error) {
	if store == nil || contract == (common.Address{}) {
		return false, nil
	}
	state, err := store.GetProjectComponentState(ctx, chainID, contract, component)
	if err != nil || state == nil {
		return false, err
	}
	return state.Status == appstore.ProjectComponentStatusSuccess && !state.LastSuccessAt.IsZero(), nil
}

func BigIntOrZero(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}
