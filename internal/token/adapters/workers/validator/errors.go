package validator

import "fmt"

func errStoreRequired() error {
	return fmt.Errorf("token store is required for project validator")
}

func errNodeWSURLRequired(chainID int64) error {
	return fmt.Errorf("node websocket URL is required for token project validator chain %d", chainID)
}

func errAthenaContractRequired(chainID int64) error {
	return fmt.Errorf("ATHENA contract address is required for token project validator chain %d", chainID)
}

func errAthenaContractInvalid(chainID int64, value string) error {
	return fmt.Errorf("invalid ATHENA contract address %q for token project validator chain %d", value, chainID)
}
