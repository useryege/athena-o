package components

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appstore "github.com/useryege/athena/internal/application/store"
)

func LoadProject(ctx context.Context, cache appcache.ProjectComponentCache, store appstore.ProjectStore, chainID int64, contract common.Address) (*appstore.ProjectRecord, error) {
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
