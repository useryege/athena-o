package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	defaultRPCURL           = "ws://65.108.192.118:8546"
	defaultClientTimeout    = 3 * time.Minute
	bscMainnetChainID       = uint64(56)
	rpcURLEnvName           = "ATHENA_E2E_BSC_TRACE_RPC_URL"
	minimumTransferValueWei = "10000000000000000"
)

var minimumTransferValue = mustParseDecimalInteger(minimumTransferValueWei)

type rpcBlock struct {
	Number       string   `json:"number"`
	Hash         string   `json:"hash"`
	Timestamp    string   `json:"timestamp"`
	Transactions []string `json:"transactions"`
}

type blockTraceResult struct {
	TransactionHash string     `json:"txHash"`
	Result          *callFrame `json:"result"`
	Error           string     `json:"error"`
}

type callFrame struct {
	Type  string       `json:"type"`
	From  string       `json:"from"`
	To    string       `json:"to"`
	Input string       `json:"input"`
	Value string       `json:"value"`
	Error string       `json:"error"`
	Calls []*callFrame `json:"calls"`
}

type traceTransfer struct {
	TransactionHash  string `json:"transaction_hash"`
	TransactionIndex uint64 `json:"transaction_index"`
	TraceAddress     []int  `json:"trace_address,omitempty"`
	FromAddress      string `json:"from_address"`
	ToAddress        string `json:"to_address"`
	ValueWei         string `json:"value_wei"`
}

