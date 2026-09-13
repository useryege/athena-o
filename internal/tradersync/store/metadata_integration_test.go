//go:build integration

package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestMetadataSchema(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	var name string
	if e := db.Pool.QueryRow(ctx, "SELECT current_database()").Scan(&name); e != nil {
		t.Fatal(e)
	}
	t.Log("isolated database", name)
	for _, table := range []string{"trader_sync_market_metadata", "trader_sync_combo_leg_index", "trader_sync_directory_refresh"} {
		var exists bool
		if e := db.Pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists); e != nil || !exists {
			t.Errorf("missing %s: %v", table, e)
		}
	}
}

func TestMetadataDirectoryPersistsPageFailureRestartAndCache(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := runtimeTestStore(t, db.Pool)
	page := pm.ComboMarket{ID: "8", ConditionID: "condition", PositionIDs: []string{"999999999999999999999999999999999999999999999999999999999"}}
	calls := 0
	fetch := func(_ context.Context, cursor string, limit int) (pm.ComboMarketPage, error) {
		calls++
		if cursor != "" || limit != 100 {
			t.Fatal(cursor, limit)
		}
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{page}, NextCursor: "next"}, nil
	}
	if _, e := s.RefreshComboPage(ctx, fetch); e != nil {
		t.Fatal(e)
	}
	if _, e := s.RefreshComboPage(ctx, fetch); e != nil || calls != 1 {
		t.Fatal("page rate not enforced", e, calls)
	}
	rows, e := s.LookupComboPosition(ctx, page.PositionIDs[0])
	if e != nil || len(rows) != 1 || rows[0].ID != "8" {
		t.Fatal(rows, e)
	}
	// Simulate process restart, advancing only the isolated fixture's schedule.
	s = runtimeTestStore(t, db.Pool)
	_, e = db.Pool.Exec(ctx, "UPDATE trader_sync_directory_refresh SET next_page_at=clock_timestamp()-interval '1 second'")
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.RefreshComboPage(ctx, func(_ context.Context, cursor string, _ int) (pm.ComboMarketPage, error) {
		if cursor != "next" {
			t.Fatal(cursor)
		}
		return pm.ComboMarketPage{}, errors.New("provider offline")
	})
	if e == nil {
		t.Fatal("failure hidden")
	}
	var cursor string
	var paced bool
	if e = db.Pool.QueryRow(ctx, "SELECT cursor,next_page_at>clock_timestamp() FROM trader_sync_directory_refresh").Scan(&cursor, &paced); e != nil || cursor != "next" || !paced {
		t.Fatal(cursor, paced, e)
	}
	_, _ = db.Pool.Exec(ctx, "UPDATE trader_sync_directory_refresh SET next_page_at=clock_timestamp()-interval '1 second'")
	_, e = s.RefreshComboPage(ctx, func(_ context.Context, cursor string, _ int) (pm.ComboMarketPage, error) {
		if cursor != "next" {
			t.Fatal(cursor)
		}
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{page}}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	var count int
	var completed bool
	if e = db.Pool.QueryRow(ctx, "SELECT count(*) FROM trader_sync_combo_leg_index").Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
	if e = db.Pool.QueryRow(ctx, "SELECT cursor,round_completed_at IS NOT NULL AND next_page_at>=round_started_at+interval '10 minutes' FROM trader_sync_directory_refresh").Scan(&cursor, &completed); e != nil || cursor != "" || !completed {
		t.Fatal(cursor, completed, e)
	}
	m := tm.TradeMetadata{Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "available"}, PositionID: page.PositionIDs[0], Title: "closed retained"}}
	if e = s.SaveMetadata(ctx, "key", m); e != nil {
		t.Fatal(e)
	}
	got, ok, e := NewSQLStore(db.Pool).LoadMetadata(ctx, "key")
	if e != nil || !ok || got.Market.Title != "closed retained" {
		t.Fatal(got, ok, e)
	}
	// A later empty directory round must not delete previously seen closed records.
	_, _ = db.Pool.Exec(ctx, "UPDATE trader_sync_directory_refresh SET next_page_at=clock_timestamp()-interval '1 second'")
	_, e = s.RefreshComboPage(ctx, func(context.Context, string, int) (pm.ComboMarketPage, error) {
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{}}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	rows, e = s.LookupComboPosition(ctx, page.PositionIDs[0])
	if e != nil || len(rows) != 1 {
		t.Fatal(rows, e)
	}
}
func TestMetadataDirectoryConcurrentInstancesAndAtomicInvalidPage(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	other, e := pgxpool.New(ctx, db.Pool.Config().ConnString())
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 2)
	var calls atomic.Int32
	fetch := func(context.Context, string, int) (pm.ComboMarketPage, error) {
		if calls.Add(1) == 1 {
			close(entered)
			<-release
		}
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{{ID: "1", ConditionID: "c", PositionIDs: []string{"1"}}}, NextCursor: "p2"}, nil
	}
	go func() { _, e := runtimeTestStore(t, db.Pool).RefreshComboPage(ctx, fetch); done <- e }()
	<-entered
	go func() { _, e := runtimeTestStore(t, other).RefreshComboPage(ctx, fetch); done <- e }()
	close(release)
	for i := 0; i < 2; i++ {
		if e := <-done; e != nil {
			t.Fatal(e)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("concurrent page requests", calls.Load())
	}
	_, _ = db.Pool.Exec(ctx, "UPDATE trader_sync_directory_refresh SET next_page_at=clock_timestamp()-interval '1 second'")
	_, e = runtimeTestStore(t, db.Pool).RefreshComboPage(ctx, func(context.Context, string, int) (pm.ComboMarketPage, error) {
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{{ID: "2", ConditionID: "c", PositionIDs: []string{"2"}}, {ID: "3", ConditionID: "c", PositionIDs: []string{"not-decimal"}}}, NextCursor: "p3"}, nil
	})
	if e == nil {
		t.Fatal("invalid page accepted")
	}
	var count int
	var cursor string
	if e = db.Pool.QueryRow(ctx, "SELECT count(*) FROM trader_sync_combo_leg_index").Scan(&count); e != nil || count != 1 {
		t.Fatal(count, e)
	}
	if e = db.Pool.QueryRow(ctx, "SELECT cursor FROM trader_sync_directory_refresh").Scan(&cursor); e != nil || cursor != "p2" {
		t.Fatal(cursor, e)
	}
}

