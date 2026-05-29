package app

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/SummerXaa-Z/agent-harbor/internal/db"
	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type App struct {
	server *httpapi.Server
	close  func()
}

func New(ctx context.Context) (*App, error) {
	repo := store.Repository(store.NewMemory())
	closeFn := func() {}
	if databaseURL := os.Getenv("AGENT_HARBOR_DATABASE_URL"); databaseURL != "" {
		pool, err := pgxpool.New(ctx, databaseURL)
		if err != nil {
			return nil, fmt.Errorf("connect postgres: %w", err)
		}
		if err := db.Migrate(ctx, pool); err != nil {
			pool.Close()
			return nil, err
		}
		repo = store.NewPostgres(pool)
		closeFn = pool.Close
	}
	return &App{
		server: httpapi.New(repo, httpapi.WithAdminKey(os.Getenv("AGENT_HARBOR_ADMIN_KEY"))),
		close:  closeFn,
	}, nil
}

func (a *App) Router() http.Handler {
	return a.server.Router()
}

func (a *App) Close() {
	a.close()
}