type traceOutput struct {
	RPCURL                string          `json:"rpc_url"`
	ChainID               uint64          `json:"chain_id"`
	BlockNumber           uint64          `json:"block_number"`
	BlockNumberHex        string          `json:"block_number_hex"`
	BlockHash             string          `json:"block_hash"`
	BlockTimestamp        uint64          `json:"block_timestamp"`
	BlockTimestampUTC     time.Time       `json:"block_timestamp_utc"`
	TransactionCount      int             `json:"transaction_count"`
	TraceResultCount      int             `json:"trace_result_count"`
	TraceStartedAt        time.Time       `json:"trace_started_at"`
	TraceElapsed          string          `json:"trace_elapsed"`
	TraceElapsedMillis    float64         `json:"trace_elapsed_ms"`
	TraceResponseBytes    int             `json:"trace_response_bytes"`
	MinimumValueWei       string          `json:"minimum_value_wei"`
	TotalCallFrames       int             `json:"total_call_frames"`
	NormalTransferCount   int             `json:"normal_transfer_count"`
	InternalTransferCount int             `json:"internal_transfer_count"`
	NormalTransfers       []traceTransfer `json:"normal_transfers"`
	InternalTransfers     []traceTransfer `json:"internal_transfers"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("bsc-block-trace", flag.ContinueOnError)
	flags.SetOutput(stderr)
	blockValue := flags.String("block", "", "BSC block number in decimal or 0x-prefixed hexadecimal")
	rpcURL := flags.String("rpc-url", defaultRPCURLFromEnv(), "BSC Mainnet WebSocket RPC URL")
	timeout := flags.Duration("timeout", defaultClientTimeout, "total client timeout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "bsc-block-trace: unexpected positional arguments: %s\n", strings.Join(flags.Args(), " "))
		return 2
	}
	if *timeout <= 0 {
		fmt.Fprintln(stderr, "bsc-block-trace: timeout must be positive")
		return 2
	}

	requestedBlock, err := readBlockNumber(*blockValue, stdin, stderr)
	if err != nil {
		fmt.Fprintf(stderr, "bsc-block-trace: %v\n", err)
		return 2
	}
	if strings.TrimSpace(*rpcURL) == "" {
		fmt.Fprintln(stderr, "bsc-block-trace: RPC URL is required")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	client, err := rpc.DialContext(ctx, strings.TrimSpace(*rpcURL))
	if err != nil {
		fmt.Fprintf(stderr, "bsc-block-trace: connect to BSC RPC: %v\n", err)
		return 1
	}
	defer client.Close()

	chainID, err := readChainID(ctx, client)
	if err != nil {
		fmt.Fprintf(stderr, "bsc-block-trace: %v\n", err)
		return 1
	}
	if chainID != bscMainnetChainID {
		fmt.Fprintf(stderr, "bsc-block-trace: RPC must report BSC Mainnet chain ID %d, got %d\n", bscMainnetChainID, chainID)
		return 1
	}

	block, err := readBlock(ctx, client, requestedBlock)
	if err != nil {
		fmt.Fprintf(stderr, "bsc-block-trace: %v\n", err)
		return 1
	}
	blockTimestamp, err := hexutil.DecodeUint64(block.Timestamp)
	if err != nil {
		fmt.Fprintf(stderr, "bsc-block-trace: decode block %d timestamp %q: %v\n", requestedBlock, block.Timestamp, err)
		return 1
	}

	traceStartedAt := time.Now().UTC()
	started := time.Now()
	var trace json.RawMessage
	err = client.CallContext(ctx, &trace, "debug_traceBlockByNumber", hexutil.EncodeUint64(requestedBlock), map[string]any{
		"tracer":  "callTracer",
		"timeout": "120s",
	})
	traceElapsed := time.Since(started)
	if err != nil {
		fmt.Fprintf(stderr, "bsc-block-trace: trace block %d: %v\n", requestedBlock, err)
		return 1
	}
	trace = bytes.TrimSpace(trace)
	if len(trace) == 0 || bytes.Equal(trace, []byte("null")) || !json.Valid(trace) {
		fmt.Fprintf(stderr, "bsc-block-trace: trace block %d returned invalid JSON\n", requestedBlock)
		return 1
	}
	var traceResults []blockTraceResult
	if err := json.Unmarshal(trace, &traceResults); err != nil {
		fmt.Fprintf(stderr, "bsc-block-trace: trace block %d did not return an array: %v\n", requestedBlock, err)
		return 1
	}
	normalTransfers, internalTransfers, totalCallFrames, err := filterTransfers(block, traceResults)
	if err != nil {
		fmt.Fprintf(stderr, "bsc-block-trace: validate trace block %d: %v\n", requestedBlock, err)
		return 1
	}

	result := traceOutput{
		RPCURL:                strings.TrimSpace(*rpcURL),
		ChainID:               chainID,
		BlockNumber:           requestedBlock,
		BlockNumberHex:        hexutil.EncodeUint64(requestedBlock),
		BlockHash:             block.Hash,
		BlockTimestamp:        blockTimestamp,
		BlockTimestampUTC:     time.Unix(int64(blockTimestamp), 0).UTC(),
		TransactionCount:      len(block.Transactions),
		TraceResultCount:      len(traceResults),
		TraceStartedAt:        traceStartedAt,
		TraceElapsed:          traceElapsed.String(),
		TraceElapsedMillis:    float64(traceElapsed) / float64(time.Millisecond),
		TraceResponseBytes:    len(trace),
		MinimumValueWei:       minimumTransferValueWei,
		TotalCallFrames:       totalCallFrames,
		NormalTransferCount:   len(normalTransfers),
		InternalTransferCount: len(internalTransfers),
		NormalTransfers:       normalTransfers,
		InternalTransfers:     internalTransfers,
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(stderr, "bsc-block-trace: write trace output: %v\n", err)
		return 1
	}
	return 0
}

func filterTransfers(block *rpcBlock, results []blockTraceResult) ([]traceTransfer, []traceTransfer, int, error) {
	if len(results) != len(block.Transactions) {
		return nil, nil, 0, fmt.Errorf(
			"trace returned %d results for %d block transactions",
			len(results), len(block.Transactions),
		)
	}

	normalTransfers := make([]traceTransfer, 0)
	internalTransfers := make([]traceTransfer, 0)
	totalCallFrames := 0
	for transactionIndex, result := range results {
		blockHash, err := normalizeFixedHex("block transaction hash", block.Transactions[transactionIndex], 32)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("transaction %d: %w", transactionIndex, err)
		}
		traceHash, err := normalizeFixedHex("trace transaction hash", result.TransactionHash, 32)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("transaction %d: %w", transactionIndex, err)
		}
		if traceHash != blockHash {
			return nil, nil, 0, fmt.Errorf(
				"transaction %d trace hash %s does not match block transaction hash %s",
				transactionIndex, traceHash, blockHash,
			)
		}
		if strings.TrimSpace(result.Error) != "" {
			return nil, nil, 0, fmt.Errorf("transaction %d trace failed: %s", transactionIndex, result.Error)
		}
		if result.Result == nil {
			return nil, nil, 0, fmt.Errorf("transaction %d trace result is empty", transactionIndex)
		}

		rootTransfer, rootSucceeded, frameCount, transactionInternalTransfers, err := inspectCallFrame(
			result.Result,
			traceHash,
			uint64(transactionIndex),
			nil,
			true,
		)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("transaction %d: %w", transactionIndex, err)
		}
		totalCallFrames += frameCount
		if rootSucceeded && rootTransfer != nil {
			normalTransfers = append(normalTransfers, *rootTransfer)
		}
		internalTransfers = append(internalTransfers, transactionInternalTransfers...)
	}
	return normalTransfers, internalTransfers, totalCallFrames, nil
}

func inspectCallFrame(
	frame *callFrame,
	transactionHash string,
	transactionIndex uint64,
	traceAddress []int,
	ancestorsSucceeded bool,
) (*traceTransfer, bool, int, []traceTransfer, error) {
	if frame == nil {
		return nil, false, 0, nil, fmt.Errorf("call frame is empty")
	}
	frameType := strings.ToUpper(strings.TrimSpace(frame.Type))
	if frameType == "" {
		return nil, false, 0, nil, fmt.Errorf("call frame %v has no type", traceAddress)
	}
	fromAddress, err := normalizeFixedHex("from address", frame.From, 20)
	if err != nil {
		return nil, false, 0, nil, fmt.Errorf("call frame %v: %w", traceAddress, err)
	}
	toAddress := ""
	if strings.TrimSpace(frame.To) != "" {
		toAddress, err = normalizeFixedHex("to address", frame.To, 20)
		if err != nil {
			return nil, false, 0, nil, fmt.Errorf("call frame %v: %w", traceAddress, err)
		}
	}
	if frameType == "CALL" && toAddress == "" {
		return nil, false, 0, nil, fmt.Errorf("call frame %v CALL has no to address", traceAddress)
	}
	value, err := decodeTraceValue(frame.Value)
	if err != nil {
		return nil, false, 0, nil, fmt.Errorf("call frame %v value %q: %w", traceAddress, frame.Value, err)
	}

	frameSucceeded := ancestorsSucceeded && strings.TrimSpace(frame.Error) == ""
	isRoot := len(traceAddress) == 0
	var rootInput []byte
	if isRoot && frameType == "CALL" {
		rootInput, err = hexutil.Decode(frame.Input)
		if err != nil {
			return nil, false, 0, nil, fmt.Errorf("root input %q: %w", frame.Input, err)
		}
	}
	var normalTransfer *traceTransfer
	internalTransfers := make([]traceTransfer, 0)
	if isRoot && frameSucceeded && frameType == "CALL" && value.Cmp(minimumTransferValue) > 0 {
		if len(rootInput) == 0 {
			normalTransfer = &traceTransfer{
				TransactionHash: transactionHash, TransactionIndex: transactionIndex,
				FromAddress: fromAddress, ToAddress: toAddress, ValueWei: value.String(),
			}
		}
	}
	if !isRoot && frameSucceeded && frameType == "CALL" && value.Cmp(minimumTransferValue) > 0 {
		internalTransfers = append(internalTransfers, traceTransfer{
			TransactionHash: transactionHash, TransactionIndex: transactionIndex,
			TraceAddress: append([]int(nil), traceAddress...),
			FromAddress:  fromAddress, ToAddress: toAddress, ValueWei: value.String(),
		})
	}

	frameCount := 1
	for childIndex, child := range frame.Calls {
		childTraceAddress := append(append([]int(nil), traceAddress...), childIndex)
		_, _, childFrameCount, childTransfers, childErr := inspectCallFrame(
			child,
			transactionHash,
			transactionIndex,
			childTraceAddress,
			frameSucceeded,
		)
		if childErr != nil {
			return nil, false, 0, nil, childErr
		}
		frameCount += childFrameCount
		internalTransfers = append(internalTransfers, childTransfers...)
	}
	return normalTransfer, frameSucceeded, frameCount, internalTransfers, nil
}

func decodeTraceValue(encoded string) (*big.Int, error) {
	if strings.TrimSpace(encoded) == "" {
		return new(big.Int), nil
	}
	value, err := hexutil.DecodeBig(encoded)
	if err != nil {
		return nil, err
	}
	if value.Sign() < 0 {
		return nil, fmt.Errorf("value must not be negative")
	}
	return value, nil
}

func normalizeFixedHex(field, encoded string, expectedLength int) (string, error) {
	decoded, err := hexutil.Decode(strings.TrimSpace(encoded))
	if err != nil {
		return "", fmt.Errorf("decode %s %q: %w", field, encoded, err)
	}
	if len(decoded) != expectedLength {
		return "", fmt.Errorf("%s must contain %d bytes, got %d", field, expectedLength, len(decoded))
	}
	return hexutil.Encode(decoded), nil
}

func mustParseDecimalInteger(encoded string) *big.Int {
	value, ok := new(big.Int).SetString(encoded, 10)
	if !ok {
		panic(fmt.Sprintf("invalid decimal integer %q", encoded))
	}
	return value
}

func defaultRPCURLFromEnv() string {
	if value := strings.TrimSpace(os.Getenv(rpcURLEnvName)); value != "" {
		return value
	}
	return defaultRPCURL
}

func readBlockNumber(value string, stdin io.Reader, stderr io.Writer) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		fmt.Fprint(stderr, "请输入 BSC 区块号（十进制或 0x 十六进制）: ")
		line, err := bufio.NewReader(stdin).ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return 0, fmt.Errorf("read block number: %w", err)
		}
		value = strings.TrimSpace(line)
	}
	if value == "" {
		return 0, fmt.Errorf("block number is required")
	}
	base := 10
	digits := value
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		base = 16
		digits = value[2:]
	}
	if digits == "" {
		return 0, fmt.Errorf("invalid block number %q", value)
	}
	blockNumber, err := strconv.ParseUint(digits, base, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid block number %q: %w", value, err)
	}
	return blockNumber, nil
}

func readChainID(ctx context.Context, client *rpc.Client) (uint64, error) {
	var encoded string
	if err := client.CallContext(ctx, &encoded, "eth_chainId"); err != nil {
		return 0, fmt.Errorf("query chain ID: %w", err)
	}
	chainID, err := hexutil.DecodeUint64(encoded)
	if err != nil {
		return 0, fmt.Errorf("decode chain ID %q: %w", encoded, err)
	}
	return chainID, nil
}

func readBlock(ctx context.Context, client *rpc.Client, blockNumber uint64) (*rpcBlock, error) {
	var block *rpcBlock
	if err := client.CallContext(ctx, &block, "eth_getBlockByNumber", hexutil.EncodeUint64(blockNumber), false); err != nil {
		return nil, fmt.Errorf("query block %d: %w", blockNumber, err)
	}
	if block == nil {
		return nil, fmt.Errorf("block %d does not exist", blockNumber)
	}
	if block.Number != hexutil.EncodeUint64(blockNumber) {
		return nil, fmt.Errorf("RPC returned block number %q for requested block %d", block.Number, blockNumber)
	}
	if strings.TrimSpace(block.Hash) == "" || strings.TrimSpace(block.Timestamp) == "" {
		return nil, fmt.Errorf("block %d metadata is incomplete", blockNumber)
	}
	return block, nil
}
