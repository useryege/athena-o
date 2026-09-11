//go:build integration

package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestDirectoryAdmissionSurvivesPostSendRollback(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	other, err := pgxpool.New(ctx, db.Pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(http.StatusOK) }))
	defer server.Close()
	fetch := func(ctx context.Context, cursor string, limit int) (pm.ComboMarketPage, error) {
		if cursor != "" || limit != 100 {
			t.Errorf("request boundary %q %d", cursor, limit)
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		resp, e := server.Client().Do(req)
		if e != nil {
			return pm.ComboMarketPage{}, e
		}
		resp.Body.Close()
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{{ID: "17", ConditionID: "condition", PositionIDs: []string{"123"}}}, NextCursor: "next"}, nil
	}
	if _, err = db.Pool.Exec(ctx, `CREATE FUNCTION fail_directory_mapping() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'injected mapping failure'; END$$; CREATE TRIGGER fail_directory_mapping BEFORE INSERT ON trader_sync_combo_leg_index FOR EACH ROW EXECUTE FUNCTION fail_directory_mapping()`); err != nil {
		t.Fatal(err)
	}
	_, err = NewSQLStore(db.Pool).RefreshComboPage(ctx, fetch)
	if err == nil {
		t.Fatal("mapping fault was not exercised")
	}
	if requests.Load() != 1 {
		t.Fatal("provider was not called before rollback")
	}
	var mappings int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_combo_leg_index`).Scan(&mappings); err != nil || mappings != 0 {
		t.Fatal(mappings, err)
	}
	// A different instance must observe the committed admission even though the
	// mapping/cursor transaction failed after the provider request returned.
	if _, err = other.Exec(ctx, `DROP TRIGGER fail_directory_mapping ON trader_sync_combo_leg_index`); err != nil {
		t.Fatal(err)
	}
	next, err := NewSQLStore(other).RefreshComboPage(ctx, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Fatalf("post-send rollback lost shared pacing: requests=%d", requests.Load())
	}
	var cursor string
	var wait time.Duration
	if err = other.QueryRow(ctx, `SELECT cursor,EXTRACT(EPOCH FROM(next_page_at-clock_timestamp()))::double precision FROM trader_sync_directory_refresh WHERE name='combo_markets'`).Scan(&cursor, new(float64)); err != nil {
		t.Fatal(err)
	}
	if cursor != "" || next.IsZero() {
		t.Fatal("rollback advanced cursor or lost reservation", cursor, next)
	}
	var seconds float64
	if err = other.QueryRow(ctx, `SELECT EXTRACT(EPOCH FROM(next_page_at-clock_timestamp()))::double precision FROM trader_sync_directory_refresh WHERE name='combo_markets'`).Scan(&seconds); err != nil {
		t.Fatal(err)
	}
	wait = time.Duration(seconds * float64(time.Second))
	if wait > 0 {
		timer := time.NewTimer(wait + 20*time.Millisecond)
		defer timer.Stop()
		<-timer.C
	}
	if _, err = NewSQLStore(other).RefreshComboPage(ctx, fetch); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 {
		t.Fatal("due admission did not recover", requests.Load())
	}
}

// directoryFaults decorates real PG transactions: every query still reaches the
// isolated database, while commit acknowledgements/connection outcomes are lost.
type directoryFaults struct {
	pool  *pgxpool.Pool
	count int
	wrap  func(int, pgx.Tx) pgx.Tx
}

func (f *directoryFaults) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	tx, e := f.pool.BeginTx(ctx, opts)
	if e != nil {
		return nil, e
	}
	f.count++
	return f.wrap(f.count, tx), nil
}

type directoryFaultTx struct {
	pgx.Tx
	commit func(context.Context) error
}

func (t *directoryFaultTx) Commit(ctx context.Context) error { return t.commit(ctx) }

func TestDirectoryAdmissionUnknownAndSettlementFailure(t *testing.T) {
	for _, mode := range []string{"admission_committed_ack_lost", "admission_rolled_back_ack_lost", "provider_error_pacing_rollback", "page_committed_ack_lost", "page_rolled_back_ack_lost"} {
		t.Run(mode, func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			other, e := pgxpool.New(ctx, db.Pool.Config().ConnString())
			if e != nil {
				t.Fatal(e)
			}
			defer other.Close()
			s := NewSQLStore(db.Pool)
			requests := 0
			fetch := func(context.Context, string, int) (pm.ComboMarketPage, error) {
				requests++
				if mode == "provider_error_pacing_rollback" {
					return pm.ComboMarketPage{}, errors.New("provider failure")
				}
				return pm.ComboMarketPage{Markets: []pm.ComboMarket{{ID: "17", ConditionID: "c", PositionIDs: []string{"123"}}}, NextCursor: "next"}, nil
			}
			s.directoryTransactions = &directoryFaults{pool: db.Pool, wrap: func(n int, tx pgx.Tx) pgx.Tx {
				if (strings.HasPrefix(mode, "admission_") && n == 1) || (!strings.HasPrefix(mode, "admission_") && n == 2) {
					return &directoryFaultTx{Tx: tx, commit: func(c context.Context) error {
						if strings.Contains(mode, "committed") {
							if err := tx.Commit(c); err != nil {
								t.Fatal(err)
							}
						} else {
							if err := tx.Rollback(c); err != nil {
								t.Fatal(err)
							}
						}
						return errors.New("injected missing commit ACK")
					}}
				}
				return tx
			}}
			_, e = s.RefreshComboPage(ctx, fetch)
			var detail *tm.DirectoryError
			if !errors.As(e, &detail) || detail.CommitKnown {
				t.Fatalf("unknown commit outcome missing: %v", e)
			}
			if detail.HTTPAttempted != (!strings.HasPrefix(mode, "admission_")) {
				t.Fatal("incorrect request phase", detail)
			}
			expectedRequests := 1
			if strings.HasPrefix(mode, "admission_") {
				expectedRequests = 0
			}
			if requests != expectedRequests {
				t.Fatal(requests)
			}
			var cursor string
			var mappings int
			// A rolled-back first admission also rolled back creation of the state row.
			e = other.QueryRow(ctx, `SELECT cursor FROM trader_sync_directory_refresh WHERE name='combo_markets'`).Scan(&cursor)
			if mode == "admission_rolled_back_ack_lost" {
				if !errors.Is(e, pgx.ErrNoRows) {
					t.Fatal(e)
				}
			} else if e != nil {
				t.Fatal(e)
			}
			if e = other.QueryRow(ctx, `SELECT count(*) FROM trader_sync_combo_leg_index`).Scan(&mappings); e != nil {
				t.Fatal(e)
			}
			if mode == "page_committed_ack_lost" {
				if cursor != "next" || mappings != 1 {
					t.Fatal(cursor, mappings)
				}
			} else if cursor != "" || mappings != 0 {
				t.Fatal("mapping and cursor must commit atomically", cursor, mappings)
			}
			before := requests
			_, e = NewSQLStore(other).RefreshComboPage(ctx, fetch)
			if e != nil {
				t.Fatal(e)
			}
			if mode == "admission_rolled_back_ack_lost" {
				if requests != before+1 {
					t.Fatal("uncommitted admission should be recoverable")
				}
			} else if requests != before {
				t.Fatal("another instance bypassed persistent reservation", requests, before)
			}
		})
	}
}

func TestDirectoryAdmissionOriginalDeadlineIncludesAckAndLockWait(t *testing.T) {
	for _, mode := range []string{"late_ack", "second_lock_wait"} {
		t.Run(mode, func(t *testing.T) {
			t.Parallel()
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			other, e := pgxpool.New(ctx, db.Pool.Config().ConnString())
			if e != nil {
				t.Fatal(e)
			}
			defer other.Close()
			var blocker pgx.Tx
			defer func() {
				if blocker != nil {
					_ = blocker.Rollback(ctx)
				}
			}()
			s := NewSQLStore(db.Pool)
			s.directoryTransactions = &directoryFaults{pool: db.Pool, wrap: func(n int, tx pgx.Tx) pgx.Tx {
				if n != 1 {
					return tx
				}
				return &directoryFaultTx{Tx: tx, commit: func(c context.Context) error {
					if err := tx.Commit(c); err != nil {
						return err
					}
					if mode == "late_ack" {
						timer := time.NewTimer(5100 * time.Millisecond)
						defer timer.Stop()
						<-timer.C
					} else {
						var err error
						blocker, err = other.Begin(ctx)
						if err != nil {
							t.Fatal(err)
						}
						if _, err = blocker.Exec(ctx, `SELECT 1 FROM trader_sync_directory_refresh WHERE name='combo_markets' FOR UPDATE`); err != nil {
							t.Fatal(err)
						}
					}
					return nil
				}}
			}}
			requests := 0
			started := time.Now()
			_, e = s.RefreshComboPage(ctx, func(context.Context, string, int) (pm.ComboMarketPage, error) {
				requests++
				return pm.ComboMarketPage{Markets: []pm.ComboMarket{}}, nil
			})
			if !errors.Is(e, context.DeadlineExceeded) || requests != 0 {
				t.Fatal("expired admission sent HTTP", e, requests)
			}
			if elapsed := time.Since(started); elapsed > 6500*time.Millisecond {
				t.Fatal("second transaction reset budget", elapsed)
			}
			if blocker != nil {
				_ = blocker.Rollback(ctx)
				blocker = nil
			}
			var reserved bool
			if e = other.QueryRow(ctx, `SELECT admission_id IS NOT NULL FROM trader_sync_directory_refresh WHERE name='combo_markets'`).Scan(&reserved); e != nil || !reserved {
				t.Fatal("expired caller erased admission", reserved, e)
			}
		})
	}
}

func TestDirectoryLostConnectionCancelsOldHTTPAndRejectsStaleToken(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	other, e := pgxpool.New(ctx, db.Pool.Config().ConnString())
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	entered := make(chan struct{})
	cancelled := make(chan struct{})
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); close(cancelled) }))
	defer provider.Close()
	s := NewSQLStore(db.Pool)
	var pid uint32
	s.directoryTransactions = &directoryFaults{pool: db.Pool, wrap: func(n int, tx pgx.Tx) pgx.Tx {
		if n == 2 {
			pid = tx.Conn().PgConn().PID()
		}
		return tx
	}}
	done := make(chan error, 1)
	go func() {
		_, err := s.RefreshComboPage(ctx, func(c context.Context, _ string, _ int) (pm.ComboMarketPage, error) {
			req, _ := http.NewRequestWithContext(c, http.MethodGet, provider.URL, nil)
			response, err := provider.Client().Do(req)
			if response != nil {
				response.Body.Close()
			}
			return pm.ComboMarketPage{}, err
		})
		done <- err
	}()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("provider not entered")
	}
	var oldToken pgtype.UUID
	if e = other.QueryRow(ctx, `SELECT admission_id FROM trader_sync_directory_refresh WHERE name='combo_markets'`).Scan(&oldToken); e != nil {
		t.Fatal(e)
	}
	if _, e = other.Exec(ctx, `SELECT pg_terminate_backend($1)`, pid); e != nil {
		t.Fatal(e)
	}
	requests := 0
	fetch := func(context.Context, string, int) (pm.ComboMarketPage, error) {
		requests++
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{{ID: "18", ConditionID: "new", PositionIDs: []string{"124"}}}, NextCursor: "new-cursor"}, nil
	}
	next, e := NewSQLStore(other).RefreshComboPage(ctx, fetch)
	if e != nil || requests != 0 {
		t.Fatal("lost lock bypassed reservation", requests, e)
	}
	select {
	case <-cancelled:
	case <-time.After(6 * time.Second):
		t.Fatal("old network call did not end within original deadline")
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("lost page transaction reported success")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("old page cleanup did not join")
	}
	if wait := time.Until(next); wait > 0 {
		timer := time.NewTimer(wait + 20*time.Millisecond)
		<-timer.C
	}
	if _, e = NewSQLStore(other).RefreshComboPage(ctx, fetch); e != nil || requests != 1 {
		t.Fatal("successor could not acquire due admission", requests, e)
	}
	_, e = q.New(other).AdvanceComboPage(ctx, q.AdvanceComboPageParams{NextCursor: "stale", ExpectedCursor: "", AdmissionID: oldToken})
	if !errors.Is(e, pgx.ErrNoRows) {
		t.Fatal("old token overwrote successor", e)
	}
	var cursor string
	var count int
	if e = other.QueryRow(ctx, `SELECT cursor FROM trader_sync_directory_refresh WHERE name='combo_markets'`).Scan(&cursor); e != nil || cursor != "new-cursor" {
		t.Fatal(cursor, e)
	}
	if e = other.QueryRow(ctx, `SELECT count(*) FROM trader_sync_combo_leg_index WHERE market_id='18'`).Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
}
