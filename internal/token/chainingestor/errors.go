package chainingestor

import "fmt"

func errStoreRequired() error {
	return fmt.Errorf("token store is required for chain-ingestor")
}

func errNodeWSURLRequired(chainID int64) error {
	return fmt.Errorf("node websocket URL is required for token chain-ingestor chain %d", chainID)
}
