package token

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/useryege/athena/internal/token/chainscanner"
	"github.com/useryege/athena/internal/token/projectdatacollector"
	"github.com/useryege/athena/internal/token/projectreportbuilder"
	"github.com/useryege/athena/internal/token/projectresearchscheduler"
	"github.com/useryege/athena/internal/token/projectselectionevaluator"
	"github.com/useryege/athena/internal/token/projectvalidator"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

const (
	ModeDiscovery = "discovery"
	ModeResearch  = "research"
)

type worker interface {
	Start(context.Context) error
	Stop(context.Context) error
}
type Runtime struct {
	opts    RuntimeOptions
	store   *tokenstore.SQLStore
	workers []worker
}
type RuntimeOptions struct {
	Mode                   string
	StoreSrc               func(context.Context) (*tokenstore.SQLStore, error)
	EthNodeWSURLs          []string
	BSCNodeWSURLs          []string
	EthAthenaContract      string
	BSCAthenaContract      string
	EthEnabled             bool
	BSCEnabled             bool
	NodeWSUseProxy         bool
	AveAPIKey              string
	AveAPIBaseURL          string
	EthereumAPIAddress     string
	ChainStateInterval     time.Duration
	WalletAssetInterval    time.Duration
	SimulationInterval     time.Duration
	AveInterval            time.Duration
	ContractSourceInterval time.Duration
	ResearchTTL            time.Duration
}

func NormalizeMode(mode string) string { return strings.TrimSpace(mode) }
func NewRuntime(opts RuntimeOptions) (*Runtime, error) {
	mode := NormalizeMode(opts.Mode)
	if mode != ModeDiscovery && mode != ModeResearch {
		return nil, fmt.Errorf("unsupported athena-token mode %q; expected discovery or research", opts.Mode)
	}
	if opts.StoreSrc == nil {
		return nil, fmt.Errorf("token store source is required")
	}
	opts.Mode = mode
	return &Runtime{opts: opts}, nil
}
func (r *Runtime) Start(ctx context.Context) error {
	store, err := r.opts.StoreSrc(ctx)
	if err != nil {
		return err
	}
	r.store = store
	if r.opts.Mode == ModeDiscovery {
		r.workers = []worker{chainscanner.NewWorker(chainscanner.Options{Store: store, EthNodeWSURLs: r.opts.EthNodeWSURLs, BSCNodeWSURLs: r.opts.BSCNodeWSURLs, EthEnabled: r.opts.EthEnabled, BSCEnabled: r.opts.BSCEnabled, NodeWSUseProxy: r.opts.NodeWSUseProxy}), projectvalidator.NewWorker(projectvalidator.Options{Store: store, EthNodeWSURLs: r.opts.EthNodeWSURLs, BSCNodeWSURLs: r.opts.BSCNodeWSURLs, EthAthenaContract: r.opts.EthAthenaContract, BSCAthenaContract: r.opts.BSCAthenaContract, EthEnabled: r.opts.EthEnabled, BSCEnabled: r.opts.BSCEnabled, NodeWSUseProxy: r.opts.NodeWSUseProxy})}
	} else {
		r.workers = []worker{projectresearchscheduler.NewWorker(projectresearchscheduler.Options{Store: store, ChainStateInterval: r.opts.ChainStateInterval, WalletAssetInterval: r.opts.WalletAssetInterval, SimulationInterval: r.opts.SimulationInterval, AveInterval: r.opts.AveInterval, ContractSourceInterval: r.opts.ContractSourceInterval, ResearchTTL: r.opts.ResearchTTL}), projectdatacollector.NewWorker(projectdatacollector.Options{Store: store, EthNodeWSURLs: r.opts.EthNodeWSURLs, BSCNodeWSURLs: r.opts.BSCNodeWSURLs, EthAthenaContract: r.opts.EthAthenaContract, BSCAthenaContract: r.opts.BSCAthenaContract, EthEnabled: r.opts.EthEnabled, BSCEnabled: r.opts.BSCEnabled, NodeWSUseProxy: r.opts.NodeWSUseProxy, AveAPIKey: r.opts.AveAPIKey, AveAPIBaseURL: r.opts.AveAPIBaseURL, EthereumAPIAddress: r.opts.EthereumAPIAddress}), projectreportbuilder.NewWorker(projectreportbuilder.Options{Store: store}), projectselectionevaluator.NewWorker(projectselectionevaluator.Options{Store: store})}
	}
	for i, w := range r.workers {
		if err = w.Start(ctx); err != nil {
			r.stopWorkers(i)
			_ = store.Close()
			r.store = nil
			return err
		}
	}
	return nil
}
func (r *Runtime) stopWorkers(start int) {
	for i := start - 1; i >= 0; i-- {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = r.workers[i].Stop(ctx)
		cancel()
	}
}
func (r *Runtime) Stop() error {
	var result error
	for i := len(r.workers) - 1; i >= 0; i-- {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := r.workers[i].Stop(ctx); err != nil && result == nil {
			result = err
		}
		cancel()
	}
	r.workers = nil
	if r.store != nil {
		if err := r.store.Close(); err != nil && result == nil {
			result = err
		}
		r.store = nil
	}
	return result
}
