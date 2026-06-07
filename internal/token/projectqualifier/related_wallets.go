package projectqualifier

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	tokenstore "github.com/useryege/athena/internal/token/store"
	erc20contract "github.com/useryege/athena/pkg/abi/ERC20"
)

const initialRecipientWalletLimit = 10

var errProjectCreationReceiptNil = errors.New("project creation transaction receipt is nil")

func fetchProjectCreationReceipt(ctx context.Context, client *ethclient.Client, candidate tokenstore.ProjectCandidate) (*types.Receipt, error) {
	receipt, err := client.TransactionReceipt(ctx, candidate.TxHash)
	if err != nil {
		return nil, err
	}
	if receipt == nil {
		return nil, errProjectCreationReceiptNil
	}
	return receipt, nil
}

func extractInitialRecipientWallets(logs []*types.Log, tokenContract common.Address, limit int) []common.Address {
	if len(logs) == 0 || tokenContract == (common.Address{}) || limit == 0 {
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
		transferEvent, err := filterer.ParseTransfer(*entry)
		if err != nil || transferEvent.Tokens == nil || transferEvent.Tokens.Sign() <= 0 {
			continue
		}
		if transferEvent.To != (common.Address{}) {
			candidates[transferEvent.To] = struct{}{}
			if _, exists := netBalance[transferEvent.To]; !exists {
				netBalance[transferEvent.To] = new(big.Int)
			}
			netBalance[transferEvent.To].Add(netBalance[transferEvent.To], transferEvent.Tokens)
		}
		if transferEvent.From != (common.Address{}) {
			if _, exists := netBalance[transferEvent.From]; !exists {
				netBalance[transferEvent.From] = new(big.Int)
			}
			netBalance[transferEvent.From].Sub(netBalance[transferEvent.From], transferEvent.Tokens)
		}
	}
	recipients := make([]initialRecipientWallet, 0, len(candidates))
	for wallet := range candidates {
		amount, exists := netBalance[wallet]
		if !exists || amount.Sign() <= 0 {
			continue
		}
		recipients = append(recipients, initialRecipientWallet{
			wallet: wallet,
			amount: new(big.Int).Set(amount),
		})
	}
	sort.Slice(recipients, func(i, j int) bool {
		amountCmp := recipients[i].amount.Cmp(recipients[j].amount)
		if amountCmp != 0 {
			return amountCmp > 0
		}
		return bytes.Compare(recipients[i].wallet.Bytes(), recipients[j].wallet.Bytes()) < 0
	})
	if limit > 0 && len(recipients) > limit {
		recipients = recipients[:limit]
	}
	wallets := make([]common.Address, 0, len(recipients))
	for _, recipient := range recipients {
		wallets = append(wallets, recipient.wallet)
	}
	return wallets
}

func buildProjectRelatedWallets(candidate tokenstore.ProjectCandidate, initialRecipients []common.Address) []tokenstore.ProjectRelatedWallet {
	wallets := make([]tokenstore.ProjectRelatedWallet, 0, 1+len(initialRecipients))
	if candidate.Creator != (common.Address{}) {
		wallets = append(wallets, tokenstore.ProjectRelatedWallet{
			Wallet: candidate.Creator,
			Role:   tokenstore.ProjectRelatedWalletRoleCreator,
		})
	}
	for _, wallet := range initialRecipients {
		if wallet == (common.Address{}) {
			continue
		}
		wallets = append(wallets, tokenstore.ProjectRelatedWallet{
			Wallet: wallet,
			Role:   tokenstore.ProjectRelatedWalletRoleInitialRecipient,
		})
	}
	return wallets
}

func buildWalletAssetStates(candidate tokenstore.ProjectCandidate, initialRecipients []common.Address) []tokenstore.WalletAssetState {
	wallets := make([]tokenstore.WalletAssetState, 0, 1+len(initialRecipients))
	seen := make(map[common.Address]struct{}, 1+len(initialRecipients))
	if candidate.Creator != (common.Address{}) {
		seen[candidate.Creator] = struct{}{}
		wallets = append(wallets, tokenstore.WalletAssetState{
			ChainID: candidate.ChainID,
			Wallet:  candidate.Creator,
		})
	}
	for _, wallet := range initialRecipients {
		if wallet == (common.Address{}) {
			continue
		}
		if _, exists := seen[wallet]; exists {
			continue
		}
		seen[wallet] = struct{}{}
		wallets = append(wallets, tokenstore.WalletAssetState{
			ChainID: candidate.ChainID,
			Wallet:  wallet,
		})
	}
	return wallets
}

type initialRecipientWallet struct {
	wallet common.Address
	amount *big.Int
}
