package ave

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
	utilave "github.com/useryege/athena/util/ave"
)

type fakeFetcher struct {
	response *utilave.TokenDetailResponse
	err      error
	calls    int
	tokenID  string
}

func (f *fakeFetcher) FetchDetail(_ context.Context, tokenID string) (*utilave.TokenDetailResponse, error) {
	f.calls++
	f.tokenID = tokenID
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

type fakeCache struct {
	details map[common.Address]appstore.ProjectAveDetail
}

func (c *fakeCache) SetAveDetail(_ context.Context, _ int64, contract common.Address, item appstore.ProjectAveDetail) error {
	if c.details == nil {
		c.details = map[common.Address]appstore.ProjectAveDetail{}
	}
	c.details[contract] = item
	return nil
}

func (c *fakeCache) GetAveDetail(_ context.Context, _ int64, contract common.Address) (*appstore.ProjectAveDetail, bool, error) {
	item, ok := c.details[contract]
	if !ok {
		return nil, false, nil
	}
	return &item, true, nil
}

type fakeStore struct {
	details    map[common.Address]appstore.ProjectAveDetail
	states     map[common.Address]appstore.ProjectComponentState
	candidates []common.Address
}

func (s *fakeStore) UpsertProjectAveDetail(_ context.Context, _ int64, contract common.Address, detail appstore.ProjectAveDetail) error {
	if s.details == nil {
		s.details = map[common.Address]appstore.ProjectAveDetail{}
	}
	s.details[contract] = detail
	return nil
}

func (s *fakeStore) GetProjectAveDetail(_ context.Context, _ int64, contract common.Address) (*appstore.ProjectAveDetail, error) {
	item, ok := s.details[contract]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (s *fakeStore) ListProjectAveDetailsByContracts(_ context.Context, _ int64, contracts []common.Address) (map[common.Address]appstore.ProjectAveDetail, error) {
	result := map[common.Address]appstore.ProjectAveDetail{}
	for _, contract := range contracts {
		if item, ok := s.details[contract]; ok {
			result[contract] = item
		}
	}
	return result, nil
}

func (s *fakeStore) ListProjectAveRefreshCandidates(context.Context, int64, time.Time, time.Time, int32) ([]common.Address, error) {
	return append([]common.Address(nil), s.candidates...), nil
}

func (s *fakeStore) ScheduleProjectAveRefresh(_ context.Context, _ int64, contract common.Address, nextRunAt time.Time) error {
	s.setState(contract, appstore.ProjectComponentStatusPending, time.Time{}, time.Time{}, nextRunAt, "")
	return nil
}

func (s *fakeStore) MarkProjectAveRefreshRunning(_ context.Context, _ int64, contract common.Address, at time.Time) error {
	s.setState(contract, appstore.ProjectComponentStatusRunning, at, time.Time{}, at, "")
	return nil
}

func (s *fakeStore) MarkProjectAveRefreshSuccess(_ context.Context, _ int64, contract common.Address, successAt time.Time, nextRunAt time.Time) error {
	s.setState(contract, appstore.ProjectComponentStatusSuccess, successAt, successAt, nextRunAt, "")
	return nil
}

func (s *fakeStore) MarkProjectAveRefreshFailed(_ context.Context, _ int64, contract common.Address, attemptAt time.Time, nextRunAt time.Time, lastError string) error {
	s.setState(contract, appstore.ProjectComponentStatusFailed, attemptAt, time.Time{}, nextRunAt, lastError)
	return nil
}

func (s *fakeStore) GetProjectAveComponentState(_ context.Context, _ int64, contract common.Address) (*appstore.ProjectComponentState, error) {
	item, ok := s.states[contract]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (s *fakeStore) setState(contract common.Address, status string, lastAttemptAt time.Time, lastSuccessAt time.Time, nextRunAt time.Time, lastError string) {
	if s.states == nil {
		s.states = map[common.Address]appstore.ProjectComponentState{}
	}
	s.states[contract] = appstore.ProjectComponentState{
		ProjectContract: contract,
		Component:       appstore.ProjectComponentAveDetail,
		Status:          status,
		LastAttemptAt:   lastAttemptAt,
		LastSuccessAt:   lastSuccessAt,
		NextRunAt:       nextRunAt,
		LastError:       lastError,
	}
}

func TestScheduleRefreshSetsPendingState(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	store := &fakeStore{}
	component, err := NewComponent(Options{
		Store:   store,
		Fetcher: &fakeFetcher{},
		Chain:   "bsc",
		Now:     func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new component: %v", err)
	}

	if err := component.ScheduleRefresh(ctx, contract); err != nil {
		t.Fatalf("schedule refresh: %v", err)
	}

	state := store.states[contract]
	if state.Status != appstore.ProjectComponentStatusPending || !state.NextRunAt.Equal(now) {
		t.Fatalf("state = %#v, want pending at now", state)
	}
}

func TestStateReturnsDetailAndStaleFlag(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	store := &fakeStore{
		details: map[common.Address]appstore.ProjectAveDetail{
			contract: {FetchedAt: now.Add(-time.Hour), Token: appstore.ProjectAveTokenDetail{LogoURL: "https://example.com/logo.png"}},
		},
	}
	component, err := NewComponent(Options{
		Store:     store,
		Fetcher:   &fakeFetcher{},
		Chain:     "bsc",
		DetailTTL: 24 * time.Hour,
		Now:       func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new component: %v", err)
	}

	state, err := component.State(ctx, contract)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if !state.DetailAvailable || state.Stale {
		t.Fatalf("state = %#v, want available and fresh", state)
	}
}

func TestRunOnceRefreshSuccessWritesDetailCacheAndState(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000A2")
	store := &fakeStore{candidates: []common.Address{contract}}
	cache := &fakeCache{}
	fetcher := &fakeFetcher{response: &utilave.TokenDetailResponse{
		Status:   1,
		Msg:      "SUCCESS",
		DataType: 1,
		Data: utilave.TokenDetailData{
			Token:     utilave.Token{LogoURL: " https://example.com/logo.png ", Token: "token", Chain: "bsc"},
			Pairs:     []utilave.Pair{{Pair: "pair-1", Chain: "bsc"}},
			IsAudited: true,
		},
	}}
	component, err := NewComponent(Options{
		Store:     store,
		Cache:     cache,
		Fetcher:   fetcher,
		Chain:     "bsc",
		DetailTTL: 24 * time.Hour,
		Now:       func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new component: %v", err)
	}

	component.runOnce(ctx)

	if fetcher.calls != 1 {
		t.Fatalf("fetcher calls = %d, want 1", fetcher.calls)
	}
	if fetcher.tokenID != strings.ToLower(contract.Hex())+"-bsc" {
		t.Fatalf("token id = %q, want contract-bsc", fetcher.tokenID)
	}
	detail := store.details[contract]
	if detail.Token.LogoURL != "https://example.com/logo.png" || len(detail.Pairs) != 1 {
		t.Fatalf("detail = %#v, want trimmed logo and pair", detail)
	}
	if cache.details[contract].Token.LogoURL != "https://example.com/logo.png" {
		t.Fatalf("cached detail = %#v, want cached logo", cache.details[contract])
	}
	state := store.states[contract]
	if state.Status != appstore.ProjectComponentStatusSuccess || !state.LastSuccessAt.Equal(now) || !state.NextRunAt.Equal(now.Add(24*time.Hour)) {
		t.Fatalf("state = %#v, want success with next run", state)
	}
}

func TestRunOnceRefreshFailureWritesRetryState(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a3")
	store := &fakeStore{candidates: []common.Address{contract}}
	component, err := NewComponent(Options{
		Store:             store,
		Fetcher:           &fakeFetcher{err: errors.New("ave down")},
		Chain:             "bsc",
		FailureRetryDelay: 10 * time.Minute,
		Now:               func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("new component: %v", err)
	}

	component.runOnce(ctx)

	state := store.states[contract]
	if state.Status != appstore.ProjectComponentStatusFailed || state.LastError != "ave down" || !state.NextRunAt.Equal(now.Add(10*time.Minute)) {
		t.Fatalf("state = %#v, want failed retry", state)
	}
	if _, ok := store.details[contract]; ok {
		t.Fatalf("detail was written on failure")
	}
}
