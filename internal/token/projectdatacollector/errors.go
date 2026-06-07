package projectdatacollector

import "fmt"

func errStoreRequired() error {
	return fmt.Errorf("token project data collector store is required")
}

func errAveAPIKeyRequired() error {
	return fmt.Errorf("token project data collector ave api key is required")
}

func errEtherscanAPIKeyRequired() error {
	return fmt.Errorf("token project data collector etherscan api key is required")
}

func errNodeWSURLRequired(chainID int64) error {
	return fmt.Errorf("token project data collector node websocket url is required for chain %d", chainID)
}

func errAthenaContractRequired(chainID int64) error {
	return fmt.Errorf("token project data collector ATHENA contract is required for chain %d", chainID)
}

func errAthenaContractInvalid(chainID int64, value string) error {
	return fmt.Errorf("token project data collector ATHENA contract %q is invalid for chain %d", value, chainID)
}
