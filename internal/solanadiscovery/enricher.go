package solanadiscovery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// Enricher borrows the scanner's node gate: both loops share budget, cooldown and
// request timeout. Queue state and retry times survive process restarts in Store.
type Enricher struct{ scanner *Scanner }

func NewEnricher(scanner *Scanner) *Enricher { return &Enricher{scanner: scanner} }
func (e *Enricher) nodeCall(ctx context.Context, call func(context.Context) error) error {
	return e.scanner.nodeCall(ctx, call)
}
func enrichmentRetry(now time.Time, err error) time.Time {
	delay := 5 * time.Minute
	var response *HTTPError
	if errors.As(err, &response) && response.RetryAfter > delay {
		delay = response.RetryAfter
	}
	return now.Add(delay)
}

func (e *Enricher) EnrichOnce(ctx context.Context) (int, error) {
	work, err := e.scanner.store.PendingEnrichments(ctx, time.Now(), 10)
	if err != nil || len(work) == 0 {
		return 0, err
	}
	if err := e.scanner.verifyGenesis(ctx); err != nil {
		return 0, err
	}
	var writes []error
	// Account requests are paired and bounded at 20; no remote call runs inside a DB transaction.
	var addresses []string
	var metadataWork []EnrichmentWork
	for _, w := range work {
		if !w.MetadataDue {
			continue
		}
		pda, err := MetadataAddress(w.Project.Mint)
		if err != nil {
			writes = append(writes, e.scanner.store.WriteMetadata(ctx, w.Project.Mint, MetadataResult{Status: "error"}, time.Time{}, enrichmentRetry(time.Now(), err)))
			continue
		}
		addresses = append(addresses, w.Project.Mint, pda)
		metadataWork = append(metadataWork, w)
	}
	if len(addresses) > 0 {
		var snapshot AccountSnapshot
		batchErr := e.nodeCall(ctx, func(callCtx context.Context) error {
			var err error
			snapshot, err = e.scanner.rpc.MultipleAccounts(callCtx, addresses)
			return err
		})
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		for i, w := range metadataWork {
			result := MetadataResult{}
			readErr := batchErr
			if readErr == nil {
				if snapshot.Slot < w.Project.Slot {
					readErr = errors.New("metadata bank predates initialization")
				} else {
					result, readErr = ReadMetadata(w.Project, snapshot.Accounts[2*i], snapshot.Accounts[2*i+1])
					result.ObservedSlot = snapshot.Slot
				}
			}
			now := time.Now()
			var next time.Time
			if readErr != nil {
				result = MetadataResult{Status: "error"}
				next = enrichmentRetry(now, readErr)
				slog.WarnContext(ctx, "Solana metadata enrichment failed", "mint", w.Project.Mint, "error", readErr)
			} else if result.Status == "unavailable" {
				next = now.Add(time.Hour)
			}
			writes = append(writes, e.scanner.store.WriteMetadata(ctx, w.Project.Mint, result, now, next))
		}
	}
	// One finalized transaction fetch per original signature, even for several mints.
	type transactionEvidence struct {
		projects []Project
		err      error
	}
	transactions := make(map[string]transactionEvidence)
	for _, w := range work {
		if !w.SourceDue {
			continue
		}
		evidence, cached := transactions[w.Project.Signature]
		if !cached {
			evidence.err = e.nodeCall(ctx, func(callCtx context.Context) error {
				data, err := e.scanner.rpc.Transaction(callCtx, w.Project.Signature)
				if err != nil {
					return err
				}
				evidence.projects, err = ParseTransaction(data)
				return err
			})
			transactions[w.Project.Signature] = evidence
		}
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		result := SourceResult{Status: "error"}
		readErr := evidence.err
		if readErr == nil {
			readErr = errors.New("initialization evidence not found in transaction")
			for _, p := range evidence.projects {
				if p.Mint == w.Project.Mint && p.Signature == w.Project.Signature && p.TokenProgram == w.Project.TokenProgram && p.Slot == w.Project.Slot {
					result = SourceResult{Source: p.IssuanceSource, Program: p.IssuanceProgram, Status: p.SourceStatus}
					readErr = nil
					break
				}
			}
		}
		var next time.Time
		if readErr != nil {
			next = enrichmentRetry(time.Now(), readErr)
			slog.WarnContext(ctx, "Solana issuance enrichment failed", "mint", w.Project.Mint, "error", readErr)
		}
		writes = append(writes, e.scanner.store.WriteSource(ctx, w.Project.Mint, result, next))
	}
	return len(work), errors.Join(writes...)
}
func (e *Enricher) Run(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		if _, err := e.EnrichOnce(ctx); err != nil && ctx.Err() == nil {
			slog.WarnContext(ctx, "Solana enrichment batch failed", "error", fmt.Errorf("enrich candidates: %w", err))
		}
		if !waitContext(ctx, 10*time.Second) {
			return nil
		}
	}
}
