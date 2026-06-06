package migration

import (
	"fmt"
	"io/fs"

	appstore "github.com/useryege/athena/internal/application/store"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	tokenstore "github.com/useryege/athena/internal/token/store"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	wormstore "github.com/useryege/athena/internal/worm/store"
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
	{Name: "application", DSNEnv: "ATHENA_APPLICATION_POSTGRES_DSN", Database: "application", Migrations: appstore.Migrations()},
	{Name: "worm", DSNEnv: "ATHENA_WORM_POSTGRES_DSN", Database: "worm", Migrations: wormstore.Migrations()},
	{Name: "notification", DSNEnv: "ATHENA_NOTIFICATION_POSTGRES_DSN", Database: "notification", Migrations: notificationstore.Migrations()},
	{Name: "wallet", DSNEnv: "ATHENA_WALLET_POSTGRES_DSN", Database: "wallet", Migrations: walletstore.Migrations()},
	{Name: "polymarket", DSNEnv: "ATHENA_POLYMARKET_POSTGRES_DSN", Database: "polymarket", Migrations: polymarketstore.Migrations()},
	{Name: "token", DSNEnv: "ATHENA_TOKEN_POSTGRES_DSN", Database: "token", Migrations: tokenstore.Migrations()},
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
