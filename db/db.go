package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pgxpool mannages a pool of connections rather than one. 
// That allows the server to manage many requests concurently. 


// Pool is the global connection pool. Other packages import this and call db.Pool.
var Pool *pgxpool.Pool

func Init() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return fmt.Errorf("failed to create pool: %w", err)
	}

	// Verify the connection actually works
	if err := pool.Ping(context.Background()); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	Pool = pool
	return nil
}