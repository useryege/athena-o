package evm

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type AthenaTokenValidator struct {
	caller *athenacontract.ATHENACaller
}

func NewAthenaTokenValidator(nodeClient bind.ContractBackend, athenaContractAddress common.Address) (*AthenaTokenValidator, error) {
	caller, err := athenacontract.NewATHENACaller(athenaContractAddress, nodeClient)
	if err != nil {
		return nil, err
	}
	return &AthenaTokenValidator{caller: caller}, nil
}

func (v *AthenaTokenValidator) ValidateERC20(ctx context.Context, contracts []common.Address) ([]model.TokenValidation, error) {
	if len(contracts) == 0 {
		return nil, nil
	}
	if v == nil || v.caller == nil {
		return nil, errors.New("athena token validator is nil")
	}
	items, err := v.caller.ValidateERC20(&bind.CallOpts{Context: ctx}, contracts)
	if err != nil {
		return nil, err
	}
	results := make([]model.TokenValidation, 0, len(items))
	for _, item := range items {
		results = append(results, model.TokenValidation{
			IsValidERC20: item.IsValidERC20,
			WethPair:     item.WethPair,
			UsdtPair:     item.UsdtPair,
		})
	}
	return results, nil
}
