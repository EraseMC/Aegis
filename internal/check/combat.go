package check

import (
	"fmt"
	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const ReachA = "Reach_A"
const KillAuraA = "KillAura_A"

type combat struct {
	violations
	aim      bool
	buffer   int
	lastFlag uint64
}

func newCombat(name string, settings config.Check, aim bool) *combat {
	return &combat{violations: newViolations(name, settings), aim: aim}
}
func (c *combat) Name() string { return c.name }

func (c *combat) Serverbound(p *player.Player, pk packet.Packet, at uint64) {
	if c.aim && p.Touch() {
		return
	}
	tx, ok := pk.(*packet.InventoryTransaction)
	if !ok || !p.Ready(at) || at < p.LastInput || at-p.LastInput > 250 || p.DirectionCount == 0 {
		return
	}
	attack, ok := tx.TransactionData.(*protocol.UseItemOnEntityTransactionData)
	if !ok || attack.ActionType != protocol.UseItemOnEntityActionAttack {
		return
	}
	distance, hit, known := p.View.Distance(attack.TargetEntityRuntimeID, at, p.Position, p.Directions[:p.DirectionCount])
	if !known {
		return
	}
	bad := distance > 3.35
	if c.aim {
		bad = distance > 0.8 && distance <= 3.35 && !hit
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
	c.fail(p, 1, fmt.Sprintf("distance=%.3f ray=%t rtt=%dms", distance, hit, p.View.RTT))
}
