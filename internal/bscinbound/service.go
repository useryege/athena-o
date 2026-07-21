package bscinbound

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/bscinbound/apiclient"
	"github.com/useryege/athena/internal/bscinbound/store"
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
	apiclient.UnimplementedBscInboundTransactionServiceServer
	store   *store.Store
	metrics *Metrics
}

func NewService(repository *store.Store, metrics *Metrics) *Service {
	return &Service{store: repository, metrics: metrics}
}

func (s *Service) ListInboundNormalTransactions(ctx context.Context, request *apiclient.ListInboundNormalTransactionsRequest) (*apiclient.ListInboundNormalTransactionsResponse, error) {
	startedAt := time.Now()
	if s.metrics != nil {
		defer s.metrics.ObserveQuery(startedAt)
	}
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "BSC inbound transaction store is required")
	}
	address, pageSize, cursor, err := normalizeListRequest(request)
	if err != nil {
		return nil, err
	}
	items, checkpoint, err := s.store.ListInboundNormalTransactions(ctx, address, cursor, pageSize+1)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.FailedPrecondition, "BSC inbound transaction index is not initialized")
		}
		return nil, status.Errorf(codes.Internal, "list inbound normal transactions: %v", err)
	}

	hasNext := len(items) > int(pageSize)
	if hasNext {
		items = items[:pageSize]
	}
	transactions := make([]*apiclient.InboundNormalTransaction, 0, len(items))
	for _, item := range items {
		transactions = append(transactions, transactionResponse(item))
	}
	nextPageToken := ""
	if hasNext && len(items) > 0 {
		last := items[len(items)-1]
		nextPageToken = encodePageToken(address, store.PageCursor{
			BlockNumber: last.BlockNumber, TransactionIndex: last.TransactionIndex,
		})
	}
	return &apiclient.ListInboundNormalTransactionsResponse{
		Transactions: transactions, NextPageToken: nextPageToken,
		IndexedThroughBlock:     checkpoint.CursorBlockNumber,
		IndexedThroughTimestamp: checkpoint.CursorBlockTimestamp,
	}, nil
}

func normalizeListRequest(request *apiclient.ListInboundNormalTransactionsRequest) (common.Address, int32, *store.PageCursor, error) {
	if request == nil {
		return common.Address{}, 0, nil, status.Error(codes.InvalidArgument, "request is required")
	}
	if !common.IsHexAddress(request.GetAddress()) {
		return common.Address{}, 0, nil, status.Error(codes.InvalidArgument, "address must be a valid EVM address")
	}
	address := common.HexToAddress(request.GetAddress())
	pageSize := request.GetPageSize()
	if pageSize == 0 {
		pageSize = defaultPageSize
	}
	if pageSize < 1 || pageSize > maxPageSize {
		return common.Address{}, 0, nil, status.Errorf(codes.InvalidArgument, "page_size must be between 1 and %d", maxPageSize)
	}
	pageToken := request.GetPageToken()
	beforePosition := request.GetBeforePosition()
	if pageToken != "" && beforePosition != nil {
		return common.Address{}, 0, nil, status.Error(codes.InvalidArgument, "page_token and before_position cannot be used together")
	}
	if beforePosition != nil {
		if beforePosition.GetBlockNumber() > math.MaxInt64 || beforePosition.GetTransactionIndex() > math.MaxInt64 {
			return common.Address{}, 0, nil, status.Error(codes.InvalidArgument, "before_position is invalid")
		}
		return address, pageSize, &store.PageCursor{
			BlockNumber: beforePosition.GetBlockNumber(), TransactionIndex: beforePosition.GetTransactionIndex(),
		}, nil
	}
	if pageToken == "" {
		return address, pageSize, nil, nil
	}
	cursor, tokenAddress, err := decodePageToken(pageToken)
	if err != nil {
		return common.Address{}, 0, nil, status.Error(codes.InvalidArgument, "page_token is invalid")
	}
	if tokenAddress != address {
		return common.Address{}, 0, nil, status.Error(codes.InvalidArgument, "page_token does not belong to address")
	}
	if cursor.BlockNumber > math.MaxInt64 || cursor.TransactionIndex > math.MaxInt64 {
		return common.Address{}, 0, nil, status.Error(codes.InvalidArgument, "page_token position is invalid")
	}
	return address, pageSize, &cursor, nil
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

func transactionResponse(transaction store.InboundNormalTransaction) *apiclient.InboundNormalTransaction {
	valueWei := "0"
	if transaction.ValueWei != nil {
		valueWei = transaction.ValueWei.String()
	}
	return &apiclient.InboundNormalTransaction{
		BlockNumber: transaction.BlockNumber, BlockHash: transaction.BlockHash.Hex(),
		BlockTimestamp: transaction.BlockTimestamp, TransactionHash: transaction.TransactionHash.Hex(),
		TransactionIndex: transaction.TransactionIndex, FromAddress: transaction.FromAddress.Hex(),
		ToAddress: transaction.ToAddress.Hex(), ValueWei: valueWei,
	}
}
