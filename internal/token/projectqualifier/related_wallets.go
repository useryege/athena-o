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
const basisPointsDenominator int64 = 10000

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
	return recipients
}

func filterInitialRecipientWallets(ctx context.Context, client *ethclient.Client, recipients []initialRecipientWallet, limit int) ([]initialRecipientWallet, error) {
	if client == nil || len(recipients) == 0 || limit == 0 {
		return nil, nil
	}
	filtered := make([]initialRecipientWallet, 0, len(recipients))
	for _, recipient := range recipients {
		code, err := client.CodeAt(ctx, recipient.wallet, nil)
		if err != nil {
			return nil, err
		}
		if len(code) != 0 {
			continue
		}
		filtered = append(filtered, recipient)
	}
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[:limit]
	}
	return filtered, nil
}

func buildProjectRelatedWallets(candidate tokenstore.ProjectCandidate, initialRecipients []initialRecipientWallet) []tokenstore.ProjectRelatedWallet {
	wallets := make([]tokenstore.ProjectRelatedWallet, 0, 1+len(initialRecipients))
	if candidate.TxSender != (common.Address{}) {
		wallets = append(wallets, tokenstore.ProjectRelatedWallet{
			Wallet: candidate.TxSender,
			Role:   tokenstore.ProjectRelatedWalletRoleCreator,
		})
	}
	for _, recipient := range initialRecipients {
		wallet := recipient.wallet
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

func buildWalletAssetStates(candidate tokenstore.ProjectCandidate, initialRecipients []initialRecipientWallet) []tokenstore.WalletAssetState {
	wallets := make([]tokenstore.WalletAssetState, 0, 1+len(initialRecipients))
	seen := make(map[common.Address]struct{}, 1+len(initialRecipients))
	if candidate.TxSender != (common.Address{}) {
		seen[candidate.TxSender] = struct{}{}
		wallets = append(wallets, tokenstore.WalletAssetState{
			ChainID: candidate.ChainID,
			Wallet:  candidate.TxSender,
		})
	}
	for _, recipient := range initialRecipients {
		wallet := recipient.wallet
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

func buildProjectInitialRecipients(candidate tokenstore.ProjectCandidate, initialRecipients []initialRecipientWallet, totalSupply *big.Int) []tokenstore.ProjectInitialRecipient {
	if len(initialRecipients) == 0 || totalSupply == nil || totalSupply.Sign() <= 0 {
		return nil
	}
	items := make([]tokenstore.ProjectInitialRecipient, 0, len(initialRecipients))
	for i, recipient := range initialRecipients {
		if recipient.wallet == (common.Address{}) || recipient.amount == nil || recipient.amount.Sign() <= 0 {
			continue
		}
		ratio := new(big.Int).Mul(recipient.amount, big.NewInt(basisPointsDenominator))
		ratio.Div(ratio, totalSupply)
		items = append(items, tokenstore.ProjectInitialRecipient{
			Wallet:            recipient.wallet,
			RatioBPS:          ratio.Int64(),
			RankIndex:         int32(i),
			SourceTxHash:      candidate.TxHash,
			SourceBlockNumber: candidate.BlockNumber,
		})
	}
	return items
}

type initialRecipientWallet struct {
	wallet common.Address
	amount *big.Int
}
