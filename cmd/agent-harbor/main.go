package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/SummerXaa-Z/agent-harbor/internal/app"
)

func main() {
	addr := os.Getenv("AGENT_HARBOR_ADDR")
	if addr == "" {
		addr = ":9090"
	}
	if err := app.RunServer(context.Background(), addr); err != nil {
		slog.Error("server exited", "error", err)
		os.Exit(1)
	}
}
