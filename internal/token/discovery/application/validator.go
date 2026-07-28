package application

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/shared"
)

const (
	defaultCandidateLimit      = int32(100)
	defaultCandidateLease      = 5 * time.Minute
	defaultCandidateLeaseRenew = time.Minute
)

type CandidateLease struct {
	ID         string
	Candidates []discovery.ProjectCandidate
}

type CandidateLeasePort interface {
	ClaimCandidates(context.Context, int64, time.Duration, int32) (*CandidateLease, error)
	RenewCandidateLease(context.Context, string, time.Duration) error
	ReleaseCandidateLease(context.Context, string) error
}

type CandidateInspector interface {
	InspectCandidates(context.Context, int64, []discovery.ProjectCandidate, int) ([]CandidateInspection, error)
}

type CandidateInspection struct {
	Candidate         discovery.ProjectCandidate
	Accepted          bool
	CodeHash          shared.Hash
	Name              string
	Symbol            string
	Decimals          uint8
	TotalSupply       *big.Int
	WethPair          shared.Address
	UsdtPair          shared.Address
	RelatedWallets    []RelatedWallet
	InitialRecipients []InitialRecipient
}

type RelatedWallet struct {
	Wallet shared.Address
	Role   string
}

type InitialRecipient struct {
	Wallet            shared.Address
	RatioBPS          int64
	RankIndex         int32
	SourceTxHash      shared.Hash
	SourceBlockNumber uint64
}

type CollectionScheduleSeed struct {
	DataType      string
	RetryInterval time.Duration
	NextRunAt     time.Time
}

type PromoteCandidateCommand struct {
	Inspection CandidateInspection
	Schedules  []CollectionScheduleSeed
}

type CandidateTransaction interface {
	PromoteCandidateAndInitializeResearch(context.Context, PromoteCandidateCommand) (shared.ProjectID, error)
	RejectCandidate(context.Context, discovery.ProjectCandidate) error
}

type ValidatorOptions struct {
	CandidateLimit       int32
	CandidateConcurrency int
	LeaseDuration        time.Duration
	LeaseRenewInterval   time.Duration
	Now                  func() time.Time
}

type Validator struct {
	leases      CandidateLeasePort
	inspector   CandidateInspector
	transaction CandidateTransaction
	options     ValidatorOptions
}

func NewValidator(leases CandidateLeasePort, inspector CandidateInspector, transaction CandidateTransaction, options ValidatorOptions) *Validator {
	if options.CandidateLimit <= 0 {
		options.CandidateLimit = defaultCandidateLimit
	}
	if options.CandidateConcurrency <= 0 {
		options.CandidateConcurrency = 10
	}
	if options.LeaseDuration <= 0 {
		options.LeaseDuration = defaultCandidateLease
	}
	if options.LeaseRenewInterval <= 0 {
		options.LeaseRenewInterval = defaultCandidateLeaseRenew
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Validator{leases: leases, inspector: inspector, transaction: transaction, options: options}
}

func (validator *Validator) RunOnce(ctx context.Context, chainID int64) (int, error) {
	if validator == nil || validator.leases == nil || validator.inspector == nil || validator.transaction == nil {
		return 0, fmt.Errorf("token validator application is not configured")
	}
	lease, err := validator.leases.ClaimCandidates(ctx, chainID, validator.options.LeaseDuration, validator.options.CandidateLimit)
	if err != nil || lease == nil || len(lease.Candidates) == 0 {
		return 0, err
	}
	stopRenewal := validator.renewLease(ctx, lease.ID)
	defer func() {
		stopRenewal()
		_ = validator.leases.ReleaseCandidateLease(context.WithoutCancel(ctx), lease.ID)
	}()
	inspections, err := validator.inspector.InspectCandidates(ctx, chainID, lease.Candidates, validator.options.CandidateConcurrency)
	if err != nil {
		return 0, err
	}
	if len(inspections) != len(lease.Candidates) {
		return 0, fmt.Errorf("inspect candidates returned %d results for %d candidates", len(inspections), len(lease.Candidates))
	}
	processed := 0
	for _, inspection := range inspections {
		if !inspection.Accepted {
			if err := validator.transaction.RejectCandidate(ctx, inspection.Candidate); err != nil {
				return processed, err
			}
			processed++
			continue
		}
		if _, err := validator.transaction.PromoteCandidateAndInitializeResearch(ctx, PromoteCandidateCommand{Inspection: inspection, Schedules: defaultResearchSchedules(validator.options.Now().UTC())}); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (validator *Validator) renewLease(ctx context.Context, leaseID string) func() {
	renewCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(validator.options.LeaseRenewInterval)
		defer ticker.Stop()
		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				_ = validator.leases.RenewCandidateLease(renewCtx, leaseID, validator.options.LeaseDuration)
			}
		}
	}()
	return func() { cancel(); <-done }
}

func defaultResearchSchedules(now time.Time) []CollectionScheduleSeed {
	return []CollectionScheduleSeed{
		{DataType: "chain_state", RetryInterval: 15 * time.Second, NextRunAt: now},
		{DataType: "wallet_asset_state", RetryInterval: time.Minute, NextRunAt: now},
		{DataType: "simulation_result", RetryInterval: time.Minute, NextRunAt: now},
		{DataType: "ave", RetryInterval: 5 * time.Minute, NextRunAt: now},
		{DataType: "contract_code_source", RetryInterval: 10 * time.Minute, NextRunAt: now},
		{DataType: "wallet_normal_transactions", RetryInterval: 10 * time.Minute, NextRunAt: now},
	}
}
