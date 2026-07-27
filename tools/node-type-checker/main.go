package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/useryege/athena/util/ethws"
)

const (
	checkTimeout = 30 * time.Second

	ethereumChainID int64 = 1
	bscChainID      int64 = 56

	statusHealthy     = "healthy"
	statusSyncing     = "syncing"
	statusLagging     = "lagging"
	statusStale       = "stale"
	statusUnreachable = "unreachable"
	statusUnsupported = "unsupported"
	statusUnknown     = "unknown"

	evidenceComplete = "complete"
	evidencePruned   = "pruned"
	evidenceUnknown  = "unknown"

	typeArchive       = "archive"
	typeFullHistory   = "full-history"
	typePrunedHistory = "pruned-history"
	typeUnknown       = "unknown"
)

var chainPolicies = map[int64]chainPolicy{
	ethereumChainID: {
		name:         "ETH",
		referenceURL: "https://ethereum-rpc.publicnode.com",
		maxBlockLag:  2,
		maxBlockAge:  60 * time.Second,
	},
	bscChainID: {
		name:         "BSC",
		referenceURL: "https://bsc-dataseed.bnbchain.org",
		maxBlockLag:  3,
		maxBlockAge:  30 * time.Second,
	},
}

type chainPolicy struct {
	name         string
	referenceURL string
	maxBlockLag  uint64
	maxBlockAge  time.Duration
}

type rpcBlock struct {
	Hash             common.Hash    `json:"hash"`
	Number           hexutil.Uint64 `json:"number"`
	Timestamp        hexutil.Uint64 `json:"timestamp"`
	TransactionsRoot common.Hash    `json:"transactionsRoot"`
	ReceiptsRoot     common.Hash    `json:"receiptsRoot"`
	Transactions     []common.Hash  `json:"transactions"`
}

type sampleResult struct {
	height   uint64
	blocks   string
	receipts string
	state    string
	details  []string
}

type historyResult struct {
	blocks   string
	receipts string
	state    string
	nodeType string
	details  []string
}

type referenceResult struct {
	height uint64
	err    error
}

type resultRow struct {
	endpoint string
	chain    string
	client   string
	latest   string
	lag      string
	status   string
	blocks   string
	receipts string
	state    string
	nodeType string
	detail   string
	failed   bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		writeUsage(stdout)
		return 0
	}
	if len(args) != 1 {
		fmt.Fprintln(stderr, "error: exactly one WebSocket endpoint is required")
		writeUsage(stderr)
		return 1
	}

	endpoint, err := validateEndpoint(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		writeUsage(stderr)
		return 1
	}

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, checkTimeout)
	defer cancel()

	row := inspectNode(ctx, endpoint)
	writeResultTable(stdout, row)
	if row.failed {
		return 1
	}
	return 0
}

func writeUsage(output io.Writer) {
	fmt.Fprintln(output, "usage: node-type-checker <ws-url>")
}

func validateEndpoint(raw string) (string, error) {
	endpoint := strings.TrimSpace(raw)
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse endpoint: %w", err)
	}
	if parsed.Scheme != "ws" && parsed.Scheme != "wss" {
		return "", fmt.Errorf("endpoint scheme must be ws or wss")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("endpoint host is required")
	}
	return endpoint, nil
}

