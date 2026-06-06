package runtime

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func loadAthenaContractOptions(ctx context.Context, nodeClient *ethclient.Client, athenaContract string) (ethcommon.Address, ethcommon.Address, ethcommon.Address, ethcommon.Address, uint8, uint8, error) {
	athenaContractAddress, err := parseRequiredAddress("ATHENA contract address", athenaContract, "--athena-contract", "ATHENA_APPLICATION_ATHENA_CONTRACT")
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	athenaClient, err := athenacontract.NewATHENA(athenaContractAddress, nodeClient)
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	wethContractAddress, err := athenaClient.WethContract(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	usdtContractAddress, err := athenaClient.UsdtContract(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	wethDecimals, err := athenaClient.WethDecimals(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	usdtDecimals, err := athenaClient.UsdtDecimals(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	v2FactoryContractAddress, err := athenaClient.FactoryContract(&bind.CallOpts{Context: ctx})
	if err != nil {
		return ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, ethcommon.Address{}, 0, 0, err
	}
	return athenaContractAddress, v2FactoryContractAddress, wethContractAddress, usdtContractAddress, wethDecimals, usdtDecimals, nil
}

func parseRequiredAddress(name string, value string, flag string, envVar string) (ethcommon.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ethcommon.Address{}, fmt.Errorf("%s is required.Set it by %s flag or %s environment variable", name, flag, envVar)
	}
	if !ethcommon.IsHexAddress(value) {
		return ethcommon.Address{}, fmt.Errorf("invalid %s %q", name, value)
	}

	address := ethcommon.HexToAddress(value)
	if address == (ethcommon.Address{}) {
		return ethcommon.Address{}, fmt.Errorf("%s cannot be zero address", name)
	}
	return address, nil
}

func parseLiquidityLockerAddresses(values []string) ([]ethcommon.Address, error) {
	if len(values) == 0 {
		return nil, nil
	}

	addresses := make([]ethcommon.Address, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("liquidity locker address cannot be empty")
		}
		if !ethcommon.IsHexAddress(value) {
			return nil, fmt.Errorf("invalid liquidity locker address %q", value)
		}
		addresses = append(addresses, ethcommon.HexToAddress(value))
	}
	return addresses, nil
}
