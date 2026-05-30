package discovery

import (
	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
)

func projectMetaFromStore(meta appstore.ProjectMeta) ProjectMeta {
	return meta
}

func uniqueProjectPairAddresses(items []common.Address) []common.Address {
	seen := make(map[common.Address]struct{}, len(items))
	result := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item == (common.Address{}) {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func projectPairAddresses(project *Project) []common.Address {
	if project == nil {
		return nil
	}
	pairs := make([]common.Address, 0, 2)
	if project.Meta.WethPair != (common.Address{}) {
		pairs = append(pairs, project.Meta.WethPair)
	}
	if project.Meta.UsdtPair != (common.Address{}) {
		pairs = append(pairs, project.Meta.UsdtPair)
	}
	return pairs
}
