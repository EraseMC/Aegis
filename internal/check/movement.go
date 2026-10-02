package check

import (
	"fmt"
	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/EraseMC/Aegis/internal/wire"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"math"
)

const SpeedA = "Speed_A"
const FlyA = "Fly_A"

type movement struct {
	violations
	fly                bool
	last               [3]float32
	tick, at, lastFlag uint64
	vertical           float64
	air, buffer        int
}

func newMovement(name string, s config.Check, fly bool) *movement {
	return &movement{violations: newViolations(name, s), fly: fly}
}
func (m *movement) Name() string { return m.name }

func (m *movement) Serverbound(p *player.Player, pk packet.Packet, at uint64) {
	in, ok := pk.(*packet.PlayerAuthInput)
	if !ok {
		return
	}
	pos := [3]float32(in.Position)
	pos[1] -= p.State.Offset
	dt := in.Tick - m.tick
	if m.tick == 0 || in.Tick <= m.tick || dt > 3 || at < m.at || at-m.at > 250 || !p.Ready(at) {
		m.last, m.tick, m.at, m.air, m.buffer = pos, in.Tick, at, 0, 0
		m.vertical = 0
		return
	}
	dx, dy, dz := float64(pos[0]-m.last[0]), float64(pos[1]-m.last[1]), float64(pos[2]-m.last[2])
	horizontal := math.Hypot(dx, dz) / float64(dt)
	vertical := dy / float64(dt)
	bad := horizontal > max(0.72, float64(p.State.Speed)*7.2)
	if m.fly {
		ground := p.State.Flags&wire.StateGround != 0 && math.Abs(float64(pos[1]-p.State.Position[1])) < 0.35
		if ground || p.State.Flags&wire.StateCeiling != 0 {
			m.air = 0
			m.vertical = vertical
			m.buffer = 0
		} else {
			m.air += int(dt)
		}
		predicted := (m.vertical - 0.08) * 0.98
		bad = m.air > 15 && vertical > predicted+0.09 && vertical > -0.3
		bad = bad || m.air > 25 && vertical >= -0.02
		if vertical > float64(p.State.Jump)+0.12 {
			bad = true
		}
		m.vertical = vertical
	}
	m.last, m.tick, m.at = pos, in.Tick, at
	if !bad {
		m.buffer = max(0, m.buffer-1)
		m.pass(0.002)
		return
	}
	m.buffer = min(m.buffer+1, 30)
	if m.buffer < 10 || at-m.lastFlag < 1000 {
		return
	}
	m.lastFlag = at
	m.fail(p, 1, fmt.Sprintf("horizontal=%.3f vertical=%.3f air=%d", horizontal, vertical, m.air))
}
