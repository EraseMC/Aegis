package engine

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/EraseMC/Aegis/internal/check"
	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/EraseMC/Aegis/internal/wire"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type Engine struct {
	cfg      config.Config
	log      *slog.Logger
	send     func([]byte)
	shield   int32
	players  map[uint64]*player.Player
	fromUser packet.Pool
	failures map[uint32]int
}

func New(cfg config.Config, log *slog.Logger, send func([]byte)) *Engine {
	return &Engine{
		cfg:      cfg,
		log:      log,
		send:     send,
		players:  make(map[uint64]*player.Player),
		fromUser: packet.NewClientPool(),
		failures: make(map[uint32]int),
	}
}

func (e *Engine) Players() int {
	return len(e.players)
}

func (e *Engine) Handle(t wire.Type, body []byte) error {
	switch t {
	case wire.TypeHello:
		hello, err := wire.DecodeHello(body)
		if err != nil {
			return err
		}
		if hello.Version != wire.Version {
			return fmt.Errorf("server speaks wire version %d, aegis speaks %d", hello.Version, wire.Version)
		}
		e.shield = hello.ShieldID
		e.log.Info("server connected", "server", hello.Server, "shield", hello.ShieldID)
		e.send(wire.Welcome{Version: wire.Version}.Encode())
	case wire.TypeJoin:
		join, err := wire.DecodeJoin(body)
		if err != nil {
			return err
		}
		e.players[join.Session] = player.New(join, e, check.Build(e.cfg))
		e.log.Debug("join", "player", join.Name, "protocol", join.Protocol, "version", join.Version, "os", join.DeviceOS)
	case wire.TypeQuit:
		quit, err := wire.DecodeQuit(body)
		if err != nil {
			return err
		}
		delete(e.players, quit.Session)
	case wire.TypeServerbound:
		pk, err := wire.DecodePacket(body)
		if err != nil {
			return err
		}
		e.serverbound(pk)
	case wire.TypeClientbound, wire.TypeTick:
	default:
		return fmt.Errorf("unknown frame type 0x%02x", uint8(t))
	}
	return nil
}

func (e *Engine) serverbound(frame wire.Packet) {
	p, ok := e.players[frame.Session]
	if !ok {
		return
	}
	pk, id, err := e.decode(e.fromUser, frame.Payload)
	if err != nil {
		e.failures[id]++
		if e.failures[id] == 1 || e.failures[id]%1000 == 0 {
			e.log.Warn("undecodable serverbound packet", "id", id, "player", p.Name, "protocol", p.Protocol, "count", e.failures[id], "err", err)
		}
		return
	}
	if pk != nil {
		p.Serverbound(pk, frame.Time)
	}
}

func (e *Engine) decode(pool packet.Pool, payload []byte) (pk packet.Packet, id uint32, err error) {
	buf := bytes.NewBuffer(payload)
	var header packet.Header
	if err := header.Read(buf); err != nil {
		return nil, 0, err
	}
	factory, ok := pool[header.PacketID]
	if !ok {
		return nil, header.PacketID, nil
	}
	defer func() {
		if r := recover(); r != nil {
			pk, err = nil, fmt.Errorf("%v", r)
		}
	}()
	pk = factory()
	pk.Marshal(protocol.NewReader(buf, e.shield, true))
	return pk, header.PacketID, nil
}

func (e *Engine) Flag(p *player.Player, name string, violations, max float64, data string) {
	e.log.Info("flag", "player", p.Name, "check", name, "vl", fmt.Sprintf("%.1f/%.0f", violations, max), "data", data)
	e.send(wire.Flag{Session: p.Session, Check: name, Violations: float32(violations), Max: float32(max), Data: data}.Encode())
}

func (e *Engine) Punish(p *player.Player, name string, violations float64, action wire.Action, data string) {
	e.log.Warn("punish", "player", p.Name, "check", name, "action", action, "data", data)
	e.send(wire.Punish{Session: p.Session, Check: name, Violations: float32(violations), Action: action, Data: data}.Encode())
}
