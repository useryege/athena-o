package migration

import (
	"fmt"
	"io/fs"

	accountstatestore "github.com/useryege/athena/internal/accountstate/store"
	managedoostore "github.com/useryege/athena/internal/managedoo/store"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	profitsharingstore "github.com/useryege/athena/internal/profitsharing/store"
	sportshistorystore "github.com/useryege/athena/internal/sportshistory/store"
	sportslivestore "github.com/useryege/athena/internal/sportslive/store"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	wormmarketsstore "github.com/useryege/athena/internal/wormmarkets/store"
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
}

var modules = []Module{
	{Name: "account-state", DSNEnv: "ATHENA_SERVER_POSTGRES_DSN", Database: "athena", Migrations: accountstatestore.Migrations()},
	{Name: "worm-markets", DSNEnv: "ATHENA_WORM_MARKETS_POSTGRES_DSN", Database: "worm_markets", Migrations: wormmarketsstore.Migrations()},
	{Name: "notification", DSNEnv: "ATHENA_NOTIFICATION_POSTGRES_DSN", Database: "notification", Migrations: notificationstore.Migrations()},
	{Name: "wallet", DSNEnv: "ATHENA_WALLET_POSTGRES_DSN", Database: "wallet", Migrations: walletstore.Migrations()},
	{Name: "sports-live", DSNEnv: "ATHENA_SPORTS_LIVE_POSTGRES_DSN", Database: "sports_live", Migrations: sportslivestore.Migrations()},
	{Name: "sports-history", DSNEnv: "ATHENA_SPORTS_HISTORY_POSTGRES_DSN", Database: "sports_history", Migrations: sportshistorystore.Migrations()},
	{Name: "managed-oo", DSNEnv: "ATHENA_MANAGED_OO_POSTGRES_DSN", Database: "managed_oo", Migrations: managedoostore.Migrations()},
	{Name: "profit-sharing", DSNEnv: "ATHENA_PROFIT_SHARING_POSTGRES_DSN", Database: "profit_sharing", Migrations: profitsharingstore.Migrations()},
	{Name: "token", DSNEnv: "ATHENA_TOKEN_POSTGRES_DSN", Database: "token", Migrations: tokenpostgres.Migrations()},
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
