package player

import (
	"github.com/EraseMC/Aegis/internal/wire"
	"testing"
)

func TestConfirmationAndBounds(t *testing.T) {
	v := NewView()
	box := [6]float32{0, 0, 2, 1, 2, 3}
	v.Observe(wire.Observation{Target: 2, Box: box})
	v.Mark(42, 1000)
	if v.Ack(99, 1050) {
		t.Fatal("accepted invented marker")
	}
	if _, _, known := v.Distance(2, 1050, [3]float32{}, nil); known {
		t.Fatal("used unconfirmed spawn")
	}
	if !v.Ack(42, 1050) {
		t.Fatal("real marker rejected")
	}
	if d, _, known := v.Distance(2, 1100, [3]float32{}, nil); !known || d != 2 {
		t.Fatal(d, known)
	}
	v.Observe(wire.Observation{Target: 2, Kind: wire.EntityRemove})
	v.Mark(43, 1100)
	v.Ack(43, 1150)
	if _, _, known := v.Distance(2, 1150, [3]float32{}, nil); known {
		t.Fatal("removed entity used")
	}
	for i := uint64(1); i < 1000; i++ {
		v.Observe(wire.Observation{Target: i, Box: box})
		v.Mark(int64(i+100), i+2000)
		v.Ack(int64(i+100), i+2050)
	}
	if len(v.Targets) > MaxTargets {
		t.Fatal("unbounded targets")
	}
	for i := int64(1); i < 100; i++ {
		v.Mark(i, 4000)
	}
	if len(v.markers) > maxMarkers {
		t.Fatal("unbounded pending markers")
	}
}

func BenchmarkViewDistance(b *testing.B) {
	v := NewView()
	for tick := uint64(1); tick <= 16; tick++ {
		for id := uint64(1); id <= 100; id++ {
			v.Observe(wire.Observation{Target: id, Box: [6]float32{0, 0, 2, 1, 2, 3}})
		}
		v.Mark(int64(tick), tick*50)
		v.Ack(int64(tick), tick*50+50)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.Distance(50, 900, [3]float32{0, 1.621, 0}, [][3]float32{{0, 0, 1}})
	}
}

func TestInterpolationAndOverflow(t *testing.T) {
	v := NewView()
	for i, x := range []float32{-2, 2} {
		v.Observe(wire.Observation{Target: 2, Box: [6]float32{x - .3, 0, 2, x + .3, 1.8, 2.6}})
		v.Mark(int64(i), uint64(1000+i*50))
		v.Ack(int64(i), uint64(1050+i*50))
	}
	if _, hit, known := v.Distance(2, 1100, [3]float32{0, 1.62, 0}, [][3]float32{{0, 0, 1}}); !known || !hit {
		t.Fatal("interpolated target rejected")
	}
	for i := 0; i <= MaxTargets*2; i++ {
		v.Observe(wire.Observation{Target: 2, Box: [6]float32{0, 0, 2, 1, 2, 3}})
	}
	if v.Synced(1100) {
		t.Fatal("overflowed history remained usable")
	}
	if v.Ack(1, 1100) {
		t.Fatal("old acknowledgment repaired lost history")
	}
}
