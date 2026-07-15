package collector

import (
	"context"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	athenacommon "github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token/research"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func (r *dataCollectorRunner) ensureClient(ctx context.Context, dataType research.DataCollectionType, chainID int64) (*ethclient.Client, error) {
	key := chainResourceKey{dataType: dataType, chainID: chainID}
	resource := r.chainResource(key)
	resource.mu.Lock()
	defer resource.mu.Unlock()
	return r.ensureClientLocked(ctx, key, resource)
}

func (r *dataCollectorRunner) ensureClientLocked(ctx context.Context, key chainResourceKey, resource *chainResource) (*ethclient.Client, error) {
	return r.opts.clients.Client(ctx, key.chainID)
}

func (r *dataCollectorRunner) ensureCaller(ctx context.Context, dataType research.DataCollectionType, chainID int64) (*athenacontract.ATHENACaller, error) {
	key := chainResourceKey{dataType: dataType, chainID: chainID}
	resource := r.chainResource(key)
	resource.mu.Lock()
	defer resource.mu.Unlock()
	if resource.caller != nil {
		return resource.caller, nil
	}
	client, err := r.ensureClientLocked(ctx, key, resource)
	if err != nil {
		return nil, err
	}
	athenaContract := r.opts.athenaContracts[chainID]
	if athenaContract == (ethcommon.Address{}) {
		return nil, errAthenaContractRequired(chainID)
	}
	caller, err := athenacontract.NewATHENACaller(athenaContract, client)
	if err != nil {
		return nil, err
	}
	resource.caller = caller
	log.WithFields(log.Fields{
		"chain_id":        chainID,
		"chain_name":      athenacommon.ChainName(chainID),
		"athena_contract": athenaContract.Hex(),
		"data_type":       dataType,
	}).Info("token project data collector initialized ATHENA caller")
	return caller, nil
}

func (r *dataCollectorRunner) resetChain(dataType research.DataCollectionType, chainID int64) {
	key := chainResourceKey{dataType: dataType, chainID: chainID}
	resource := r.existingChainResource(key)
	if resource == nil {
		return
	}
	resource.mu.Lock()
	defer resource.mu.Unlock()
	r.opts.clients.Reset(chainID)
	resource.caller = nil
}

func (r *dataCollectorRunner) close() {
	r.resourceMu.Lock()
	resources := r.resources
	r.resources = make(map[chainResourceKey]*chainResource)
	r.resourceMu.Unlock()
	for _, resource := range resources {
		resource.mu.Lock()
		resource.caller = nil
		resource.mu.Unlock()
	}
}
