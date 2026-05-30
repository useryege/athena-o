package store

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

type fakeApplicationQuerier struct {
	addCommentParams      appsqlc.AddProjectCommentParams
	listCommentParams     appsqlc.ListProjectCommentsByContractParams
	addEventLogParams     appsqlc.AddProjectEventLogParams
	listEventLogsContract []byte
	listAveRefreshParams  appsqlc.ListProjectAveRefreshCandidatesParams
	scheduleAveParams     appsqlc.ScheduleProjectAveRefreshParams
	markAveRunningParams  appsqlc.MarkProjectAveRefreshRunningParams
	markAveSuccessParams  appsqlc.MarkProjectAveRefreshSuccessParams
	markAveFailedParams   appsqlc.MarkProjectAveRefreshFailedParams
	aveStateContract      []byte

	addCommentResult   appsqlc.ProjectComment
	countCommentResult int64
	listCommentResult  []appsqlc.ProjectComment
	listEventLogResult []appsqlc.ProjectEventLog
	listAveResult      [][]byte
	aveStateResult     appsqlc.ProjectComponentState
}

func (f *fakeApplicationQuerier) AddProjectComment(_ context.Context, arg appsqlc.AddProjectCommentParams) (appsqlc.ProjectComment, error) {
	f.addCommentParams = arg
	return f.addCommentResult, nil
}

func (f *fakeApplicationQuerier) AddProjectEventLog(_ context.Context, arg appsqlc.AddProjectEventLogParams) error {
	f.addEventLogParams = arg
	return nil
}

func (f *fakeApplicationQuerier) CountProjectCommentsByContract(context.Context, []byte) (int64, error) {
	return f.countCommentResult, nil
}

func (f *fakeApplicationQuerier) GetProjectAveComponentState(_ context.Context, contract []byte) (appsqlc.ProjectComponentState, error) {
	f.aveStateContract = contract
	return f.aveStateResult, nil
}

func (f *fakeApplicationQuerier) ListProjectAveRefreshCandidates(_ context.Context, arg appsqlc.ListProjectAveRefreshCandidatesParams) ([][]byte, error) {
	f.listAveRefreshParams = arg
	return f.listAveResult, nil
}

func (f *fakeApplicationQuerier) ListProjectCommentsByContract(_ context.Context, arg appsqlc.ListProjectCommentsByContractParams) ([]appsqlc.ProjectComment, error) {
	f.listCommentParams = arg
	return f.listCommentResult, nil
}

func (f *fakeApplicationQuerier) ListProjectEventLogsByContract(_ context.Context, contract []byte) ([]appsqlc.ProjectEventLog, error) {
	f.listEventLogsContract = contract
	return f.listEventLogResult, nil
}

func (f *fakeApplicationQuerier) MarkProjectAveRefreshFailed(_ context.Context, arg appsqlc.MarkProjectAveRefreshFailedParams) error {
	f.markAveFailedParams = arg
	return nil
}

func (f *fakeApplicationQuerier) MarkProjectAveRefreshRunning(_ context.Context, arg appsqlc.MarkProjectAveRefreshRunningParams) error {
	f.markAveRunningParams = arg
	return nil
}

func (f *fakeApplicationQuerier) MarkProjectAveRefreshSuccess(_ context.Context, arg appsqlc.MarkProjectAveRefreshSuccessParams) error {
	f.markAveSuccessParams = arg
	return nil
}

func (f *fakeApplicationQuerier) ScheduleProjectAveRefresh(_ context.Context, arg appsqlc.ScheduleProjectAveRefreshParams) error {
	f.scheduleAveParams = arg
	return nil
}

func TestProjectCommentsUseQuerier(t *testing.T) {
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	createdAt := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	querier := &fakeApplicationQuerier{
		addCommentResult: appsqlc.ProjectComment{
			ID:              7,
			ProjectContract: contract.Bytes(),
			Username:        "alice",
			Content:         "hello",
			CreatedAt:       pgtype.Timestamptz{Time: createdAt, Valid: true},
		},
		countCommentResult: 1,
		listCommentResult: []appsqlc.ProjectComment{{
			ID:              7,
			ProjectContract: contract.Bytes(),
			Username:        "alice",
			Content:         "hello",
			CreatedAt:       pgtype.Timestamptz{Time: createdAt, Valid: true},
		}},
	}
	store := NewSQLStoreWithQuerier(querier)

	created, err := store.AddProjectComment(context.Background(), ProjectComment{
		Contract: contract,
		Username: " alice ",
		Content:  " hello ",
	})
	if err != nil {
		t.Fatalf("AddProjectComment: %v", err)
	}
	if created.ID != 7 || created.Username != "alice" || created.Content != "hello" {
		t.Fatalf("created = %#v, want generated comment mapping", created)
	}
	if querier.addCommentParams.Username != "alice" || querier.addCommentParams.Content != "hello" {
		t.Fatalf("add params = %#v, want trimmed fields", querier.addCommentParams)
	}

	items, total, page, pageSize, err := store.ListProjectCommentsByContract(context.Background(), contract, 2, 5)
	if err != nil {
		t.Fatalf("ListProjectCommentsByContract: %v", err)
	}
	if total != 1 || page != 2 || pageSize != 5 || len(items) != 1 {
		t.Fatalf("items/total/page/pageSize = %d/%d/%d/%d", len(items), total, page, pageSize)
	}
	if querier.listCommentParams.Limit != 5 || querier.listCommentParams.Offset != 5 {
		t.Fatalf("list params = %#v, want page 2 offset", querier.listCommentParams)
	}
}

