package chainscanner

import (
	"context"

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
	client, endpoint, err := ethws.DialFastestContext(ctx, r.opts.nodeWSURLs, r.opts.chainID, r.opts.nodeWSUseProxy)
	if err != nil {
		return nil, err
	}
	r.client = client
	log.WithFields(log.Fields{
		"chain_id":    r.opts.chainID,
		"chain_name":  r.opts.chainName,
		"node_ws_url": ethws.RedactEndpoint(endpoint),
	}).Info("token chain scanner connected to node websocket")
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
