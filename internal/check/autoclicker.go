package check

import (
	"fmt"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const AutoclickerA = "Autoclicker_A"

const (
	clickWindow = 20
	limitCPS    = 20
	clickBuffer = 3
)

type autoclicker struct {
	violations
	clicks   []uint64
	sounds   int
	nextEval uint64
	buffer   int
}

func newAutoclicker(settings config.Check) *autoclicker {
	return &autoclicker{violations: newViolations(AutoclickerA, settings)}
}

func (a *autoclicker) Name() string {
	return AutoclickerA
}

func (a *autoclicker) Serverbound(p *player.Player, pk packet.Packet, _ uint64) {
	switch pk := pk.(type) {
	case *packet.InventoryTransaction:
		if data, ok := pk.TransactionData.(*protocol.UseItemOnEntityTransactionData); ok && data.ActionType == protocol.UseItemOnEntityActionAttack {
			a.clicks = append(a.clicks, p.ClientTick)
		}
	case *packet.LevelSoundEvent:
		if pk.SoundType == packet.SoundEventAttackNoDamage {
			a.sounds++
		}
	case *packet.PlayerAuthInput:
		misses := a.sounds
		if misses == 0 && player.Input(pk.InputData, packet.InputFlagMissedSwing) {
			misses = 1
		}
		for range misses {
			a.clicks = append(a.clicks, pk.Tick)
		}
		a.sounds = 0
		a.evaluate(p, pk.Tick)
	}
}

func (a *autoclicker) evaluate(p *player.Player, tick uint64) {
	if tick < a.nextEval {
		return
	}
	a.nextEval = tick + clickWindow

	kept := a.clicks[:0]
	for _, c := range a.clicks {
		if c+clickWindow > tick {
			kept = append(kept, c)
		}
	}
	a.clicks = kept

	cps := len(a.clicks)
	if cps <= limitCPS {
		a.buffer = max(0, a.buffer-1)
		a.pass(0.05)
		return
	}
	a.buffer++
	if a.buffer >= clickBuffer {
		a.fail(p, 1, fmt.Sprintf("cps=%d limit=%d", cps, limitCPS))
	}
}