func inspectNode(ctx context.Context, endpoint string) resultRow {
	row := resultRow{
		endpoint: ethws.RedactEndpoint(endpoint),
		chain:    "N/A",
		client:   "unknown",
		latest:   "N/A",
		lag:      "N/A",
		status:   statusUnknown,
		blocks:   evidenceUnknown,
		receipts: evidenceUnknown,
		state:    evidenceUnknown,
		nodeType: typeUnknown,
	}

	client, err := ethws.DialContext(ctx, endpoint, "")
	if err != nil {
		row.status = statusUnreachable
		row.detail = compactError(fmt.Errorf("connect target: %w", err))
		row.failed = true
		return row
	}
	defer client.Close()

	details := make([]string, 0, 6)
	var clientVersion string
	if err := client.Client().CallContext(ctx, &clientVersion, "web3_clientVersion"); err != nil {
		appendDetail(&details, "client-version: "+compactError(err))
	} else if strings.TrimSpace(clientVersion) != "" {
		row.client = compactCell(clientVersion)
	}

	chainID, err := client.ChainID(ctx)
	if err != nil || chainID == nil {
		if err == nil {
			err = fmt.Errorf("empty response")
		}
		row.status = statusUnreachable
		appendDetail(&details, "chain-id: "+compactError(err))
		row.detail = strings.Join(details, "; ")
		row.failed = true
		return row
	}
	policy, supported := chainPolicies[chainID.Int64()]
	if !supported {
		row.chain = chainID.String()
		row.status = statusUnsupported
		row.detail = fmt.Sprintf("unsupported chain id %s", chainID.String())
		row.failed = true
		return row
	}
	row.chain = fmt.Sprintf("%s(%d)", policy.name, chainID.Int64())

	syncProgress, syncErr := client.SyncProgress(ctx)
	if syncErr != nil {
		appendDetail(&details, "syncing: "+compactError(syncErr))
	}

	latest, err := client.BlockNumber(ctx)
	if err != nil {
		row.status = statusUnreachable
		appendDetail(&details, "latest-block: "+compactError(err))
		row.detail = strings.Join(details, "; ")
		row.failed = true
		return row
	}
	row.latest = strconv.FormatUint(latest, 10)

	latestHeader, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(latest))
	if err != nil || latestHeader == nil {
		if err == nil {
			err = fmt.Errorf("empty response")
		}
		row.status = statusUnreachable
		appendDetail(&details, "latest-header: "+compactError(err))
		row.detail = strings.Join(details, "; ")
		row.failed = true
		return row
	}

	var history historyResult
	var reference referenceResult
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		history = inspectHistory(ctx, client, latest)
	}()
	go func() {
		defer wait.Done()
		reference = fetchReferenceHeight(ctx, chainID.Int64(), policy.referenceURL)
	}()
	wait.Wait()

	row.blocks = history.blocks
	row.receipts = history.receipts
	row.state = history.state
	row.nodeType = history.nodeType
	for _, detail := range history.details {
		appendDetail(&details, detail)
	}

	if reference.err != nil {
		row.status = statusUnknown
		appendDetail(&details, "reference: "+compactError(reference.err))
		row.failed = true
	} else {
		lag := uint64(0)
		if reference.height > latest {
			lag = reference.height - latest
		}
		row.lag = strconv.FormatUint(lag, 10)
		row.status = evaluateStatus(syncProgress != nil, syncErr, latestHeader.Time, lag, policy)
		if row.status == statusUnknown {
			row.failed = true
		}
	}
	if history.nodeType == typeUnknown {
		row.failed = true
	}
	if len(details) == 0 {
		details = append(details, fmt.Sprintf("sampled %d heights", len(sampleHeights(latest))))
	}
	row.detail = strings.Join(details, "; ")
	return row
}

func fetchReferenceHeight(ctx context.Context, expectedChainID int64, endpoint string) referenceResult {
	client, err := ethclient.DialContext(ctx, endpoint)
	if err != nil {
		return referenceResult{err: fmt.Errorf("connect %s: %w", endpoint, err)}
	}
	defer client.Close()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return referenceResult{err: fmt.Errorf("get chain id from %s: %w", endpoint, err)}
	}
	if chainID == nil || chainID.Int64() != expectedChainID {
		return referenceResult{err: fmt.Errorf("unexpected chain id from %s: got %v, expected %d", endpoint, chainID, expectedChainID)}
	}
	height, err := client.BlockNumber(ctx)
	if err != nil {
		return referenceResult{err: fmt.Errorf("get block number from %s: %w", endpoint, err)}
	}
	return referenceResult{height: height}
}

