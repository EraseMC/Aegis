package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/link"
)

func main() {
	path := flag.String("config", "aegis.json", "path to the config file")
	flag.Parse()

	cfg, err := config.Load(*path)
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	level := slog.LevelInfo
	if cfg.Debug {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := link.NewServer(cfg, log).Serve(ctx); err != nil {
		log.Error("server", "err", err)
		os.Exit(1)
	}
}
