package solanadiscovery

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

var ErrCheckpointConflict = errors.New("Solana discovery checkpoint changed concurrently")

// Store borrows the caller's pool; the process that created it closes it.
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) Migrate(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return errors.New("Solana discovery store has no database")
	}
	sql, err := migrationFiles.ReadFile("migrations/001_init.sql")
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, string(sql))
	return err
}

// Initialize records the first scan start exactly once, including across restarts.
func (s *Store) Initialize(ctx context.Context, startSlot uint64) error {
	if startSlot == 0 || startSlot > math.MaxInt64 {
		return fmt.Errorf("invalid start slot %d", startSlot)
	}
	_, err := s.pool.Exec(ctx, `INSERT INTO solana_discovery.scan_state
        (id, start_slot, last_processed_slot) VALUES (1, $1, $2)
		ON CONFLICT (id) DO UPDATE SET
		start_slot = COALESCE(solana_discovery.scan_state.start_slot, EXCLUDED.start_slot),
		last_processed_slot = COALESCE(solana_discovery.scan_state.last_processed_slot, EXCLUDED.last_processed_slot)`,
		int64(startSlot), int64(startSlot-1))
	return err
}

// CommitRange saves every candidate and its range checkpoint in one short transaction.
func (s *Store) CommitRange(ctx context.Context, expectedLastProcessed, lastSlot uint64, projects []Project) error {
	if expectedLastProcessed > math.MaxInt64 || lastSlot > math.MaxInt64 || lastSlot <= expectedLastProcessed {
		return fmt.Errorf("invalid range %d..%d", expectedLastProcessed, lastSlot)
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var current int64
	if err := tx.QueryRow(ctx, `SELECT last_processed_slot FROM solana_discovery.scan_state WHERE id = 1 FOR UPDATE`).Scan(&current); err != nil {
		return err
	}
	if current != int64(expectedLastProcessed) {
		return ErrCheckpointConflict
	}
	for _, p := range projects {
		if p.Slot <= expectedLastProcessed || p.Slot > lastSlot || !validPublicKey(p.Mint) {
			return fmt.Errorf("candidate %q is outside range or has invalid Mint", p.Mint)
		}
		_, err = tx.Exec(ctx, `INSERT INTO solana_discovery.projects
            (mint, token_program, signature, fee_payer, mint_authority, freeze_authority, decimals, slot, block_time)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (mint) DO NOTHING`,
			p.Mint, p.TokenProgram, p.Signature, p.FeePayer, p.MintAuthority, p.FreezeAuthority,
			int32(p.Decimals), int64(p.Slot), p.BlockTime)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE solana_discovery.scan_state
        SET last_processed_slot = $1, last_success_at = now(), last_error = '' WHERE id = 1`, int64(lastSlot))
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SetLatestFinalized(ctx context.Context, slot uint64) error {
	if slot > math.MaxInt64 {
		return fmt.Errorf("invalid finalized slot %d", slot)
	}
	_, err := s.pool.Exec(ctx, `UPDATE solana_discovery.scan_state
		SET latest_finalized_slot = GREATEST(latest_finalized_slot, $1),
        last_error = CASE WHEN last_processed_slot >= $1 THEN '' ELSE last_error END WHERE id = 1`, int64(slot))
	return err
}

func (s *Store) RecordError(ctx context.Context, message string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO solana_discovery.scan_state (id, last_error)
		VALUES (1, $1) ON CONFLICT (id) DO UPDATE SET last_error = EXCLUDED.last_error`, message)
	return err
}

func (s *Store) GetDiscoveryStatus(ctx context.Context) (DiscoveryStatus, error) {
	var status DiscoveryStatus
	var start, processed *int64
	var latest int64
	var success *time.Time
	err := s.pool.QueryRow(ctx, `SELECT start_slot, last_processed_slot, latest_finalized_slot,
        last_success_at, last_error FROM solana_discovery.scan_state WHERE id = 1`).
		Scan(&start, &processed, &latest, &success, &status.LastError)
	if errors.Is(err, pgx.ErrNoRows) {
		status.Status = "starting"
		return status, nil
	}
	if err != nil {
		return status, err
	}
	if start != nil {
		status.StartSlot = uint64(*start)
	}
	if processed != nil {
		status.LastProcessedSlot = uint64(*processed)
	}
	status.LatestFinalizedSlot = uint64(latest)
	if success != nil {
		status.LastSuccessAt = *success
	}
	err = s.pool.QueryRow(ctx, `SELECT count(*) FROM solana_discovery.projects`).Scan(&status.TotalProjects)
	if err != nil {
		return status, err
	}
	switch {
	case status.LastError != "":
		status.Status = "error"
	case start == nil:
		status.Status = "starting"
	case status.LastProcessedSlot < status.LatestFinalizedSlot:
		status.Status = "catching_up"
	default:
		status.Status = "current"
	}
	return status, nil
}

func (s *Store) ListProjects(ctx context.Context, page, pageSize uint32, query string) ([]Project, int64, error) {
	if page < 1 || pageSize < 1 || pageSize > 100 || len([]rune(query)) > 128 {
		return nil, 0, errors.New("invalid project page or query")
	}
	var total int64
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM solana_discovery.projects
        WHERE ($1 = '' OR position($1 in mint) > 0)`, query).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	offset := (uint64(page) - 1) * uint64(pageSize)
	if offset > math.MaxInt64 {
		return nil, 0, errors.New("page offset is too large")
	}
	rows, err := s.pool.Query(ctx, `SELECT mint, token_program, signature, fee_payer,
        mint_authority, freeze_authority, decimals, slot, block_time, discovered_at
        FROM solana_discovery.projects WHERE ($1 = '' OR position($1 in mint) > 0)
        ORDER BY slot DESC, mint ASC LIMIT $2 OFFSET $3`, query, int64(pageSize), int64(offset))
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]Project, 0)
	for rows.Next() {
		var p Project
		var decimals int32
		var slot int64
		if err := rows.Scan(&p.Mint, &p.TokenProgram, &p.Signature, &p.FeePayer,
			&p.MintAuthority, &p.FreezeAuthority, &decimals, &slot, &p.BlockTime, &p.DiscoveredAt); err != nil {
			return nil, 0, err
		}
		p.Decimals, p.Slot = uint32(decimals), uint64(slot)
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
