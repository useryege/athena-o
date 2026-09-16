package migration

import (
	"fmt"
	"io/fs"

	"github.com/useryege/athena/internal/accountstate/schema"
	accountstatemigrations "github.com/useryege/athena/internal/accountstate/store/migrations"
	managedoostore "github.com/useryege/athena/internal/managedoo/store"
	profitsharingstore "github.com/useryege/athena/internal/profitsharing/store"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	wormtradingstore "github.com/useryege/athena/internal/wormtrading/store"
)

const (
	AllModules   = "all"
	MigrationDir = "migrations"
)

type Module struct {
	Name       string
	DSNEnv     string
	Database   string
	Migrations fs.FS
	Dir        string
}

var modules = []Module{
	{Name: "account-state", DSNEnv: schema.DSNEnv, Database: "athena", Migrations: accountstatemigrations.FS, Dir: accountstatemigrations.Dir},
	{Name: "worm-trading", DSNEnv: "ATHENA_WORM_TRADING_POSTGRES_DSN", Database: "worm_trading", Migrations: wormtradingstore.Migrations(), Dir: MigrationDir},
	{Name: "wallet", DSNEnv: "ATHENA_WALLET_POSTGRES_DSN", Database: "wallet", Migrations: walletstore.Migrations(), Dir: MigrationDir},
	{Name: "managed-oo", DSNEnv: "ATHENA_MANAGED_OO_POSTGRES_DSN", Database: "managed_oo", Migrations: managedoostore.Migrations(), Dir: MigrationDir},
	{Name: "profit-sharing", DSNEnv: "ATHENA_PROFIT_SHARING_POSTGRES_DSN", Database: "profit_sharing", Migrations: profitsharingstore.Migrations(), Dir: MigrationDir},
	{Name: "token", DSNEnv: "ATHENA_TOKEN_POSTGRES_DSN", Database: "token", Migrations: tokenpostgres.Migrations(), Dir: MigrationDir},
}

func Modules() []Module {
	result := make([]Module, len(modules))
	copy(result, modules)
	return result
}

func Select(module string) ([]Module, error) {
	if module == AllModules {
		return Modules(), nil
	}
	for _, candidate := range modules {
		if candidate.Name == module {
			return []Module{candidate}, nil
		}
	}
	return nil, fmt.Errorf("unknown module %q", module)
}
