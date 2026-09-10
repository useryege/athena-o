//go:build integration

package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
	"sync/atomic"
	"testing"
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
	s := NewSQLStore(db.Pool)
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
	s = NewSQLStore(db.Pool)
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
	go func() { _, e := NewSQLStore(db.Pool).RefreshComboPage(ctx, fetch); done <- e }()
	<-entered
	go func() { _, e := NewSQLStore(other).RefreshComboPage(ctx, fetch); done <- e }()
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
	_, e = NewSQLStore(db.Pool).RefreshComboPage(ctx, func(context.Context, string, int) (pm.ComboMarketPage, error) {
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
	s := NewSQLStore(db.Pool)
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
