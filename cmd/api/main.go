package main

import (
	"log/slog"
	"os"

	"github.com/chuuch/gorest/internal/config"
	"github.com/chuuch/gorest/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg)
	if err != nil {
		slog.Error("initialize server", "error", err)
		os.Exit(1)
	}

	if err := srv.Run(*cfg); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}
