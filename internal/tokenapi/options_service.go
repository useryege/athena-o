package tokenapi

import (
	"context"

	"github.com/useryege/athena/internal/tokenapi/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

func (s *Service) GetRuntimeConfiguration(ctx context.Context, _ *apiclient.GetRuntimeConfigurationRequest) (*apiclient.GetRuntimeConfigurationResponse, error) {
	store, err := s.operationsApplication()
	if err != nil {
		return nil, err
	}
	chains, err := store.ListChains(ctx)
	if err != nil {
		return nil, wrapStoreError("get token api options", err)
	}
	return &apiclient.GetRuntimeConfigurationResponse{
		Configuration: &v1alpha1.TokenRuntimeConfiguration{
			Chains: mapChainOptions(chains),
		},
	}, nil
}
