package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/EraseMC/Aegis/internal/app"
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", "oomph.hjson", "path to the Oomph configuration")
	debugModes := flag.String("debug", "", "comma separated Oomph debug modes logged for every player")
	offline := flag.Bool("offline", false, "skip Xbox Live authentication, for local testing only")
	flag.Parse()

	level := slog.LevelInfo
	if *debugModes != "" {
		level = slog.LevelDebug
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opts := app.Options{ConfigPath: *configPath, Offline: *offline}
	if *debugModes != "" {
		opts.DebugModes = strings.Split(*debugModes, ",")
	}

	if err := app.Run(ctx, log, opts); err != nil {
		log.Error("aegis stopped", "err", err)

		return 1
	}

	return 0
}
