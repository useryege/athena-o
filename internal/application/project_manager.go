package application

import "github.com/ethereum/go-ethereum/ethclient"

type ProjectManager struct {
	nodeClient *ethclient.Client
}

func NewProjectManager(nodeClient *ethclient.Client) *ProjectManager {
	return &ProjectManager{nodeClient: nodeClient}
}

func (p *ProjectManager) Start() error {
	return nil
}

func (p *ProjectManager) Stop() error {
	return nil
}
