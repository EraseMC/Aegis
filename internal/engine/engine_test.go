package engine

import (
	"bytes"
	"encoding/binary"
	"io"
	"log/slog"
	"testing"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/wire"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestCompactInput(t *testing.T) {
	var replies [][]byte
	e := New(config.Default(), slog.New(slog.NewTextHandler(io.Discard, nil)), func(f []byte) { replies = append(replies, f) })
	for _, f := range [][]byte{
		wire.Hello{Version: wire.Version}.Encode(),
		wire.Join{Session: 1, Name: "Compact", Protocol: 419, InputMode: 1}.Encode(),
		wire.Input{Session: 1, Time: 1000, Tick: 1, Position: [3]float32{2, 65, 3}, Pitch: 120, Mode: 1}.Encode(),
	} {
		typ, body, err := wire.ReadFrame(bytes.NewReader(f), nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Handle(typ, body); err != nil {
			t.Fatal(err)
		}
	}
	if e.players[1].Position != [3]float32{2, 65, 3} || e.players[1].ClientTick != 1 {
		t.Fatal("compact input changed position/tick")
	}
	if len(replies) != 3 {
		t.Fatalf("want welcome, flag and punishment; got %d", len(replies))
	}
	if err := e.Handle(wire.TypeInput, []byte{1}); err == nil {
		t.Fatal("accepted truncated input")
	}
}

// One iteration is a full 50ms game tick for 100 players, including 20 CPS,
// 4Hz synchronization/state and ten visible moving opponents per player.
func BenchmarkEngine100Players(b *testing.B) {
	e := New(config.Default(), slog.New(slog.NewTextHandler(io.Discard, nil)), func([]byte) {})
	inputs := make([][]byte, 100)
	attacks := make([][]byte, 100)
	states := make([][]byte, 100)
	observations := make([][][]byte, 100)
	for i := range inputs {
		id := uint64(i + 1)
		f := wire.Join{Session: id, Name: "Load", InputMode: 1}.Encode()
		if err := e.Handle(wire.TypeJoin, f[5:]); err != nil {
			b.Fatal(err)
		}
		inputs[i] = wire.Input{Session: id, Position: [3]float32{0, 1.621, 0}, Mode: 1}.Encode()[5:]
		attacks[i] = wire.Attack{Session: id, Target: (id % 100) + 1}.Encode()[5:]
		states[i] = wire.State{Session: id, Flags: wire.StateGround, Speed: .1, Jump: .42, Offset: 1.621}.Encode()[5:]
		for n := uint64(1); n <= 10; n++ {
			observations[i] = append(observations[i], wire.Observation{Session: id, Target: (id+n-1)%100 + 1, Box: [6]float32{-.3, 0, 2.2, .3, 1.8, 2.8}}.Encode()[5:])
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for tick := 1; tick <= b.N; tick++ {
		at := uint64(10000 + tick*50)
		for i, input := range inputs {
			id := uint64(i + 1)
			if tick%5 == 1 {
				binary.LittleEndian.PutUint64(states[i][8:], at)
				_ = e.Handle(wire.TypeState, states[i])
				for _, obs := range observations[i] {
					_ = e.Handle(wire.TypeObservation, obs)
				}
				p := e.players[id]
				p.View.Mark(int64(tick), at-50)
				p.Serverbound(&packet.NetworkStackLatency{Timestamp: int64(tick)}, at)
			}
			binary.LittleEndian.PutUint64(input[8:], at)
			binary.LittleEndian.PutUint64(input[16:], uint64(tick))
			_ = e.Handle(wire.TypeInput, input)
			binary.LittleEndian.PutUint64(attacks[i][8:], at)
			_ = e.Handle(wire.TypeAttack, attacks[i])
		}
	}
}
