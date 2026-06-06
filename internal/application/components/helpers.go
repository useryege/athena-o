package components

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func ProjectFromCandidate(candidate model.DiscoveredProjectCandidate) appstore.ProjectRecord {
	return appstore.ProjectRecord{
		ChainID:     candidate.ChainID,
		BlockTime:   candidate.BlockTime,
		BlockNumber: candidate.BlockNumber,
		Contract:    candidate.Contract,
		Creator:     candidate.Creator,
		Tx:          candidate.Tx,
		TxHash:      candidate.TxHash,
		TxIndex:     candidate.TxIndex,
	}
}

func GenesisWalletAddresses(items []appstore.ProjectGenesisWallet) []common.Address {
	addresses := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item.Wallet != (common.Address{}) {
			addresses = append(addresses, item.Wallet)
		}
	}
	return addresses
}

func ProjectQuery(base appstore.ProjectRecord, genesisWallets []appstore.ProjectGenesisWallet) athenacontract.AthenaProjectQuery {
	return athenacontract.AthenaProjectQuery{
		TokenContract:  base.Contract,
		MsgCaller:      base.Creator,
		GenesisWallets: GenesisWalletAddresses(genesisWallets),
	}
}

func ChainStateFromSnapshot(chainID int64, contract common.Address, snapshot athenacontract.AthenaProject, fetchedAt time.Time) (appstore.ProjectChainState, error) {
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return appstore.ProjectChainState{}, err
	}
	return appstore.ProjectChainState{
		ChainID:         chainID,
		ProjectContract: contract,
		ChainState:      snapshot,
		RawChainState:   raw,
		WethPair:        snapshot.WethPair.ContractAddress,
		UsdtPair:        snapshot.UsdtPair.ContractAddress,
		TokenName:       snapshot.Token.Name,
		TokenSymbol:     snapshot.Token.Symbol,
		FetchedAt:       fetchedAt,
	}, nil
}

func BigIntOrZero(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}

func NowUTC() time.Time {
	return time.Now().UTC()
}
