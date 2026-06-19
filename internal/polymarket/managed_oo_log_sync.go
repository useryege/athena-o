package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
)

const (
	managedOOProposePriceSyncName        = "managed_oo_propose_price"
	managedOOContractAddress             = "0x2C0367a9DB231dDeBd88a94b4f6461a6e47C58B1"
	managedOOProposePriceTopic           = "0x6e51dd00371aabffa82cd401592f76ed51e98a9ea4b58751c70463a2c78b5ca1"
	managedOOLogPollInterval             = 10 * time.Second
	managedOOLogQueryTimeout             = 20 * time.Second
	managedOOLogMaxBlockRange     uint64 = 2000
)

var managedOOProposePriceABI = mustManagedOOProposePriceABI()
var managedOOMarketIDPattern = regexp.MustCompile(`(?i)market_id\s*:\s*([^\s,]+)`)

func (s *Service) runManagedOOProposePriceLogSyncLoop(ctx context.Context) {
	defer s.runWG.Done()
	if err := s.syncManagedOOProposePriceLogs(ctx); err != nil {
		log.WithError(err).Warn("initial polymarket managed oo propose price log sync failed")
	}
	ticker := time.NewTicker(managedOOLogPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.syncManagedOOProposePriceLogs(ctx); err != nil {
				log.WithError(err).Warn("periodic polymarket managed oo propose price log sync failed")
			}
		}
	}
}

func (s *Service) syncManagedOOProposePriceLogs(ctx context.Context) error {
	if strings.TrimSpace(s.fifaPolygonRPCURL) == "" {
		return fmt.Errorf("polymarket polygon rpc url is required")
	}
	queryCtx, cancel := context.WithTimeout(ctx, managedOOLogQueryTimeout)
	defer cancel()

	client, err := ethclient.DialContext(queryCtx, s.fifaPolygonRPCURL)
	if err != nil {
		return fmt.Errorf("dial polygon rpc for managed oo logs: %w", err)
	}
	defer client.Close()

	latest, err := client.BlockNumber(queryCtx)
	if err != nil {
		return fmt.Errorf("get polygon latest block for managed oo logs: %w", err)
	}
	now := s.now().UTC()
	cursor, err := s.store.GetPolymarketChainLogCursor(ctx, managedOOProposePriceSyncName)
	if err != nil {
		return err
	}
	if cursor == nil {
		_, err := s.store.UpsertPolymarketChainLogCursor(ctx, polymarketstore.ChainLogCursor{
			SyncName:        managedOOProposePriceSyncName,
			ContractAddress: managedOOContractAddress,
			Topic:           managedOOProposePriceTopic,
			LastBlockNumber: latest,
			LastPolledAt:    now,
		})
		if err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"sync_name":    managedOOProposePriceSyncName,
			"block_number": latest,
		}).Info("initialized polymarket managed oo propose price log cursor")
		return nil
	}

	if latest <= cursor.LastBlockNumber {
		_, err := s.store.UpsertPolymarketChainLogCursor(ctx, polymarketstore.ChainLogCursor{
			SyncName:        managedOOProposePriceSyncName,
			ContractAddress: managedOOContractAddress,
			Topic:           managedOOProposePriceTopic,
			LastBlockNumber: cursor.LastBlockNumber,
			LastPolledAt:    now,
		})
		return err
	}

	fromBlock := cursor.LastBlockNumber + 1
	items, err := s.fetchManagedOOProposePriceLogs(ctx, client, fromBlock, latest, now)
	if err != nil {
		return err
	}
	if err := s.store.IngestManagedOOProposePriceLogs(ctx, polymarketstore.ChainLogCursor{
		SyncName:        managedOOProposePriceSyncName,
		ContractAddress: managedOOContractAddress,
		Topic:           managedOOProposePriceTopic,
		LastBlockNumber: latest,
		LastPolledAt:    now,
	}, items); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"sync_name":    managedOOProposePriceSyncName,
		"from_block":   fromBlock,
		"to_block":     latest,
		"log_count":    len(items),
		"rpc_endpoint": sanitizedRPCURL(s.fifaPolygonRPCURL),
	}).Debug("synced polymarket managed oo propose price logs")
	return nil
}

func (s *Service) fetchManagedOOProposePriceLogs(ctx context.Context, client *ethclient.Client, fromBlock, toBlock uint64, fetchedAt time.Time) ([]polymarketstore.ManagedOOProposePriceLog, error) {
	contractAddress := ethcommon.HexToAddress(managedOOContractAddress)
	topic := ethcommon.HexToHash(managedOOProposePriceTopic)
	out := make([]polymarketstore.ManagedOOProposePriceLog, 0)
	for start := fromBlock; start <= toBlock; {
		end := start + managedOOLogMaxBlockRange - 1
		if end > toBlock {
			end = toBlock
		}
		queryCtx, cancel := context.WithTimeout(ctx, managedOOLogQueryTimeout)
		logs, err := client.FilterLogs(queryCtx, ethereum.FilterQuery{
			FromBlock: new(big.Int).SetUint64(start),
			ToBlock:   new(big.Int).SetUint64(end),
			Addresses: []ethcommon.Address{contractAddress},
			Topics:    [][]ethcommon.Hash{{topic}},
		})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("filter managed oo propose price logs %d-%d: %w", start, end, err)
		}
		for _, item := range logs {
			parsed, err := managedOOProposePriceLogFromEthereumLog(item, fetchedAt)
			if err != nil {
				return nil, err
			}
			out = append(out, parsed)
		}
		if end == toBlock {
			break
		}
		start = end + 1
	}
	return out, nil
}

