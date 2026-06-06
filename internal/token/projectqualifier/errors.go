package projectqualifier

import "fmt"

func errStoreRequired() error {
	return fmt.Errorf("token store is required for project-qualifier")
}

func errNodeWSURLRequired(chainID int64) error {
	return fmt.Errorf("node websocket URL is required for token project-qualifier chain %d", chainID)
}
