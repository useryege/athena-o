package ethereumapi

import (
	"context"
	stderrors "errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/ethereumapi/apiclient"
	ethereumapistore "github.com/useryege/athena/internal/ethereumapi/store"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultNormalTransactionEndBlock = uint64(999999999)
	defaultNormalTransactionPage     = int32(1)
	defaultNormalTransactionPageSize = int32(100)
	maxNormalTransactionPageSize     = int32(1000)
)

func (s *Service) ListNormalTransactions(
	ctx context.Context,
	req *apiclient.ListNormalTransactionsRequest,
) (*apiclient.ListNormalTransactionsResponse, error) {
	query, err := normalizeNormalTransactionQuery(req)
	if err != nil {
		return nil, err
	}
	cache, err := s.store.GetNormalTransactionCache(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "read normal transaction cache: %v", err)
	}
	if req.GetForceRefresh() {
		refreshed, err := s.refreshNormalTransactions(ctx, query)
		if err != nil {
			return nil, err
		}
		return normalTransactionResponse(refreshed, false, false), nil
	}
	if cache == nil {
		refreshed, err := s.refreshNormalTransactions(ctx, query)
		if err != nil {
			return nil, err
		}
		return normalTransactionResponse(refreshed, false, false), nil
	}

	now := s.nowFn().UTC()
	stale := !cache.ExpiresAt.After(now)
	if stale {
		s.scheduleNormalTransactionRefresh(query)
	}
	return normalTransactionResponse(cache, true, stale), nil
}

func normalizeNormalTransactionQuery(req *apiclient.ListNormalTransactionsRequest) (ethereumapistore.NormalTransactionQuery, error) {
	if req == nil {
		return ethereumapistore.NormalTransactionQuery{}, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.GetChainId() <= 0 {
		return ethereumapistore.NormalTransactionQuery{}, status.Error(codes.InvalidArgument, "chain_id must be positive")
	}
	addressText := strings.TrimSpace(req.GetAddress())
	if !ethcommon.IsHexAddress(addressText) {
		return ethereumapistore.NormalTransactionQuery{}, status.Error(codes.InvalidArgument, "address must be a valid EVM address")
	}

	startBlock := uint64(0)
	endBlock := defaultNormalTransactionEndBlock
	if blockRange := req.GetBlockRange(); blockRange != nil {
		startBlock = blockRange.GetStartBlock()
		endBlock = blockRange.GetEndBlock()
		if startBlock > endBlock {
			return ethereumapistore.NormalTransactionQuery{}, status.Error(codes.InvalidArgument, "start_block must not exceed end_block")
		}
	}
	if startBlock > math.MaxInt64 || endBlock > math.MaxInt64 {
		return ethereumapistore.NormalTransactionQuery{}, status.Error(codes.InvalidArgument, "block range exceeds the supported range")
	}

	page := req.GetPage()
	if page == 0 {
		page = defaultNormalTransactionPage
	}
	if page < 1 {
		return ethereumapistore.NormalTransactionQuery{}, status.Error(codes.InvalidArgument, "page must be positive")
	}
	pageSize := req.GetPageSize()
	if pageSize == 0 {
		pageSize = defaultNormalTransactionPageSize
	}
	if pageSize < 1 || pageSize > maxNormalTransactionPageSize {
		return ethereumapistore.NormalTransactionQuery{}, status.Errorf(
			codes.InvalidArgument,
			"page_size must be between 1 and %d",
			maxNormalTransactionPageSize,
		)
	}

	sortOrder := utilethereumapi.NormalTransactionSortASC
	switch req.GetSort() {
	case apiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_UNSPECIFIED,
		apiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_ASC:
	case apiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_DESC:
		sortOrder = utilethereumapi.NormalTransactionSortDESC
	default:
		return ethereumapistore.NormalTransactionQuery{}, status.Error(codes.InvalidArgument, "sort is invalid")
	}

	return ethereumapistore.NormalTransactionQuery{
		ChainID:    req.GetChainId(),
		Address:    ethcommon.HexToAddress(addressText),
		StartBlock: startBlock,
		EndBlock:   endBlock,
		Page:       page,
		PageSize:   pageSize,
		SortOrder:  string(sortOrder),
	}, nil
}

func (s *Service) refreshNormalTransactions(
	ctx context.Context,
	query ethereumapistore.NormalTransactionQuery,
) (*ethereumapistore.NormalTransactionCacheEntry, error) {
	value, err, _ := s.refreshGroup.Do(query.CacheKey(), func() (any, error) {
		return s.fetchAndStoreNormalTransactions(ctx, query)
	})
	if err != nil {
		return nil, err
	}
	cache, ok := value.(*ethereumapistore.NormalTransactionCacheEntry)
	if !ok || cache == nil {
		return nil, status.Error(codes.Internal, "normal transaction refresh returned an invalid result")
	}
	return cache, nil
}

func (s *Service) scheduleNormalTransactionRefresh(query ethereumapistore.NormalTransactionQuery) {
	key := query.CacheKey()
	s.backgroundMu.Lock()
	if _, exists := s.backgroundRefresh[key]; exists {
		s.backgroundMu.Unlock()
		return
	}
	s.backgroundRefresh[key] = struct{}{}
	s.backgroundMu.Unlock()

	s.startStopMu.Lock()
	if !s.started || s.runCtx == nil {
		s.startStopMu.Unlock()
		s.backgroundMu.Lock()
		delete(s.backgroundRefresh, key)
		s.backgroundMu.Unlock()
		return
	}
	runCtx := s.runCtx
	s.runWG.Add(1)
	s.startStopMu.Unlock()

	go func() {
		defer s.runWG.Done()
		defer func() {
			s.backgroundMu.Lock()
			delete(s.backgroundRefresh, key)
			s.backgroundMu.Unlock()
		}()

		refreshCtx, cancel := context.WithTimeout(runCtx, s.refreshTimeout)
		defer cancel()
		if _, err := s.refreshNormalTransactions(refreshCtx, query); err != nil && !stderrors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"chain_id": query.ChainID,
				"address":  query.Address.Hex(),
				"page":     query.Page,
			}).Warnf("failed to refresh stale normal transaction cache: %v", err)
		}
	}()
}

