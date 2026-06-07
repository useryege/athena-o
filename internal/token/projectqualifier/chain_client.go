package projectqualifier

import (
	"context"
	"fmt"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	athenacommon "github.com/useryege/athena/common"
	"github.com/useryege/athena/util/ethws"
)

func (r *qualifierRunner) ensureClient(ctx context.Context, chainID int64) (*ethclient.Client, error) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if client := r.clients[chainID]; client != nil {
		return client, nil
	}
	nodeWSURL := r.opts.nodeWSURLs[chainID]
	if nodeWSURL == "" {
		return nil, errNodeWSURLRequired(chainID)
	}
	client, err := ethws.DialContext(ctx, nodeWSURL, r.opts.nodeWSUseProxy)
	if err != nil {
		return nil, err
	}
	nodeChainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return nil, err
	}
	if nodeChainID == nil || nodeChainID.Int64() != chainID {
		client.Close()
		return nil, fmt.Errorf("token project qualifier node returned chain_id %v for configured chain_id %d", nodeChainID, chainID)
	}
	r.clients[chainID] = client
	log.WithFields(log.Fields{
		"chain_id":   chainID,
		"chain_name": athenacommon.ChainName(chainID),
	}).Info("token project qualifier connected to node websocket")
	return client, nil
}

func (r *qualifierRunner) ensureValidator(ctx context.Context, chainID int64) (*athenaValidator, error) {
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
	}).Info("token project qualifier initialized ATHENA validator")
	return validator, nil
}

func (r *qualifierRunner) resetChain(chainID int64) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if client := r.clients[chainID]; client != nil {
		client.Close()
		delete(r.clients, chainID)
	}
	delete(r.validators, chainID)
}

func (r *qualifierRunner) close() {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	for chainID, client := range r.clients {
		if client != nil {
			client.Close()
		}
		delete(r.clients, chainID)
	}
	for chainID := range r.validators {
		delete(r.validators, chainID)
	}
}
