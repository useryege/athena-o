package sourcecode

import (
	"context"
	"fmt"
	"strings"

	ethereumapiapiclient "github.com/useryege/athena/internal/ethereumapi/apiclient"
	"github.com/useryege/athena/internal/token/shared"
	utilio "github.com/useryege/athena/util/io"
)

type Provider struct {
	client ethereumapiapiclient.EthereumAPIServiceClient
	closer utilio.Closer
}

func New(address string) (*Provider, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("Ethereum API address is required")
	}
	closer, client, err := ethereumapiapiclient.NewEthereumAPIClientset(address).NewEthereumAPIServiceClient()
	if err != nil {
		return nil, err
	}
	return &Provider{client: client, closer: closer}, nil
}

func (p *Provider) GetSourceCode(ctx context.Context, chainID int64, contract shared.Address) (string, error) {
	response, err := p.client.GetSourceCode(ctx, &ethereumapiapiclient.GetSourceCodeRequest{
		ChainId:         chainID,
		ContractAddress: contract.Hex(),
	})
	if err != nil {
		return "", fmt.Errorf("fetch ethereum-api source code chain_id=%d contract=%s: %w", chainID, contract.Hex(), err)
	}
	if response == nil || len(response.GetItems()) == 0 {
		return "", fmt.Errorf("fetch ethereum-api source code chain_id=%d contract=%s returned empty result", chainID, contract.Hex())
	}
	return strings.TrimSpace(response.GetItems()[0].GetSourceCode()), nil
}

func (p *Provider) Close() error {
	if p == nil || p.closer == nil {
		return nil
	}
	err := p.closer.Close()
	p.closer = nil
	p.client = nil
	return err
}
