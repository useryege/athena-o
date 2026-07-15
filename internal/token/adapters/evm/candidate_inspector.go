package evm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/useryege/athena/internal/token/chainregistry"
	"github.com/useryege/athena/internal/token/discovery"
	discoveryapp "github.com/useryege/athena/internal/token/discovery/application"
	"github.com/useryege/athena/internal/token/shared"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	erc20contract "github.com/useryege/athena/pkg/abi/ERC20"
)

const (
	initialRecipientWalletLimit = 10
	basisPointsDenominator      = int64(10000)
)

var errProjectCreationReceiptNil = errors.New("project creation transaction receipt is nil")

type CandidateInspector struct {
	registry *chainregistry.Registry
	clients  *ChainClientRegistry
}

func NewCandidateInspector(registry *chainregistry.Registry, clients *ChainClientRegistry) *CandidateInspector {
	return &CandidateInspector{registry: registry, clients: clients}
}

func (inspector *CandidateInspector) InspectCandidates(ctx context.Context, chainID int64, candidates []discovery.ProjectCandidate, concurrency int) ([]discoveryapp.CandidateInspection, error) {
	client, err := inspector.clients.Client(ctx, chainID)
	if err != nil {
		return nil, err
	}
	chain, ok := inspector.registry.Chain(chainID)
	if !ok || !common.IsHexAddress(chain.AthenaContract) {
		return nil, fmt.Errorf("token chain %d ATHENA contract is invalid", chainID)
	}
	caller, err := athenacontract.NewATHENACaller(common.HexToAddress(chain.AthenaContract), client)
	if err != nil {
		return nil, err
	}
	contracts := make([]common.Address, 0, len(candidates))
	for _, candidate := range candidates {
		contracts = append(contracts, common.Address(candidate.Contract))
	}
	validations, err := caller.ValidateERC20(&bind.CallOpts{Context: ctx}, contracts)
	if err != nil {
		inspector.clients.Reset(chainID)
		return nil, err
	}
	if len(validations) != len(candidates) {
		return nil, fmt.Errorf("validate ERC20 returned %d results for %d candidates", len(validations), len(candidates))
	}
	if concurrency <= 0 {
		concurrency = 1
	}
	results := make([]discoveryapp.CandidateInspection, len(candidates))
	errorsByIndex := make([]error, len(candidates))
	semaphore := make(chan struct{}, concurrency)
	var group sync.WaitGroup
	for index, candidate := range candidates {
		index, candidate, validation := index, candidate, validations[index]
		semaphore <- struct{}{}
		group.Add(1)
		go func() {
			defer group.Done()
			defer func() { <-semaphore }()
			results[index], errorsByIndex[index] = inspectCandidate(ctx, client, candidate, validation)
		}()
	}
	group.Wait()
	for _, err := range errorsByIndex {
		if err != nil {
			inspector.clients.Reset(chainID)
			return nil, err
		}
	}
	return results, nil
}

