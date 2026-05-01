package application

import (
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	watcher        *Watcher
	projectManager *ProjectManager
}

func NewService(nodeClient *ethclient.Client) *Service {
	return &Service{
		watcher:        NewWatcher(nodeClient),
		projectManager: NewProjectManager(nodeClient),
	}
}

func (s *Service) Start() error {
	if err := s.watcher.Start(); err != nil {
		return err
	}
	if err := s.projectManager.Start(); err != nil {
		return err
	}
	return nil
}

func (s *Service) Stop() error {
	if err := s.watcher.Stop(); err != nil {
		return err
	}
	if err := s.projectManager.Stop(); err != nil {
		return err
	}
	return nil
}
