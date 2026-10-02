package check

import (
	"fmt"
	"math"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const BadPacketA = "BadPacket_A"

type badPacket struct {
	violations
}

func newBadPacket(settings config.Check) *badPacket {
	return &badPacket{violations: newViolations(BadPacketA, settings)}
}

func (b *badPacket) Name() string {
	return BadPacketA
}

func (b *badPacket) Serverbound(p *player.Player, pk packet.Packet, _ uint64) {
	input, ok := pk.(*packet.PlayerAuthInput)
	if !ok {
		return
	}
	values := []float32{input.Pitch, input.Yaw, input.HeadYaw, input.Position[0], input.Position[1], input.Position[2]}
	for _, v := range values {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			b.fail(p, 1, "non-finite rotation or position")
			return
		}
	}
	if math.Abs(float64(input.Pitch)) > 90.01 {
		b.fail(p, 1, fmt.Sprintf("pitch=%.2f", input.Pitch))
	}
}
