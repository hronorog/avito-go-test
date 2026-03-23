package repo

import (
	"context"
	"database/sql"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}