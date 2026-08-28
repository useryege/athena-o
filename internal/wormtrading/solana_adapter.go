package wormtrading

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"
	rpcjson "github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
	"golang.org/x/time/rate"
)

const (
	SolanaNetwork                         = "solana-mainnet-beta"
	SolanaCommitment                      = "confirmed"
	SolanaMainnetGenesisHash              = "5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d"
	SolanaNativeUSDCMint                  = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	DefaultSolanaMainnetRPCEndpoint       = "https://api.mainnet-beta.solana.com"
	SolanaDecimals                  int32 = 9
	USDCDecimals                    int32 = 6

	SolanaProbeInterval = 30 * time.Second

	DefaultSolanaRPCAttemptTimeout = 4 * time.Second
	DefaultSolanaBalanceBudget     = 12 * time.Second
	DefaultSolanaRPCRateLimit      = 40.0
	DefaultSolanaRPCRateBurst      = 40

	solanaBatchSize        = 25
	solanaBatchConcurrency = 2
	solanaRetryBaseDelay   = 200 * time.Millisecond
	legacyTokenAccountSize = 165

	availabilityAvailable   = "AVAILABLE"
	availabilityUnavailable = "UNAVAILABLE"

	walletBalanceComplete    = "COMPLETE"
	walletBalancePartial     = "PARTIAL"
	walletBalanceUnavailable = "UNAVAILABLE"

	errorCancelled            = "CANCELLED"
	errorDecode               = "DECODE_ERROR"
	errorInvalidResponse      = "INVALID_RESPONSE"
	errorInvalidWalletAddress = "INVALID_WALLET_ADDRESS"
	errorRateLimited          = "RATE_LIMITED"
	errorRPCRejected          = "RPC_REJECTED"
	errorRPCUnavailable       = "RPC_UNAVAILABLE"
	errorTimeout              = "TIMEOUT"
)

var (
	usdcMintPublicKey = solana.MustPublicKeyFromBase58(SolanaNativeUSDCMint)
)

// WalletBalanceReference is a caller-resolved Wallet identifier and public
// address. The adapter deliberately has no Wallet service or database access.
type WalletBalanceReference struct {
	WalletID int64
	Address  string
}

// BalanceObservation preserves atomic precision and the independent Solana
// slot at which one asset was evaluated.
type BalanceObservation struct {
	AtomicAmount      string
	Amount            string
	Decimals          int32
	ObservedSlot      uint64
	Availability      string
	ErrorCode         string
	TokenAccountCount int32
}

// WalletBalanceResult is one ordered, independently fallible wallet result.
type WalletBalanceResult struct {
	WalletID int64
	Address  string
	SOL      BalanceObservation
	USDC     BalanceObservation
	Status   string
}

// SolanaAdapterStatus is the redacted operational state exposed by the
// service. It never contains the RPC endpoint or provider response body.
type SolanaAdapterStatus struct {
	Status              string
	RPCConfigured       bool
	RPCReachable        bool
	BatchSupported      bool
	GenesisVerified     bool
	GenesisHash         string
	USDCProgramVerified bool
	USDCDecimals        int32
	LatestConfirmedSlot uint64
	LastProbeAt         int64
	LastSuccessAt       int64
	LatencyMS           int64
	ConsecutiveFailures int32
	LastErrorCode       string
	ConfigurationError  bool
}

func (s SolanaAdapterStatus) readyForReads() bool {
	return !s.ConfigurationError && s.GenesisVerified && s.USDCProgramVerified && s.USDCDecimals == USDCDecimals
}

type SolanaBalanceAdapter struct {
	client            *rpc.Client
	limiter           *rate.Limiter
	rpcAttemptTimeout time.Duration
	balanceBudget     time.Duration

	probeMu sync.Mutex
	stateMu sync.RWMutex
	status  SolanaAdapterStatus

	lifecycleMu  sync.RWMutex
	lifecycleCtx context.Context

	reads singleflight.Group
}

// SolanaBalanceAdapterConfig contains the provider endpoint and bounded
// runtime controls. Chain, mint and commitment are intentionally not
// configurable.
type SolanaBalanceAdapterConfig struct {
	RPCURL            string
	RPCAttemptTimeout time.Duration
	BalanceBudget     time.Duration
	RPCRateLimit      float64
	RPCRateBurst      int
}

