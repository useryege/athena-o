package validator

import (
	"context"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	athenacommon "github.com/useryege/athena/common"
)

func (r *validatorRunner) ensureClient(ctx context.Context, chainID int64) (*ethclient.Client, error) {
	return r.opts.clients.Client(ctx, chainID)
}

func (r *validatorRunner) ensureValidator(ctx context.Context, chainID int64) (*athenaValidator, error) {
	r.clientMu.Lock()
	if validator := r.validators[chainID]; validator != nil {
		r.clientMu.Unlock()
		return validator, nil
	}
	r.clientMu.Unlock()

	client, err := r.ensureClient(ctx, chainID)
	if err != nil {
		return nil, err
	}
	athenaContract := r.opts.athenaContracts[chainID]
	if athenaContract == (ethcommon.Address{}) {
		return nil, errAthenaContractRequired(chainID)
	}
	validator, err := newAthenaValidator(athenaContract, client)
	if err != nil {
		return nil, err
	}
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if current := r.validators[chainID]; current != nil {
		return current, nil
	}
	r.validators[chainID] = validator
	log.WithFields(log.Fields{
		"chain_id":        chainID,
		"chain_name":      athenacommon.ChainName(chainID),
		"athena_contract": athenaContract.Hex(),
	}).Info("token project validator initialized ATHENA validator")
	return validator, nil
}

func (r *validatorRunner) resetChain(chainID int64) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	r.opts.clients.Reset(chainID)
	delete(r.validators, chainID)
}

func (r *validatorRunner) close() {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	for chainID := range r.validators {
		delete(r.validators, chainID)
	}
}
