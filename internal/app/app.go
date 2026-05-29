package app

import (
	"net/http"

	"github.com/SummerXaa-Z/agent-harbor/internal/httpapi"
	"github.com/SummerXaa-Z/agent-harbor/internal/store"
)

type App struct {
	server *httpapi.Server
}

func New() *App {
	return &App{
		server: httpapi.New(store.NewMemory()),
	}
}

func (a *App) Router() http.Handler {
	return a.server.Router()
}