func inspectHistory(ctx context.Context, client *ethclient.Client, latest uint64) historyResult {
	heights := sampleHeights(latest)
	stateHeights := historicalStateHeights(latest)
	results := make([]sampleResult, len(heights))

	var wait sync.WaitGroup
	wait.Add(len(heights))
	for index, height := range heights {
		go func(index int, height uint64) {
			defer wait.Done()
			_, checkState := stateHeights[height]
			results[index] = inspectSample(ctx, client, height, checkState)
		}(index, height)
	}
	wait.Wait()

	result := historyResult{
		blocks:   evidenceComplete,
		receipts: evidenceComplete,
		state:    evidenceComplete,
		nodeType: typeUnknown,
	}
	stateChecks := 0
	for _, sample := range results {
		result.blocks = mergeEvidence(result.blocks, sample.blocks)
		result.receipts = mergeEvidence(result.receipts, sample.receipts)
		if sample.state != "" {
			stateChecks++
			result.state = mergeEvidence(result.state, sample.state)
		}
		for _, detail := range sample.details {
			appendDetail(&result.details, detail)
		}
	}
	if stateChecks == 0 {
		result.state = evidenceUnknown
	}

	switch {
	case result.blocks == evidencePruned || result.receipts == evidencePruned:
		result.nodeType = typePrunedHistory
	case result.blocks == evidenceComplete && result.receipts == evidenceComplete && result.state == evidenceComplete:
		result.state = typeArchive
		result.nodeType = typeArchive
	case result.blocks == evidenceComplete && result.receipts == evidenceComplete && result.state == evidencePruned:
		result.nodeType = typeFullHistory
	default:
		result.nodeType = typeUnknown
	}
	return result
}

func sampleHeights(latest uint64) []uint64 {
	candidates := []uint64{
		0,
		percentage(latest, 1),
		percentage(latest, 10),
		percentage(latest, 25),
		percentage(latest, 50),
		percentage(latest, 75),
	}
	if latest > 1000 {
		candidates = append(candidates, latest-1000)
	} else {
		candidates = append(candidates, 0)
	}

	heights := make([]uint64, 0, len(candidates))
	seen := make(map[uint64]struct{}, len(candidates))
	for _, height := range candidates {
		if _, exists := seen[height]; exists {
			continue
		}
		seen[height] = struct{}{}
		heights = append(heights, height)
	}
	return heights
}

func historicalStateHeights(latest uint64) map[uint64]struct{} {
	heights := make(map[uint64]struct{}, 5)
	for _, percent := range []uint64{1, 10, 25, 50, 75} {
		height := percentage(latest, percent)
		if height != 0 {
			heights[height] = struct{}{}
		}
	}
	return heights
}

func percentage(value, percent uint64) uint64 {
	return value/100*percent + value%100*percent/100
}

func inspectSample(ctx context.Context, client *ethclient.Client, height uint64, checkState bool) sampleResult {
	result := sampleResult{height: height, blocks: evidenceUnknown, receipts: evidenceUnknown}
	var block *rpcBlock
	err := client.Client().CallContext(ctx, &block, "eth_getBlockByNumber", hexutil.EncodeUint64(height), false)
	if err != nil {
		if isPruningError(err) {
			result.blocks = evidencePruned
			result.receipts = evidencePruned
			result.details = append(result.details, fmt.Sprintf("history-pruned@%d", height))
		} else {
			result.details = append(result.details, fmt.Sprintf("block@%d: %s", height, compactError(err)))
		}
		result.state = inspectHistoricalState(ctx, client, height, checkState, &result.details)
		return result
	}
	if block == nil {
		result.blocks = evidencePruned
		result.receipts = evidencePruned
		result.details = append(result.details, fmt.Sprintf("history-missing@%d", height))
		result.state = inspectHistoricalState(ctx, client, height, checkState, &result.details)
		return result
	}
	if block.Hash == (common.Hash{}) || block.TransactionsRoot == (common.Hash{}) || block.ReceiptsRoot == (common.Hash{}) {
		result.details = append(result.details, fmt.Sprintf("invalid-block-roots@%d", height))
	} else if block.TransactionsRoot != types.EmptyTxsHash && len(block.Transactions) == 0 {
		result.blocks = evidencePruned
		result.details = append(result.details, fmt.Sprintf("block-body-pruned@%d", height))
	} else {
		result.blocks = evidenceComplete
	}

	result.receipts, err = inspectReceipts(ctx, client, height, block, result.blocks)
	if err != nil {
		result.details = append(result.details, fmt.Sprintf("receipts@%d: %s", height, compactError(err)))
	} else if result.receipts == evidencePruned {
		result.details = append(result.details, fmt.Sprintf("receipts-pruned@%d", height))
	}
	result.state = inspectHistoricalState(ctx, client, height, checkState, &result.details)
	return result
}

