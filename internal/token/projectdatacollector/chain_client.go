package projectdatacollector

import (
	"context"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	athenacommon "github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token/domain"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ethws"
	utilio "github.com/useryege/athena/util/io"
)

func (r *dataCollectorRunner) ensureClient(ctx context.Context, dataType domain.DataCollectionType, chainID int64) (*ethclient.Client, error) {
	key := chainResourceKey{dataType: dataType, chainID: chainID}
	resource := r.chainResource(key)
	resource.mu.Lock()
	defer resource.mu.Unlock()
	return r.ensureClientLocked(ctx, key, resource)
}

func (r *dataCollectorRunner) ensureClientLocked(ctx context.Context, key chainResourceKey, resource *chainResource) (*ethclient.Client, error) {
	if resource.client != nil {
		return resource.client, nil
	}
	chainID := key.chainID
	nodeWSURLs := r.opts.nodeWSURLs[chainID]
	if len(nodeWSURLs) == 0 {
		return nil, errNodeWSURLRequired(chainID)
	}
	client, endpoint, err := ethws.DialFastestContext(ctx, nodeWSURLs, chainID, r.opts.nodeWSUseProxy)
	if err != nil {
		return nil, err
	}
	resource.client = client
	log.WithFields(log.Fields{
		"chain_id":    chainID,
		"chain_name":  athenacommon.ChainName(chainID),
		"node_ws_url": ethws.RedactEndpoint(endpoint),
		"data_type":   key.dataType,
	}).Info("token project data collector connected to node websocket")
	return client, nil
}

func (r *dataCollectorRunner) ensureCaller(ctx context.Context, dataType domain.DataCollectionType, chainID int64) (*athenacontract.ATHENACaller, error) {
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

func (r *dataCollectorRunner) resetChain(dataType domain.DataCollectionType, chainID int64) {
	key := chainResourceKey{dataType: dataType, chainID: chainID}
	resource := r.existingChainResource(key)
	if resource == nil {
		return
	}
	resource.mu.Lock()
	defer resource.mu.Unlock()
	if resource.client != nil {
		resource.client.Close()
		resource.client = nil
	}
	resource.caller = nil
}

func (r *dataCollectorRunner) close() {
	if r.opts.ethereumAPIConn != nil {
		utilio.Close(r.opts.ethereumAPIConn)
		r.opts.ethereumAPIConn = nil
	}
	r.opts.ethereumAPI = nil
	r.resourceMu.Lock()
	resources := r.resources
	r.resources = make(map[chainResourceKey]*chainResource)
	r.resourceMu.Unlock()
	for _, resource := range resources {
		resource.mu.Lock()
		if resource.client != nil {
			resource.client.Close()
			resource.client = nil
		}
		resource.caller = nil
		resource.mu.Unlock()
	}
}
