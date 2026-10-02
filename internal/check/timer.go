package check

import (
	"fmt"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const TimerA = "Timer_A"

const (
	tickMillis       = 50
	timerGrace       = 5000
	timerWindow      = 10000
	timerAheadMillis = 300
)

type drift struct {
	at     uint64
	amount int64
}

type timer struct {
	violations
	started  bool
	start    uint64
	lastTick uint64
	base     int64
	drifts   []drift
}

func newTimer(settings config.Check) *timer {
	return &timer{violations: newViolations(TimerA, settings)}
}

func (t *timer) Name() string {
	return TimerA
}

func (t *timer) Serverbound(p *player.Player, pk packet.Packet, at uint64) {
	input, ok := pk.(*packet.PlayerAuthInput)
	if !ok {
		return
	}
	offset := int64(at) - int64(input.Tick)*tickMillis

	if !t.started || input.Tick <= t.lastTick {
		t.started, t.start, t.base, t.drifts = true, at, offset, t.drifts[:0]
		t.lastTick = input.Tick
		return
	}
	t.lastTick = input.Tick

	if at-t.start < timerGrace || offset >= t.base {
		t.base = min(t.base, offset)
		return
	}
	t.drifts = append(t.drifts, drift{at: at, amount: t.base - offset})
	t.base = offset

	var ahead int64
	kept := t.drifts[:0]
	for _, d := range t.drifts {
		if at-d.at <= timerWindow {
			kept = append(kept, d)
			ahead += d.amount
		}
	}
	t.drifts = kept

	if ahead > timerAheadMillis {
		t.drifts = t.drifts[:0]
		t.fail(p, 1, fmt.Sprintf("ahead=%dms", ahead))
	}
}
