package tokenapi

import (
	"context"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

func (s *Service) GetOptions(ctx context.Context, _ *apiclient.GetOptionsRequest) (*apiclient.GetOptionsResponse, error) {
	store, err := requiredStore(s.tokenStore())
	if err != nil {
		return nil, err
	}
	chains, err := store.ListChains(ctx)
	if err != nil {
		return nil, wrapStoreError("get token api options", err)
	}
	return &apiclient.GetOptionsResponse{
		Options: &v1alpha1.TokenAPIOptions{
			Chains: mapChainOptions(chains),
		},
	}, nil
}
