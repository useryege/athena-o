package bytecode

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type ContractSourceResolver interface {
	GetContractSourceInfo(ctx context.Context, req *applicationpkg.GetContractSourceInfoRequest) (*v1alpha1.ContractSourceInfo, error)
}

type Options struct {
	Store    appstore.Store
	Cache    appcache.ProjectComponentCache
	Resolver ContractSourceResolver
	ChainID  int64
}

type Component struct {
	store    appstore.Store
	cache    appcache.ProjectComponentCache
	resolver ContractSourceResolver
	chainID  int64
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil || opts.Resolver == nil || opts.ChainID <= 0 {
		return nil
	}
	return &Component{store: opts.Store, cache: opts.Cache, resolver: opts.Resolver, chainID: opts.ChainID}
}

func (c *Component) Start(context.Context) error { return nil }

func (c *Component) Stop() error { return nil }

func (c *Component) Collect(ctx context.Context, chainID int64, contract common.Address) error {
	if c == nil || c.store == nil || c.resolver == nil {
		return nil
	}
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, chainID, contract); err != nil {
		_ = appcomponents.MarkComponentFailed(ctx, c.store, chainID, contract, appstore.ProjectComponentBytecodeFact, err, nowUTC())
		return err
	}
	return nil
}

func (c *Component) refresh(ctx context.Context, chainID int64, contract common.Address) error {
	if _, err := appcomponents.LoadProjectBase(ctx, c.cache, c.store, chainID, contract); err != nil {
		return err
	}
	if err := appcomponents.MarkComponentRunning(ctx, c.store, chainID, contract, appstore.ProjectComponentBytecodeFact, nowUTC()); err != nil {
		return err
	}
	info, err := c.resolver.GetContractSourceInfo(ctx, &applicationpkg.GetContractSourceInfoRequest{
		ChainId:  chainID,
		Contract: contract.Hex(),
	})
	if err != nil || info == nil {
		return err
	}
	item := appstore.ProjectBytecodeFact{
		ChainID:         chainID,
		ProjectContract: contract,
		FetchedAt:       nowUTC(),
	}
	if strings.TrimSpace(info.CodeBinHash) != "" {
		item.CodeHash = common.HexToHash(info.CodeBinHash)
	}
	if err := c.store.UpsertProjectBytecodeFact(ctx, item); err != nil {
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetBytecodeFact(ctx, item); err != nil {
			return err
		}
	}
	if err := appcomponents.MarkComponentSuccess(ctx, c.store, chainID, contract, appstore.ProjectComponentBytecodeFact, item.FetchedAt); err != nil {
		return err
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
