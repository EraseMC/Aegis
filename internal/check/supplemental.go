package check

import (
	"fmt"
	"math"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/EraseMC/Aegis/internal/wire"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	KillAuraB    = "KillAura_B"
	AutoCritA    = "AutoCrit_A"
	VelocityA    = "Velocity_A"
	NoJumpDelayA = "NoJumpDelay_A"
)

type aura struct {
	violations
	targets       [16]uint64
	count, buffer int
	lastFlag      uint64
}

func newAura(s config.Check) *aura { return &aura{violations: newViolations(KillAuraB, s)} }
func (c *aura) Name() string       { return c.name }
func (c *aura) Serverbound(p *player.Player, pk packet.Packet, at uint64) {
	if tx, ok := pk.(*packet.InventoryTransaction); ok {
		if hit, ok := tx.TransactionData.(*protocol.UseItemOnEntityTransactionData); ok && hit.ActionType == protocol.UseItemOnEntityActionAttack && c.count < len(c.targets) {
			for _, id := range c.targets[:c.count] {
				if id == hit.TargetEntityRuntimeID {
					return
				}
			}
			c.targets[c.count] = hit.TargetEntityRuntimeID
			c.count++
		}
		return
	}
	if _, ok := pk.(*packet.PlayerAuthInput); !ok {
		return
	}
	count := c.count
	c.count = 0
	if count == 0 || !p.CombatReady(at) || p.LastInput == 0 || at < p.LastInput || at-p.LastInput < 20 || at-p.LastInput > 100 {
		c.buffer = max(0, c.buffer-1)
		return
	}
	known := 0
	for _, id := range c.targets[:count] {
		if e := p.View.Measure(id, at, [][3]float32{p.Position, p.PreviousPosition}, nil); e.Known && e.Raw < 4 {
			known++
		}
	}
	missingSwing := p.ControlsKnown && p.LastSwing > 0 && at >= p.LastSwing && at-p.LastSwing > 750
	if known < 3 && (!missingSwing || known == 0) {
		c.buffer = max(0, c.buffer-1)
		c.pass(0.01)
		return
	}
	c.buffer = min(20, c.buffer+1)
	if c.buffer < 4 || at-c.lastFlag < 500 {
		return
	}
	c.lastFlag = at
	c.fail(p, 1, fmt.Sprintf("targets=%d swing_missing=%t touch=%t", known, missingSwing, p.Touch()))
}

type velocity struct {
	violations
	ack, lastInput uint64
	ticks          int
	x, y           float64
	base           [3]float32
	active         bool
}

func newVelocity(s config.Check) *velocity { return &velocity{violations: newViolations(VelocityA, s)} }
func (v *velocity) Name() string           { return v.name }
func (v *velocity) Serverbound(p *player.Player, pk packet.Packet, at uint64) {
	in, ok := pk.(*packet.PlayerAuthInput)
	if !ok {
		return
	}
	if !p.CombatReady(at) || p.State.Flags&(wire.StateSpecial|wire.StateCeiling|wire.StateObstructed) != 0 {
		v.active = false
		v.ack = p.View.VelocityAt
		return
	}
	if p.View.VelocityAt != v.ack {
		v.ack, v.ticks, v.x, v.y = p.View.VelocityAt, 0, 0, 0
		v.lastInput = in.Tick - 1
		v.base = p.PreviousPosition
		motion := p.View.Velocity
		v.active = v.ack > 0 && at >= v.ack && at-v.ack < 150 && motion[1] > 0.12 && math.Hypot(float64(motion[0]), float64(motion[2])) > 0.15
	}
	if !v.active {
		return
	}
	if in.Tick <= v.lastInput || (v.lastInput > 0 && in.Tick-v.lastInput > 1) || at < v.ack || at-v.ack > 500 {
		v.active = false
		v.lastInput = in.Tick
		return
	}
	v.lastInput = in.Tick
	motion := p.View.Velocity
	horizontal := math.Hypot(float64(motion[0]), float64(motion[2]))
	projection := (float64(p.Position[0]-v.base[0])*float64(motion[0]) + float64(p.Position[2]-v.base[2])*float64(motion[2])) / horizontal
	v.x = max(v.x, projection)
	v.y = max(v.y, float64(p.Position[1]-v.base[1]))
	v.ticks++
	if v.ticks < 6 {
		return
	}
	v.active = false
	if v.x < horizontal*0.12 && v.y < float64(motion[1])*0.25 {
		v.fail(p, 1, fmt.Sprintf("response_x=%.3f response_y=%.3f motion_x=%.3f motion_y=%.3f", v.x, v.y, horizontal, motion[1]))
	} else {
		v.pass(0.1)
	}
}