func inspectCandidate(ctx context.Context, client interface {
	CodeAt(context.Context, common.Address, *big.Int) ([]byte, error)
	TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error)
}, candidate discovery.ProjectCandidate, validation athenacontract.AthenaToken) (discoveryapp.CandidateInspection, error) {
	result := discoveryapp.CandidateInspection{Candidate: candidate}
	if !validation.IsValidERC20 {
		return result, nil
	}
	contract := common.Address(candidate.Contract)
	code, err := client.CodeAt(ctx, contract, nil)
	if err != nil {
		return result, err
	}
	if len(code) == 0 {
		return result, nil
	}
	receipt, err := client.TransactionReceipt(ctx, common.Hash(candidate.TxHash))
	if err != nil {
		return result, err
	}
	if receipt == nil {
		return result, errProjectCreationReceiptNil
	}
	recipients := extractInitialRecipientWallets(receipt.Logs, contract)
	recipients, err = filterInitialRecipientWallets(ctx, client, recipients, initialRecipientWalletLimit)
	if err != nil {
		return result, err
	}
	result.Accepted = true
	result.CodeHash = shared.Hash(crypto.Keccak256Hash(code))
	result.Name = validation.Name
	result.Symbol = validation.Symbol
	result.Decimals = validation.Decimals
	result.TotalSupply = cloneBigInt(validation.TotalSupply)
	result.WethPair = shared.Address(validation.WethPair)
	result.UsdtPair = shared.Address(validation.UsdtPair)
	if !candidate.TxSender.IsZero() {
		result.RelatedWallets = append(result.RelatedWallets, discoveryapp.RelatedWallet{Wallet: candidate.TxSender, Role: "creator"})
	}
	for index, recipient := range recipients {
		wallet := shared.Address(recipient.wallet)
		result.RelatedWallets = append(result.RelatedWallets, discoveryapp.RelatedWallet{Wallet: wallet, Role: "initial_recipient"})
		if validation.TotalSupply != nil && validation.TotalSupply.Sign() > 0 {
			ratio := new(big.Int).Mul(recipient.amount, big.NewInt(basisPointsDenominator))
			ratio.Div(ratio, validation.TotalSupply)
			result.InitialRecipients = append(result.InitialRecipients, discoveryapp.InitialRecipient{Wallet: wallet, RatioBPS: ratio.Int64(), RankIndex: int32(index), SourceTxHash: candidate.TxHash, SourceBlockNumber: candidate.BlockNumber})
		}
	}
	return result, nil
}

type initialRecipientWallet struct {
	wallet common.Address
	amount *big.Int
}

func extractInitialRecipientWallets(logs []*types.Log, tokenContract common.Address) []initialRecipientWallet {
	if len(logs) == 0 || tokenContract == (common.Address{}) {
		return nil
	}
	filterer, err := erc20contract.NewERC20Filterer(tokenContract, nil)
	if err != nil {
		return nil
	}
	candidates := make(map[common.Address]struct{})
	netBalance := make(map[common.Address]*big.Int)
	for _, entry := range logs {
		if entry == nil || entry.Address != tokenContract {
			continue
		}
		transfer, err := filterer.ParseTransfer(*entry)
		if err != nil || transfer.Tokens == nil || transfer.Tokens.Sign() <= 0 {
			continue
		}
		if transfer.To != (common.Address{}) {
			candidates[transfer.To] = struct{}{}
			if netBalance[transfer.To] == nil {
				netBalance[transfer.To] = new(big.Int)
			}
			netBalance[transfer.To].Add(netBalance[transfer.To], transfer.Tokens)
		}
		if transfer.From != (common.Address{}) {
			if netBalance[transfer.From] == nil {
				netBalance[transfer.From] = new(big.Int)
			}
			netBalance[transfer.From].Sub(netBalance[transfer.From], transfer.Tokens)
		}
	}
	result := make([]initialRecipientWallet, 0, len(candidates))
	for wallet := range candidates {
		amount := netBalance[wallet]
		if amount != nil && amount.Sign() > 0 {
			result = append(result, initialRecipientWallet{wallet: wallet, amount: cloneBigInt(amount)})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if comparison := result[i].amount.Cmp(result[j].amount); comparison != 0 {
			return comparison > 0
		}
		return bytes.Compare(result[i].wallet.Bytes(), result[j].wallet.Bytes()) < 0
	})
	return result
}

func filterInitialRecipientWallets(ctx context.Context, client interface {
	CodeAt(context.Context, common.Address, *big.Int) ([]byte, error)
}, recipients []initialRecipientWallet, limit int) ([]initialRecipientWallet, error) {
	result := make([]initialRecipientWallet, 0, len(recipients))
	for _, recipient := range recipients {
		code, err := client.CodeAt(ctx, recipient.wallet, nil)
		if err != nil {
			return nil, err
		}
		if len(code) == 0 {
			result = append(result, recipient)
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func cloneBigInt(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}