func DefaultSolanaBalanceAdapterConfig() SolanaBalanceAdapterConfig {
	return SolanaBalanceAdapterConfig{
		RPCURL:            DefaultSolanaMainnetRPCEndpoint,
		RPCAttemptTimeout: DefaultSolanaRPCAttemptTimeout,
		BalanceBudget:     DefaultSolanaBalanceBudget,
		RPCRateLimit:      DefaultSolanaRPCRateLimit,
		RPCRateBurst:      DefaultSolanaRPCRateBurst,
	}
}

// NewSolanaBalanceAdapter creates the fixed-mainnet, fixed-USDC adapter. The
// endpoint is configurable, but its chain identity is not trusted until Probe
// verifies the mainnet genesis and native USDC mint.
func NewSolanaBalanceAdapter(config SolanaBalanceAdapterConfig) (*SolanaBalanceAdapter, error) {
	endpoint := strings.TrimSpace(config.RPCURL)
	if endpoint == "" {
		return nil, fmt.Errorf("Solana RPC URL is required")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("Solana RPC URL must be an absolute HTTP or HTTPS URL")
	}
	if config.RPCAttemptTimeout <= 0 {
		return nil, fmt.Errorf("Solana RPC attempt timeout must be positive")
	}
	if config.BalanceBudget <= 0 {
		return nil, fmt.Errorf("Solana balance budget must be positive")
	}
	if config.RPCAttemptTimeout > config.BalanceBudget {
		return nil, fmt.Errorf("Solana RPC attempt timeout must not exceed the balance budget")
	}
	if math.IsNaN(config.RPCRateLimit) || math.IsInf(config.RPCRateLimit, 0) || config.RPCRateLimit <= 0 {
		return nil, fmt.Errorf("Solana RPC rate limit must be a finite positive number")
	}
	if config.RPCRateBurst < solanaBatchSize+1 {
		return nil, fmt.Errorf("Solana RPC rate burst must be at least %d", solanaBatchSize+1)
	}
	return &SolanaBalanceAdapter{
		client:            rpc.NewWithTimeoutAndCommitment(endpoint, config.RPCAttemptTimeout, rpc.CommitmentConfirmed),
		limiter:           rate.NewLimiter(rate.Limit(config.RPCRateLimit), config.RPCRateBurst),
		rpcAttemptTimeout: config.RPCAttemptTimeout,
		balanceBudget:     config.BalanceBudget,
		status: SolanaAdapterStatus{
			Status:        "degraded",
			RPCConfigured: true,
			USDCDecimals:  USDCDecimals,
		},
	}, nil
}

// Start binds coalesced reads to the service lifecycle. The caller owns ctx.
func (a *SolanaBalanceAdapter) Start(ctx context.Context) {
	a.lifecycleMu.Lock()
	a.lifecycleCtx = ctx
	a.lifecycleMu.Unlock()
}

func (a *SolanaBalanceAdapter) lifecycleContext() context.Context {
	a.lifecycleMu.RLock()
	ctx := a.lifecycleCtx
	a.lifecycleMu.RUnlock()
	return ctx
}

func (a *SolanaBalanceAdapter) Status() SolanaAdapterStatus {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.status
}

