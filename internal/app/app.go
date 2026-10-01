package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/oomph-ac/oomph/anticheat/integration/proxy"
	"github.com/oomph-ac/oomph/anticheat/oconfig"
	"github.com/oomph-ac/oomph/anticheat/utils"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/resource"

	"github.com/EraseMC/Aegis/internal/backend"
	"github.com/EraseMC/Aegis/internal/session"
)

const (
	dialTimeout     = 10 * time.Second
	contentKeysFile = "content_keys.json"
)

type Options struct {
	ConfigPath string
	DebugModes []string
}

func Run(ctx context.Context, log *slog.Logger, opts Options) error {
	if err := loadConfig(opts.ConfigPath); err != nil {
		return err
	}

	debugModes, err := session.ParseDebugModes(opts.DebugModes)
	if err != nil {
		return err
	}

	tuneRuntime()
	cfg := oconfig.Global

	status, err := minecraft.NewForeignStatusProvider(cfg.RemoteAddress)
	if err != nil {
		return fmt.Errorf("query status of %s: %w", cfg.RemoteAddress, err)
	}

	packs, err := resourcePacks(cfg.Resource.ResourceFolder)
	if err != nil {
		return err
	}

	utils.InitializeBlockNameMapping()

	sessions := session.NewRegistry(log, debugModes)
	serveCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	defer cancel()

	p, err := proxy.Listen(serveCtx, proxy.Config{
		LocalAddress:  cfg.LocalAddress,
		RemoteAddress: cfg.RemoteAddress,
		Log:           log,
		Listen: minecraft.ListenConfig{
			StatusProvider:       status,
			ResourcePacks:        packs,
			TexturePacksRequired: cfg.Resource.RequirePacks,
			FlushRate:            -1,
		},
		Dial:      backend.Dial(cfg.BackupAddress, dialTimeout),
		Configure: sessions.Configure,
	})
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.LocalAddress, err)
	}
	defer func() {
		_ = p.Close()
	}()

	served := make(chan error, 1)
	go func() {
		served <- p.Serve(serveCtx)
	}()
	log.Info("aegis is listening", "local", cfg.LocalAddress, "remote", cfg.RemoteAddress)

	select {
	case <-ctx.Done():
		sessions.DisconnectAll(cfg.ShutdownMessage)

		return nil
	case err := <-served:
		return err
	}
}

func loadConfig(path string) error {
	err := oconfig.ParseJSON(path)
	if errors.Is(err, oconfig.ErrConfigCreated) || errors.Is(err, oconfig.ErrConfigUpdated) {
		return fmt.Errorf("review %s and start again: %w", path, err)
	}
	if err != nil {
		return fmt.Errorf("load %s: %w", path, err)
	}

	return nil
}

func tuneRuntime() {
	percent := oconfig.Global.GCPercent
	if percent != -1 {
		percent = max(percent, 100)
	}
	debug.SetGCPercent(percent)
	debug.SetMemoryLimit(int64(oconfig.Global.MemThreshold) << 20)
}

func resourcePacks(folder string) ([]*resource.Pack, error) {
	if folder == "" {
		return nil, nil
	}

	packs, err := utils.ResourcePacks(folder, contentKeysFile)
	if err != nil {
		return nil, fmt.Errorf("load resource packs from %s: %w", folder, err)
	}

	return packs, nil
}