func (s *Service) fetchAndStoreNormalTransactions(
	ctx context.Context,
	query ethereumapistore.NormalTransactionQuery,
) (*ethereumapistore.NormalTransactionCacheEntry, error) {
	response, err := s.ethereumAPI.ListNormalTransactions(ctx, utilethereumapi.ListNormalTransactionsOptions{
		ChainID:    query.ChainID,
		Address:    query.Address.Hex(),
		StartBlock: query.StartBlock,
		EndBlock:   query.EndBlock,
		Page:       query.Page,
		PageSize:   query.PageSize,
		Sort:       utilethereumapi.NormalTransactionSort(query.SortOrder),
	})
	if err != nil {
		return nil, grpcErrorFromEthereumAPI(err)
	}

	fetchedAt := s.nowFn().UTC()
	transactions := make([]*ethereumapistore.NormalTransaction, 0, len(response.Result))
	for index, item := range response.Result {
		transaction, err := normalTransactionFromEthereumAPI(query.ChainID, item, fetchedAt)
		if err != nil {
			return nil, status.Errorf(codes.DataLoss, "decode normal transaction %d: %v", index, err)
		}
		transactions = append(transactions, transaction)
	}
	cache, err := s.store.ReplaceNormalTransactionCache(
		ctx,
		query,
		transactions,
		fetchedAt,
		fetchedAt.Add(s.cacheTTL),
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "persist normal transaction cache: %v", err)
	}
	return cache, nil
}

