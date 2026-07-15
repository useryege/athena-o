package postgres

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	postgresutil "github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS { return migrations }

type Connection struct{ pool *pgxpool.Pool }

func NewConnection(pool *pgxpool.Pool) *Connection { return &Connection{pool: pool} }

func NewConnectionSource() func(context.Context) (*Connection, error) {
	return func(ctx context.Context) (*Connection, error) {
		pool, err := postgresutil.ConnectAndMigrate(ctx, postgresutil.Options{Module: "token", DSNEnv: "ATHENA_TOKEN_POSTGRES_DSN", Database: "token", Migrations: migrations, MigrationDir: "migrations"})
		if err != nil {
			return nil, fmt.Errorf("connect token postgres: %w", err)
		}
		log.Info("token postgres migrations are up to date")
		return NewConnection(pool), nil
	}
}

func (connection *Connection) Close() error {
	if connection != nil && connection.pool != nil {
		connection.pool.Close()
	}
	return nil
}

func (connection *Connection) Ping(ctx context.Context) error {
	if connection == nil || connection.pool == nil {
		return fmt.Errorf("token postgres connection is not configured")
	}
	return connection.pool.Ping(ctx)
}

type baseRepository struct {
	pool    *pgxpool.Pool
	queries tokensqlc.Querier
}

func newBaseRepository(connection *Connection) *baseRepository {
	if connection == nil || connection.pool == nil {
		return &baseRepository{}
	}
	return &baseRepository{pool: connection.pool, queries: tokensqlc.New(connection.pool)}
}

func (repository *baseRepository) querier() (tokensqlc.Querier, error) {
	if repository == nil || repository.queries == nil {
		return nil, fmt.Errorf("token postgres repository is not configured")
	}
	return repository.queries, nil
}

type ChainRepository struct{ *baseRepository }
type CandidateRepository struct{ *baseRepository }
type SchedulerRepository struct{ *baseRepository }
type CollectionRepository struct{ *baseRepository }
type CatalogRepository struct{ *baseRepository }
type ReportingRepository struct{ *baseRepository }
type ResearchReadRepository struct{ *baseRepository }
type SelectionRepository struct{ *baseRepository }
type PolicyRepository struct{ *baseRepository }
type DiagnosticsRepository struct{ *baseRepository }

func NewChainRepository(connection *Connection) *ChainRepository {
	return &ChainRepository{newBaseRepository(connection)}
}
func NewCandidateRepository(connection *Connection) *CandidateRepository {
	return &CandidateRepository{newBaseRepository(connection)}
}
func NewSchedulerRepository(connection *Connection) *SchedulerRepository {
	return &SchedulerRepository{newBaseRepository(connection)}
}
func NewCollectionRepository(connection *Connection) *CollectionRepository {
	return &CollectionRepository{newBaseRepository(connection)}
}
func NewCatalogRepository(connection *Connection) *CatalogRepository {
	return &CatalogRepository{newBaseRepository(connection)}
}
func NewReportingRepository(connection *Connection) *ReportingRepository {
	return &ReportingRepository{newBaseRepository(connection)}
}
func NewResearchReadRepository(connection *Connection) *ResearchReadRepository {
	return &ResearchReadRepository{newBaseRepository(connection)}
}
func NewSelectionRepository(connection *Connection) *SelectionRepository {
	return &SelectionRepository{newBaseRepository(connection)}
}
func NewPolicyRepository(connection *Connection) *PolicyRepository {
	return &PolicyRepository{newBaseRepository(connection)}
}
func NewDiagnosticsRepository(connection *Connection) *DiagnosticsRepository {
	return &DiagnosticsRepository{newBaseRepository(connection)}
}
