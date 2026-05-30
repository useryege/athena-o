package redisrepo

import (
	"fmt"

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

func (k Keyspace) ProjectBase(contract common.Address) string {
	return k.join("project", contract.Hex(), "base")
}

func (k Keyspace) ProjectChainState(contract common.Address) string {
	return k.join("project", contract.Hex(), "chain_state")
}

func (k Keyspace) ProjectSimulation(contract common.Address) string {
	return k.join("project", contract.Hex(), "simulation")
}

func (k Keyspace) ProjectReport(contract common.Address) string {
	return k.join("project", contract.Hex(), "report")
}

func (k Keyspace) ProjectBytecodeFact(contract common.Address) string {
	return k.join("project", contract.Hex(), "bytecode_fact")
}

func (k Keyspace) ProjectAveDetail(contract common.Address) string {
	return k.join("project", contract.Hex(), "ave_detail")
}

func (k Keyspace) ProjectGenesisWallets(contract common.Address) string {
	return k.join("project", contract.Hex(), "genesis_wallets")
}

func (k Keyspace) ProjectCreatorHistory(contract common.Address) string {
	return k.join("project", contract.Hex(), "creator_history")
}

func (k Keyspace) ProjectComponentState(contract common.Address, component string) string {
	return k.join("project", contract.Hex(), "component_state", component)
}

func (k Keyspace) ProjectEventLogItem(contract common.Address, member string) string {
	return k.join("project", contract.Hex(), "event_log", member)
}

func (k Keyspace) ProjectIndexBase() string {
	return k.join("index", "project", "base")
}

func (k Keyspace) ProjectIndexCreator(creator common.Address) string {
	return k.join("index", "project", "creator", creator.Hex())
}

func (k Keyspace) ProjectIndexPair(pair common.Address) string {
	return k.join("index", "project", "pair", pair.Hex())
}

func (k Keyspace) ProjectIndexComponentNextRun(component string) string {
	return k.join("index", "component", "next_run", component)
}

func (k Keyspace) ProjectComponentStream() string {
	return k.join("stream", "component")
}

func (k Keyspace) BufferItem(kind string, member string) string {
	return k.join("buffer", kind, member)
}

func (k Keyspace) BufferDirty(kind string) string {
	return k.join("dirty", kind)
}

func (k Keyspace) BufferIndexBase() string {
	return k.join("index", "buffer", "base")
}

func (k Keyspace) BufferIndexCreator(creator common.Address) string {
	return k.join("index", "buffer", "creator", creator.Hex())
}

func (k Keyspace) BufferIndexPair(pair common.Address) string {
	return k.join("index", "buffer", "pair", pair.Hex())
}

func (k Keyspace) BufferIndexComponentNextRun(component string) string {
	return k.join("index", "buffer", "component_next_run", component)
}

func (k Keyspace) BufferIndexEventLog(contract common.Address) string {
	return k.join("index", "buffer", "event_log", contract.Hex())
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
