package cache

import (
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

const (
	DefaultProjectCacheTTLSeconds = 24 * 60 * 60
)

type Keyspace struct {
	prefix string
}

func NewKeyspace(prefix string) Keyspace {
	if prefix == "" {
		prefix = "application"
	}
	return Keyspace{prefix: prefix}
}

func (k Keyspace) Project(chainID int64, contract common.Address) string {
	return k.join("project", chainIDPart(chainID), contract.Hex())
}

func (k Keyspace) ProjectChainState(chainID int64, contract common.Address) string {
	return k.join("project", chainIDPart(chainID), contract.Hex(), "chain_state")
}

func (k Keyspace) ProjectSimulation(chainID int64, contract common.Address) string {
	return k.join("project", chainIDPart(chainID), contract.Hex(), "simulation")
}

func (k Keyspace) ProjectGenesisWallets(chainID int64, contract common.Address) string {
	return k.join("project", chainIDPart(chainID), contract.Hex(), "genesis_wallets")
}

func (k Keyspace) ProjectCreatorHistory(chainID int64, contract common.Address) string {
	return k.join("project", chainIDPart(chainID), contract.Hex(), "creator_history")
}

func (k Keyspace) ProjectComponentState(chainID int64, contract common.Address, component string) string {
	return k.join("project", chainIDPart(chainID), contract.Hex(), "component_state", component)
}

func (k Keyspace) ProjectIndex(chainID int64) string {
	return k.join("index", "project", chainIDPart(chainID))
}

func (k Keyspace) ProjectIndexCreator(chainID int64, creator common.Address) string {
	return k.join("index", "project", chainIDPart(chainID), "creator", creator.Hex())
}

func (k Keyspace) ProjectIndexPair(chainID int64, pair common.Address) string {
	return k.join("index", "project", chainIDPart(chainID), "pair", pair.Hex())
}

func (k Keyspace) ProjectIndexComponentNextRun(chainID int64, component string) string {
	return k.join("index", "component", chainIDPart(chainID), "next_run", component)
}

func (k Keyspace) join(parts ...string) string {
	if len(parts) == 0 {
		return k.prefix
	}
	key := k.prefix
	for _, part := range parts {
		key = fmt.Sprintf("%s:%s", key, part)
	}
	return key
}

func chainIDPart(chainID int64) string {
	return strconv.FormatInt(chainID, 10)
}
