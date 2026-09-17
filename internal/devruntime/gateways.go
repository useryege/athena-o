package devruntime

import (
	"context"
	"strings"
	"sync"
	"time"
)

// GatewayResult reports only remote Health. It is separate from Manager readiness
// and performs no quota, business, deployment or lifecycle operation.
type GatewayResult struct {
	Address string `json:"address"`
	Serving bool   `json:"serving"`
	Error   string `json:"error,omitempty"`
}

func ProbeGateways(ctx context.Context, env map[string]string) ([]GatewayResult, error) {
	config := map[string]string{"ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS": env["ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS"], "ETHERSCAN_GATEWAY_IPS": env["ETHERSCAN_GATEWAY_IPS"]}
	if err := normalizeGatewayConfiguration(config); err != nil {
		return nil, err
	}
	if config["ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS"] == "" {
		return nil, nil
	}
	return probeGatewayAddresses(ctx, strings.Split(config["ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS"], ",")), nil
}
func probeGatewayAddresses(ctx context.Context, addresses []string) []GatewayResult {
	results := make([]GatewayResult, len(addresses))
	var workers sync.WaitGroup
	for i, address := range addresses {
		workers.Add(1)
		go func() {
			defer workers.Done()
			deadline, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			err := probeService(deadline, "etherscan-manager", address, nil)
			results[i] = GatewayResult{Address: address, Serving: err == nil}
			if err != nil {
				results[i].Error = err.Error()
			}
		}()
	}
	workers.Wait()
	return results
}
