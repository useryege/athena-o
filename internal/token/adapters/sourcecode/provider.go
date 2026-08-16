package sourcecode

import (
	"context"
	"fmt"
	"strings"

	etherscanmanagerapiclient "github.com/useryege/athena/internal/etherscanmanager/apiclient"
	"github.com/useryege/athena/internal/token/shared"
)

type Provider struct {
	client    etherscanmanagerapiclient.EtherscanManagerServiceClient
	clientset etherscanmanagerapiclient.Clientset
}

func New(address string) (*Provider, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("Etherscan Manager address is required")
	}
	clientset, err := etherscanmanagerapiclient.NewEtherscanManagerClientset(address)
	if err != nil {
		return nil, err
	}
	return &Provider{client: clientset.EtherscanManager(), clientset: clientset}, nil
}

func (p *Provider) GetSourceCode(ctx context.Context, chainID int64, contract shared.Address) (string, error) {
	response, err := p.client.GetSourceCode(ctx, &etherscanmanagerapiclient.GetSourceCodeRequest{
		ChainId:         chainID,
		ContractAddress: contract.Hex(),
	})
	if err != nil {
		return "", fmt.Errorf("fetch etherscan-manager source code chain_id=%d contract=%s: %w", chainID, contract.Hex(), err)
	}
	if response == nil || len(response.GetItems()) == 0 {
		return "", fmt.Errorf("fetch etherscan-manager source code chain_id=%d contract=%s returned empty result", chainID, contract.Hex())
	}
	return strings.TrimSpace(response.GetItems()[0].GetSourceCode()), nil
}

func (p *Provider) Close() error {
	if p == nil || p.clientset == nil {
		return nil
	}
	err := p.clientset.Close()
	p.clientset = nil
	p.client = nil
	return err
}
