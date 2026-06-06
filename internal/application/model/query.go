package model

import (
	"github.com/ethereum/go-ethereum/common"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func BuildProjectQueries(projects []Project) ([]athenacontract.AthenaProjectQuery, []common.Address) {
	queries := make([]athenacontract.AthenaProjectQuery, 0, len(projects))
	contracts := make([]common.Address, 0, len(projects))
	for _, project := range projects {
		queries = append(queries, athenacontract.AthenaProjectQuery{
			TokenContract:  project.Contract,
			MsgCaller:      project.Creator,
			GenesisWallets: GenesisWalletAddresses(project.GenesisWallets),
		})
		contracts = append(contracts, project.Contract)
	}
	return queries, contracts
}

func GenesisWalletAddresses(items []GenesisWalletMeta) []common.Address {
	if len(items) == 0 {
		return nil
	}
	addresses := make([]common.Address, 0, len(items))
	for _, item := range items {
		addresses = append(addresses, item.Wallet)
	}
	return addresses
}
