package scanner

import "fmt"

func errStoreRequired() error {
	return fmt.Errorf("token store is required for chain scanner")
}

func errNodeWSURLRequired(chainID int64) error {
	return fmt.Errorf("node websocket URL is required for token chain scanner chain %d", chainID)
}
