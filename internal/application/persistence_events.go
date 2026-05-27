package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
)

const (
	persistenceEventVersion = 1

	persistenceStreamKey           = "application:persist:stream"
	persistenceGroupName           = "application:persist:group"
	persistenceConsumerID          = "application-persist-consumer"
	persistenceDeadLetterStreamKey = "application:persist:dead_letter_stream"

	persistenceReadCount      = int64(64)
	persistenceReadBlock      = 2 * time.Second
	persistencePendingBackoff = 200 * time.Millisecond
	persistenceMaxRetries     = 3
)

const (
	PersistenceOpProjectMetaSave                 = "project_meta_save"
	PersistenceOpProjectEventLogAdd              = "project_event_log_add"
	PersistenceOpProjectGenesisReplace           = "project_genesis_wallet_replace"
	PersistenceOpProjectCreatorHistoricalReplace = "project_creator_historical_project_replace"
	PersistenceOpProjectAveDetail                = "project_ave_detail_upsert"
	PersistenceOpProjectCreatorResult            = "project_creator_result_update"
	PersistenceOpProjectReport                   = "project_report_update"
	PersistenceOpWalletBlacklistAdd              = "wallet_blacklist_add"
	PersistenceOpWalletBlacklistNote             = "wallet_blacklist_update_note"
	PersistenceOpWalletBlacklistDel              = "wallet_blacklist_delete"
)

