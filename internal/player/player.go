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

	ClientTick uint64
	Punished   bool

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
		reporter:  reporter,
	}
}

func (p *Player) Touch() bool {
	return p.InputMode == InputModeTouch
}

func (p *Player) Serverbound(pk packet.Packet, at uint64) {
	if input, ok := pk.(*packet.PlayerAuthInput); ok {
		p.InputMode = input.InputMode
	}
	for _, c := range p.checks {
		c.Serverbound(p, pk, at)
	}
	if input, ok := pk.(*packet.PlayerAuthInput); ok {
		p.ClientTick = input.Tick
	}
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