func normalTransactionFromEthereumAPI(
	chainID int64,
	item utilethereumapi.NormalTransactionResult,
	fetchedAt time.Time,
) (*ethereumapistore.NormalTransaction, error) {
	blockNumber, err := parseUint64Field("blockNumber", item.BlockNumber)
	if err != nil {
		return nil, err
	}
	blockHash, err := parseHashField("blockHash", item.BlockHash)
	if err != nil {
		return nil, err
	}
	blockTimestamp, err := parseUint64Field("timeStamp", item.TimeStamp)
	if err != nil {
		return nil, err
	}
	txHash, err := parseHashField("hash", item.Hash)
	if err != nil {
		return nil, err
	}
	nonce, err := parseUint64Field("nonce", item.Nonce)
	if err != nil {
		return nil, err
	}
	transactionIndex, err := parseUint64Field("transactionIndex", item.TransactionIndex)
	if err != nil {
		return nil, err
	}
	fromAddress, err := parseAddressField("from", item.From)
	if err != nil {
		return nil, err
	}
	toAddress, err := parseOptionalAddressField("to", item.To)
	if err != nil {
		return nil, err
	}
	value, err := parseDecimalField("value", item.Value)
	if err != nil {
		return nil, err
	}
	gas, err := parseUint64Field("gas", item.Gas)
	if err != nil {
		return nil, err
	}
	gasPrice, err := parseDecimalField("gasPrice", item.GasPrice)
	if err != nil {
		return nil, err
	}
	methodID, err := parseOptionalFixedBytesField("methodId", item.MethodID, 4)
	if err != nil {
		return nil, err
	}
	contractAddress, err := parseOptionalAddressField("contractAddress", item.ContractAddress)
	if err != nil {
		return nil, err
	}
	cumulativeGasUsed, err := parseUint64Field("cumulativeGasUsed", item.CumulativeGasUsed)
	if err != nil {
		return nil, err
	}
	receiptStatus, err := parseReceiptStatus(item.TxReceiptStatus)
	if err != nil {
		return nil, err
	}
	gasUsed, err := parseUint64Field("gasUsed", item.GasUsed)
	if err != nil {
		return nil, err
	}
	confirmations, err := parseUint64Field("confirmations", item.Confirmations)
	if err != nil {
		return nil, err
	}
	isError, err := parseZeroOneBool("isError", item.IsError)
	if err != nil {
		return nil, err
	}
	return &ethereumapistore.NormalTransaction{
		ChainID:           chainID,
		TxHash:            txHash,
		BlockNumber:       blockNumber,
		BlockHash:         blockHash,
		BlockTimestamp:    blockTimestamp,
		Nonce:             nonce,
		TransactionIndex:  transactionIndex,
		FromAddress:       fromAddress,
		ToAddress:         toAddress,
		Value:             value,
		Gas:               gas,
		GasPrice:          gasPrice,
		Input:             item.Input,
		MethodID:          methodID,
		FunctionName:      item.FunctionName,
		ContractAddress:   contractAddress,
		CumulativeGasUsed: cumulativeGasUsed,
		ReceiptStatus:     receiptStatus,
		GasUsed:           gasUsed,
		Confirmations:     confirmations,
		IsError:           isError,
		FetchedAt:         fetchedAt,
	}, nil
}

func normalTransactionResponse(
	cache *ethereumapistore.NormalTransactionCacheEntry,
	cacheHit bool,
	stale bool,
) *apiclient.ListNormalTransactionsResponse {
	items := make([]*apiclient.NormalTransaction, 0, len(cache.Transactions))
	for _, transaction := range cache.Transactions {
		items = append(items, normalTransactionAPIItem(transaction))
	}
	return &apiclient.ListNormalTransactionsResponse{
		Transactions: items,
		Page:         cache.Query.Page,
		PageSize:     cache.Query.PageSize,
		Cache: &apiclient.NormalTransactionCacheMetadata{
			CacheHit:  cacheHit,
			Stale:     stale,
			FetchedAt: formatTime(cache.FetchedAt),
			ExpiresAt: formatTime(cache.ExpiresAt),
		},
	}
}

func normalTransactionAPIItem(item *ethereumapistore.NormalTransaction) *apiclient.NormalTransaction {
	if item == nil {
		return nil
	}
	toAddress := ""
	if item.ToAddress != nil {
		toAddress = item.ToAddress.Hex()
	}
	contractAddress := ""
	if item.ContractAddress != nil {
		contractAddress = item.ContractAddress.Hex()
	}
	methodID := ""
	if len(item.MethodID) > 0 {
		methodID = hexutil.Encode(item.MethodID)
	}
	return &apiclient.NormalTransaction{
		BlockNumber:       item.BlockNumber,
		BlockHash:         item.BlockHash.Hex(),
		BlockTimestamp:    item.BlockTimestamp,
		TransactionHash:   item.TxHash.Hex(),
		Nonce:             item.Nonce,
		TransactionIndex:  item.TransactionIndex,
		FromAddress:       item.FromAddress.Hex(),
		ToAddress:         toAddress,
		Value:             item.Value,
		Gas:               item.Gas,
		GasPrice:          item.GasPrice,
		Input:             item.Input,
		MethodId:          methodID,
		FunctionName:      item.FunctionName,
		ContractAddress:   contractAddress,
		CumulativeGasUsed: item.CumulativeGasUsed,
		ReceiptStatus:     receiptStatusAPIValue(item.ReceiptStatus),
		GasUsed:           item.GasUsed,
		Confirmations:     item.Confirmations,
		IsError:           item.IsError,
	}
}

func receiptStatusAPIValue(value int16) apiclient.NormalTransactionReceiptStatus {
	switch value {
	case ethereumapistore.ReceiptStatusFailed:
		return apiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_FAILED
	case ethereumapistore.ReceiptStatusSuccess:
		return apiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_SUCCESS
	default:
		return apiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_UNSPECIFIED
	}
}

