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
	"github.com/useryege/athena/internal/token/telemetry"
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
	opts              RuntimeOptions
	store             *tokenstore.SQLStore
	workers           []worker
	tracker           *telemetry.Tracker
	health            *telemetry.Server
	diagnosticsCancel context.CancelFunc
	diagnosticsDone   chan struct{}
}
type RuntimeOptions struct {
	Mode                string
	StoreSrc            func(context.Context) (*tokenstore.SQLStore, error)
	Discovery           DiscoveryOptions
	Research            ResearchOptions
	HealthListenAddress string
	HealthStaleAfter    time.Duration
}

type ChainRuntimeOptions struct {
	EthNodeWSURLs     []string
	BSCNodeWSURLs     []string
	EthAthenaContract string
	BSCAthenaContract string
	EthEnabled        bool
	BSCEnabled        bool
	NodeWSUseProxy    bool
}

type DiscoveryOptions struct {
	Chains ChainRuntimeOptions
}

type ResearchOptions struct {
	Chains                   ChainRuntimeOptions
	AveAPIKey                string
	AveAPIBaseURL            string
	EthereumAPIAddress       string
	ChainStateInterval       time.Duration
	WalletAssetInterval      time.Duration
	SimulationInterval       time.Duration
	AveInterval              time.Duration
	ContractSourceInterval   time.Duration
	ResearchTTL              time.Duration
	SelectionStrategyKey     string
	SelectionStrategyVersion string
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
	if strings.TrimSpace(opts.Research.SelectionStrategyKey) == "" {
		opts.Research.SelectionStrategyKey = "default"
	}
	if strings.TrimSpace(opts.Research.SelectionStrategyVersion) == "" {
		opts.Research.SelectionStrategyVersion = "1"
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
	r.tracker = telemetry.NewTracker(r.opts.Mode, r.opts.HealthStaleAfter)
	healthAddress := strings.TrimSpace(r.opts.HealthListenAddress)
	if healthAddress == "" {
		healthAddress = defaultHealthListenAddress(r.opts.Mode)
	}
	r.health = telemetry.NewServer(healthAddress, r.tracker, store.Ping)
	if err = r.health.Start(); err != nil {
		_ = store.Close()
		r.store = nil
		return err
	}
	if r.opts.Mode == ModeDiscovery {
		r.workers = r.buildDiscoveryWorkers(store, r.tracker)
	} else {
		r.workers, err = r.buildResearchWorkers(store, r.tracker)
		if err != nil {
			_ = r.health.Stop(context.Background())
			_ = store.Close()
			r.store = nil
			return err
		}
	}
	for i, w := range r.workers {
		if err = w.Start(ctx); err != nil {
			r.stopWorkers(i)
			_ = r.health.Stop(context.Background())
			_ = store.Close()
			r.store = nil
			return err
		}
	}
	r.tracker.SetRunning(true)
	r.startDiagnosticsMonitor()
	return nil
}

func (r *Runtime) buildDiscoveryWorkers(store *tokenstore.SQLStore, reporter telemetry.Reporter) []worker {
	chains := r.opts.Discovery.Chains
	return []worker{
		chainscanner.NewWorker(chainscanner.Options{Store: store, EthNodeWSURLs: chains.EthNodeWSURLs, BSCNodeWSURLs: chains.BSCNodeWSURLs, EthEnabled: chains.EthEnabled, BSCEnabled: chains.BSCEnabled, NodeWSUseProxy: chains.NodeWSUseProxy, Telemetry: reporter}),
		projectvalidator.NewWorker(projectvalidator.Options{Store: store, EthNodeWSURLs: chains.EthNodeWSURLs, BSCNodeWSURLs: chains.BSCNodeWSURLs, EthAthenaContract: chains.EthAthenaContract, BSCAthenaContract: chains.BSCAthenaContract, EthEnabled: chains.EthEnabled, BSCEnabled: chains.BSCEnabled, NodeWSUseProxy: chains.NodeWSUseProxy, Telemetry: reporter}),
	}
}

func (r *Runtime) buildResearchWorkers(store *tokenstore.SQLStore, reporter telemetry.Reporter) ([]worker, error) {
	opts := r.opts.Research
	chains := opts.Chains
	registry, err := projectselectionevaluator.NewRegistry(opts.SelectionStrategyKey, opts.SelectionStrategyVersion)
	if err != nil {
		return nil, err
	}
	return []worker{
		projectresearchscheduler.NewWorker(projectresearchscheduler.Options{Store: store, ChainStateInterval: opts.ChainStateInterval, WalletAssetInterval: opts.WalletAssetInterval, SimulationInterval: opts.SimulationInterval, AveInterval: opts.AveInterval, ContractSourceInterval: opts.ContractSourceInterval, ResearchTTL: opts.ResearchTTL, Telemetry: reporter}),
		projectdatacollector.NewWorker(projectdatacollector.Options{Store: store, EthNodeWSURLs: chains.EthNodeWSURLs, BSCNodeWSURLs: chains.BSCNodeWSURLs, EthAthenaContract: chains.EthAthenaContract, BSCAthenaContract: chains.BSCAthenaContract, EthEnabled: chains.EthEnabled, BSCEnabled: chains.BSCEnabled, NodeWSUseProxy: chains.NodeWSUseProxy, AveAPIKey: opts.AveAPIKey, AveAPIBaseURL: opts.AveAPIBaseURL, EthereumAPIAddress: opts.EthereumAPIAddress, Telemetry: reporter}),
		projectreportbuilder.NewWorker(projectreportbuilder.Options{Store: store, Telemetry: reporter}),
		projectselectionevaluator.NewWorker(projectselectionevaluator.Options{Store: store, Registry: registry, Telemetry: reporter}),
	}, nil
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
	if r.tracker != nil {
		r.tracker.SetRunning(false)
	}
	if r.diagnosticsCancel != nil {
		r.diagnosticsCancel()
		r.diagnosticsCancel = nil
	}
	if r.diagnosticsDone != nil {
		<-r.diagnosticsDone
		r.diagnosticsDone = nil
	}
	for i := len(r.workers) - 1; i >= 0; i-- {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := r.workers[i].Stop(ctx); err != nil && result == nil {
			result = err
		}
		cancel()
	}
	r.workers = nil
	if r.health != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := r.health.Stop(ctx); err != nil && result == nil {
			result = err
		}
		cancel()
		r.health = nil
	}
	if r.store != nil {
		if err := r.store.Close(); err != nil && result == nil {
			result = err
		}
		r.store = nil
	}
	r.tracker = nil
	return result
}

