package link

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/wire"
)

func TestShutdownWithUnreadVerdicts(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		NewServer(config.Default(), slog.New(slog.NewTextHandler(io.Discard, nil))).handle(ctx, server)
		close(done)
	}()
	if err := client.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Write(wire.Hello{Version: wire.Version, Server: "test"}.Encode()); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("connection did not stop while its verdict writer was blocked")
	}
}