func inspectReceipts(
	ctx context.Context,
	client *ethclient.Client,
	height uint64,
	block *rpcBlock,
	blockEvidence string,
) (string, error) {
	if block.ReceiptsRoot == types.EmptyReceiptsHash {
		return evidenceComplete, nil
	}

	var receipts []*types.Receipt
	err := client.Client().CallContext(ctx, &receipts, "eth_getBlockReceipts", hexutil.EncodeUint64(height))
	if err == nil {
		if len(receipts) == 0 {
			return evidencePruned, nil
		}
		if blockEvidence == evidenceComplete && len(block.Transactions) != 0 && len(receipts) != len(block.Transactions) {
			return evidencePruned, fmt.Errorf("got %d receipts for %d transactions", len(receipts), len(block.Transactions))
		}
		return evidenceComplete, nil
	}
	if isPruningError(err) {
		return evidencePruned, nil
	}
	if len(block.Transactions) == 0 {
		return evidenceUnknown, fmt.Errorf("block receipts unavailable and no transaction hash for fallback: %w", err)
	}

	receipt, fallbackErr := client.TransactionReceipt(ctx, block.Transactions[0])
	if fallbackErr == nil && receipt != nil {
		return evidenceComplete, nil
	}
	if fallbackErr != nil && isPruningError(fallbackErr) {
		return evidencePruned, nil
	}
	if fallbackErr == nil {
		fallbackErr = ethereum.NotFound
	}
	return evidenceUnknown, fmt.Errorf("block receipts unavailable (%v), transaction receipt fallback failed: %w", err, fallbackErr)
}

func inspectHistoricalState(
	ctx context.Context,
	client *ethclient.Client,
	height uint64,
	check bool,
	details *[]string,
) string {
	if !check {
		return ""
	}
	_, err := client.BalanceAt(ctx, common.Address{}, new(big.Int).SetUint64(height))
	if err == nil {
		return evidenceComplete
	}
	if isPruningError(err) {
		*details = append(*details, fmt.Sprintf("state-pruned@%d", height))
		return evidencePruned
	}
	*details = append(*details, fmt.Sprintf("state@%d: %s", height, compactError(err)))
	return evidenceUnknown
}

func mergeEvidence(current, next string) string {
	if current == evidencePruned || next == evidencePruned {
		return evidencePruned
	}
	if current == evidenceUnknown || next == evidenceUnknown {
		return evidenceUnknown
	}
	return evidenceComplete
}

func evaluateStatus(syncing bool, syncErr error, latestBlockTime uint64, lag uint64, policy chainPolicy) string {
	if syncErr != nil {
		return statusUnknown
	}
	if syncing {
		return statusSyncing
	}
	if time.Since(time.Unix(int64(latestBlockTime), 0)) > policy.maxBlockAge {
		return statusStale
	}
	if lag > policy.maxBlockLag {
		return statusLagging
	}
	return statusHealthy
}

func isPruningError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ethereum.NotFound) {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"prun",
		"old data not available",
		"history is available from",
		"historical state",
		"missing trie node",
		"no state available",
		"no transactions snapshot",
		"header not found",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func appendDetail(details *[]string, value string) {
	if len(*details) >= 6 {
		return
	}
	value = compactCell(value)
	if value == "" {
		return
	}
	*details = append(*details, value)
}

func compactError(err error) string {
	if err == nil {
		return ""
	}
	return compactCell(err.Error())
}

func compactCell(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	const maxLength = 180
	if len(value) > maxLength {
		return value[:maxLength-3] + "..."
	}
	return value
}

func writeResultTable(output io.Writer, row resultRow) {
	table := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(table, "ENDPOINT\tCHAIN\tCLIENT\tLATEST\tLAG\tSTATUS\tBLOCKS\tRECEIPTS\tSTATE\tTYPE\tDETAIL")
	fmt.Fprintf(
		table,
		"%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		compactCell(row.endpoint),
		compactCell(row.chain),
		compactCell(row.client),
		compactCell(row.latest),
		compactCell(row.lag),
		compactCell(row.status),
		compactCell(row.blocks),
		compactCell(row.receipts),
		compactCell(row.state),
		compactCell(row.nodeType),
		compactCell(row.detail),
	)
	_ = table.Flush()
}
