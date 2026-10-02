package check

import (
	"fmt"
	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"math"
)

const ReachA = "Reach_A"
const KillAuraA = "KillAura_A"

type combat struct {
	violations
	aim      bool
	buffer   int
	lastFlag uint64
	pending  [8]wireAttack
	count    int
}

type wireAttack struct{ target, at uint64 }

func newCombat(name string, settings config.Check, aim bool) *combat {
	return &combat{violations: newViolations(name, settings), aim: aim}
}
func (c *combat) Name() string { return c.name }

func (c *combat) Serverbound(p *player.Player, pk packet.Packet, at uint64) {
	if tx, ok := pk.(*packet.InventoryTransaction); ok {
		attack, valid := tx.TransactionData.(*protocol.UseItemOnEntityTransactionData)
		if valid && attack.ActionType == protocol.UseItemOnEntityActionAttack && c.count < len(c.pending) {
			c.pending[c.count] = wireAttack{target: attack.TargetEntityRuntimeID, at: at}
			c.count++
		}
		return
	}
	if _, ok := pk.(*packet.PlayerAuthInput); !ok || c.count == 0 {
		return
	}
	count := c.count
	c.count = 0
	// Attack packets may precede the input carrying the attack's look direction.
	if !p.CombatReady(at) || (c.aim && p.Touch()) || p.DirectionCount == 0 || p.LastInput == 0 || at < p.LastInput || at-p.LastInput > 250 {
		c.buffer = 0
		return
	}
	positions := [2][3]float32{p.Position, p.PreviousPosition}
	for _, attack := range c.pending[:count] {
		if at >= attack.at && at-attack.at <= 200 {
			c.evaluate(p, attack.target, at, positions[:])
		}
	}
}

func (c *combat) evaluate(p *player.Player, target, at uint64, positions [][3]float32) {
	e := p.View.Measure(target, at, positions, p.Directions[:p.DirectionCount])
	if !e.Known {
		return
	}
	// The raw distance is a conservative lower bound. A ray is more precise
	// for non-touch attacks; its 0.1-block box expansion covers hitbox tolerance.
	bad := e.Raw > 3.05 || (!p.Touch() && !math.IsInf(e.Ray, 1) && e.Ray > 3.01)
	if c.aim {
		bad = e.Raw > 0.8 && e.Raw <= 3.05 && !e.AimHit
	}
	if !bad {
		c.buffer = max(0, c.buffer-1)
		c.pass(0.025)
		return
	}
	c.buffer = min(c.buffer+1, 10)
	if c.buffer < 5 || at-c.lastFlag < 1000 {
		return
	}
	c.lastFlag = at
	c.fail(p, 1, fmt.Sprintf("distance=%.3f ray_distance=%.3f aim=%t rtt=%dms", e.Raw, e.Ray, e.AimHit, p.View.RTT))
}