// Probe verifies JSON-RPC batch behavior, mainnet identity, the native USDC
// token program and decimal scale, and confirmed-slot reachability.
func (a *SolanaBalanceAdapter) Probe(ctx context.Context) SolanaAdapterStatus {
	a.probeMu.Lock()
	defer a.probeMu.Unlock()

	current := a.Status()
	if current.ConfigurationError {
		return current
	}

	startedAt := time.Now()
	probeCtx, cancel := context.WithTimeout(ctx, a.balanceBudget)
	defer cancel()

	descriptors := []batchDescriptor{
		{request: rpcjson.NewRequest("getGenesisHash")},
		{request: rpcjson.NewRequest(
			"getAccountInfo",
			usdcMintPublicKey,
			rpc.M{"commitment": rpc.CommitmentConfirmed, "encoding": solana.EncodingBase64},
		)},
		{request: rpcjson.NewRequest(
			"getTokenSupply",
			usdcMintPublicKey,
			rpc.M{"commitment": rpc.CommitmentConfirmed},
		)},
		{request: rpcjson.NewRequest(
			"getSlot",
			rpc.M{"commitment": rpc.CommitmentConfirmed},
		)},
	}
	execution := a.executeBatch(probeCtx, descriptors)
	observedAt := time.Now()
	latency := observedAt.Sub(startedAt).Milliseconds()

	probeStatus := current
	probeStatus.RPCReachable = execution.rpcReachable
	probeStatus.BatchSupported = current.BatchSupported || execution.batchSupported
	probeStatus.LastProbeAt = observedAt.Unix()
	probeStatus.LatencyMS = latency

	for _, outcome := range execution.outcomes {
		if outcome.failure != nil {
			return a.recordProbeFailure(probeStatus, *outcome.failure)
		}
	}
	if len(execution.outcomes) != len(descriptors) {
		return a.recordProbeFailure(probeStatus, batchFailure{
			code: errorInvalidResponse,
		})
	}

	var genesisHash string
	if err := execution.outcomes[0].response.GetObject(&genesisHash); err != nil || genesisHash == "" {
		return a.recordProbeFailure(probeStatus, batchFailure{code: errorInvalidResponse})
	}
	probeStatus.GenesisHash = genesisHash
	probeStatus.GenesisVerified = genesisHash == SolanaMainnetGenesisHash
	if !probeStatus.GenesisVerified {
		return a.recordProbeFailure(probeStatus, configurationFailure())
	}

	var mintAccount rpc.GetAccountInfoResult
	if err := execution.outcomes[1].response.GetObject(&mintAccount); err != nil || mintAccount.Value == nil {
		return a.recordProbeFailure(probeStatus, batchFailure{code: errorInvalidResponse})
	}
	if !mintAccount.Value.Owner.Equals(solana.TokenProgramID) {
		probeStatus.USDCProgramVerified = false
		return a.recordProbeFailure(probeStatus, configurationFailure())
	}
	probeStatus.USDCProgramVerified = true

	var tokenSupply rpc.GetTokenSupplyResult
	if err := execution.outcomes[2].response.GetObject(&tokenSupply); err != nil || tokenSupply.Value == nil {
		return a.recordProbeFailure(probeStatus, batchFailure{code: errorInvalidResponse})
	}
	probeStatus.USDCDecimals = int32(tokenSupply.Value.Decimals)
	if tokenSupply.Value.Decimals != uint8(USDCDecimals) {
		return a.recordProbeFailure(probeStatus, configurationFailure())
	}

	var confirmedSlot uint64
	if err := execution.outcomes[3].response.GetObject(&confirmedSlot); err != nil || confirmedSlot == 0 {
		return a.recordProbeFailure(probeStatus, batchFailure{code: errorInvalidResponse})
	}
	probeStatus.LatestConfirmedSlot = confirmedSlot

	probeStatus.Status = "running"
	probeStatus.LastSuccessAt = observedAt.Unix()
	probeStatus.ConsecutiveFailures = 0
	probeStatus.LastErrorCode = ""
	a.storeStatus(probeStatus)
	return probeStatus
}

func (a *SolanaBalanceAdapter) recordProbeFailure(status SolanaAdapterStatus, failure batchFailure) SolanaAdapterStatus {
	status.ConsecutiveFailures++
	status.LastErrorCode = failure.code
	status.ConfigurationError = failure.permanent
	if failure.batchUnsupported {
		status.BatchSupported = false
	}
	if failure.permanent {
		status.Status = "configuration_error"
	} else {
		status.Status = "degraded"
	}
	a.storeStatus(status)
	return status
}

func (a *SolanaBalanceAdapter) storeStatus(status SolanaAdapterStatus) {
	a.stateMu.Lock()
	a.status = status
	a.stateMu.Unlock()
}

