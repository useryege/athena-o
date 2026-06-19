package polymarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
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
	utilpolymarket "github.com/useryege/athena/util/polymarket"
)

const (
	managedOOProposePriceSyncName          = "managed_oo_propose_price"
	managedOODisputePriceSyncName          = "managed_oo_dispute_price"
	managedOOContractAddress               = "0x2C0367a9DB231dDeBd88a94b4f6461a6e47C58B1"
	managedOOProposePriceTopic             = "0x6e51dd00371aabffa82cd401592f76ed51e98a9ea4b58751c70463a2c78b5ca1"
	managedOODisputePriceTopic             = "0x5165909c3d1c01c5d1e121ac6f6d01dda1ba24bc9e1f975b5a375339c15be7f3"
	managedOOLogPollInterval               = 2 * time.Second
	managedOOLogQueryTimeout               = 20 * time.Second
	managedOOMarketDataQueryTimeout        = 20 * time.Second
	managedOOMarketDataMissingLimit        = 100
	managedOOLogMaxBlockRange       uint64 = 2000
)

var managedOOProposePriceABI = mustManagedOOProposePriceABI()
var managedOODisputePriceABI = mustManagedOODisputePriceABI()
var managedOOMarketIDPattern = regexp.MustCompile(`(?i)market_id\s*:\s*([^\s,]+)`)

func (s *Service) runManagedOOProposePriceLogSyncLoop(ctx context.Context) {
	defer s.runWG.Done()
	s.syncManagedOOProposePriceLogsMarketsAndAlerts(ctx, "initial")
	ticker := time.NewTicker(managedOOLogPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncManagedOOProposePriceLogsMarketsAndAlerts(ctx, "periodic")
		}
	}
}

func (s *Service) syncManagedOOProposePriceLogsMarketsAndAlerts(ctx context.Context, phase string) {
	if err := s.syncManagedOOProposePriceLogs(ctx); err != nil {
		log.WithError(err).Warnf("%s polymarket managed oo propose price log sync failed", phase)
		return
	}
	if err := s.syncManagedOODisputePriceLogs(ctx); err != nil {
		log.WithError(err).Warnf("%s polymarket managed oo dispute price log sync failed", phase)
		return
	}
	if err := s.syncManagedOOMarketData(ctx); err != nil {
		log.WithError(err).Warnf("%s polymarket managed oo market data sync failed", phase)
		return
	}
	s.sendManagedOOProposePriceAlerts(ctx)
	s.sendManagedOODisputePriceAlerts(ctx)
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

func (s *Service) syncManagedOODisputePriceLogs(ctx context.Context) error {
	if strings.TrimSpace(s.fifaPolygonRPCURL) == "" {
		return fmt.Errorf("polymarket polygon rpc url is required")
	}
	queryCtx, cancel := context.WithTimeout(ctx, managedOOLogQueryTimeout)
	defer cancel()

	client, err := ethclient.DialContext(queryCtx, s.fifaPolygonRPCURL)
	if err != nil {
		return fmt.Errorf("dial polygon rpc for managed oo dispute logs: %w", err)
	}
	defer client.Close()

	latest, err := client.BlockNumber(queryCtx)
	if err != nil {
		return fmt.Errorf("get polygon latest block for managed oo dispute logs: %w", err)
	}
	now := s.now().UTC()
	cursor, err := s.store.GetPolymarketChainLogCursor(ctx, managedOODisputePriceSyncName)
	if err != nil {
		return err
	}
	if cursor == nil {
		_, err := s.store.UpsertPolymarketChainLogCursor(ctx, polymarketstore.ChainLogCursor{
			SyncName:        managedOODisputePriceSyncName,
			ContractAddress: managedOOContractAddress,
			Topic:           managedOODisputePriceTopic,
			LastBlockNumber: latest,
			LastPolledAt:    now,
		})
		if err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"sync_name":    managedOODisputePriceSyncName,
			"block_number": latest,
		}).Info("initialized polymarket managed oo dispute price log cursor")
		return nil
	}

	if latest <= cursor.LastBlockNumber {
		_, err := s.store.UpsertPolymarketChainLogCursor(ctx, polymarketstore.ChainLogCursor{
			SyncName:        managedOODisputePriceSyncName,
			ContractAddress: managedOOContractAddress,
			Topic:           managedOODisputePriceTopic,
			LastBlockNumber: cursor.LastBlockNumber,
			LastPolledAt:    now,
		})
		return err
	}

	fromBlock := cursor.LastBlockNumber + 1
	items, err := s.fetchManagedOODisputePriceLogs(ctx, client, fromBlock, latest, now)
	if err != nil {
		return err
	}
	if err := s.store.IngestManagedOODisputePriceLogs(ctx, polymarketstore.ChainLogCursor{
		SyncName:        managedOODisputePriceSyncName,
		ContractAddress: managedOOContractAddress,
		Topic:           managedOODisputePriceTopic,
		LastBlockNumber: latest,
		LastPolledAt:    now,
	}, items); err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"sync_name":    managedOODisputePriceSyncName,
		"from_block":   fromBlock,
		"to_block":     latest,
		"log_count":    len(items),
		"rpc_endpoint": sanitizedRPCURL(s.fifaPolygonRPCURL),
	}).Debug("synced polymarket managed oo dispute price logs")
	return nil
}

