package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	appstore "github.com/useryege/athena/internal/application/store"
)

const (
	persistenceEventVersion = 1

	persistenceStreamKey  = "application:persist:stream"
	persistenceGroupName  = "application:persist:group"
	persistenceConsumerID = "application-persist-consumer"

	persistenceReadCount      = int64(64)
	persistenceReadBlock      = 2 * time.Second
	persistencePendingBackoff = 200 * time.Millisecond
)

const (
	PersistenceOpProjectMetaSave       = "project_meta_save"
	PersistenceOpProjectEventLogAdd    = "project_event_log_add"
	PersistenceOpProjectGenesisReplace = "project_genesis_wallet_replace"
	PersistenceOpProjectSourceCode     = "project_source_code_update"
	PersistenceOpProjectArchive        = "project_archive"
	PersistenceOpProjectUnarchive      = "project_unarchive"
	PersistenceOpSourceBlacklistAdd    = "blacklist_add"
	PersistenceOpSourceBlacklistDelete = "blacklist_delete"
	PersistenceOpBytecodeBlacklistAdd  = "bytecode_blacklist_add"
	PersistenceOpBytecodeBlacklistNote = "bytecode_blacklist_update_note"
	PersistenceOpBytecodeBlacklistDel  = "bytecode_blacklist_delete"
	PersistenceOpWalletBlacklistAdd    = "wallet_blacklist_add"
	PersistenceOpWalletBlacklistNote   = "wallet_blacklist_update_note"
	PersistenceOpWalletBlacklistDel    = "wallet_blacklist_delete"
)

