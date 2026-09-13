package migration

import (
	"fmt"
	"io/fs"

	"github.com/useryege/athena/internal/accountstate/schema"
	accountstatemigrations "github.com/useryege/athena/internal/accountstate/store/migrations"
	managedoostore "github.com/useryege/athena/internal/managedoo/store"
	profitsharingstore "github.com/useryege/athena/internal/profitsharing/store"
	sportshistorystore "github.com/useryege/athena/internal/sportshistory/store"
	sportslivestore "github.com/useryege/athena/internal/sportslive/store"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	wormmarketsstore "github.com/useryege/athena/internal/wormmarkets/store"
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
	{Name: "worm-markets", DSNEnv: "ATHENA_WORM_MARKETS_POSTGRES_DSN", Database: "worm_markets", Migrations: wormmarketsstore.Migrations(), Dir: MigrationDir},
	{Name: "worm-trading", DSNEnv: "ATHENA_WORM_TRADING_POSTGRES_DSN", Database: "worm_trading", Migrations: wormtradingstore.Migrations(), Dir: MigrationDir},
	{Name: "wallet", DSNEnv: "ATHENA_WALLET_POSTGRES_DSN", Database: "wallet", Migrations: walletstore.Migrations(), Dir: MigrationDir},
	{Name: "sports-live", DSNEnv: "ATHENA_SPORTS_LIVE_POSTGRES_DSN", Database: "sports_live", Migrations: sportslivestore.Migrations(), Dir: MigrationDir},
	{Name: "sports-history", DSNEnv: "ATHENA_SPORTS_HISTORY_POSTGRES_DSN", Database: "sports_history", Migrations: sportshistorystore.Migrations(), Dir: MigrationDir},
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
