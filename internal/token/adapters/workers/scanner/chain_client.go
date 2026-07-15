package scanner

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
)

func (r *chainRunner) ensureClient(ctx context.Context) (*ethclient.Client, error) {
	return r.opts.clients.Client(ctx, r.opts.chainID)
}

func (r *chainRunner) resetClient() {
	r.opts.clients.Reset(r.opts.chainID)
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