func parseUint64Field(name string, value string) (uint64, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a uint64 decimal integer", name)
	}
	return parsed, nil
}

func parseDecimalField(name string, value string) (string, error) {
	integer, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
	if !ok || integer.Sign() < 0 {
		return "", fmt.Errorf("%s must be a nonnegative decimal integer", name)
	}
	return integer.String(), nil
}

func parseHashField(name string, value string) (ethcommon.Hash, error) {
	decoded, err := hexutil.Decode(strings.TrimSpace(value))
	if err != nil || len(decoded) != ethcommon.HashLength {
		return ethcommon.Hash{}, fmt.Errorf("%s must be a 32-byte hex hash", name)
	}
	return ethcommon.BytesToHash(decoded), nil
}

func parseAddressField(name string, value string) (ethcommon.Address, error) {
	if !ethcommon.IsHexAddress(strings.TrimSpace(value)) {
		return ethcommon.Address{}, fmt.Errorf("%s must be a valid EVM address", name)
	}
	return ethcommon.HexToAddress(value), nil
}

func parseOptionalAddressField(name string, value string) (*ethcommon.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0x" {
		return nil, nil
	}
	address, err := parseAddressField(name, value)
	if err != nil {
		return nil, err
	}
	return &address, nil
}

func parseOptionalFixedBytesField(name string, value string, length int) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0x" {
		return nil, nil
	}
	decoded, err := hexutil.Decode(value)
	if err != nil || len(decoded) != length {
		return nil, fmt.Errorf("%s must be %d bytes of hex data", name, length)
	}
	return decoded, nil
}

func parseReceiptStatus(value string) (int16, error) {
	switch strings.TrimSpace(value) {
	case "":
		return ethereumapistore.ReceiptStatusUnknown, nil
	case "0":
		return ethereumapistore.ReceiptStatusFailed, nil
	case "1":
		return ethereumapistore.ReceiptStatusSuccess, nil
	default:
		return 0, fmt.Errorf("txreceipt_status must be empty, 0, or 1")
	}
}

func parseZeroOneBool(name string, value string) (bool, error) {
	switch strings.TrimSpace(value) {
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, fmt.Errorf("%s must be 0 or 1", name)
	}
}

func grpcErrorFromEthereumAPI(err error) error {
	if err == nil {
		return nil
	}
	if contextError := contextErrorStatus(err); contextError != nil {
		return contextError
	}
	var apiErr *utilethereumapi.APIError
	if stderrors.As(err, &apiErr) {
		switch apiErr.Type {
		case utilethereumapi.APIErrorTypeRateLimit:
			return status.Error(codes.ResourceExhausted, apiErr.Error())
		case utilethereumapi.APIErrorTypeAuthentication, utilethereumapi.APIErrorTypePlan:
			return status.Error(codes.FailedPrecondition, apiErr.Error())
		case utilethereumapi.APIErrorTypeInvalidRequest:
			return status.Error(codes.InvalidArgument, apiErr.Error())
		case utilethereumapi.APIErrorTypeMalformed:
			return status.Error(codes.DataLoss, apiErr.Error())
		}
		return status.Error(codes.Unavailable, apiErr.Error())
	}
	if strings.Contains(strings.ToLower(err.Error()), "decode etherscan") {
		return status.Errorf(codes.DataLoss, "invalid etherscan response: %v", err)
	}
	return status.Errorf(codes.Unavailable, "etherscan request failed: %v", err)
}

func contextErrorStatus(err error) error {
	switch {
	case stderrors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, context.Canceled.Error())
	case stderrors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error())
	default:
		return nil
	}
}

func (s *Service) runCacheCleanup(ctx context.Context) {
	defer s.runWG.Done()
	s.cleanupNormalTransactionCaches(ctx)
	ticker := time.NewTicker(normalTransactionCacheCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanupNormalTransactionCaches(ctx)
		}
	}
}

func (s *Service) cleanupNormalTransactionCaches(ctx context.Context) {
	deleteBefore := s.nowFn().UTC().Add(-s.cacheRetention)
	deleted, err := s.store.DeleteExpiredNormalTransactionCaches(ctx, deleteBefore)
	if err != nil {
		if !stderrors.Is(err, context.Canceled) {
			log.Warnf("failed to clean up normal transaction query caches: %v", err)
		}
		return
	}
	if deleted > 0 {
		log.Infof("deleted %d expired normal transaction query caches", deleted)
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