// BatchGetBalances reads at most 100 refs and preserves their request order.
// Invalid addresses are per-wallet failures, while identical concurrent address
// sets share one non-cached RPC operation.
func (a *SolanaBalanceAdapter) BatchGetBalances(ctx context.Context, refs []WalletBalanceReference) ([]WalletBalanceResult, error) {
	lifecycleCtx := a.lifecycleContext()
	if lifecycleCtx == nil {
		return nil, fmt.Errorf("Solana balance adapter is not started")
	}

	results := make([]WalletBalanceResult, len(refs))
	validByAddress := make(map[string]solana.PublicKey, len(refs))
	for index, ref := range refs {
		address := strings.TrimSpace(ref.Address)
		results[index] = unavailableWalletResult(ref.WalletID, address, errorInvalidWalletAddress)
		publicKey, err := solana.PublicKeyFromBase58(address)
		if err != nil || publicKey.String() != address {
			continue
		}
		validByAddress[address] = publicKey
	}
	if len(validByAddress) == 0 {
		return results, nil
	}

	addresses := make([]string, 0, len(validByAddress))
	for address := range validByAddress {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)

	key := balanceSingleflightKey(addresses)
	resultChan := a.reads.DoChan(key, func() (any, error) {
		sharedCtx, cancel := context.WithTimeout(lifecycleCtx, a.balanceBudget)
		defer cancel()
		return a.readUniqueBalances(sharedCtx, addresses, validByAddress), nil
	})

	var uniqueResults map[string]addressBalanceResult
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case shared := <-resultChan:
		if shared.Err != nil {
			return nil, shared.Err
		}
		var ok bool
		uniqueResults, ok = shared.Val.(map[string]addressBalanceResult)
		if !ok {
			return nil, fmt.Errorf("unexpected coalesced Solana balance result")
		}
	}

	for index, ref := range refs {
		address := strings.TrimSpace(ref.Address)
		balance, ok := uniqueResults[address]
		if !ok {
			continue
		}
		results[index] = WalletBalanceResult{
			WalletID: ref.WalletID,
			Address:  address,
			SOL:      balance.SOL,
			USDC:     balance.USDC,
			Status:   walletBalanceStatus(balance.SOL, balance.USDC),
		}
	}
	return results, nil
}

type addressBalanceResult struct {
	SOL  BalanceObservation
	USDC BalanceObservation
}

type addressSpec struct {
	address   string
	publicKey solana.PublicKey
}

func (a *SolanaBalanceAdapter) readUniqueBalances(
	ctx context.Context,
	addresses []string,
	publicKeys map[string]solana.PublicKey,
) map[string]addressBalanceResult {
	output := make(map[string]addressBalanceResult, len(addresses))
	var outputMu sync.Mutex
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(solanaBatchConcurrency)

	for start := 0; start < len(addresses); start += solanaBatchSize {
		end := start + solanaBatchSize
		if end > len(addresses) {
			end = len(addresses)
		}
		chunkAddresses := append([]string(nil), addresses[start:end]...)
		group.Go(func() error {
			chunk := make([]addressSpec, 0, len(chunkAddresses))
			for _, address := range chunkAddresses {
				chunk = append(chunk, addressSpec{address: address, publicKey: publicKeys[address]})
			}
			chunkOutput := a.readBalanceChunk(groupCtx, chunk)
			outputMu.Lock()
			for address, balance := range chunkOutput {
				output[address] = balance
			}
			outputMu.Unlock()
			return nil
		})
	}
	_ = group.Wait()
	return output
}

