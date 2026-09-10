package store

import "github.com/jackc/pgx/v5/pgxpool"

type SQLStore struct {
	pool            *pgxpool.Pool
	activitySiteURL string
}

func NewSQLStore(pool *pgxpool.Pool) *SQLStore { return &SQLStore{pool: pool} }
