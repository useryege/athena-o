package application

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
	err             error
	sourceCodeCalls int
}

func (w *persistenceEventWriterFake) WriteProjectMeta(context.Context, appstore.ProjectMeta) error {
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectEventLog(context.Context, appstore.ProjectEventLog) error {
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectSourceCode(context.Context, common.Address, string) error {
	if w.err == nil {
		w.sourceCodeCalls++
	}
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectCodeBinHash(context.Context, common.Address, common.Hash) error {
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectSourceQualityReport(context.Context, common.Address, string) error {
	return w.err
}

func (w *persistenceEventWriterFake) WriteProjectCreatorHistoricalProjects(context.Context, common.Address, []appstore.ProjectCreatorHistoricalProject) error {
	return w.err
}

func (w *persistenceEventWriterFake) AddBytecodeBlacklistContract(context.Context, appstore.BytecodeBlacklistContract) error {
	return w.err
}

func (w *persistenceEventWriterFake) UpdateBytecodeBlacklistContractNote(context.Context, common.Address, string) error {
	return w.err
}

func (w *persistenceEventWriterFake) DeleteBytecodeBlacklistContract(context.Context, common.Address) error {
	return w.err
}

func (w *persistenceEventWriterFake) AddSourcecodeBlacklistContract(context.Context, appstore.SourcecodeBlacklistContract) error {
	return w.err
}

func (w *persistenceEventWriterFake) UpdateSourcecodeBlacklistContractNote(context.Context, common.Address, string) error {
	return w.err
}

func (w *persistenceEventWriterFake) DeleteSourcecodeBlacklistContract(context.Context, common.Address) error {
	return w.err
}

func (w *persistenceEventWriterFake) AddWalletBlacklistEntry(context.Context, appstore.WalletBlacklistEntry) error {
	return w.err
}

func (w *persistenceEventWriterFake) UpdateWalletBlacklistEntryNote(context.Context, common.Address, string) error {
	return w.err
}

func (w *persistenceEventWriterFake) DeleteWalletBlacklistEntry(context.Context, common.Address) error {
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
	if err := (*RedisPersistenceEventBus)(nil).Publish(context.Background(), PersistenceEvent{Op: PersistenceOpProjectSourceCode}); err == nil {
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

	if err := bus.PublishProjectSourceCodeUpdate(ctx, contract, "contract Source {}"); err != nil {
		t.Fatalf("publish source code: %v", err)
	}
	writer := &persistenceEventWriterFake{}
	processed, err := bus.consume(ctx, ">", writer, time.Millisecond)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if writer.sourceCodeCalls != 1 {
		t.Fatalf("source code calls = %d, want 1", writer.sourceCodeCalls)
	}
	pending, err := client.XPending(ctx, persistenceStreamKey, persistenceGroupName).Result()
	if err != nil {
		t.Fatalf("xpending: %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("pending count = %d, want 0", pending.Count)
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
			values: map[string]any{"event": `{"version":1,"op":"project_source_code_update","contract":"0x1000000000000000000000000000000000000001","payload":{"contract":"0x1000000000000000000000000000000000000001","source_code":"contract Source {}"},"occurred_at":"2026-05-22T00:00:00Z"}`},
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
