package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestMain resets the PostgreSQL test database before the suite runs. The
// store integration tests query by fixed secondary identifiers (agent key
// names, subject selectors), so rows left by an earlier run leak into later
// results. CI avoids this by provisioning a fresh service container per job;
// this reset gives consecutive local runs (for example `make test` followed by
// `make test-race`, both with AGENT_HARBOR_TEST_DATABASE_URL set) the same
// empty starting point. The suite itself applies db.Migrate on its own
// connections, so recreation is a plain schema drop.
func TestMain(m *testing.M) {
	databaseURL := os.Getenv("AGENT_HARBOR_TEST_DATABASE_URL")
	if databaseURL == "" {
		os.Exit(m.Run())
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect postgres: %v\n", err)
		os.Exit(1)
	}
	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		pool.Close()
		fmt.Fprintf(os.Stderr, "reset postgres test database: %v\n", err)
		os.Exit(1)
	}
	pool.Close()
	os.Exit(m.Run())
}