func (s *Service) fetchManagedOODisputePriceLogs(ctx context.Context, client *ethclient.Client, fromBlock, toBlock uint64, fetchedAt time.Time) ([]polymarketstore.ManagedOODisputePriceLog, error) {
	contractAddress := ethcommon.HexToAddress(managedOOContractAddress)
	topic := ethcommon.HexToHash(managedOODisputePriceTopic)
	out := make([]polymarketstore.ManagedOODisputePriceLog, 0)
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
			return nil, fmt.Errorf("filter managed oo dispute price logs %d-%d: %w", start, end, err)
		}
		for _, item := range logs {
			parsed, err := managedOODisputePriceLogFromEthereumLog(item, fetchedAt)
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

func (s *Service) syncManagedOOMarketData(ctx context.Context) error {
	if s.gammaClient == nil {
		return fmt.Errorf("polymarket gamma client is required")
	}
	marketIDs, err := s.store.ListManagedOOMarketIDsMissingData(ctx, managedOOMarketDataMissingLimit)
	if err != nil {
		return err
	}
	if len(marketIDs) == 0 {
		return nil
	}

	fetched := 0
	notFound := 0
	for _, marketID := range marketIDs {
		gammaID, err := strconv.ParseInt(strings.TrimSpace(marketID), 10, 64)
		if err != nil {
			continue
		}

		fetchedAt := s.now().UTC()
		queryCtx, cancel := context.WithTimeout(ctx, managedOOMarketDataQueryTimeout)
		market, err := s.gammaClient.GetMarketByID(queryCtx, gammaID, utilpolymarket.GetMarketOptions{
			IncludeTag: ptrBool(true),
		})
		cancel()
		if err != nil {
			if isPolymarketAPIStatus(err, 404) {
				if storeErr := s.store.UpsertManagedOOMarketNotFound(ctx, marketID, err.Error(), fetchedAt); storeErr != nil {
					return storeErr
				}
				notFound++
				continue
			}
			return fmt.Errorf("fetch managed oo gamma market %s: %w", marketID, err)
		}
		if market == nil {
			return fmt.Errorf("fetch managed oo gamma market %s: empty response", marketID)
		}
		if err := s.store.UpsertManagedOOMarket(ctx, managedOOMarketFromGamma(marketID, *market, fetchedAt)); err != nil {
			return err
		}
		fetched++
	}

	log.WithFields(log.Fields{
		"requested": len(marketIDs),
		"fetched":   fetched,
		"not_found": notFound,
	}).Debug("synced polymarket managed oo market data")
	return nil
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

func managedOODisputePriceLogFromEthereumLog(item types.Log, fetchedAt time.Time) (polymarketstore.ManagedOODisputePriceLog, error) {
	if len(item.Topics) < 4 {
		return polymarketstore.ManagedOODisputePriceLog{}, fmt.Errorf("managed oo dispute price log has %d topics", len(item.Topics))
	}
	values, err := managedOODisputePriceABI.Unpack("DisputePrice", item.Data)
	if err != nil {
		return polymarketstore.ManagedOODisputePriceLog{}, fmt.Errorf("unpack managed oo dispute price log %s/%d: %w", item.TxHash.Hex(), item.Index, err)
	}
	if len(values) != 4 {
		return polymarketstore.ManagedOODisputePriceLog{}, fmt.Errorf("managed oo dispute price log %s/%d returned %d values", item.TxHash.Hex(), item.Index, len(values))
	}
	identifier, ok := values[0].([32]byte)
	if !ok {
		return polymarketstore.ManagedOODisputePriceLog{}, fmt.Errorf("managed oo dispute price log %s/%d has invalid identifier", item.TxHash.Hex(), item.Index)
	}
	requestTimestamp, err := abiUint64(values[1], "request timestamp", item)
	if err != nil {
		return polymarketstore.ManagedOODisputePriceLog{}, err
	}
	ancillaryData, ok := values[2].([]byte)
	if !ok {
		return polymarketstore.ManagedOODisputePriceLog{}, fmt.Errorf("managed oo dispute price log %s/%d has invalid ancillary data", item.TxHash.Hex(), item.Index)
	}
	proposedPrice, ok := values[3].(*big.Int)
	if !ok || proposedPrice == nil {
		return polymarketstore.ManagedOODisputePriceLog{}, fmt.Errorf("managed oo dispute price log %s/%d has invalid proposed price", item.TxHash.Hex(), item.Index)
	}
	rawTopics, err := marshalLogTopics(item.Topics)
	if err != nil {
		return polymarketstore.ManagedOODisputePriceLog{}, err
	}
	ancillaryText := ancillaryDataText(ancillaryData)

	return polymarketstore.ManagedOODisputePriceLog{
		TxHash:            item.TxHash.Hex(),
		LogIndex:          item.Index,
		BlockNumber:       item.BlockNumber,
		BlockHash:         item.BlockHash.Hex(),
		TxIndex:           item.TxIndex,
		ContractAddress:   item.Address.Hex(),
		Topic:             item.Topics[0].Hex(),
		Requester:         ethcommon.BytesToAddress(item.Topics[1].Bytes()).Hex(),
		Proposer:          ethcommon.BytesToAddress(item.Topics[2].Bytes()).Hex(),
		Disputer:          ethcommon.BytesToAddress(item.Topics[3].Bytes()).Hex(),
		Identifier:        ethcommon.BytesToHash(identifier[:]).Hex(),
		RequestTimestamp:  requestTimestamp,
		AncillaryDataHex:  hexutil.Encode(ancillaryData),
		AncillaryDataText: ancillaryText,
		MarketID:          marketIDFromAncillaryDataText(ancillaryText),
		ProposedPrice:     proposedPrice.String(),
		RawTopics:         rawTopics,
		RawData:           hexutil.Encode(item.Data),
		FetchedAt:         fetchedAt,
	}, nil
}

func managedOOMarketFromGamma(marketID string, market utilpolymarket.Market, fetchedAt time.Time) polymarketstore.ManagedOOMarket {
	return polymarketstore.ManagedOOMarket{
		MarketID:         strings.TrimSpace(marketID),
		ConditionID:      strings.TrimSpace(stringValue(market.ConditionID)),
		Slug:             strings.TrimSpace(stringValue(market.Slug)),
		EventSlug:        managedOOMarketEventSlugFromGamma(market.Events),
		Question:         strings.TrimSpace(stringValue(market.Question)),
		Description:      strings.TrimSpace(stringValue(market.Description)),
		ResolutionSource: strings.TrimSpace(stringValue(market.ResolutionSource)),
		QuestionID:       strings.TrimSpace(stringValue(market.QuestionID)),
		SportsMarketType: strings.TrimSpace(stringValue(market.SportsMarketType)),
		GroupItemTitle:   strings.TrimSpace(stringValue(market.GroupItemTitle)),
		Image:            strings.TrimSpace(stringValue(market.Image)),
		Icon:             strings.TrimSpace(stringValue(market.Icon)),
		Outcomes:         strings.TrimSpace(stringValue(market.Outcomes)),
		OutcomePrices:    strings.TrimSpace(stringValue(market.OutcomePrices)),
		ClobTokenIDs:     strings.TrimSpace(stringValue(market.ClobTokenIDs)),
		Active:           boolValue(market.Active),
		Closed:           boolValue(market.Closed),
		Archived:         boolValue(market.Archived),
		Restricted:       boolValue(market.Restricted),
		EnableOrderBook:  boolValue(market.EnableOrderBook),
		AcceptingOrders:  boolValue(market.AcceptingOrders),
		Volume:           strings.TrimSpace(stringValue(market.Volume)),
		VolumeNum:        float64Value(market.VolumeNum),
		LiquidityNum:     float64Value(market.LiquidityNum),
		Volume24hr:       float64Value(market.Volume24hr),
		Volume1wk:        float64Value(market.Volume1wk),
		Volume1mo:        float64Value(market.Volume1mo),
		Volume1yr:        float64Value(market.Volume1yr),
		Spread:           float64Value(market.Spread),
		BestBid:          float64Value(market.BestBid),
		BestAsk:          float64Value(market.BestAsk),
		LastTradePrice:   float64Value(market.LastTradePrice),
		StartDate:        timePtrValue(market.StartDate),
		EndDate:          timePtrValue(market.EndDate),
		CreatedAtGamma:   timePtrValue(market.CreatedAt),
		UpdatedAtGamma:   timePtrValue(market.UpdatedAt),
		Tags:             marshalArrayOrFallback(market.Tags),
		Raw:              rawObjectOrMarshal(market.Raw, market),
		FetchedAt:        fetchedAt,
		Labels:           managedOOMarketLabelsFromGamma(marketID, market.Tags, fetchedAt),
	}
}

func managedOOMarketEventSlugFromGamma(events []utilpolymarket.Event) string {
	for i := range events {
		if slug := strings.TrimSpace(stringValue(events[i].Slug)); slug != "" {
			return slug
		}
	}
	return ""
}

func managedOOMarketLabelsFromGamma(marketID string, tags []utilpolymarket.Tag, fetchedAt time.Time) []polymarketstore.ManagedOOMarketLabel {
	out := make([]polymarketstore.ManagedOOMarketLabel, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for i := range tags {
		label := strings.TrimSpace(stringValue(tags[i].Label))
		if label == "" {
			continue
		}
		if _, ok := seen[label]; ok {
			continue
		}
		seen[label] = struct{}{}
		out = append(out, polymarketstore.ManagedOOMarketLabel{
			MarketID:  strings.TrimSpace(marketID),
			Label:     label,
			TagID:     strings.TrimSpace(tags[i].ID),
			Slug:      strings.TrimSpace(stringValue(tags[i].Slug)),
			Position:  int64(i),
			FetchedAt: fetchedAt,
		})
	}
	return out
}

func isPolymarketAPIStatus(err error, statusCode int) bool {
	var apiErr *utilpolymarket.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == statusCode
}

func mustManagedOOProposePriceABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"requester","type":"address"},{"indexed":true,"internalType":"address","name":"proposer","type":"address"},{"indexed":false,"internalType":"bytes32","name":"identifier","type":"bytes32"},{"indexed":false,"internalType":"uint256","name":"timestamp","type":"uint256"},{"indexed":false,"internalType":"bytes","name":"ancillaryData","type":"bytes"},{"indexed":false,"internalType":"int256","name":"proposedPrice","type":"int256"},{"indexed":false,"internalType":"uint256","name":"expirationTimestamp","type":"uint256"},{"indexed":false,"internalType":"address","name":"currency","type":"address"}],"name":"ProposePrice","type":"event"}]`))
	if err != nil {
		panic(err)
	}
	return parsed
}

func mustManagedOODisputePriceABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(`[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"requester","type":"address"},{"indexed":true,"internalType":"address","name":"proposer","type":"address"},{"indexed":true,"internalType":"address","name":"disputer","type":"address"},{"indexed":false,"internalType":"bytes32","name":"identifier","type":"bytes32"},{"indexed":false,"internalType":"uint256","name":"timestamp","type":"uint256"},{"indexed":false,"internalType":"bytes","name":"ancillaryData","type":"bytes"},{"indexed":false,"internalType":"int256","name":"proposedPrice","type":"int256"}],"name":"DisputePrice","type":"event"}]`))
	if err != nil {
		panic(err)
	}
	return parsed
}

func abiUint64(value any, label string, item types.Log) (uint64, error) {
	raw, ok := value.(*big.Int)
	if !ok || raw == nil || raw.Sign() < 0 || !raw.IsUint64() {
		return 0, fmt.Errorf("managed oo log %s/%d has invalid %s", item.TxHash.Hex(), item.Index, label)
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
