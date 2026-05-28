package model

type SimulateResult struct {
	CanMintFromDeadViaTransferFrom     bool
	CanMintFromZeroViaTransferFrom     bool
	CanMintFromWethPairViaTransferFrom bool
	CanMintFromUsdtPairViaTransferFrom bool
	CanMintViaTransferToWethPair       bool
	CanMintViaTransferToUsdtPair       bool
}

func (r SimulateResult) HasMintRisk() bool {
	return len(r.MintablePaths()) > 0
}

func (r SimulateResult) MintablePaths() []string {
	paths := make([]string, 0, 6)
	if r.CanMintFromDeadViaTransferFrom {
		paths = append(paths, "can_mint_from_dead_via_transfer_from")
	}
	if r.CanMintFromZeroViaTransferFrom {
		paths = append(paths, "can_mint_from_zero_via_transfer_from")
	}
	if r.CanMintFromWethPairViaTransferFrom {
		paths = append(paths, "can_mint_from_weth_pair_via_transfer_from")
	}
	if r.CanMintFromUsdtPairViaTransferFrom {
		paths = append(paths, "can_mint_from_usdt_pair_via_transfer_from")
	}
	if r.CanMintViaTransferToWethPair {
		paths = append(paths, "can_mint_via_transfer_to_weth_pair")
	}
	if r.CanMintViaTransferToUsdtPair {
		paths = append(paths, "can_mint_via_transfer_to_usdt_pair")
	}
	return paths
}