func TestProjectEventLogsUseQuerier(t *testing.T) {
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	occurredAt := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	querier := &fakeApplicationQuerier{
		listEventLogResult: []appsqlc.ProjectEventLog{{
			ID:             3,
			Contract:       contract.Bytes(),
			EventType:      1,
			OccurredAt:     pgtype.Timestamptz{Time: occurredAt, Valid: true},
			Message:        pgtype.Text{String: "created", Valid: true},
			Payload:        []byte(`{"ok":true}`),
			IdempotencyKey: "event-1",
			CreatedAt:      pgtype.Timestamptz{Time: occurredAt, Valid: true},
		}},
	}
	store := NewSQLStoreWithQuerier(querier)

	if err := store.AddProjectEventLog(context.Background(), ProjectEventLog{
		Contract:       contract,
		EventType:      1,
		OccurredAt:     occurredAt,
		Message:        "",
		Payload:        "",
		IdempotencyKey: "event-1",
	}); err != nil {
		t.Fatalf("AddProjectEventLog: %v", err)
	}
	if querier.addEventLogParams.Message.Valid {
		t.Fatalf("message = %#v, want null for empty message", querier.addEventLogParams.Message)
	}
	if string(querier.addEventLogParams.Column5) != "{}" {
		t.Fatalf("payload = %q, want default JSON object", string(querier.addEventLogParams.Column5))
	}

	items, err := store.ListProjectEventLogsByContract(context.Background(), contract)
	if err != nil {
		t.Fatalf("ListProjectEventLogsByContract: %v", err)
	}
	if len(items) != 1 || items[0].ID != 3 || items[0].Payload != `{"ok":true}` {
		t.Fatalf("items = %#v, want mapped event log", items)
	}
}

func TestProjectAveRefreshUsesQuerier(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	staleBefore := time.Date(2026, time.May, 29, 12, 0, 0, 0, time.UTC)
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	nextRunAt := now.Add(time.Hour)
	querier := &fakeApplicationQuerier{
		listAveResult: [][]byte{contract.Bytes()},
		aveStateResult: appsqlc.ProjectComponentState{
			ProjectContract: contract.Bytes(),
			Component:       ProjectComponentAveDetail,
			Status:          ProjectComponentStatusPending,
			NextRunAt:       pgtype.Timestamptz{Time: nextRunAt, Valid: true},
		},
	}
	store := NewSQLStoreWithQuerier(querier)

	candidates, err := store.ListProjectAveRefreshCandidates(ctx, staleBefore, now, 20)
	if err != nil {
		t.Fatalf("ListProjectAveRefreshCandidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0] != contract {
		t.Fatalf("candidates = %#v, want contract", candidates)
	}
	if !querier.listAveRefreshParams.StaleBefore.Time.Equal(staleBefore) || !querier.listAveRefreshParams.Now.Time.Equal(now) || querier.listAveRefreshParams.LimitCount != 20 {
		t.Fatalf("list params = %#v, want stale/now/limit", querier.listAveRefreshParams)
	}

	if err := store.ScheduleProjectAveRefresh(ctx, contract, now); err != nil {
		t.Fatalf("ScheduleProjectAveRefresh: %v", err)
	}
	if !querier.scheduleAveParams.NextRunAt.Time.Equal(now) {
		t.Fatalf("schedule params = %#v, want now", querier.scheduleAveParams)
	}

	if err := store.MarkProjectAveRefreshRunning(ctx, contract, now); err != nil {
		t.Fatalf("MarkProjectAveRefreshRunning: %v", err)
	}
	if !querier.markAveRunningParams.LastAttemptAt.Time.Equal(now) {
		t.Fatalf("running params = %#v, want attempt", querier.markAveRunningParams)
	}

	if err := store.MarkProjectAveRefreshSuccess(ctx, contract, now, nextRunAt); err != nil {
		t.Fatalf("MarkProjectAveRefreshSuccess: %v", err)
	}
	if !querier.markAveSuccessParams.LastAttemptAt.Time.Equal(now) || !querier.markAveSuccessParams.NextRunAt.Time.Equal(nextRunAt) {
		t.Fatalf("success params = %#v, want success and next run", querier.markAveSuccessParams)
	}

	if err := store.MarkProjectAveRefreshFailed(ctx, contract, now, nextRunAt, " ave down "); err != nil {
		t.Fatalf("MarkProjectAveRefreshFailed: %v", err)
	}
	if querier.markAveFailedParams.LastError.String != "ave down" {
		t.Fatalf("failed params = %#v, want trimmed error", querier.markAveFailedParams)
	}

	state, err := store.GetProjectAveComponentState(ctx, contract)
	if err != nil {
		t.Fatalf("GetProjectAveComponentState: %v", err)
	}
	if state == nil || state.Status != ProjectComponentStatusPending || !state.NextRunAt.Equal(nextRunAt) {
		t.Fatalf("state = %#v, want mapped pending state", state)
	}
}