func (a *SolanaBalanceAdapter) readBalanceChunk(ctx context.Context, chunk []addressSpec) map[string]addressBalanceResult {
	publicKeys := make([]solana.PublicKey, 0, len(chunk))
	for _, spec := range chunk {
		publicKeys = append(publicKeys, spec.publicKey)
	}

	descriptors := make([]batchDescriptor, 0, len(chunk)+1)
	descriptors = append(descriptors, batchDescriptor{request: rpcjson.NewRequest(
		"getMultipleAccounts",
		publicKeys,
		rpc.M{"commitment": rpc.CommitmentConfirmed, "encoding": solana.EncodingBase64},
	)})
	for _, spec := range chunk {
		descriptors = append(descriptors, batchDescriptor{request: rpcjson.NewRequest(
			"getTokenAccountsByOwner",
			spec.publicKey,
			rpc.M{"mint": usdcMintPublicKey},
			rpc.M{"commitment": rpc.CommitmentConfirmed, "encoding": solana.EncodingBase64},
		)})
	}

	execution := a.executeBatch(ctx, descriptors)
	output := make(map[string]addressBalanceResult, len(chunk))
	for _, spec := range chunk {
		output[spec.address] = addressBalanceResult{
			SOL:  unavailableObservation(SolanaDecimals, errorInvalidResponse),
			USDC: unavailableObservation(USDCDecimals, errorInvalidResponse),
		}
	}
	if len(execution.outcomes) != len(descriptors) {
		return output
	}

	solOutcome := execution.outcomes[0]
	if solOutcome.failure != nil {
		for _, spec := range chunk {
			balance := output[spec.address]
			balance.SOL = unavailableObservation(SolanaDecimals, solOutcome.failure.code)
			output[spec.address] = balance
		}
	} else {
		var accounts rpc.GetMultipleAccountsResult
		if err := solOutcome.response.GetObject(&accounts); err != nil || len(accounts.Value) != len(chunk) || accounts.Context.Slot == 0 {
			for _, spec := range chunk {
				balance := output[spec.address]
				balance.SOL = unavailableObservation(SolanaDecimals, errorInvalidResponse)
				output[spec.address] = balance
			}
		} else {
			for index, spec := range chunk {
				lamports := uint64(0)
				if accounts.Value[index] != nil {
					lamports = accounts.Value[index].Lamports
				}
				balance := output[spec.address]
				balance.SOL = availableObservation(lamports, SolanaDecimals, accounts.Context.Slot)
				output[spec.address] = balance
			}
		}
	}

	for index, spec := range chunk {
		outcome := execution.outcomes[index+1]
		balance := output[spec.address]
		if outcome.failure != nil {
			balance.USDC = unavailableObservation(USDCDecimals, outcome.failure.code)
			output[spec.address] = balance
			continue
		}
		observation, errCode := decodeUSDCBalance(outcome.response, spec.publicKey)
		if errCode != "" {
			balance.USDC = unavailableObservation(USDCDecimals, errCode)
		} else {
			balance.USDC = observation
		}
		output[spec.address] = balance
	}
	return output
}

type rawTokenAccountsResult struct {
	Context rpc.Context          `json:"context"`
	Value   []stdjson.RawMessage `json:"value"`
}

func decodeUSDCBalance(response *rpcjson.RPCResponse, owner solana.PublicKey) (BalanceObservation, string) {
	var accounts rawTokenAccountsResult
	if err := response.GetObject(&accounts); err != nil || accounts.Value == nil || accounts.Context.Slot == 0 {
		return BalanceObservation{}, errorInvalidResponse
	}

	var total uint64
	for _, raw := range accounts.Value {
		var rawAccount rpc.TokenAccount
		if len(raw) == 0 || stdjson.Unmarshal(raw, &rawAccount) != nil ||
			!rawAccount.Account.Owner.Equals(solana.TokenProgramID) || rawAccount.Account.Data == nil {
			return BalanceObservation{}, errorDecode
		}
		data := rawAccount.Account.Data.GetBinary()
		if len(data) != legacyTokenAccountSize {
			return BalanceObservation{}, errorDecode
		}
		var tokenAccount token.Account
		if err := bin.NewBinDecoder(data).Decode(&tokenAccount); err != nil {
			return BalanceObservation{}, errorDecode
		}
		if !tokenAccount.Mint.Equals(usdcMintPublicKey) ||
			!tokenAccount.Owner.Equals(owner) ||
			(tokenAccount.State != token.Initialized && tokenAccount.State != token.Frozen) {
			return BalanceObservation{}, errorDecode
		}
		if ^uint64(0)-total < tokenAccount.Amount {
			return BalanceObservation{}, errorDecode
		}
		total += tokenAccount.Amount
	}
	observation := availableObservation(total, USDCDecimals, accounts.Context.Slot)
	observation.TokenAccountCount = int32(len(accounts.Value))
	return observation, ""
}

func availableObservation(value uint64, decimals int32, slot uint64) BalanceObservation {
	return BalanceObservation{
		AtomicAmount: strconv.FormatUint(value, 10),
		Amount:       formatAtomicAmount(value, decimals),
		Decimals:     decimals,
		ObservedSlot: slot,
		Availability: availabilityAvailable,
	}
}

func unavailableObservation(decimals int32, errorCode string) BalanceObservation {
	return BalanceObservation{
		Decimals:     decimals,
		Availability: availabilityUnavailable,
		ErrorCode:    errorCode,
	}
}

func unavailableWalletResult(walletID int64, address, errorCode string) WalletBalanceResult {
	return WalletBalanceResult{
		WalletID: walletID,
		Address:  address,
		SOL:      unavailableObservation(SolanaDecimals, errorCode),
		USDC:     unavailableObservation(USDCDecimals, errorCode),
		Status:   walletBalanceUnavailable,
	}
}