type PersistenceEvent struct {
	Version    int             `json:"version"`
	Op         string          `json:"op"`
	Contract   string          `json:"contract,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	OccurredAt time.Time       `json:"occurred_at"`
}

type projectMetaSavePayload struct {
	BlockTime                          uint64 `json:"block_time"`
	BlockNumber                        uint64 `json:"block_number"`
	Contract                           string `json:"contract"`
	Creator                            string `json:"creator"`
	WethPair                           string `json:"weth_pair"`
	UsdtPair                           string `json:"usdt_pair"`
	FetchAt                            string `json:"fetch_at"`
	TxHash                             string `json:"tx_hash"`
	TxIndex                            uint64 `json:"tx_index"`
	ReportIsPolicyEvaluated            bool   `json:"report_is_policy_evaluated,omitempty"`
	ReportIsBlacklistedCreatorWallet   bool   `json:"report_is_blacklisted_creator_wallet,omitempty"`
	ReportIsBlacklistedGenesisWallet   bool   `json:"report_is_blacklisted_genesis_wallet,omitempty"`
	ReportIsBlacklistedBytecode        bool   `json:"report_is_blacklisted_bytecode,omitempty"`
	ReportHasMintRisk                  bool   `json:"report_has_mint_risk,omitempty"`
	GenesisWalletsFetchedAt            string `json:"genesis_wallets_fetched_at,omitempty"`
	CreatorHistoricalProjectsFetchedAt string `json:"creator_historical_projects_fetched_at,omitempty"`
}

type projectAveDetailUpsertPayload struct {
	Contract string                    `json:"contract"`
	Detail   appstore.ProjectAveDetail `json:"detail"`
}

type projectCreatorResultUpdatePayload struct {
	Contract                           string `json:"contract"`
	CanMintFromDeadViaTransferFrom     bool   `json:"can_mint_from_dead_via_transfer_from"`
	CanMintFromZeroViaTransferFrom     bool   `json:"can_mint_from_zero_via_transfer_from"`
	CanMintFromWethPairViaTransferFrom bool   `json:"can_mint_from_weth_pair_via_transfer_from"`
	CanMintFromUsdtPairViaTransferFrom bool   `json:"can_mint_from_usdt_pair_via_transfer_from"`
	CanMintViaTransferToWethPair       bool   `json:"can_mint_via_transfer_to_weth_pair"`
	CanMintViaTransferToUsdtPair       bool   `json:"can_mint_via_transfer_to_usdt_pair"`
}

type projectReportUpdatePayload struct {
	Contract                   string `json:"contract"`
	IsPolicyEvaluated          bool   `json:"is_policy_evaluated"`
	IsBlacklistedCreatorWallet bool   `json:"is_blacklisted_creator_wallet"`
	IsBlacklistedGenesisWallet bool   `json:"is_blacklisted_genesis_wallet"`
	IsBlacklistedBytecode      bool   `json:"is_blacklisted_bytecode"`
	HasMintRisk                bool   `json:"has_mint_risk"`
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

type projectCreatorHistoricalProjectReplacePayload struct {
	Contract string                                       `json:"contract"`
	Items    []projectCreatorHistoricalProjectItemPayload `json:"items"`
}

type projectCreatorHistoricalProjectItemPayload struct {
	Contract  string `json:"contract"`
	RankIndex int32  `json:"rank_index"`
}

type walletBlacklistAddPayload struct {
	Wallet string `json:"wallet"`
	Note   string `json:"note,omitempty"`
}

type walletBlacklistUpdateNotePayload struct {
	Wallet string `json:"wallet"`
	Note   string `json:"note,omitempty"`
}

type walletBlacklistDeletePayload struct {
	Wallet string `json:"wallet"`
}

type PersistenceEventPublisher interface {
	Publish(ctx context.Context, event PersistenceEvent) error
	PublishProjectMetaSave(ctx context.Context, meta appstore.ProjectMeta) error
	PublishProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error
	PublishProjectAveDetailUpsert(ctx context.Context, contract common.Address, detail appstore.ProjectAveDetail) error
	PublishProjectCreatorResultUpdate(ctx context.Context, contract common.Address, result SimulateResult) error
	PublishProjectReportUpdate(ctx context.Context, contract common.Address, report ProjectReport) error
	PublishProjectCreatorHistoricalProjectsReplace(ctx context.Context, contract common.Address, items []appstore.ProjectCreatorHistoricalProject) error
	PublishWalletBlacklistAdd(ctx context.Context, item appstore.WalletBlacklistEntry) error
	PublishWalletBlacklistUpdateNote(ctx context.Context, wallet common.Address, note string) error
	PublishWalletBlacklistDelete(ctx context.Context, wallet common.Address) error
}

type PersistenceEventWriter interface {
	WriteProjectMeta(ctx context.Context, meta appstore.ProjectMeta) error
	WriteProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error
	WriteProjectAveDetail(ctx context.Context, contract common.Address, detail appstore.ProjectAveDetail) error
	WriteProjectCreatorResult(ctx context.Context, contract common.Address, result appstore.SimulateResult) error
	WriteProjectReport(ctx context.Context, contract common.Address, report appstore.ProjectReport) error
	WriteProjectCreatorHistoricalProjects(ctx context.Context, contract common.Address, items []appstore.ProjectCreatorHistoricalProject) error
	AddWalletBlacklistEntry(ctx context.Context, item appstore.WalletBlacklistEntry) error
	UpdateWalletBlacklistEntryNote(ctx context.Context, wallet common.Address, note string) error
	DeleteWalletBlacklistEntry(ctx context.Context, wallet common.Address) error
}

type projectGenesisWalletWriter interface {
	WriteProjectGenesisWallets(ctx context.Context, contract common.Address, items []appstore.ProjectGenesisWallet) error
}

type PersistenceEventConsumer interface {
	Start(ctx context.Context, writer PersistenceEventWriter) error
}

type persistenceRedisClient interface {
	redisport.StreamClient
	redisport.KVReaderWriter
}

type RedisPersistenceEventBus struct {
	client           persistenceRedisClient
	stream           string
	group            string
	consumer         string
	deadLetterStream string
}

func NewRedisPersistenceEventBus(client persistenceRedisClient) *RedisPersistenceEventBus {
	if client == nil {
		return nil
	}
	return &RedisPersistenceEventBus{
		client:           client,
		stream:           persistenceStreamKey,
		group:            persistenceGroupName,
		consumer:         persistenceConsumerID,
		deadLetterStream: persistenceDeadLetterStreamKey,
	}
}

func (b *RedisPersistenceEventBus) Publish(ctx context.Context, event PersistenceEvent) error {
	if b == nil || b.client == nil {
		return errors.New("persistence event redis client is nil")
	}
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
	if err := b.client.XAdd(ctx, redisport.XAddInput{
		Stream: b.stream,
		Values: map[string]any{"event": string(encoded)},
	}); err != nil {
		return fmt.Errorf("publish persistence event: %w", err)
	}
	return nil
}

func (b *RedisPersistenceEventBus) PublishProjectMetaSave(ctx context.Context, meta appstore.ProjectMeta) error {
	payload := projectMetaSavePayload{
		BlockTime:                          meta.BlockTime,
		BlockNumber:                        meta.BlockNumber,
		Contract:                           meta.Contract.Hex(),
		Creator:                            meta.Creator.Hex(),
		WethPair:                           meta.WethPair.Hex(),
		UsdtPair:                           meta.UsdtPair.Hex(),
		FetchAt:                            timeToPayload(meta.FetchAt),
		TxHash:                             meta.TxHash.Hex(),
		TxIndex:                            meta.TxIndex,
		ReportIsPolicyEvaluated:            meta.Report.IsPolicyEvaluated,
		ReportIsBlacklistedCreatorWallet:   meta.Report.IsBlacklistedCreatorWallet,
		ReportIsBlacklistedGenesisWallet:   meta.Report.IsBlacklistedGenesisWallet,
		ReportIsBlacklistedBytecode:        meta.Report.IsBlacklistedBytecode,
		ReportHasMintRisk:                  meta.Report.HasMintRisk,
		GenesisWalletsFetchedAt:            timeToPayload(meta.GenesisWalletsFetchedAt),
		CreatorHistoricalProjectsFetchedAt: timeToPayload(meta.CreatorHistoricalProjectsFetchedAt),
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

func timeToPayload(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func timeFromPayload(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func (b *RedisPersistenceEventBus) PublishProjectAveDetailUpsert(ctx context.Context, contract common.Address, detail appstore.ProjectAveDetail) error {
	payload := projectAveDetailUpsertPayload{Contract: contract.Hex(), Detail: detail}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal project ave detail payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectAveDetail,
		Contract:   contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishProjectCreatorResultUpdate(ctx context.Context, contract common.Address, result SimulateResult) error {
	payload := projectCreatorResultUpdatePayload{
		Contract:                           contract.Hex(),
		CanMintFromDeadViaTransferFrom:     result.CanMintFromDeadViaTransferFrom,
		CanMintFromZeroViaTransferFrom:     result.CanMintFromZeroViaTransferFrom,
		CanMintFromWethPairViaTransferFrom: result.CanMintFromWethPairViaTransferFrom,
		CanMintFromUsdtPairViaTransferFrom: result.CanMintFromUsdtPairViaTransferFrom,
		CanMintViaTransferToWethPair:       result.CanMintViaTransferToWethPair,
		CanMintViaTransferToUsdtPair:       result.CanMintViaTransferToUsdtPair,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal project creator result payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectCreatorResult,
		Contract:   contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishProjectReportUpdate(ctx context.Context, contract common.Address, report ProjectReport) error {
	payload := projectReportUpdatePayload{
		Contract:                   contract.Hex(),
		IsPolicyEvaluated:          report.IsPolicyEvaluated,
		IsBlacklistedCreatorWallet: report.IsBlacklistedCreatorWallet,
		IsBlacklistedGenesisWallet: report.IsBlacklistedGenesisWallet,
		IsBlacklistedBytecode:      report.IsBlacklistedBytecode,
		HasMintRisk:                report.HasMintRisk,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal project report payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectReport,
		Contract:   contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishProjectCreatorHistoricalProjectsReplace(ctx context.Context, contract common.Address, items []appstore.ProjectCreatorHistoricalProject) error {
	payload := projectCreatorHistoricalProjectReplacePayload{
		Contract: contract.Hex(),
		Items:    make([]projectCreatorHistoricalProjectItemPayload, 0, len(items)),
	}
	for _, item := range items {
		payload.Items = append(payload.Items, projectCreatorHistoricalProjectItemPayload{
			Contract:  item.HistoricalProjectContract.Hex(),
			RankIndex: item.RankIndex,
		})
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal project creator historical projects payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectCreatorHistoricalReplace,
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

func (b *RedisPersistenceEventBus) PublishWalletBlacklistAdd(ctx context.Context, item appstore.WalletBlacklistEntry) error {
	payload := walletBlacklistAddPayload{
		Wallet: item.Wallet.Hex(),
		Note:   item.Note,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal wallet blacklist add payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpWalletBlacklistAdd,
		Contract:   item.Wallet.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishWalletBlacklistUpdateNote(ctx context.Context, wallet common.Address, note string) error {
	payload := walletBlacklistUpdateNotePayload{
		Wallet: wallet.Hex(),
		Note:   note,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal wallet blacklist update note payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpWalletBlacklistNote,
		Contract:   wallet.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) PublishWalletBlacklistDelete(ctx context.Context, wallet common.Address) error {
	payload := walletBlacklistDeletePayload{
		Wallet: wallet.Hex(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal wallet blacklist delete payload: %w", err)
	}
	return b.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpWalletBlacklistDel,
		Contract:   wallet.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (b *RedisPersistenceEventBus) Start(ctx context.Context, writer PersistenceEventWriter) error {
	if b == nil || b.client == nil {
		return errors.New("persistence event redis client is nil")
	}
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
	if b == nil || b.client == nil {
		return 0, errors.New("persistence event redis client is nil")
	}
	streams, err := b.client.XReadGroup(ctx, redisport.XReadGroupInput{
		Group:    b.group,
		Consumer: b.consumer,
		Streams:  []string{b.stream, id},
		Count:    persistenceReadCount,
		Block:    block,
	})
	if err != nil {
		if errors.Is(err, redisport.ErrNotFound) {
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

func (b *RedisPersistenceEventBus) handleMessage(ctx context.Context, writer PersistenceEventWriter, message redisport.XMessage) error {
	if b == nil || b.client == nil {
		return errors.New("persistence event redis client is nil")
	}
	if err := b.processMessage(ctx, writer, message); err != nil {
		return b.handleMessageFailure(ctx, message, err)
	}
	if err := b.client.XAck(ctx, b.stream, b.group, message.ID); err != nil {
		return fmt.Errorf("ack persistence event %s: %w", message.ID, err)
	}
	if err := b.client.Del(ctx, persistenceRetryKey(message.ID)); err != nil {
		return fmt.Errorf("clear persistence event retry count %s: %w", message.ID, err)
	}
	return nil
}

func (b *RedisPersistenceEventBus) processMessage(ctx context.Context, writer PersistenceEventWriter, message redisport.XMessage) error {
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
	return b.applyEvent(ctx, writer, event)
}

func (b *RedisPersistenceEventBus) handleMessageFailure(ctx context.Context, message redisport.XMessage, cause error) error {
	attempts, err := b.incrementRetryCount(ctx, message.ID)
	if err != nil {
		return err
	}
	if attempts < persistenceMaxRetries {
		return cause
	}
	if err := b.publishDeadLetter(ctx, message, cause, attempts); err != nil {
		return err
	}
	if err := b.client.XAck(ctx, b.stream, b.group, message.ID); err != nil {
		return fmt.Errorf("ack persistence event %s: %w", message.ID, err)
	}
	if err := b.client.Del(ctx, persistenceRetryKey(message.ID)); err != nil {
		return fmt.Errorf("clear persistence event retry count %s: %w", message.ID, err)
	}
	return nil
}

func (b *RedisPersistenceEventBus) incrementRetryCount(ctx context.Context, messageID string) (int, error) {
	key := persistenceRetryKey(messageID)
	value, err := b.client.Get(ctx, key)
	if err != nil && !errors.Is(err, redisport.ErrNotFound) {
		return 0, fmt.Errorf("get persistence event retry count %s: %w", messageID, err)
	}
	attempts := 0
	if value != "" {
		parsed, parseErr := strconv.Atoi(value)
		if parseErr != nil {
			return 0, fmt.Errorf("parse persistence event retry count %s: %w", messageID, parseErr)
		}
		attempts = parsed
	}
	attempts++
	if err := b.client.Set(ctx, key, strconv.Itoa(attempts), 0); err != nil {
		return 0, fmt.Errorf("set persistence event retry count %s: %w", messageID, err)
	}
	return attempts, nil
}

func (b *RedisPersistenceEventBus) publishDeadLetter(ctx context.Context, message redisport.XMessage, cause error, attempts int) error {
	rawValues, err := json.Marshal(message.Values)
	if err != nil {
		rawValues = []byte(fmt.Sprintf("%v", message.Values))
	}
	payload := ""
	if value, ok := message.Values["event"].(string); ok {
		payload = value
	}
	if err := b.client.XAdd(ctx, redisport.XAddInput{
		Stream: b.deadLetterStream,
		Values: map[string]any{
			"message_id": message.ID,
			"payload":    payload,
			"raw_values": string(rawValues),
			"error":      cause.Error(),
			"attempts":   strconv.Itoa(attempts),
			"failed_at":  time.Now().UTC().Format(time.RFC3339Nano),
		},
	}); err != nil {
		return fmt.Errorf("publish dead-letter persistence event %s: %w", message.ID, err)
	}
	return nil
}

func persistenceRetryKey(messageID string) string {
	return "application:persist:retry:" + messageID
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
			WethPair:    common.HexToAddress(payload.WethPair),
			UsdtPair:    common.HexToAddress(payload.UsdtPair),
			TxHash:      txHash,
			TxIndex:     payload.TxIndex,
			Report: appstore.ProjectReport{
				IsPolicyEvaluated:          payload.ReportIsPolicyEvaluated,
				IsBlacklistedCreatorWallet: payload.ReportIsBlacklistedCreatorWallet,
				IsBlacklistedGenesisWallet: payload.ReportIsBlacklistedGenesisWallet,
				IsBlacklistedBytecode:      payload.ReportIsBlacklistedBytecode,
				HasMintRisk:                payload.ReportHasMintRisk,
			},
		}
		var err error
		if meta.FetchAt, err = timeFromPayload(payload.FetchAt); err != nil {
			return fmt.Errorf("parse fetch_at %q: %w", payload.FetchAt, err)
		}
		if meta.GenesisWalletsFetchedAt, err = timeFromPayload(payload.GenesisWalletsFetchedAt); err != nil {
			return fmt.Errorf("parse genesis_wallets_fetched_at %q: %w", payload.GenesisWalletsFetchedAt, err)
		}
		if meta.CreatorHistoricalProjectsFetchedAt, err = timeFromPayload(payload.CreatorHistoricalProjectsFetchedAt); err != nil {
			return fmt.Errorf("parse creator_historical_projects_fetched_at %q: %w", payload.CreatorHistoricalProjectsFetchedAt, err)
		}
		return writer.WriteProjectMeta(ctx, meta)
	case PersistenceOpProjectAveDetail:
		var payload projectAveDetailUpsertPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal project ave detail payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		return writer.WriteProjectAveDetail(ctx, common.HexToAddress(payload.Contract), payload.Detail)
	case PersistenceOpProjectCreatorResult:
		var payload projectCreatorResultUpdatePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal project creator result payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		return writer.WriteProjectCreatorResult(ctx, common.HexToAddress(payload.Contract), appstore.SimulateResult{
			CanMintFromDeadViaTransferFrom:     payload.CanMintFromDeadViaTransferFrom,
			CanMintFromZeroViaTransferFrom:     payload.CanMintFromZeroViaTransferFrom,
			CanMintFromWethPairViaTransferFrom: payload.CanMintFromWethPairViaTransferFrom,
			CanMintFromUsdtPairViaTransferFrom: payload.CanMintFromUsdtPairViaTransferFrom,
			CanMintViaTransferToWethPair:       payload.CanMintViaTransferToWethPair,
			CanMintViaTransferToUsdtPair:       payload.CanMintViaTransferToUsdtPair,
		})
	case PersistenceOpProjectReport:
		var payload projectReportUpdatePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal project report payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		return writer.WriteProjectReport(ctx, common.HexToAddress(payload.Contract), appstore.ProjectReport{
			IsPolicyEvaluated:          payload.IsPolicyEvaluated,
			IsBlacklistedCreatorWallet: payload.IsBlacklistedCreatorWallet,
			IsBlacklistedGenesisWallet: payload.IsBlacklistedGenesisWallet,
			IsBlacklistedBytecode:      payload.IsBlacklistedBytecode,
			HasMintRisk:                payload.HasMintRisk,
		})
	case PersistenceOpProjectCreatorHistoricalReplace:
		var payload projectCreatorHistoricalProjectReplacePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal project creator historical projects replace payload: %w", err)
		}
		if !common.IsHexAddress(payload.Contract) {
			return fmt.Errorf("invalid contract %q", payload.Contract)
		}
		items := make([]appstore.ProjectCreatorHistoricalProject, 0, len(payload.Items))
		for _, entry := range payload.Items {
			if !common.IsHexAddress(entry.Contract) {
				return fmt.Errorf("invalid historical project contract %q", entry.Contract)
			}
			items = append(items, appstore.ProjectCreatorHistoricalProject{
				ProjectContract:           common.HexToAddress(payload.Contract),
				HistoricalProjectContract: common.HexToAddress(entry.Contract),
				RankIndex:                 entry.RankIndex,
			})
		}
		return writer.WriteProjectCreatorHistoricalProjects(ctx, common.HexToAddress(payload.Contract), items)
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
	case PersistenceOpWalletBlacklistAdd:
		var payload walletBlacklistAddPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal wallet blacklist add payload: %w", err)
		}
		if !common.IsHexAddress(payload.Wallet) {
			return fmt.Errorf("invalid wallet %q", payload.Wallet)
		}
		err := writer.AddWalletBlacklistEntry(ctx, appstore.WalletBlacklistEntry{
			Wallet: common.HexToAddress(payload.Wallet),
			Note:   payload.Note,
		})
		if errors.Is(err, appstore.ErrWalletBlacklistEntryAlreadyExists) {
			return nil
		}
		return err
	case PersistenceOpWalletBlacklistNote:
		var payload walletBlacklistUpdateNotePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal wallet blacklist update note payload: %w", err)
		}
		if !common.IsHexAddress(payload.Wallet) {
			return fmt.Errorf("invalid wallet %q", payload.Wallet)
		}
		return writer.UpdateWalletBlacklistEntryNote(ctx, common.HexToAddress(payload.Wallet), payload.Note)
	case PersistenceOpWalletBlacklistDel:
		var payload walletBlacklistDeletePayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return fmt.Errorf("unmarshal wallet blacklist delete payload: %w", err)
		}
		if !common.IsHexAddress(payload.Wallet) {
			return fmt.Errorf("invalid wallet %q", payload.Wallet)
		}
		err := writer.DeleteWalletBlacklistEntry(ctx, common.HexToAddress(payload.Wallet))
		if errors.Is(err, appstore.ErrWalletBlacklistEntryNotFound) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unsupported persistence event op %q", event.Op)
	}
}

func (b *RedisPersistenceEventBus) ensureConsumerGroup(ctx context.Context) error {
	if b == nil || b.client == nil {
		return errors.New("persistence event redis client is nil")
	}
	err := b.client.XGroupCreateMkStream(ctx, b.stream, b.group, "0")
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

func (w *storePersistenceWriter) WriteProjectAveDetail(ctx context.Context, contract common.Address, detail appstore.ProjectAveDetail) error {
	return w.store.UpsertProjectAveDetail(ctx, contract, detail)
}

func (w *storePersistenceWriter) WriteProjectCreatorResult(ctx context.Context, contract common.Address, result appstore.SimulateResult) error {
	return w.store.UpdateProjectCreatorResult(ctx, contract, result)
}

func (w *storePersistenceWriter) WriteProjectReport(ctx context.Context, contract common.Address, report appstore.ProjectReport) error {
	return w.store.UpdateProjectReport(ctx, contract, report)
}

func (w *storePersistenceWriter) WriteProjectCreatorHistoricalProjects(ctx context.Context, contract common.Address, items []appstore.ProjectCreatorHistoricalProject) error {
	store, ok := w.store.(appstore.ProjectCreatorHistoricalProjectStore)
	if !ok || store == nil {
		return errors.New("project creator historical project store is not configured")
	}
	return store.ReplaceProjectCreatorHistoricalProjects(ctx, contract, items)
}

func (w *storePersistenceWriter) AddWalletBlacklistEntry(ctx context.Context, item appstore.WalletBlacklistEntry) error {
	store, ok := w.store.(appstore.WalletBlacklistStore)
	if !ok || store == nil {
		return errors.New("wallet blacklist entry store is not configured")
	}
	return store.AddWalletBlacklistEntry(ctx, item)
}

func (w *storePersistenceWriter) UpdateWalletBlacklistEntryNote(ctx context.Context, wallet common.Address, note string) error {
	store, ok := w.store.(appstore.WalletBlacklistStore)
	if !ok || store == nil {
		return errors.New("wallet blacklist entry store is not configured")
	}
	return store.UpdateWalletBlacklistEntryNote(ctx, wallet, note)
}

func (w *storePersistenceWriter) DeleteWalletBlacklistEntry(ctx context.Context, wallet common.Address) error {
	store, ok := w.store.(appstore.WalletBlacklistStore)
	if !ok || store == nil {
		return errors.New("wallet blacklist entry store is not configured")
	}
	return store.DeleteWalletBlacklistEntry(ctx, wallet)
}

func (w *storePersistenceWriter) WriteProjectGenesisWallets(ctx context.Context, contract common.Address, items []appstore.ProjectGenesisWallet) error {
	store, ok := w.store.(appstore.ProjectGenesisWalletStore)
	if !ok || store == nil {
		return errors.New("project genesis wallet store is not configured")
	}
	return store.ReplaceProjectGenesisWallets(ctx, contract, items)
}
