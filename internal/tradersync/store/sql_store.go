package store

import "github.com/jackc/pgx/v5/pgxpool"
import "github.com/useryege/athena/internal/accountstate/txgate"

type SQLStore struct {
	pool                  *pgxpool.Pool
	directoryTransactions txgate.Beginner
	activitySiteURL       string
}

func NewSQLStore(pool *pgxpool.Pool) *SQLStore { return &SQLStore{pool: pool} }
