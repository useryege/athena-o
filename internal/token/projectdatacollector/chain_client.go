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
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	key := chainResourceKey{dataType: dataType, chainID: chainID}
	if client := r.clients[key]; client != nil {
		return client, nil
	}
	nodeWSURLs := r.opts.nodeWSURLs[chainID]
	if len(nodeWSURLs) == 0 {
		return nil, errNodeWSURLRequired(chainID)
	}
	client, endpoint, err := ethws.DialFastestContext(ctx, nodeWSURLs, chainID, r.opts.nodeWSUseProxy)
	if err != nil {
		return nil, err
	}
	r.clients[key] = client
	log.WithFields(log.Fields{
		"chain_id":    chainID,
		"chain_name":  athenacommon.ChainName(chainID),
		"node_ws_url": ethws.RedactEndpoint(endpoint),
		"data_type":   dataType,
	}).Info("token project data collector connected to node websocket")
	return client, nil
}

func (r *dataCollectorRunner) ensureCaller(ctx context.Context, dataType domain.DataCollectionType, chainID int64) (*athenacontract.ATHENACaller, error) {
	key := chainResourceKey{dataType: dataType, chainID: chainID}
	r.clientMu.Lock()
	if caller := r.callers[key]; caller != nil {
		r.clientMu.Unlock()
		return caller, nil
	}
	r.clientMu.Unlock()

	client, err := r.ensureClient(ctx, dataType, chainID)
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
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if current := r.callers[key]; current != nil {
		return current, nil
	}
	r.callers[key] = caller
	log.WithFields(log.Fields{
		"chain_id":        chainID,
		"chain_name":      athenacommon.ChainName(chainID),
		"athena_contract": athenaContract.Hex(),
		"data_type":       dataType,
	}).Info("token project data collector initialized ATHENA caller")
	return caller, nil
}

func (r *dataCollectorRunner) resetChain(dataType domain.DataCollectionType, chainID int64) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	key := chainResourceKey{dataType: dataType, chainID: chainID}
	if client := r.clients[key]; client != nil {
		client.Close()
		delete(r.clients, key)
	}
	delete(r.callers, key)
}

func (r *dataCollectorRunner) close() {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if r.opts.ethereumAPIConn != nil {
		utilio.Close(r.opts.ethereumAPIConn)
		r.opts.ethereumAPIConn = nil
	}
	r.opts.ethereumAPI = nil
	for key, client := range r.clients {
		if client != nil {
			client.Close()
		}
		delete(r.clients, key)
	}
	for key := range r.callers {
		delete(r.callers, key)
	}
}
