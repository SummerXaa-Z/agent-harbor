package app

import (
	"net/http"

	"github.com/SummerXaa-Z/ai-nexus-go-rebirth/internal/httpapi"
	"github.com/SummerXaa-Z/ai-nexus-go-rebirth/internal/store"
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
