package projectdatacollector

import (
	"context"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	athenacommon "github.com/useryege/athena/common"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ethws"
	utilio "github.com/useryege/athena/util/io"
)

func (r *dataCollectorRunner) ensureClient(ctx context.Context, chainID int64) (*ethclient.Client, error) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if client := r.clients[chainID]; client != nil {
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
	r.clients[chainID] = client
	log.WithFields(log.Fields{
		"chain_id":    chainID,
		"chain_name":  athenacommon.ChainName(chainID),
		"node_ws_url": ethws.RedactEndpoint(endpoint),
	}).Info("token project data collector connected to node websocket")
	return client, nil
}

func (r *dataCollectorRunner) ensureCaller(ctx context.Context, chainID int64) (*athenacontract.ATHENACaller, error) {
	r.clientMu.Lock()
	if caller := r.callers[chainID]; caller != nil {
		r.clientMu.Unlock()
		return caller, nil
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
	caller, err := athenacontract.NewATHENACaller(athenaContract, client)
	if err != nil {
		return nil, err
	}
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if current := r.callers[chainID]; current != nil {
		return current, nil
	}
	r.callers[chainID] = caller
	log.WithFields(log.Fields{
		"chain_id":        chainID,
		"chain_name":      athenacommon.ChainName(chainID),
		"athena_contract": athenaContract.Hex(),
	}).Info("token project data collector initialized ATHENA caller")
	return caller, nil
}

func (r *dataCollectorRunner) resetChain(chainID int64) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if client := r.clients[chainID]; client != nil {
		client.Close()
		delete(r.clients, chainID)
	}
	delete(r.callers, chainID)
}

func (r *dataCollectorRunner) close() {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if r.opts.ethereumAPIConn != nil {
		utilio.Close(r.opts.ethereumAPIConn)
		r.opts.ethereumAPIConn = nil
	}
	r.opts.ethereumAPI = nil
	for chainID, client := range r.clients {
		if client != nil {
			client.Close()
		}
		delete(r.clients, chainID)
	}
	for chainID := range r.callers {
		delete(r.callers, chainID)
	}
}
