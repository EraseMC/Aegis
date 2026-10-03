package engine

import (
	"encoding/binary"
	"io"
	"log/slog"
	"testing"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/wire"
)

func TestCompactAnimation(t *testing.T) {
	e := New(config.Default(), slog.New(slog.NewTextHandler(io.Discard, nil)), func([]byte) {})
	if err := e.Handle(wire.TypeJoin, wire.Join{Session: 1}.Encode()[5:]); err != nil {
		t.Fatal(err)
	}
	body := make([]byte, 16)
	binary.LittleEndian.PutUint64(body, 1)
	binary.LittleEndian.PutUint64(body[8:], 1200)
	if err := e.Handle(wire.TypeAnimation, body); err != nil {
		t.Fatal(err)
	}
	if e.players[1].LastSwing != 1200 {
		t.Fatal("animation did not update swing time")
	}
	if err := e.Handle(wire.TypeAnimation, body[:15]); err == nil {
		t.Fatal("accepted truncated animation")
	}
}