func walletBalanceStatus(solBalance, usdcBalance BalanceObservation) string {
	solAvailable := solBalance.Availability == availabilityAvailable
	usdcAvailable := usdcBalance.Availability == availabilityAvailable
	switch {
	case solAvailable && usdcAvailable:
		return walletBalanceComplete
	case solAvailable || usdcAvailable:
		return walletBalancePartial
	default:
		return walletBalanceUnavailable
	}
}

func formatAtomicAmount(value uint64, decimals int32) string {
	digits := strconv.FormatUint(value, 10)
	if decimals <= 0 {
		return digits
	}
	scale := int(decimals)
	if len(digits) <= scale {
		return "0." + strings.Repeat("0", scale-len(digits)) + digits
	}
	return digits[:len(digits)-scale] + "." + digits[len(digits)-scale:]
}

func balanceSingleflightKey(addresses []string) string {
	digest := sha256.Sum256([]byte(strings.Join([]string{
		SolanaNetwork,
		SolanaCommitment,
		SolanaNativeUSDCMint,
		strings.Join(addresses, "\x00"),
	}, "\x01")))
	return hex.EncodeToString(digest[:])
}

type batchDescriptor struct {
	request *rpcjson.RPCRequest
}

type batchOutcome struct {
	response *rpcjson.RPCResponse
	failure  *batchFailure
}

type batchFailure struct {
	code             string
	retryable        bool
	permanent        bool
	batchUnsupported bool
}

type batchExecution struct {
	outcomes       []batchOutcome
	rpcReachable   bool
	batchSupported bool
}

func permanentFailure(code string) batchFailure {
	return batchFailure{code: code, permanent: true}
}

func configurationFailure() batchFailure {
	return permanentFailure(errorInvalidResponse)
}

func unsupportedBatchFailure(code string) batchFailure {
	return batchFailure{code: code, permanent: true, batchUnsupported: true}
}

func (a *SolanaBalanceAdapter) executeBatch(ctx context.Context, descriptors []batchDescriptor) batchExecution {
	first, callFailure := a.callBatchOnce(ctx, descriptors)
	if callFailure != nil {
		if callFailure.retryable && waitForSolanaRetry(ctx) == nil {
			second, _ := a.callBatchOnce(ctx, descriptors)
			second.rpcReachable = first.rpcReachable || second.rpcReachable
			second.batchSupported = first.batchSupported || second.batchSupported
			return second
		}
		return first
	}

	retryIndexes := make([]int, 0)
	retryDescriptors := make([]batchDescriptor, 0)
	for index, outcome := range first.outcomes {
		if outcome.failure != nil && outcome.failure.retryable {
			retryIndexes = append(retryIndexes, index)
			retryDescriptors = append(retryDescriptors, descriptors[index])
		}
	}
	if len(retryDescriptors) == 0 || waitForSolanaRetry(ctx) != nil {
		return first
	}

	retryResult, _ := a.callBatchOnce(ctx, retryDescriptors)
	for retryIndex, originalIndex := range retryIndexes {
		if retryIndex < len(retryResult.outcomes) {
			first.outcomes[originalIndex] = retryResult.outcomes[retryIndex]
		}
	}
	first.rpcReachable = first.rpcReachable || retryResult.rpcReachable
	first.batchSupported = first.batchSupported || retryResult.batchSupported
	return first
}

