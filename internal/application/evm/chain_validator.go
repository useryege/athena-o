package evm

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type AthenaChainValidator struct {
	caller *athenacontract.ATHENACaller
}

func NewAthenaChainValidator(nodeClient bind.ContractBackend, athenaContractAddress common.Address) (*AthenaChainValidator, error) {
	caller, err := athenacontract.NewATHENACaller(athenaContractAddress, nodeClient)
	if err != nil {
		return nil, err
	}
	return &AthenaChainValidator{caller: caller}, nil
}

func (v *AthenaChainValidator) ValidateERC20(ctx context.Context, contracts []common.Address) ([]model.TokenValidation, error) {
	if len(contracts) == 0 {
		return nil, nil
	}
	if v == nil || v.caller == nil {
		return nil, errors.New("athena chain validator is nil")
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

func (v *AthenaChainValidator) ValidatePairs(ctx context.Context, pairs []common.Address) ([]model.PairValidation, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	if v == nil || v.caller == nil {
		return nil, errors.New("athena chain validator is nil")
	}
	items, err := v.caller.ValidatePairs(&bind.CallOpts{Context: ctx}, pairs)
	if err != nil {
		return nil, err
	}
	results := make([]model.PairValidation, 0, len(items))
	for _, item := range items {
		results = append(results, model.PairValidation{
			IsValidPancakePair: item.IsValidPancakePair,
			Token0:             item.Token0,
			Token1:             item.Token1,
		})
	}
	return results, nil
}
