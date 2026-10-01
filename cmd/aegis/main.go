package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/EraseMC/Aegis/internal/app"
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "oomph.hjson", "path to the Oomph configuration")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, log, *configPath); err != nil {
		log.Error("aegis stopped", "err", err)

		return 1
	}

	return 0
}
