package projectqualifier

import (
	"context"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type tokenValidation struct {
	IsValidERC20 bool
	WethPair     common.Address
	UsdtPair     common.Address
}

type athenaValidator struct {
	caller *athenacontract.ATHENACaller
}

func newAthenaValidator(athenaContract common.Address, client *ethclient.Client) (*athenaValidator, error) {
	caller, err := athenacontract.NewATHENACaller(athenaContract, client)
	if err != nil {
		return nil, err
	}
	return &athenaValidator{caller: caller}, nil
}

func (v *athenaValidator) validateERC20(ctx context.Context, contracts []common.Address) ([]tokenValidation, error) {
	items, err := v.caller.ValidateERC20(&bind.CallOpts{Context: ctx}, contracts)
	if err != nil {
		return nil, err
	}
	results := make([]tokenValidation, 0, len(items))
	for _, item := range items {
		results = append(results, tokenValidation{
			IsValidERC20: item.IsValidERC20,
			WethPair:     item.WethPair,
			UsdtPair:     item.UsdtPair,
		})
	}
	return results, nil
}