type jumpCheck struct {
	violations
	critical                          bool
	tick, at, jumped, hopAt, lastFlag uint64
	lastY, vertical, hopHeight        float64
	buffer                            int
	held, baseGround                  bool
}

func newJumpCheck(name string, s config.Check, critical bool) *jumpCheck {
	return &jumpCheck{violations: newViolations(name, s), critical: critical}
}
func (c *jumpCheck) Name() string { return c.name }
func (c *jumpCheck) Serverbound(p *player.Player, pk packet.Packet, at uint64) {
	if c.critical {
		if tx, ok := pk.(*packet.InventoryTransaction); ok {
			if hit, ok := tx.TransactionData.(*protocol.UseItemOnEntityTransactionData); ok && hit.ActionType == protocol.UseItemOnEntityActionAttack && c.hopAt > 0 && at >= c.hopAt && at-c.hopAt <= 250 && c.vertical < -0.001 && p.Ready(at) && p.ControlsKnown {
				c.evidence(p, at, fmt.Sprintf("micro_hop=%.4f dy=%.4f", c.hopHeight, c.vertical))
				c.hopAt = 0
			}
			return
		}
	}
	in, ok := pk.(*packet.PlayerAuthInput)
	if !ok {
		return
	}
	y := float64(in.Position[1] - p.State.Offset)
	jumping := player.Input(in.InputData, packet.InputFlagJumping) || player.Input(in.InputData, packet.InputFlagStartJumping) || player.Input(in.InputData, packet.InputFlagJumpPressedRaw)
	if !p.Ready(at) || !p.ControlsKnown || p.State.Flags&(wire.StateCeiling|wire.StateObstructed) != 0 || c.tick == 0 || in.Tick != c.tick+1 || at < c.at || at-c.at > 150 {
		c.tick, c.at, c.lastY, c.vertical, c.jumped, c.hopAt, c.held, c.baseGround = in.Tick, at, y, 0, 0, 0, false, p.Grounded([3]float32(in.Position))
		c.buffer = 0
		return
	}
	dy := y - c.lastY
	if !jumping {
		c.jumped = 0
	}
	if c.critical {
		if c.baseGround && !jumping && dy > 0.015 && dy < 0.15 {
			c.hopAt, c.hopHeight = at, dy
		}
		if jumping || dy > 0.15 || (c.hopAt > 0 && at-c.hopAt > 250) {
			c.hopAt = 0
		}
	} else if jumping && dy > 0 && math.Abs(dy-float64(p.State.Jump)) < 0.035 && c.vertical <= 0 {
		if c.held && c.jumped > 0 && in.Tick-c.jumped < 10 {
			c.evidence(p, at, fmt.Sprintf("jump_interval=%d held=true", in.Tick-c.jumped))
		} else {
			c.pass(0.03)
		}
		c.jumped = in.Tick
	}
	c.tick, c.at, c.lastY, c.vertical, c.held, c.baseGround = in.Tick, at, y, dy, jumping, p.Grounded([3]float32(in.Position))
}

func (c *jumpCheck) evidence(p *player.Player, at uint64, data string) {
	c.buffer = min(10, c.buffer+1)
	if c.buffer < 3 || at-c.lastFlag < 500 {
		return
	}
	c.lastFlag = at
	c.fail(p, 1, data)
}