func defaultHealthListenAddress(mode string) string {
	if mode == ModeResearch {
		return "127.0.0.1:8097"
	}
	return "127.0.0.1:8095"
}

func (r *Runtime) startDiagnosticsMonitor() {
	ctx, cancel := context.WithCancel(context.Background())
	r.diagnosticsCancel = cancel
	r.diagnosticsDone = make(chan struct{})
	go func() {
		defer close(r.diagnosticsDone)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			r.refreshDiagnostics(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (r *Runtime) refreshDiagnostics(parent context.Context) {
	if r.store == nil || r.tracker == nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	metrics, err := r.store.ListPipelineQueueMetrics(ctx)
	cancel()
	if err != nil {
		r.tracker.SetDiagnosticsSuccess(false)
		return
	}
	r.tracker.ResetQueues()
	queues := []string{
		"candidate_validation",
		"data_collection:ave",
		"data_collection:chain_state",
		"data_collection:contract_code_source",
		"data_collection:simulation_result",
		"data_collection:wallet_asset_state",
		"report_build",
		"selection_evaluation",
	}
	for _, queue := range queues {
		for _, status := range []string{"pending", "running", "failed"} {
			r.tracker.UpdateQueue(queue, status, 0, time.Time{})
		}
	}
	for _, metric := range metrics {
		r.tracker.UpdateQueue(metric.Queue, metric.Status, metric.Count, metric.OldestAvailableAt)
	}
	r.tracker.SetDiagnosticsSuccess(true)
}
