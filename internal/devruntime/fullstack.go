package devruntime

import (
	"context"
	"errors"
)

// FullStackServices is the explicit development graph. Commented-out modules in
// the former Procfile are intentionally absent; infrastructure is in each spec.
func FullStackServices() []string {
	return []string{"wallet", "notification", "etherscan-manager", "api-server", "ui", "trader-sync", "solana-discovery", "market-radar", "managed-oo", "profit-sharing", "worm-trading"}
}
func fullStackSpecs() []ServiceSpec { specs, _ := ResolveServices(FullStackServices()); return specs }

type schemaOwner struct {
	Name, Database, DSNEnv, BuildPackage string
	Args                                 []string
}

func schemaOwners() []schemaOwner {
	owners := []schemaOwner{
		{"account", "athena", "ATHENA_ACCOUNT_STATE_POSTGRES_DSN", "./cmd/athena-account-state-migrate", nil},
		{"solana-discovery", "athena", "ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN", "", []string{"schema"}},
		{"wallet", "wallet", "ATHENA_WALLET_POSTGRES_DSN", "", []string{"schema"}},
		{"managed-oo", "managed_oo", "ATHENA_MANAGED_OO_POSTGRES_DSN", "", []string{"schema"}},
		{"profit-sharing", "profit_sharing", "ATHENA_PROFIT_SHARING_POSTGRES_DSN", "", []string{"schema"}},
		{"worm-trading", "worm_trading", "ATHENA_WORM_TRADING_POSTGRES_DSN", "./cmd/athena-worm-trading-migrate", nil},
	}
	registry := serviceRegistry()
	for i := range owners {
		if owners[i].BuildPackage == "" {
			owners[i].BuildPackage = registry[owners[i].Name].BuildPackage
		}
	}
	return owners
}
func selectedSchemas(specs []ServiceSpec) []schemaOwner {
	var out []schemaOwner
	for _, owner := range schemaOwners() {
		if needsSchema(specs, owner.Name) {
			out = append(out, owner)
		}
	}
	return out
}

func prepareFullStackEnvironment(env map[string]string) {
	if _, ok := env["ATHENA_ACCOUNT_AVATAR_MAX_BYTES"]; !ok {
		env["ATHENA_ACCOUNT_AVATAR_MAX_BYTES"] = "2097152"
	}
}

// RunFullStack uses the same resource owner and startup sequence as a selected
// independent service run. Only selected schemas are prepared.
func RunFullStack(ctx context.Context, o RunOptions) error {
	if o.DBMode != "" && o.DBMode != "managed" {
		return errors.New("full stack requires DB_MODE=managed; use run-services for external account-state databases")
	}
	if len(o.Services) != 0 {
		return errors.New("full stack has an explicit service graph; use run-services for a selection")
	}
	o.DBMode = "managed"
	o.Services = FullStackServices()
	return runResolved(ctx, o, fullStackSpecs(), true)
}
