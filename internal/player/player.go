package player

import (
	"github.com/EraseMC/Aegis/internal/wire"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	InputModeMouse = packet.InputModeMouse
	InputModeTouch = packet.InputModeTouch
)

type Reporter interface {
	Flag(p *Player, check string, violations, max float64, data string)
	Punish(p *Player, check string, violations float64, action wire.Action, data string)
}

type Check interface {
	Name() string
	Serverbound(p *Player, pk packet.Packet, at uint64)
}

type Player struct {
	Session   uint64
	Name      string
	XUID      string
	Protocol  uint32
	Version   string
	DeviceOS  int32
	InputMode uint32

	ClientTick                    uint64
	Punished                      bool
	View                          View
	State                         wire.State
	Position                      [3]float32
	PreviousPosition              [3]float32
	Directions                    [4][3]float32
	DirectionCount, DirectionNext int
	GraceUntil                    uint64
	CombatGraceUntil              uint64
	LastInput                     uint64
	ControlsKnown                 bool
	LastSwing                     uint64
	AnimationSeen                 bool

	checks   []Check
	reporter Reporter
}

func New(join wire.Join, reporter Reporter, checks []Check) *Player {
	return &Player{
		Session:   join.Session,
		Name:      join.Name,
		XUID:      join.XUID,
		Protocol:  join.Protocol,
		Version:   join.Version,
		DeviceOS:  join.DeviceOS,
		InputMode: join.InputMode,
		checks:    checks,
		View:      NewView(),
		reporter:  reporter,
	}
}

func (p *Player) Touch() bool {
	return p.InputMode == InputModeTouch
}

func (p *Player) Serverbound(pk packet.Packet, at uint64) {
	if ack, ok := pk.(*packet.NetworkStackLatency); ok {
		p.View.Ack(ack.Timestamp, at)
		return
	}
	if input, ok := pk.(*packet.PlayerAuthInput); ok {
		p.InputMode = input.InputMode
		p.PreviousPosition = p.Position
		p.Position = [3]float32(input.Position)
		p.Directions[p.DirectionNext] = Direction(input.Yaw, input.Pitch)
		p.DirectionNext = (p.DirectionNext + 1) % len(p.Directions)
		p.DirectionCount = min(p.DirectionCount+1, len(p.Directions))
		if Input(input.InputData, packet.InputFlagMissedSwing) {
			p.LastSwing = at
		}
	}
	if swing, ok := pk.(*packet.LevelSoundEvent); ok && swing.SoundType == packet.SoundEventAttackNoDamage {
		p.LastSwing = at
	}
	if animation, ok := pk.(*packet.Animate); ok && animation.ActionType == packet.AnimateActionSwingArm {
		p.LastSwing = at
		p.AnimationSeen = true
	}
	for _, c := range p.checks {
		c.Serverbound(p, pk, at)
	}
	if input, ok := pk.(*packet.PlayerAuthInput); ok {
		p.ClientTick = input.Tick
		p.LastInput = at
	}
}

func (p *Player) Grounded(position [3]float32) bool {
	feet := position[1] - p.State.Offset
	return p.State.Flags&wire.StateGround != 0 && feet-p.State.Position[1] < 0.035 && feet-p.State.Position[1] > -0.035
}

func (p *Player) Observe(u wire.Observation) {
	if u.Target == p.Session && u.Kind == wire.Teleport {
		p.View = NewView()
		p.CombatGraceUntil = max(p.CombatGraceUntil, u.Time+2000)
	}
	if u.Target == p.Session && (u.Kind == wire.Teleport || u.Kind == wire.Velocity) {
		p.GraceUntil = max(p.GraceUntil, u.Time+2000)
	}
	if u.Kind == wire.BlockChange {
		p.GraceUntil = max(p.GraceUntil, u.Time+500)
		return
	}
	p.View.Observe(u)
}

func (p *Player) Ready(at uint64) bool {
	return p.State.Time > 0 && at >= p.State.Time && at-p.State.Time <= 750 && at >= p.GraceUntil &&
		p.State.Flags&(wire.StateFlying|wire.StateSpecial|wire.StateLagging|wire.StateDead|wire.StateFrozen) == 0 &&
		p.View.Synced(at)
}

func (p *Player) CombatReady(at uint64) bool {
	return p.State.Time > 0 && at >= p.State.Time && at-p.State.Time <= 750 && at >= p.CombatGraceUntil &&
		p.State.Flags&(wire.StateFlying|wire.StateLagging|wire.StateDead|wire.StateFrozen) == 0 && p.View.Synced(at)
}

func (p *Player) Flag(check string, violations, max float64, data string) {
	p.reporter.Flag(p, check, violations, max, data)
}

func (p *Player) Punish(check string, violations float64, action wire.Action, data string) {
	if p.Punished {
		return
	}
	p.Punished = action != wire.ActionNone
	p.reporter.Punish(p, check, violations, action, data)
}

func Input(flags protocol.InputFlags, flag int) bool {
	return flag < flags.Len() && flags.Load(flag)
}