func managedOOProposePriceLogFromEthereumLog(item types.Log, fetchedAt time.Time) (polymarketstore.ManagedOOProposePriceLog, error) {
	if len(item.Topics) < 3 {
		return polymarketstore.ManagedOOProposePriceLog{}, fmt.Errorf("managed oo propose price log has %d topics", len(item.Topics))
	}
	values, err := managedOOProposePriceABI.Unpack("ProposePrice", item.Data)
	if err != nil {
		return polymarketstore.ManagedOOProposePriceLog{}, fmt.Errorf("unpack managed oo propose price log %s/%d: %w", item.TxHash.Hex(), item.Index, err)
	}
	if len(values) != 6 {
		return polymarketstore.ManagedOOProposePriceLog{}, fmt.Errorf("managed oo propose price log %s/%d returned %d values", item.TxHash.Hex(), item.Index, len(values))
	}
	identifier, ok := values[0].([32]byte)
	if !ok {
		return polymarketstore.ManagedOOProposePriceLog{}, fmt.Errorf("managed oo propose price log %s/%d has invalid identifier", item.TxHash.Hex(), item.Index)
	}
	requestTimestamp, err := abiUint64(values[1], "request timestamp", item)
	if err != nil {
		return polymarketstore.ManagedOOProposePriceLog{}, err
	}
	ancillaryData, ok := values[2].([]byte)
	if !ok {
		return polymarketstore.ManagedOOProposePriceLog{}, fmt.Errorf("managed oo propose price log %s/%d has invalid ancillary data", item.TxHash.Hex(), item.Index)
	}
	proposedPrice, ok := values[3].(*big.Int)
	if !ok || proposedPrice == nil {
		return polymarketstore.ManagedOOProposePriceLog{}, fmt.Errorf("managed oo propose price log %s/%d has invalid proposed price", item.TxHash.Hex(), item.Index)
	}
	expirationTimestamp, err := abiUint64(values[4], "expiration timestamp", item)
	if err != nil {
		return polymarketstore.ManagedOOProposePriceLog{}, err
	}
	currency, ok := values[5].(ethcommon.Address)
	if !ok {
		return polymarketstore.ManagedOOProposePriceLog{}, fmt.Errorf("managed oo propose price log %s/%d has invalid currency", item.TxHash.Hex(), item.Index)
	}
	rawTopics, err := marshalLogTopics(item.Topics)
	if err != nil {
		return polymarketstore.ManagedOOProposePriceLog{}, err
	}
	ancillaryText := ancillaryDataText(ancillaryData)

	return polymarketstore.ManagedOOProposePriceLog{
		TxHash:              item.TxHash.Hex(),
		LogIndex:            item.Index,
		BlockNumber:         item.BlockNumber,
		BlockHash:           item.BlockHash.Hex(),
		TxIndex:             item.TxIndex,
		ContractAddress:     item.Address.Hex(),
		Topic:               item.Topics[0].Hex(),
		Requester:           ethcommon.BytesToAddress(item.Topics[1].Bytes()).Hex(),
		Proposer:            ethcommon.BytesToAddress(item.Topics[2].Bytes()).Hex(),
		Identifier:          ethcommon.BytesToHash(identifier[:]).Hex(),
		RequestTimestamp:    requestTimestamp,
		AncillaryDataHex:    hexutil.Encode(ancillaryData),
		AncillaryDataText:   ancillaryText,
		MarketID:            marketIDFromAncillaryDataText(ancillaryText),
		ProposedPrice:       proposedPrice.String(),
		ExpirationTimestamp: expirationTimestamp,
		Currency:            currency.Hex(),
		RawTopics:           rawTopics,
		RawData:             hexutil.Encode(item.Data),
		FetchedAt:           fetchedAt,
	}, nil
}

func mustManagedOOProposePriceABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"requester","type":"address"},{"indexed":true,"internalType":"address","name":"proposer","type":"address"},{"indexed":false,"internalType":"bytes32","name":"identifier","type":"bytes32"},{"indexed":false,"internalType":"uint256","name":"timestamp","type":"uint256"},{"indexed":false,"internalType":"bytes","name":"ancillaryData","type":"bytes"},{"indexed":false,"internalType":"int256","name":"proposedPrice","type":"int256"},{"indexed":false,"internalType":"uint256","name":"expirationTimestamp","type":"uint256"},{"indexed":false,"internalType":"address","name":"currency","type":"address"}],"name":"ProposePrice","type":"event"}]`))
	if err != nil {
		panic(err)
	}
	return parsed
}

func abiUint64(value any, label string, item types.Log) (uint64, error) {
	raw, ok := value.(*big.Int)
	if !ok || raw == nil || raw.Sign() < 0 || !raw.IsUint64() {
		return 0, fmt.Errorf("managed oo propose price log %s/%d has invalid %s", item.TxHash.Hex(), item.Index, label)
	}
	return raw.Uint64(), nil
}

func marshalLogTopics(topics []ethcommon.Hash) (json.RawMessage, error) {
	values := make([]string, 0, len(topics))
	for _, topic := range topics {
		values = append(values, topic.Hex())
	}
	raw, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("marshal log topics: %w", err)
	}
	return raw, nil
}

func ancillaryDataText(value []byte) string {
	if !utf8.Valid(value) {
		return ""
	}
	return string(value)
}

func marketIDFromAncillaryDataText(value string) string {
	matches := managedOOMarketIDPattern.FindStringSubmatch(value)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func sanitizedRPCURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if idx := strings.LastIndex(value, "/"); idx > len("https://") && idx < len(value)-1 {
		return value[:idx+1] + "..."
	}
	return value
}