type PersistenceEvent struct {
	Version    int             `json:"version"`
	Op         string          `json:"op"`
	Contract   string          `json:"contract,omitempty"`
	Field      string          `json:"field,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	OccurredAt time.Time       `json:"occurred_at"`
}

type projectMetaSavePayload struct {
	BlockTime   uint64 `json:"block_time"`
	BlockNumber uint64 `json:"block_number"`
	Contract    string `json:"contract"`
	Creator     string `json:"creator"`
	TxHash      string `json:"tx_hash"`
	TxIndex     uint64 `json:"tx_index"`
	SourceCode  string `json:"source_code"`
	IsArchived  bool   `json:"is_archived"`
	ArchivedAt  string `json:"archived_at,omitempty"`
}

type projectSourceCodeUpdatePayload struct {
	Contract   string `json:"contract"`
	SourceCode string `json:"source_code"`
}

type projectEventLogAddPayload struct {
	Contract       string `json:"contract"`
	EventType      int32  `json:"event_type"`
	OccurredAt     string `json:"occurred_at,omitempty"`
	Message        string `json:"message,omitempty"`
	Payload        string `json:"payload,omitempty"`
	IdempotencyKey string `json:"idempotency_key"`
}

type projectGenesisWalletReplacePayload struct {
	Contract          string                            `json:"contract"`
	SourceTxHash      string                            `json:"source_tx_hash"`
	SourceBlockNumber uint64                            `json:"source_block_number"`
	TotalSupply       string                            `json:"total_supply"`
	Items             []projectGenesisWalletItemPayload `json:"items"`
}

type projectGenesisWalletItemPayload struct {
	Wallet    string `json:"wallet"`
	NetAmount string `json:"net_amount"`
	RatioBPS  int64  `json:"ratio_bps"`
	RankIndex int32  `json:"rank_index"`
}

type blacklistFieldPayload struct {
	Field string `json:"field"`
}

type bytecodeBlacklistAddPayload struct {
	Contract string `json:"contract"`
	CodeHash string `json:"code_hash"`
	Note     string `json:"note,omitempty"`
}

type bytecodeBlacklistUpdateNotePayload struct {
	Contract string `json:"contract"`
	Note     string `json:"note,omitempty"`
}

type bytecodeBlacklistDeletePayload struct {
	Contract string `json:"contract"`
}

type walletBlacklistAddPayload struct {
	Contract string `json:"contract"`
	Note     string `json:"note,omitempty"`
}

type walletBlacklistUpdateNotePayload struct {
	Contract string `json:"contract"`
	Note     string `json:"note,omitempty"`
}

type walletBlacklistDeletePayload struct {
	Contract string `json:"contract"`
}

type PersistenceEventPublisher interface {
	Publish(ctx context.Context, event PersistenceEvent) error
	PublishProjectMetaSave(ctx context.Context, meta appstore.ProjectMeta) error
	PublishProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error
	PublishProjectSourceCodeUpdate(ctx context.Context, contract common.Address, sourceCode string) error
	PublishProjectArchive(ctx context.Context, contract common.Address) error
	PublishProjectUnarchive(ctx context.Context, contract common.Address) error
	PublishSourceCodeBlacklistAdd(ctx context.Context, field string) error
	PublishSourceCodeBlacklistDelete(ctx context.Context, field string) error
	PublishBytecodeBlacklistAdd(ctx context.Context, item appstore.BytecodeBlacklistContract) error
	PublishBytecodeBlacklistUpdateNote(ctx context.Context, contract common.Address, note string) error
	PublishBytecodeBlacklistDelete(ctx context.Context, contract common.Address) error
	PublishWalletBlacklistAdd(ctx context.Context, item appstore.WalletBlacklistContract) error
	PublishWalletBlacklistUpdateNote(ctx context.Context, contract common.Address, note string) error
	PublishWalletBlacklistDelete(ctx context.Context, contract common.Address) error
}

type PersistenceEventWriter interface {
	WriteProjectMeta(ctx context.Context, meta appstore.ProjectMeta) error
	WriteProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error
	WriteProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error
	ArchiveProject(ctx context.Context, contract common.Address) error
	UnarchiveProject(ctx context.Context, contract common.Address) error
	AddSourceCodeBlacklistField(ctx context.Context, field string) error
	DeleteSourceCodeBlacklistField(ctx context.Context, field string) error
	AddBytecodeBlacklistContract(ctx context.Context, item appstore.BytecodeBlacklistContract) error
	UpdateBytecodeBlacklistContractNote(ctx context.Context, contract common.Address, note string) error
	DeleteBytecodeBlacklistContract(ctx context.Context, contract common.Address) error
	AddWalletBlacklistContract(ctx context.Context, item appstore.WalletBlacklistContract) error
	UpdateWalletBlacklistContractNote(ctx context.Context, contract common.Address, note string) error
	DeleteWalletBlacklistContract(ctx context.Context, contract common.Address) error
}

type projectGenesisWalletWriter interface {
	WriteProjectGenesisWallets(ctx context.Context, contract common.Address, items []appstore.ProjectGenesisWallet) error
}

type PersistenceEventConsumer interface {
	Start(ctx context.Context) error
}

type RedisPersistenceEventBus struct {
	client   *redis.Client
	stream   string
	group    string
	consumer string
}

func NewRedisPersistenceEventBus(client *redis.Client) *RedisPersistenceEventBus {
	return &RedisPersistenceEventBus{
		client:   client,
		stream:   persistenceStreamKey,
		group:    persistenceGroupName,
		consumer: persistenceConsumerID,
	}
}

func (b *RedisPersistenceEventBus) Publish(ctx context.Context, event PersistenceEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if event.Version == 0 {
		event.Version = persistenceEventVersion
	}
	if event.Op == "" {
		return errors.New("persistence event op is required")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal persistence event: %w", err)
	}
	if err := b.client.XAdd(ctx, &redis.XAddArgs{
		Stream: b.stream,
		Values: map[string]any{"event": string(encoded)},
	}).Err(); err != nil {
		return fmt.Errorf("publish persistence event: %w", err)
	}
	return nil
}

func (b *RedisPersistenceEventBus) PublishProjectMetaSave(ctx context.Context, meta appstore.ProjectMeta) error {
	payload := projectMetaSavePayload{
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract.Hex(),
		Creator:     meta.Creator.Hex(),
		TxHash:      meta.TxHash.Hex(),
		TxIndex:     meta.TxIndex,
		SourceCode:  meta.SourceCode,
		IsArchived:  meta.IsArchived,
	}
	if !meta.ArchivedAt.IsZero() {
		payload.ArchivedAt = meta.ArchivedAt.UTC().Format(time.RFC3339Nano)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal project meta payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectMetaSave,
		Contract:   meta.Contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishProjectSourceCodeUpdate(ctx context.Context, contract common.Address, sourceCode string) error {
	payload := projectSourceCodeUpdatePayload{Contract: contract.Hex(), SourceCode: sourceCode}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal project source code payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectSourceCode,
		Contract:   contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error {
	payload := projectEventLogAddPayload{
		Contract:       item.Contract.Hex(),
		EventType:      int32(item.EventType),
		Message:        item.Message,
		Payload:        item.Payload,
		IdempotencyKey: item.IdempotencyKey,
	}
	if !item.OccurredAt.IsZero() {
		payload.OccurredAt = item.OccurredAt.UTC().Format(time.RFC3339Nano)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal project event log payload: %w", err)
	}
	occurredAt := item.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectEventLogAdd,
		Contract:   item.Contract.Hex(),
		Payload:    data,
		OccurredAt: occurredAt,
	})
}

func (b *RedisPersistenceEventBus) PublishProjectArchive(ctx context.Context, contract common.Address) error {
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectArchive,
		Contract:   contract.Hex(),
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishProjectUnarchive(ctx context.Context, contract common.Address) error {
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectUnarchive,
		Contract:   contract.Hex(),
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishSourceCodeBlacklistAdd(ctx context.Context, field string) error {
	payload := blacklistFieldPayload{Field: field}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal blacklist add payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpSourceBlacklistAdd,
		Field:      field,
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishSourceCodeBlacklistDelete(ctx context.Context, field string) error {
	payload := blacklistFieldPayload{Field: field}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal blacklist delete payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpSourceBlacklistDelete,
		Field:      field,
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishBytecodeBlacklistAdd(ctx context.Context, item appstore.BytecodeBlacklistContract) error {
	payload := bytecodeBlacklistAddPayload{
		Contract: item.Contract.Hex(),
		CodeHash: item.CodeHash.Hex(),
		Note:     item.Note,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal bytecode blacklist add payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpBytecodeBlacklistAdd,
		Contract:   item.Contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishBytecodeBlacklistUpdateNote(ctx context.Context, contract common.Address, note string) error {
	payload := bytecodeBlacklistUpdateNotePayload{
		Contract: contract.Hex(),
		Note:     note,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal bytecode blacklist update note payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpBytecodeBlacklistNote,
		Contract:   contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishBytecodeBlacklistDelete(ctx context.Context, contract common.Address) error {
	payload := bytecodeBlacklistDeletePayload{
		Contract: contract.Hex(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal bytecode blacklist delete payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpBytecodeBlacklistDel,
		Contract:   contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishWalletBlacklistAdd(ctx context.Context, item appstore.WalletBlacklistContract) error {
	payload := walletBlacklistAddPayload{
		Contract: item.Contract.Hex(),
		Note:     item.Note,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal wallet blacklist add payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpWalletBlacklistAdd,
		Contract:   item.Contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishWalletBlacklistUpdateNote(ctx context.Context, contract common.Address, note string) error {
	payload := walletBlacklistUpdateNotePayload{
		Contract: contract.Hex(),
		Note:     note,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal wallet blacklist update note payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpWalletBlacklistNote,
		Contract:   contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishWalletBlacklistDelete(ctx context.Context, contract common.Address) error {
	payload := walletBlacklistDeletePayload{
		Contract: contract.Hex(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal wallet blacklist delete payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpWalletBlacklistDel,
		Contract:   contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) Start(ctx context.Context, writer PersistenceEventWriter) error {
	if writer == nil {
		return errors.New("persistence event writer is nil")
	}
	if err := b.ensureConsumerGroup(ctx); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		processed, err := b.consume(ctx, "0", writer, 0)
		if err != nil {
			return err
		}
		if processed == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(persistencePendingBackoff):
		}
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := b.consume(ctx, ">", writer, persistenceReadBlock); err != nil {
			return err
		}
	}
}

func (b *RedisPersistenceEventBus) consume(ctx context.Context, id string, writer PersistenceEventWriter, block time.Duration) (int, error) {
	streams, err := b.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    b.group,
		Consumer: b.consumer,
		Streams:  []string{b.stream, id},
		Count:    persistenceReadCount,
		Block:    block,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return 0, err
		}
		if strings.Contains(err.Error(), "NOGROUP") {
			if ensureErr := b.ensureConsumerGroup(ctx); ensureErr != nil {
				return 0, ensureErr
			}
			return 0, nil
		}
		return 0, fmt.Errorf("xreadgroup persistence events: %w", err)
	}

	processed := 0
	for _, stream := range streams {
		for _, message := range stream.Messages {
			if err := b.handleMessage(ctx, writer, message); err != nil {
				return processed, err
			}
			processed++
		}
	}
	return processed, nil
}

func (b *RedisPersistenceEventBus) handleMessage(ctx context.Context, writer PersistenceEventWriter, message redis.XMessage) error {
	value, ok := message.Values["event"]
	if !ok {
		return fmt.Errorf("persistence event %s missing event payload", message.ID)
	}
	raw, ok := value.(string)
	if !ok {
		return fmt.Errorf("persistence event %s payload type %T is not string", message.ID, value)
	}
	var event PersistenceEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		return fmt.Errorf("unmarshal persistence event %s: %w", message.ID, err)
	}
	if err := b.applyEvent(ctx, writer, event); err != nil {
		return err
	}
	if err := b.client.XAck(ctx, b.stream, b.group, message.ID).Err(); err != nil {
		return fmt.Errorf("ack persistence event %s: %w", message.ID, err)
	}
	return nil
}

func (b *RedisPersistenceEventBus) applyEvent(ctx context.Context, writer PersistenceEventWriter, event PersistenceEvent) error {
	switch event.Op {
	case PersistenceOpProjectMetaSave:
		var payload projectMetaSavePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal project meta payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		contract := common.HexToAddress(payload.Contract)
		creator := common.HexToAddress(payload.Creator)
		txHash := common.HexToHash(payload.TxHash)
		meta := appstore.ProjectMeta{
			BlockTime:   payload.BlockTime,
			BlockNumber: payload.BlockNumber,
			Contract:    contract,
			Creator:     creator,
			TxHash:      txHash,
			TxIndex:     payload.TxIndex,
			SourceCode:  payload.SourceCode,
			IsArchived:  payload.IsArchived,
		}
		if payload.ArchivedAt != "" {
			archivedAt, err := time.Parse(time.RFC3339Nano, payload.ArchivedAt)
			if err != nil {
				return fmt.Errorf("parse archived_at %q: %w", payload.ArchivedAt, err)
			}
			meta.ArchivedAt = archivedAt
		}
		return writer.WriteProjectMeta(ctx, meta)
	case PersistenceOpProjectSourceCode:
		var payload projectSourceCodeUpdatePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal project source code payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		return writer.WriteProjectSourceCode(ctx, common.HexToAddress(payload.Contract), payload.SourceCode)
	case PersistenceOpProjectEventLogAdd:
		var payload projectEventLogAddPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal project event log payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		if payload.EventType <= 0 {
			return fmt.Errorf("invalid event_type %d", payload.EventType)
		}
		occurredAt := event.OccurredAt.UTC()
		if payload.OccurredAt != "" {
			parsed, err := time.Parse(time.RFC3339Nano, payload.OccurredAt)
			if err != nil {
				return fmt.Errorf("parse occurred_at %q: %w", payload.OccurredAt, err)
			}
			occurredAt = parsed.UTC()
		}
		return writer.WriteProjectEventLog(ctx, appstore.ProjectEventLog{
			Contract:       common.HexToAddress(payload.Contract),
			EventType:      int16(payload.EventType),
			OccurredAt:     occurredAt,
			Message:        payload.Message,
			Payload:        payload.Payload,
			IdempotencyKey: payload.IdempotencyKey,
		})
	case PersistenceOpProjectGenesisReplace:
		var payload projectGenesisWalletReplacePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal project genesis wallet replace payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		sourceTxHash := strings.TrimSpace(payload.SourceTxHash)
		if len(sourceTxHash) != 66 || !strings.HasPrefix(sourceTxHash, "0x") || len(common.FromHex(sourceTxHash)) != 32 {
			return fmt.Errorf("invalid source_tx_hash %q", payload.SourceTxHash)
		}
		totalSupply, ok := new(big.Int).SetString(strings.TrimSpace(payload.TotalSupply), 10)
		if !ok || totalSupply.Sign() < 0 {
			return fmt.Errorf("invalid total_supply %q", payload.TotalSupply)
		}
		items := make([]appstore.ProjectGenesisWallet, 0, len(payload.Items))
		for _, entry := range payload.Items {
			if !common.IsHexAddress(entry.Wallet) {
				return fmt.Errorf("invalid wallet %q", entry.Wallet)
			}
			netAmount, ok := new(big.Int).SetString(strings.TrimSpace(entry.NetAmount), 10)
			if !ok || netAmount.Sign() <= 0 {
				return fmt.Errorf("invalid net_amount %q for wallet %s", entry.NetAmount, entry.Wallet)
			}
			items = append(items, appstore.ProjectGenesisWallet{
				ProjectContract:   common.HexToAddress(payload.Contract),
				Wallet:            common.HexToAddress(entry.Wallet),
				NetAmount:         netAmount,
				RatioBPS:          entry.RatioBPS,
				RankIndex:         entry.RankIndex,
				TotalSupply:       new(big.Int).Set(totalSupply),
				SourceTxHash:      common.HexToHash(sourceTxHash),
				SourceBlockNumber: payload.SourceBlockNumber,
			})
		}
		gwWriter, ok := writer.(projectGenesisWalletWriter)
		if !ok || gwWriter == nil {
			return errors.New("project genesis wallet store is not configured")
		}
		return gwWriter.WriteProjectGenesisWallets(ctx, common.HexToAddress(payload.Contract), items)
	case PersistenceOpProjectArchive:
		if !common.IsHexAddress(event.Contract) {
			return fmt.Errorf("invalid contract %q", event.Contract)
		}
		return writer.ArchiveProject(ctx, common.HexToAddress(event.Contract))
	case PersistenceOpProjectUnarchive:
		if !common.IsHexAddress(event.Contract) {
			return fmt.Errorf("invalid contract %q", event.Contract)
		}
		return writer.UnarchiveProject(ctx, common.HexToAddress(event.Contract))
	case PersistenceOpSourceBlacklistAdd:
		field := event.Field
		if len(event.Payload) > 0 {
			var payload blacklistFieldPayload
			if err := json.Unmarshal(event.Payload, &payload); err == nil && payload.Field != "" {
				field = payload.Field
			}
		}
		if field == "" {
			return errors.New("blacklist field is empty")
		}
		return writer.AddSourceCodeBlacklistField(ctx, field)
	case PersistenceOpSourceBlacklistDelete:
		field := event.Field
		if len(event.Payload) > 0 {
			var payload blacklistFieldPayload
			if err := json.Unmarshal(event.Payload, &payload); err == nil && payload.Field != "" {
				field = payload.Field
			}
		}
		if field == "" {
			return errors.New("blacklist field is empty")
		}
		return writer.DeleteSourceCodeBlacklistField(ctx, field)
	case PersistenceOpBytecodeBlacklistAdd:
		var payload bytecodeBlacklistAddPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal bytecode blacklist add payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		codeHash := strings.TrimSpace(payload.CodeHash)
		if len(codeHash) != 66 || !strings.HasPrefix(codeHash, "0x") || len(common.FromHex(codeHash)) != 32 {
			return fmt.Errorf("invalid code_hash %q", payload.CodeHash)
		}
		err := writer.AddBytecodeBlacklistContract(ctx, appstore.BytecodeBlacklistContract{
			Contract: common.HexToAddress(payload.Contract),
			CodeHash: common.HexToHash(codeHash),
			Note:     payload.Note,
		})
		if errors.Is(err, appstore.ErrBytecodeBlacklistContractAlreadyExists) {
			return nil
		}
		return err
	case PersistenceOpBytecodeBlacklistNote:
		var payload bytecodeBlacklistUpdateNotePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal bytecode blacklist update note payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		return writer.UpdateBytecodeBlacklistContractNote(ctx, common.HexToAddress(payload.Contract), payload.Note)
	case PersistenceOpBytecodeBlacklistDel:
		var payload bytecodeBlacklistDeletePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal bytecode blacklist delete payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		err := writer.DeleteBytecodeBlacklistContract(ctx, common.HexToAddress(payload.Contract))
		if errors.Is(err, appstore.ErrBytecodeBlacklistContractNotFound) {
			return nil
		}
		return err
	case PersistenceOpWalletBlacklistAdd:
		var payload walletBlacklistAddPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal wallet blacklist add payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		err := writer.AddWalletBlacklistContract(ctx, appstore.WalletBlacklistContract{
			Contract: common.HexToAddress(payload.Contract),
			Note:     payload.Note,
		})
		if errors.Is(err, appstore.ErrWalletBlacklistContractAlreadyExists) {
			return nil
		}
		return err
	case PersistenceOpWalletBlacklistNote:
		var payload walletBlacklistUpdateNotePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal wallet blacklist update note payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		return writer.UpdateWalletBlacklistContractNote(ctx, common.HexToAddress(payload.Contract), payload.Note)
	case PersistenceOpWalletBlacklistDel:
		var payload walletBlacklistDeletePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal wallet blacklist delete payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		err := writer.DeleteWalletBlacklistContract(ctx, common.HexToAddress(payload.Contract))
		if errors.Is(err, appstore.ErrWalletBlacklistContractNotFound) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unsupported persistence event op %q", event.Op)
	}
}

func (b *RedisPersistenceEventBus) ensureConsumerGroup(ctx context.Context) error {
	err := b.client.XGroupCreateMkStream(ctx, b.stream, b.group, "0").Err()
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "BUSYGROUP") {
		return nil
	}
	return fmt.Errorf("ensure persistence consumer group: %w", err)
}

type storePersistenceWriter struct {
	store appstore.Store
}

func NewStorePersistenceWriter(store appstore.Store) PersistenceEventWriter {
	if store == nil {
		return nil
	}
	return &storePersistenceWriter{store: store}
}

func (w *storePersistenceWriter) WriteProjectMeta(ctx context.Context, meta appstore.ProjectMeta) error {
	return w.store.SaveProjectMeta(ctx, meta)
}

func (w *storePersistenceWriter) WriteProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error {
	store, ok := w.store.(appstore.ProjectEventLogStore)
	if !ok || store == nil {
		return errors.New("project event log store is not configured")
	}
	return store.AddProjectEventLog(ctx, item)
}

func (w *storePersistenceWriter) WriteProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	return w.store.UpdateProjectSourceCode(ctx, contract, sourceCode)
}

func (w *storePersistenceWriter) ArchiveProject(ctx context.Context, contract common.Address) error {
	return w.store.ArchiveProjectByContract(ctx, contract)
}

func (w *storePersistenceWriter) UnarchiveProject(ctx context.Context, contract common.Address) error {
	return w.store.UnarchiveProjectByContract(ctx, contract)
}

func (w *storePersistenceWriter) AddSourceCodeBlacklistField(ctx context.Context, field string) error {
	return w.store.AddSourceCodeBlacklistField(ctx, field)
}

func (w *storePersistenceWriter) DeleteSourceCodeBlacklistField(ctx context.Context, field string) error {
	return w.store.DeleteSourceCodeBlacklistField(ctx, field)
}

func (w *storePersistenceWriter) AddBytecodeBlacklistContract(ctx context.Context, item appstore.BytecodeBlacklistContract) error {
	store, ok := w.store.(appstore.BytecodeBlacklistContractStore)
	if !ok || store == nil {
		return errors.New("bytecode blacklist contract store is not configured")
	}
	return store.AddBytecodeBlacklistContract(ctx, item)
}

func (w *storePersistenceWriter) UpdateBytecodeBlacklistContractNote(ctx context.Context, contract common.Address, note string) error {
	store, ok := w.store.(appstore.BytecodeBlacklistContractStore)
	if !ok || store == nil {
		return errors.New("bytecode blacklist contract store is not configured")
	}
	return store.UpdateBytecodeBlacklistContractNote(ctx, contract, note)
}

func (w *storePersistenceWriter) DeleteBytecodeBlacklistContract(ctx context.Context, contract common.Address) error {
	store, ok := w.store.(appstore.BytecodeBlacklistContractStore)
	if !ok || store == nil {
		return errors.New("bytecode blacklist contract store is not configured")
	}
	return store.DeleteBytecodeBlacklistContract(ctx, contract)
}

func (w *storePersistenceWriter) AddWalletBlacklistContract(ctx context.Context, item appstore.WalletBlacklistContract) error {
	store, ok := w.store.(appstore.WalletBlacklistContractStore)
	if !ok || store == nil {
		return errors.New("wallet blacklist contract store is not configured")
	}
	return store.AddWalletBlacklistContract(ctx, item)
}

func (w *storePersistenceWriter) UpdateWalletBlacklistContractNote(ctx context.Context, contract common.Address, note string) error {
	store, ok := w.store.(appstore.WalletBlacklistContractStore)
	if !ok || store == nil {
		return errors.New("wallet blacklist contract store is not configured")
	}
	return store.UpdateWalletBlacklistContractNote(ctx, contract, note)
}

func (w *storePersistenceWriter) DeleteWalletBlacklistContract(ctx context.Context, contract common.Address) error {
	store, ok := w.store.(appstore.WalletBlacklistContractStore)
	if !ok || store == nil {
		return errors.New("wallet blacklist contract store is not configured")
	}
	return store.DeleteWalletBlacklistContract(ctx, contract)
}

func (w *storePersistenceWriter) WriteProjectGenesisWallets(ctx context.Context, contract common.Address, items []appstore.ProjectGenesisWallet) error {
	store, ok := w.store.(appstore.ProjectGenesisWalletStore)
	if !ok || store == nil {
		return errors.New("project genesis wallet store is not configured")
	}
	return store.ReplaceProjectGenesisWallets(ctx, contract, items)
}