func (a *SolanaBalanceAdapter) callBatchOnce(ctx context.Context, descriptors []batchDescriptor) (batchExecution, *batchFailure) {
	if len(descriptors) == 0 {
		failure := batchFailure{code: errorInvalidResponse}
		return failedBatchExecution(0, failure, false), &failure
	}
	if err := a.limiter.WaitN(ctx, len(descriptors)); err != nil {
		failure := classifyContextFailure(ctx)
		return failedBatchExecution(len(descriptors), failure, false), &failure
	}

	requests := make(rpcjson.RPCRequests, 0, len(descriptors))
	for _, descriptor := range descriptors {
		requests = append(requests, descriptor.request)
	}
	attemptCtx, cancel := context.WithTimeout(ctx, a.rpcAttemptTimeout)
	defer cancel()
	responses, err := a.client.RPCCallBatch(attemptCtx, requests)
	if err != nil {
		failure, reachable := classifyBatchCallError(attemptCtx, err)
		return failedBatchExecution(len(descriptors), failure, reachable), &failure
	}

	outcomes := make([]batchOutcome, len(descriptors))
	for index := range outcomes {
		failure := batchFailure{code: errorInvalidResponse}
		outcomes[index].failure = &failure
	}
	seen := make(map[int]struct{}, len(responses))
	for _, response := range responses {
		if response == nil {
			continue
		}
		id, ok := normalizeBatchResponseID(response.ID)
		if !ok || id < 0 || id >= len(descriptors) {
			continue
		}
		if _, duplicate := seen[id]; duplicate {
			failure := batchFailure{code: errorInvalidResponse}
			outcomes[id] = batchOutcome{failure: &failure}
			continue
		}
		seen[id] = struct{}{}
		if response.Error != nil {
			failure := classifyRPCError(response.Error)
			outcomes[id] = batchOutcome{failure: &failure}
			continue
		}
		if response.Result == nil {
			failure := batchFailure{code: errorInvalidResponse}
			outcomes[id] = batchOutcome{failure: &failure}
			continue
		}
		outcomes[id] = batchOutcome{response: response}
	}
	return batchExecution{outcomes: outcomes, rpcReachable: true, batchSupported: true}, nil
}

func failedBatchExecution(count int, failure batchFailure, reachable bool) batchExecution {
	outcomes := make([]batchOutcome, count)
	for index := range outcomes {
		copyOfFailure := failure
		outcomes[index].failure = &copyOfFailure
	}
	return batchExecution{outcomes: outcomes, rpcReachable: reachable}
}

func classifyContextFailure(ctx context.Context) batchFailure {
	if errors.Is(ctx.Err(), context.Canceled) {
		return batchFailure{code: errorCancelled}
	}
	return batchFailure{code: errorTimeout, retryable: true}
}

func classifyBatchCallError(ctx context.Context, err error) (batchFailure, bool) {
	if errors.Is(ctx.Err(), context.Canceled) {
		return batchFailure{code: errorCancelled}, false
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return batchFailure{code: errorTimeout, retryable: true}, false
	}

	var httpError *rpcjson.HTTPError
	if errors.As(err, &httpError) {
		switch httpError.Code {
		case 408, 425, 429, 500, 502, 503, 504:
			code := errorRPCUnavailable
			if httpError.Code == 429 {
				code = errorRateLimited
			}
			return batchFailure{code: code, retryable: true}, true
		case 400, 404, 405, 415:
			return unsupportedBatchFailure(errorRPCRejected), true
		default:
			return permanentFailure(errorRPCRejected), true
		}
	}
	var netError net.Error
	if errors.As(err, &netError) {
		return batchFailure{code: errorRPCUnavailable, retryable: true}, false
	}
	var syntaxError *stdjson.SyntaxError
	var typeError *stdjson.UnmarshalTypeError
	if errors.As(err, &syntaxError) || errors.As(err, &typeError) {
		return batchFailure{code: errorInvalidResponse}, true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return batchFailure{code: errorRPCUnavailable, retryable: true}, true
	}
	return batchFailure{code: errorInvalidResponse}, true
}

func classifyRPCError(rpcError *rpcjson.RPCError) batchFailure {
	if rpcError == nil {
		return permanentFailure(errorInvalidResponse)
	}
	switch rpcError.Code {
	case -32005, -32603:
		return batchFailure{code: errorRPCUnavailable, retryable: true}
	case -32600, -32601:
		return permanentFailure(errorRPCRejected)
	case -32602:
		return permanentFailure(errorRPCRejected)
	default:
		return batchFailure{code: errorRPCRejected}
	}
}

func normalizeBatchResponseID(value any) (int, bool) {
	switch id := value.(type) {
	case stdjson.Number:
		parsed, err := strconv.Atoi(id.String())
		return parsed, err == nil
	case int:
		return id, true
	case int32:
		return int(id), true
	case int64:
		if int64(int(id)) != id {
			return 0, false
		}
		return int(id), true
	case float64:
		converted := int(id)
		return converted, float64(converted) == id
	default:
		return 0, false
	}
}

func waitForSolanaRetry(ctx context.Context) error {
	jitter := time.Duration(time.Now().UnixNano() % int64(solanaRetryBaseDelay))
	timer := time.NewTimer(solanaRetryBaseDelay + jitter)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
