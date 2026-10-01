package session

import (
	"log/slog"
	"runtime/debug"
	"sync"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/oomph-ac/oomph/anticheat/player"
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
	p.HandleEvents(&handler{registry: r})
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
}

func (h *handler) HandleJoin(ctx *event.Context[*player.Player]) {
	h.registry.add(ctx.Val())
}

func (h *handler) HandleQuit(ctx *event.Context[*player.Player]) {
	h.registry.remove(ctx.Val())
}
