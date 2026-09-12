package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/gocql/gocql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/scylladb/gocqlx/v3"
	"github.com/scylladb/gocqlx/v3/migrate"
)

func MigratePgsql(ctx context.Context) error {
	pgMigrations, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		return fmt.Errorf("open pg migration conn: %w", err)
	}
	defer pgMigrations.Close()

	if ctx == nil {
		ctx = context.Background()
	}
	if err := goose.UpContext(ctx, pgMigrations, "migrations/pgsql"); err != nil {
		return fmt.Errorf("run pgsql migrations: %w", err)
	}
	return nil
}

func NewPgxPool(ctx context.Context) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("create pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping pgx pool: %w", err)
	}
	return pool, nil
}

func NewScylla(ctx context.Context) (gocqlx.Session, error) {
	cluster := gocql.NewCluster(os.Getenv("SCYLLA_URL"))
	keyspace := os.Getenv("SCYLLA_KEYSPACE")

	initSession, err := cluster.CreateSession()
	if err != nil {
		return gocqlx.Session{}, fmt.Errorf("create scylla init session: %w", err)
	}
	if err := initSession.Query(
		`CREATE KEYSPACE IF NOT EXISTS ` + keyspace +
			` WITH replication = {'class': 'NetworkTopologyStrategy', 'datacenter1': 1}`).Exec(); err != nil {
		initSession.Close()
		return gocqlx.Session{}, fmt.Errorf("create keyspace: %w", err)
	}
	initSession.Close()

	cluster.Keyspace = keyspace
	session, err := gocqlx.WrapSession(cluster.CreateSession())
	if err != nil {
		return gocqlx.Session{}, fmt.Errorf("create scylla session: %w", err)
	}

	if err := migrate.FromFS(ctx, session, os.DirFS("migrations/scylla")); err != nil {
		defer session.Close()
		return gocqlx.Session{}, fmt.Errorf("run scylla migrations: %w", err)
	}
	return session, nil
}
