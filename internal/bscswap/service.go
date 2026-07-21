package bscswap

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/bscswap/apiclient"
	"github.com/useryege/athena/internal/bscswap/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultPageSize  = int32(100)
	maxPageSize      = int32(300)
	pageTokenSize    = 1 + common.AddressLength + 8 + 8
	pageTokenVersion = byte(1)
)

type Service struct {
	apiclient.UnimplementedBscSwapTransactionServiceServer
	store   *store.Store
	metrics *Metrics
}

func NewService(repository *store.Store, metrics *Metrics) *Service {
	return &Service{store: repository, metrics: metrics}
}

func (s *Service) ListSwapTransactions(ctx context.Context, request *apiclient.ListSwapTransactionsRequest) (*apiclient.ListSwapTransactionsResponse, error) {
	startedAt := time.Now()
	if s.metrics != nil {
		defer s.metrics.ObserveQuery(startedAt)
	}
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "BSC swap transaction store is required")
	}
	address, beforeBlock, pageSize, cursor, err := normalizeListRequest(request)
	if err != nil {
		return nil, err
	}
	items, checkpoint, err := s.store.ListSwapTransactions(ctx, address, beforeBlock, cursor, pageSize+1)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.FailedPrecondition, "BSC swap transaction index is not initialized")
		}
		return nil, status.Errorf(codes.Internal, "list BSC swap transactions: %v", err)
	}

	hasNext := len(items) > int(pageSize)
	if hasNext {
		items = items[:pageSize]
	}
	transactionHashes := make([]string, 0, len(items))
	for _, item := range items {
		transactionHashes = append(transactionHashes, item.TransactionHash.Hex())
	}
	nextPageToken := ""
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		nextPageToken = encodePageToken(address, store.PageCursor{
			BlockNumber: last.BlockNumber, TransactionIndex: last.TransactionIndex,
		})
	}
	return &apiclient.ListSwapTransactionsResponse{
		TransactionHashes: transactionHashes, NextPageToken: nextPageToken,
		IndexedThroughBlock:     checkpoint.CursorBlockNumber,
		IndexedThroughTimestamp: checkpoint.CursorBlockTimestamp,
	}, nil
}

func normalizeListRequest(request *apiclient.ListSwapTransactionsRequest) (common.Address, uint64, int32, *store.PageCursor, error) {
	if request == nil {
		return common.Address{}, 0, 0, nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if !common.IsHexAddress(request.GetWalletAddress()) {
		return common.Address{}, 0, 0, nil, status.Error(codes.InvalidArgument, "wallet_address must be a valid EVM address")
	}
	address := common.HexToAddress(request.GetWalletAddress())
	pageSize := request.GetPageSize()
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize < 1 || pageSize > maxPageSize {
		return common.Address{}, 0, 0, nil, status.Errorf(codes.InvalidArgument, "page_size must be between 1 and %d", maxPageSize)
	}
	pageToken := request.GetPageToken()
	beforeBlock := request.GetBeforeBlockNumber()
	if pageToken == "" {
		if beforeBlock == 0 || beforeBlock > math.MaxInt64 {
			return common.Address{}, 0, 0, nil, status.Error(codes.InvalidArgument, "before_block_number must be between 1 and int64 max")
		}
		return address, beforeBlock, pageSize, nil, nil
	}
	if beforeBlock != 0 {
		return common.Address{}, 0, 0, nil, status.Error(codes.InvalidArgument, "page_token and before_block_number cannot be used together")
	}
	cursor, tokenAddress, err := decodePageToken(pageToken)
	if err != nil {
		return common.Address{}, 0, 0, nil, status.Error(codes.InvalidArgument, "page_token is invalid")
	}
	if tokenAddress != address {
		return common.Address{}, 0, 0, nil, status.Error(codes.InvalidArgument, "page_token does not belong to wallet_address")
	}
	if cursor.BlockNumber > math.MaxInt64 || cursor.TransactionIndex > math.MaxInt64 {
		return common.Address{}, 0, 0, nil, status.Error(codes.InvalidArgument, "page_token position is invalid")
	}
	return address, 0, pageSize, &cursor, nil
}

func encodePageToken(address common.Address, cursor store.PageCursor) string {
	data := make([]byte, pageTokenSize)
	data[0] = pageTokenVersion
	copy(data[1:1+common.AddressLength], address.Bytes())
	binary.BigEndian.PutUint64(data[1+common.AddressLength:1+common.AddressLength+8], cursor.BlockNumber)
	binary.BigEndian.PutUint64(data[1+common.AddressLength+8:], cursor.TransactionIndex)
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodePageToken(token string) (store.PageCursor, common.Address, error) {
	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(data) != pageTokenSize || data[0] != pageTokenVersion {
		return store.PageCursor{}, common.Address{}, errors.New("invalid page token")
	}
	address := common.BytesToAddress(data[1 : 1+common.AddressLength])
	cursor := store.PageCursor{
		BlockNumber:      binary.BigEndian.Uint64(data[1+common.AddressLength : 1+common.AddressLength+8]),
		TransactionIndex: binary.BigEndian.Uint64(data[1+common.AddressLength+8:]),
	}
	return cursor, address, nil
}
