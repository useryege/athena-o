package chainingestor

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/util/ethws"
)

func (r *chainRunner) ensureClient(ctx context.Context) (*ethclient.Client, error) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if r.client != nil {
		return r.client, nil
	}
	client, err := ethws.DialContext(ctx, r.opts.nodeWSURL, r.opts.nodeWSUseProxy)
	if err != nil {
		return nil, err
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return nil, err
	}
	if chainID == nil || chainID.Int64() != r.opts.chainID {
		client.Close()
		return nil, fmt.Errorf("token chain ingestor node returned chain_id %v for configured chain_id %d", chainID, r.opts.chainID)
	}
	r.client = client
	log.WithFields(log.Fields{
		"chain_id":   r.opts.chainID,
		"chain_name": r.opts.chainName,
	}).Info("token chain ingestor connected to node websocket")
	return client, nil
}

func (r *chainRunner) resetClient() {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if r.client != nil {
		r.client.Close()
		r.client = nil
	}
}

func (r *chainRunner) close() {
	r.closeOnce.Do(func() {
		if r.blockFetchCancel != nil {
			r.blockFetchCancel()
		}
		r.blockFetchWG.Wait()
		r.resetClient()
	})
}
