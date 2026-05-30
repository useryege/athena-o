package persistence

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
)

type persistenceEventWriterFake struct {
	err           error
	projectMeta   appstore.ProjectMeta
	aveDetail     appstore.ProjectAveDetail
	creatorResult appstore.SimulateResult
	projectReport appstore.ProjectReport
}

func (w *persistenceEventWriterFake) WriteProjectMeta(_ context.Context, meta appstore.ProjectMeta) error {
	if w.err == nil {
		w.projectMeta = meta
	}
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectEventLog(context.Context, appstore.ProjectEventLog) error {
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectAveDetail(_ context.Context, _ common.Address, detail appstore.ProjectAveDetail) error {
	if w.err == nil {
		w.aveDetail = detail
	}
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectCreatorResult(_ context.Context, _ common.Address, result appstore.SimulateResult) error {
	if w.err == nil {
		w.creatorResult = result
	}
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectReport(_ context.Context, _ common.Address, report appstore.ProjectReport) error {
	if w.err == nil {
		w.projectReport = report
	}
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectCreatorHistoricalProjects(context.Context, common.Address, []appstore.ProjectCreatorHistoricalProject) error {
	return w.err
}

func newPersistenceEventBusTest(t *testing.T) (*redis.Client, *RedisPersistenceEventBus) {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	bus := NewRedisPersistenceEventBus(redisport.NewGoRedisAdapter(client))
	if bus == nil {
		t.Fatal("bus is nil")
	}
	if err := bus.ensureConsumerGroup(context.Background()); err != nil {
		t.Fatalf("ensure consumer group: %v", err)
	}
	return client, bus
}

func TestNewRedisPersistenceEventBusNilClient(t *testing.T) {
	if bus := NewRedisPersistenceEventBus(nil); bus != nil {
		t.Fatal("expected nil bus")
	}
	if err := (*RedisPersistenceEventBus)(nil).Publish(context.Background(), PersistenceEvent{Op: PersistenceOpProjectReport}); err == nil {
		t.Fatal("expected publish error for nil bus")
	}
	if err := (*RedisPersistenceEventBus)(nil).Start(context.Background(), &persistenceEventWriterFake{}); err == nil {
		t.Fatal("expected start error for nil bus")
	}
}

func TestRedisPersistenceEventBusConsumesAndAcks(t *testing.T) {
	client, bus := newPersistenceEventBusTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")

	if err := bus.PublishProjectReportUpdate(ctx, contract, ProjectReport{IsReportEvaluated: true}); err != nil {
		t.Fatalf("publish project report: %v", err)
	}
	writer := &persistenceEventWriterFake{}
	processed, err := bus.consume(ctx, ">", writer, time.Millisecond)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if !writer.projectReport.IsReportEvaluated {
		t.Fatalf("project report = %+v, want report evaluated", writer.projectReport)
	}
	pending, err := client.XPending(ctx, persistenceStreamKey, persistenceGroupName).Result()
	if err != nil {
		t.Fatalf("xpending: %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("pending count = %d, want 0", pending.Count)
	}
}

func TestRedisPersistenceEventBusAppliesProjectCreatorResult(t *testing.T) {
	_, bus := newPersistenceEventBusTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	want := SimulateResult{
		CanMintFromDeadViaTransferFrom:     true,
		CanMintFromWethPairViaTransferFrom: true,
		CanMintViaTransferToWethPair:       true,
	}

	if err := bus.PublishProjectCreatorResultUpdate(ctx, contract, want); err != nil {
		t.Fatalf("publish creator result: %v", err)
	}
	writer := &persistenceEventWriterFake{}
	processed, err := bus.consume(ctx, ">", writer, time.Millisecond)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if writer.creatorResult != simulateResultToStore(want) {
		t.Fatalf("creator result = %+v, want %+v", writer.creatorResult, simulateResultToStore(want))
	}
}

func TestRedisPersistenceEventBusAppliesProjectAveDetail(t *testing.T) {
	_, bus := newPersistenceEventBusTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	detail := appstore.ProjectAveDetail{
		Status:    1,
		Msg:       "SUCCESS",
		DataType:  1,
		IsAudited: true,
		FetchedAt: time.Date(2026, 5, 23, 4, 6, 6, 0, time.UTC),
		Token: appstore.ProjectAveTokenDetail{
			Token:   "token",
			Chain:   "bsc",
			LogoURL: "https://example.com/logo.png",
		},
		Pairs: []appstore.ProjectAvePair{{Pair: "pair-1", Chain: "bsc"}},
	}

	if err := bus.PublishProjectAveDetailUpsert(ctx, contract, detail); err != nil {
		t.Fatalf("publish ave detail: %v", err)
	}
	writer := &persistenceEventWriterFake{}
	processed, err := bus.consume(ctx, ">", writer, time.Millisecond)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if writer.aveDetail.Token.LogoURL != detail.Token.LogoURL || len(writer.aveDetail.Pairs) != 1 {
		t.Fatalf("ave detail = %+v, want %+v", writer.aveDetail, detail)
	}
}

func TestRedisPersistenceEventBusAppliesProjectMetaPairAddressesAndFetchAt(t *testing.T) {
	_, bus := newPersistenceEventBusTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	creator := common.HexToAddress("0x1000000000000000000000000000000000000002")
	wethPair := common.HexToAddress("0x1000000000000000000000000000000000000003")
	usdtPair := common.HexToAddress("0x1000000000000000000000000000000000000004")
	txHash := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	fetchAt := time.Date(2026, 5, 23, 4, 5, 6, 0, time.UTC)

	if err := bus.PublishProjectMetaSave(ctx, appstore.ProjectMeta{
		BlockTime:   100,
		BlockNumber: 200,
		Contract:    contract,
		Creator:     creator,
		WethPair:    wethPair,
		UsdtPair:    usdtPair,
		FetchAt:     fetchAt,
		TxHash:      txHash,
		TxIndex:     7,
	}); err != nil {
		t.Fatalf("publish project meta: %v", err)
	}
	writer := &persistenceEventWriterFake{}
	processed, err := bus.consume(ctx, ">", writer, time.Millisecond)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if writer.projectMeta.WethPair != wethPair || writer.projectMeta.UsdtPair != usdtPair {
		t.Fatalf("pair addresses = %s/%s, want %s/%s", writer.projectMeta.WethPair.Hex(), writer.projectMeta.UsdtPair.Hex(), wethPair.Hex(), usdtPair.Hex())
	}
	if !writer.projectMeta.FetchAt.Equal(fetchAt) {
		t.Fatalf("fetch at = %s, want %s", writer.projectMeta.FetchAt, fetchAt)
	}
}

func TestRedisPersistenceEventBusAppliesProjectReport(t *testing.T) {
	_, bus := newPersistenceEventBusTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	want := ProjectReport{
		IsReportEvaluated:          true,
		IsBlacklistedCreatorWallet: true,
		IsBlacklistedBytecode:      true,
		HasMintRisk:                true,
	}

	if err := bus.PublishProjectReportUpdate(ctx, contract, want); err != nil {
		t.Fatalf("publish project report: %v", err)
	}
	writer := &persistenceEventWriterFake{}
	processed, err := bus.consume(ctx, ">", writer, time.Millisecond)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if writer.projectReport != projectReportToStore(want) {
		t.Fatalf("project report = %+v, want %+v", writer.projectReport, projectReportToStore(want))
	}
}

func TestRedisPersistenceEventBusDeadLettersPoisonMessages(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]any
		writer *persistenceEventWriterFake
	}{
		{
			name:   "bad json",
			values: map[string]any{"event": "{bad-json"},
			writer: &persistenceEventWriterFake{},
		},
		{
			name:   "missing event",
			values: map[string]any{"other": "value"},
			writer: &persistenceEventWriterFake{},
		},
		{
			name:   "writer failure",
			values: map[string]any{"event": `{"version":1,"op":"project_report_update","contract":"0x1000000000000000000000000000000000000001","payload":{"contract":"0x1000000000000000000000000000000000000001","is_report_evaluated":true},"occurred_at":"2026-05-22T00:00:00Z"}`},
			writer: &persistenceEventWriterFake{err: errors.New("store unavailable")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, bus := newPersistenceEventBusTest(t)
			ctx := context.Background()
			added, err := client.XAdd(ctx, &redis.XAddArgs{
				Stream: persistenceStreamKey,
				Values: tt.values,
			}).Result()
			if err != nil {
				t.Fatalf("xadd: %v", err)
			}

			for attempt := 1; attempt <= persistenceMaxRetries; attempt++ {
				id := ">"
				if attempt > 1 {
					id = "0"
				}
				processed, err := bus.consume(ctx, id, tt.writer, time.Millisecond)
				if attempt < persistenceMaxRetries {
					if err == nil {
						t.Fatalf("attempt %d expected error", attempt)
					}
					if processed != 0 {
						t.Fatalf("attempt %d processed = %d, want 0", attempt, processed)
					}
					value, getErr := client.Get(ctx, persistenceRetryKey(added)).Result()
					if getErr != nil {
						t.Fatalf("attempt %d retry key: %v", attempt, getErr)
					}
					if value != strconv.Itoa(attempt) {
						t.Fatalf("attempt %d retry count = %q", attempt, value)
					}
					continue
				}
				if err != nil {
					t.Fatalf("final attempt consume: %v", err)
				}
				if processed != 1 {
					t.Fatalf("final attempt processed = %d, want 1", processed)
				}
			}

			if _, err := client.Get(ctx, persistenceRetryKey(added)).Result(); !errors.Is(err, redis.Nil) {
				t.Fatalf("retry key err = %v, want redis.Nil", err)
			}
			pending, err := client.XPending(ctx, persistenceStreamKey, persistenceGroupName).Result()
			if err != nil {
				t.Fatalf("xpending: %v", err)
			}
			if pending.Count != 0 {
				t.Fatalf("pending count = %d, want 0", pending.Count)
			}
			deadLetters, err := client.XRange(ctx, persistenceDeadLetterStreamKey, "-", "+").Result()
			if err != nil {
				t.Fatalf("xrange dead letters: %v", err)
			}
			if len(deadLetters) != 1 {
				t.Fatalf("dead letters = %d, want 1", len(deadLetters))
			}
			if deadLetters[0].Values["message_id"] != added {
				t.Fatalf("dead-letter message_id = %v, want %s", deadLetters[0].Values["message_id"], added)
			}
			if deadLetters[0].Values["attempts"] != "3" {
				t.Fatalf("dead-letter attempts = %v, want 3", deadLetters[0].Values["attempts"])
			}
			if _, ok := deadLetters[0].Values["raw_values"]; !ok {
				t.Fatal("dead-letter raw_values missing")
			}
			if errText, _ := deadLetters[0].Values["error"].(string); strings.TrimSpace(errText) == "" {
				t.Fatal("dead-letter error missing")
			}
		})
	}
}

type deadLetterFailureClient struct {
	ackCount int
}

func (c *deadLetterFailureClient) Get(context.Context, string) (string, error) {
	return "2", nil
}

func (c *deadLetterFailureClient) Set(context.Context, string, any, time.Duration) error {
	return nil
}

func (c *deadLetterFailureClient) Del(context.Context, ...string) error {
	return nil
}

func (c *deadLetterFailureClient) XAdd(_ context.Context, input redisport.XAddInput) error {
	if input.Stream == persistenceDeadLetterStreamKey {
		return errors.New("dead-letter unavailable")
	}
	return nil
}

func (c *deadLetterFailureClient) XReadGroup(context.Context, redisport.XReadGroupInput) ([]redisport.XStream, error) {
	return nil, redisport.ErrNotFound
}

func (c *deadLetterFailureClient) XAck(context.Context, string, string, ...string) error {
	c.ackCount++
	return nil
}

func (c *deadLetterFailureClient) XGroupCreateMkStream(context.Context, string, string, string) error {
	return nil
}

func TestRedisPersistenceEventBusDeadLetterFailureDoesNotAck(t *testing.T) {
	client := &deadLetterFailureClient{}
	bus := &RedisPersistenceEventBus{
		client:           client,
		stream:           persistenceStreamKey,
		group:            persistenceGroupName,
		consumer:         persistenceConsumerID,
		deadLetterStream: persistenceDeadLetterStreamKey,
	}

	err := bus.handleMessageFailure(context.Background(), redisport.XMessage{
		ID:     "1-0",
		Values: map[string]any{"event": "{bad-json"},
	}, errors.New("poison"))
	if err == nil {
		t.Fatal("expected dead-letter failure")
	}
	if client.ackCount != 0 {
		t.Fatalf("ack count = %d, want 0", client.ackCount)
	}
}
