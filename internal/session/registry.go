package session

import (
	"log/slog"
	"runtime/debug"
	"sync"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/oomph-ac/oomph/anticheat/player"

	"github.com/EraseMC/Aegis/internal/bridge"
)

const crashMessage = "§cInternal proxy error, please reconnect."

type Registry struct {
	log        *slog.Logger
	debugModes []int

	mu      sync.RWMutex
	players map[*player.Player]struct{}
}

func NewRegistry(log *slog.Logger, debugModes []int) *Registry {
	return &Registry{log: log, debugModes: debugModes, players: make(map[*player.Player]struct{})}
}

func (r *Registry) Configure(p *player.Player) {
	p.SetRecoverFunc(r.recover)
	for _, mode := range r.debugModes {
		p.Dbg.Toggle(mode)
	}
	p.HandleEvents(&handler{registry: r, punished: make(map[string]struct{})})
}

func (r *Registry) DisconnectAll(message string) {
	for _, p := range r.all() {
		p.Disconnect(message)
	}
}

func (r *Registry) add(p *player.Player) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.players[p] = struct{}{}
}

func (r *Registry) remove(p *player.Player) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.players, p)
}

func (r *Registry) all() []*player.Player {
	r.mu.RLock()
	defer r.mu.RUnlock()

	players := make([]*player.Player, 0, len(r.players))
	for p := range r.players {
		players = append(players, p)
	}

	return players
}

func (r *Registry) recover(p *player.Player, err any) {
	r.log.Error("player processing panicked", "player", p.Name(), "err", err, "stack", string(debug.Stack()))
	p.Disconnect(crashMessage)
}

type handler struct {
	player.NopEventHandler

	registry *Registry
	punished map[string]struct{}
}

func (h *handler) HandleJoin(ctx *event.Context[*player.Player]) {
	h.registry.add(ctx.Val())
}

func (h *handler) HandleQuit(ctx *event.Context[*player.Player]) {
	h.registry.remove(ctx.Val())
}

func (h *handler) HandleFlag(ctx *event.Context[*player.Player], d player.Detection, extraData []any) {
	if err := bridge.SendFlag(ctx.Val(), d, extraData); err != nil {
		h.registry.log.Warn("forward flag", "player", ctx.Val().Name(), "detection", bridge.Key(d), "err", err)
	}
}

func (h *handler) HandlePunishment(ctx *event.Context[*player.Player], d player.Detection, _ *string) {
	ctx.Cancel()

	key := bridge.Key(d)
	if _, ok := h.punished[key]; ok {
		return
	}
	h.punished[key] = struct{}{}

	if err := bridge.SendPunishment(ctx.Val(), d); err != nil {
		h.registry.log.Warn("forward punishment", "player", ctx.Val().Name(), "detection", key, "err", err)
	}
}
