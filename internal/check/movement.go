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
	allowance          float64
	air, buffer        int
	ground             bool
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
	ground := p.Grounded(pos)
	pos[1] -= p.State.Offset
	dt := in.Tick - m.tick
	if m.tick == 0 || in.Tick <= m.tick || dt > 3 || at < m.at || at-m.at > 250 || !p.Ready(at) {
		m.last, m.tick, m.at, m.air, m.buffer = pos, in.Tick, at, 0, 0
		m.vertical = 0
		m.allowance, m.ground = 0, ground
		return
	}
	dx, dy, dz := float64(pos[0]-m.last[0]), float64(pos[1]-m.last[1]), float64(pos[2]-m.last[2])
	horizontal := math.Hypot(dx, dz) / float64(dt)
	vertical := dy / float64(dt)
	jump := vertical > 0 && m.vertical <= 0 && math.Abs(vertical-float64(p.State.Jump)) < 0.035
	limit := max(0.38, float64(p.State.Speed)*3.8)
	if p.State.Flags&wire.StatePhysics != 0 {
		if ground && m.ground {
			limit = max(0.025, float64(p.State.Speed)*2.2+0.006)
			m.allowance = max(0.32, limit)
		} else {
			m.allowance = max(m.allowance*0.91+0.026, float64(p.State.Speed)*2.25+0.012)
			if jump {
				m.allowance += 0.2
			}
			limit = max(0.36, m.allowance) + 0.015
		}
	} else if !ground || !m.ground {
		limit = max(0.48, float64(p.State.Speed)*4.8)
	}
	bad := horizontal > limit && dt == 1
	if m.fly {
		if ground || p.State.Flags&wire.StateCeiling != 0 {
			m.air = 0
			m.vertical = vertical
			m.buffer = 0
		} else {
			m.air += int(dt)
		}
		predicted := (m.vertical - 0.08) * 0.98
		bad = !ground && p.State.Flags&wire.StateCeiling == 0 && !jump && dt == 1 &&
			vertical > predicted+0.03 && vertical > -0.3
		if ground || jump || dt != 1 {
			m.buffer = 0
		}
		m.vertical = vertical
	}
	m.vertical, m.ground = vertical, ground
	m.last, m.tick, m.at = pos, in.Tick, at
	if !bad {
		m.buffer = max(0, m.buffer-1)
		m.pass(0.002)
		return
	}
	m.buffer = min(m.buffer+1, 30)
	required := 6
	if m.buffer < required || at-m.lastFlag < 250 {
		return
	}
	m.lastFlag = at
	m.fail(p, 1, fmt.Sprintf("horizontal=%.3f limit=%.3f vertical=%.3f air=%d", horizontal, limit, vertical, m.air))
}
