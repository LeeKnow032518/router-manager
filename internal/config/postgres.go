package config

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

func NewPostgres() *Postgres {
	url := os.Getenv("POSTGRES_GO_URL")
	if url == "" {
		url = "postgres://postgres:postgres@localhost:5432/telemetry-aggregator-local?sslmode=disable"
		//log.Fatal("Dsn id empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatal("failed to create pool:  %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Fatal("failed to ping DB: %w", err)
	}

	return &Postgres{Pool: pool}
}

func (p *Postgres) Close() {
	p.Pool.Close()
}
