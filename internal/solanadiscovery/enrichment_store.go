package solanadiscovery

import (
	"context"
	"errors"
	"math"
	"time"
)

const projectColumns = `mint, token_program, signature, fee_payer, mint_authority, freeze_authority, decimals, slot, block_time, discovered_at,
name, symbol, metadata_status, metadata_source, metadata_account, metadata_observed_slot, metadata_updated_at, issuance_source, issuance_program, source_status`

func scanProject(scan func(...any) error) (Project, error) {
	var p Project
	var decimals int32
	var slot, observed int64
	var updated *time.Time
	err := scan(&p.Mint, &p.TokenProgram, &p.Signature, &p.FeePayer, &p.MintAuthority, &p.FreezeAuthority, &decimals, &slot, &p.BlockTime, &p.DiscoveredAt, &p.Name, &p.Symbol, &p.MetadataStatus, &p.MetadataSource, &p.MetadataAccount, &observed, &updated, &p.IssuanceSource, &p.IssuanceProgram, &p.SourceStatus)
	p.Decimals = uint32(decimals)
	p.Slot = uint64(slot)
	p.MetadataObservedSlot = uint64(observed)
	if updated != nil {
		p.MetadataUpdatedAt = *updated
	}
	return p, err
}

type EnrichmentWork struct {
	Project                Project
	MetadataDue, SourceDue bool
}

// PendingEnrichments reads a bounded queue without holding a transaction across RPC.
func (s *Store) PendingEnrichments(ctx context.Context, now time.Time, limit int) ([]EnrichmentWork, error) {
	if limit < 1 || limit > 10 {
		return nil, errors.New("invalid enrichment batch limit")
	}
	rows, err := s.pool.Query(ctx, `SELECT `+projectColumns+`, COALESCE(metadata_next_at <= $1,false),COALESCE(source_next_at <= $1,false)
 FROM solana_discovery.projects WHERE metadata_next_at <= $1 OR source_next_at <= $1
 ORDER BY LEAST(metadata_next_at,source_next_at),discovered_at,mint LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]EnrichmentWork, 0)
	for rows.Next() {
		var w EnrichmentWork
		p, err := scanProject(func(dest ...any) error { return rows.Scan(append(dest, &w.MetadataDue, &w.SourceDue)...) })
		if err != nil {
			return nil, err
		}
		w.Project = p
		items = append(items, w)
	}
	return items, rows.Err()
}
func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

// Error observations update scheduling/status only; prior verified evidence remains.
func (s *Store) WriteMetadata(ctx context.Context, mint string, r MetadataResult, observedAt, next time.Time) error {
	if r.Status != "ready" && r.Status != "unavailable" && r.Status != "error" {
		return errors.New("invalid metadata result status")
	}
	if r.ObservedSlot > math.MaxInt64 {
		return errors.New("invalid metadata observed slot")
	}
	if r.Status != "error" && observedAt.IsZero() {
		return errors.New("metadata observation time missing")
	}
	_, err := s.pool.Exec(ctx, `UPDATE solana_discovery.projects SET metadata_status=$2,metadata_next_at=$3,
 name=CASE WHEN $2 <> 'error' AND $4 <> '' THEN $4 ELSE name END,
 symbol=CASE WHEN $2 <> 'error' AND $5 <> '' THEN $5 ELSE symbol END,
 metadata_source=CASE WHEN $2 <> 'error' AND $6 <> '' THEN $6 ELSE metadata_source END,
 metadata_account=CASE WHEN $2 <> 'error' AND $7 <> '' THEN $7 ELSE metadata_account END,
 metadata_observed_slot=CASE WHEN $2 <> 'error' THEN $8 ELSE metadata_observed_slot END,
 metadata_updated_at=CASE WHEN $2 <> 'error' THEN $9 ELSE metadata_updated_at END WHERE mint=$1`, mint, r.Status, nullableTime(next), r.Name, r.Symbol, r.Source, r.Account, int64(r.ObservedSlot), nullableTime(observedAt))
	return err
}
func (s *Store) WriteSource(ctx context.Context, mint string, r SourceResult, next time.Time) error {
	if r.Status != "identified" && r.Status != "unrecognized" && r.Status != "error" {
		return errors.New("invalid source result status")
	}
	_, err := s.pool.Exec(ctx, `UPDATE solana_discovery.projects SET source_status=$2,source_next_at=$3,
 issuance_source=CASE WHEN $2 <> 'error' THEN $4 ELSE issuance_source END,
 issuance_program=CASE WHEN $2 <> 'error' THEN $5 ELSE issuance_program END WHERE mint=$1`, mint, r.Status, nullableTime(next), r.Source, r.Program)
	return err
}