func TestMetadataDirectoryLongRoundContinuesCursorAndPacesNextRound(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := runtimeTestStore(t, db.Pool)
	_, e := db.Pool.Exec(ctx, `INSERT INTO trader_sync_directory_refresh(name,cursor,round_started_at,next_page_at,visited_cursors) VALUES('combo_markets','mid-round',clock_timestamp()-interval '20 minutes',clock_timestamp()-interval '1 second',ARRAY[''])`)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.RefreshComboPage(ctx, func(_ context.Context, cursor string, _ int) (pm.ComboMarketPage, error) {
		if cursor != "mid-round" {
			t.Fatal("restarted unfinished round", cursor)
		}
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{}}, nil
	})
	if e != nil {
		t.Fatal(e)
	}
	var readySoon bool
	var elapsed bool
	if e = db.Pool.QueryRow(ctx, `SELECT next_page_at<clock_timestamp()+interval '2 seconds',round_completed_at-round_started_at>interval '19 minutes' FROM trader_sync_directory_refresh`).Scan(&readySoon, &elapsed); e != nil || !readySoon || !elapsed {
		t.Fatal(readySoon, elapsed, e)
	}
}

func TestMetadataDirectoryCancelledFetchPersistsPacingAcrossPools(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	other, err := pgxpool.New(ctx, db.Pool.Config().ConnString())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	_, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_directory_refresh(name,cursor,round_started_at,next_page_at) VALUES('combo_markets','resume-cursor',clock_timestamp(),clock_timestamp()-interval '1 second')`)
	if err != nil {
		t.Fatal(err)
	}
	parent, cancel := context.WithCancel(ctx)
	defer cancel()
	entered := make(chan struct{})
	firstDone := make(chan error, 1)
	go func() {
		_, err := runtimeTestStore(t, db.Pool).RefreshComboPage(parent, func(fetchCtx context.Context, cursor string, _ int) (pm.ComboMarketPage, error) {
			if cursor != "resume-cursor" {
				return pm.ComboMarketPage{}, fmt.Errorf("wrong cursor %q", cursor)
			}
			close(entered)
			<-fetchCtx.Done()
			return pm.ComboMarketPage{}, fetchCtx.Err()
		})
		firstDone <- err
	}()
	<-entered
	var retryCalls atomic.Int32
	retry := func(_ context.Context, cursor string, _ int) (pm.ComboMarketPage, error) {
		retryCalls.Add(1)
		if cursor != "resume-cursor" {
			return pm.ComboMarketPage{}, fmt.Errorf("wrong retry cursor %q", cursor)
		}
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{}, NextCursor: "after-retry"}, nil
	}
	secondDone := make(chan error, 1)
	go func() { _, err := runtimeTestStore(t, other).RefreshComboPage(ctx, retry); secondDone <- err }()
	cancel()
	select {
	case err := <-firstDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation lost: %v", err)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("cancellation cleanup unbounded")
	}
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("second pool stalled")
	}
	if retryCalls.Load() != 0 {
		t.Fatalf("second pool fetched before persisted cooldown: calls=%d", retryCalls.Load())
	}
	var cursor string
	var next time.Time
	var future bool
	if err = db.Pool.QueryRow(ctx, `SELECT cursor,next_page_at,next_page_at>clock_timestamp() FROM trader_sync_directory_refresh`).Scan(&cursor, &next, &future); err != nil || cursor != "resume-cursor" || !future {
		t.Fatal(cursor, next, future, err)
	}
	wait := time.Until(next) + 20*time.Millisecond
	if wait > 2*time.Second {
		t.Fatal("unexpected cooldown", wait)
	}
	if wait > 0 {
		time.Sleep(wait)
	}
	if _, err = runtimeTestStore(t, other).RefreshComboPage(ctx, retry); err != nil {
		t.Fatal(err)
	}
	if retryCalls.Load() != 1 {
		t.Fatal("eligible retry did not fetch exactly once", retryCalls.Load())
	}
}

func TestMetadataDirectoryCancellationReportsPacingCleanupFailure(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	_, err := db.Pool.Exec(ctx, `INSERT INTO trader_sync_directory_refresh(name,cursor,round_started_at,next_page_at) VALUES('combo_markets','resume',clock_timestamp(),clock_timestamp()-interval '1 second');
 CREATE FUNCTION reject_directory_pacing() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected pacing storage failure'; END $$;
 CREATE TRIGGER reject_pacing BEFORE UPDATE OF next_page_at ON trader_sync_directory_refresh FOR EACH ROW WHEN (OLD.admission_id IS NOT NULL AND NEW.admission_id IS NULL) EXECUTE FUNCTION reject_directory_pacing();`)
	if err != nil {
		t.Fatal(err)
	}
	parent, cancel := context.WithCancel(ctx)
	defer cancel()
	_, err = runtimeTestStore(t, db.Pool).RefreshComboPage(parent, func(context.Context, string, int) (pm.ComboMarketPage, error) {
		cancel()
		return pm.ComboMarketPage{}, parent.Err()
	})
	var storageErr *pgconn.PgError
	if !errors.Is(err, context.Canceled) || !errors.As(err, &storageErr) || !strings.Contains(err.Error(), "directory persist_pacing") {
		t.Fatalf("cleanup failure or original cancellation hidden: %v", err)
	}
}

func TestMetadataDirectoryPageWriteFailureRollsBackMappingsAndCursor(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	_, err := db.Pool.Exec(ctx, `INSERT INTO trader_sync_directory_refresh(name,cursor,round_started_at,next_page_at) VALUES('combo_markets','page',clock_timestamp(),clock_timestamp()-interval '1 second');
 CREATE FUNCTION reject_second_directory_mapping() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.position_id='2' THEN RAISE EXCEPTION 'injected second mapping failure'; END IF; RETURN NEW; END $$;
 CREATE TRIGGER reject_mapping BEFORE INSERT ON trader_sync_combo_leg_index FOR EACH ROW EXECUTE FUNCTION reject_second_directory_mapping();`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtimeTestStore(t, db.Pool).RefreshComboPage(ctx, func(context.Context, string, int) (pm.ComboMarketPage, error) {
		return pm.ComboMarketPage{Markets: []pm.ComboMarket{{ID: "1", ConditionID: "c", PositionIDs: []string{"1", "2"}}}, NextCursor: "next"}, nil
	})
	if err == nil {
		t.Fatal("second mapping failure hidden")
	}
	var count int
	var cursor string
	if err = db.Pool.QueryRow(ctx, "SELECT count(*) FROM trader_sync_combo_leg_index").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, "SELECT cursor FROM trader_sync_directory_refresh").Scan(&cursor); err != nil {
		t.Fatal(err)
	}
	if count != 0 || cursor != "page" {
		t.Fatalf("partial page committed: mappings=%d cursor=%q", count, cursor)
	}
}

func TestMetadataPreservesKnownPartialAcrossTransientFailure(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	s := runtimeTestStore(t, db.Pool)
	ctx := context.Background()
	original := tm.TradeMetadata{Market: tm.MarketRef{PositionID: "100"}, LegsEvidence: tm.Evidence{Availability: "available"}, Legs: []tm.ComboLeg{{PositionID: "1", Market: tm.MarketRef{PositionID: "1", ID: "11", Evidence: tm.Evidence{Availability: "available", Source: "gamma"}}}, {PositionID: "2", Market: tm.MarketRef{PositionID: "2", Evidence: tm.Evidence{Availability: "unavailable", ReasonCode: "market_not_found"}}}}}
	if err := s.SaveMetadata(ctx, "combo:100:hash", original); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMetadata(ctx, "combo:100:hash", tm.TradeMetadata{Market: tm.MarketRef{PositionID: "100", Evidence: tm.Evidence{Availability: "unavailable", ReasonCode: "module_call_failed"}}}); err != nil {
		t.Fatal(err)
	}
	got, _, err := s.LoadMetadata(ctx, "combo:100:hash")
	if err != nil || len(got.Legs) != 2 || got.Legs[0].Market.ID != "11" {
		t.Fatalf("transient failure erased known legs: %+v %v", got, err)
	}
}

func TestMetadataConcurrentOldAvailableCannotHideExplicitConflict(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	other, e := pgxpool.New(ctx, db.Pool.Config().ConnString())
	if e != nil {
		t.Fatal(e)
	}
	defer other.Close()
	one, two := runtimeTestStore(t, db.Pool), runtimeTestStore(t, other)
	old := tm.TradeMetadata{Market: tm.MarketRef{PositionID: "1", ID: "known", Evidence: tm.Evidence{Availability: "available"}}, LegsEvidence: tm.Evidence{Availability: "available"}, Legs: []tm.ComboLeg{{PositionID: "2", Market: tm.MarketRef{PositionID: "2", Evidence: tm.Evidence{Availability: "available"}, ID: "leg"}}}}
	conflict := old
	conflict.Market.Availability = "unavailable"
	conflict.Market.ReasonCode = "condition_conflict"
	done := make(chan error, 2)
	go func() { done <- one.SaveMetadata(ctx, "same-version:1:hash", old) }()
	go func() { done <- two.SaveMetadata(ctx, "same-version:1:hash", conflict) }()
	for range 2 {
		if e = <-done; e != nil {
			t.Fatal(e)
		}
	}
	if e = one.SaveMetadata(ctx, "same-version:1:hash", old); e != nil {
		t.Fatal(e)
	}
	got, found, e := two.LoadMetadata(ctx, "same-version:1:hash")
	if e != nil || !found || got.Market.Availability != "unavailable" || got.Market.ReasonCode != "condition_conflict" || len(got.Legs) != 1 {
		t.Fatal(got, found, e)
	}
	if _, found, e = two.LoadMetadata(ctx, "same-version:1:other-hash"); e != nil || found {
		t.Fatal("merged across evidence keys", found, e)
	}
}
