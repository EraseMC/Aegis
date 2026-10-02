package check

import (
	"math/rand"
	"testing"

	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/EraseMC/Aegis/internal/wire"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

type recorder struct {
	flags    map[string]int
	punished map[string]wire.Action
}

func (r *recorder) Flag(_ *player.Player, check string, _, _ float64, _ string) {
	r.flags[check]++
}

func (r *recorder) Punish(_ *player.Player, check string, _ float64, action wire.Action, _ string) {
	r.punished[check] = action
}

func newPlayer(inputMode uint32) (*player.Player, *recorder) {
	r := &recorder{flags: map[string]int{}, punished: map[string]wire.Action{}}
	return player.New(wire.Join{Session: 1, Name: "test", InputMode: inputMode}, r, Build(config.Default())), r
}

func play(p *player.Player, seconds int, speed float64, jitter int, cps int) {
	rng := rand.New(rand.NewSource(1))
	interval := 1000.0 / 20 / speed
	for tick := 1; tick <= seconds*20; tick++ {
		at := uint64(10_000 + float64(tick)*interval + float64(rng.Intn(jitter+1)))
		at -= at % 50
		input := &packet.PlayerAuthInput{Tick: uint64(tick), InputMode: p.InputMode, InputData: protocol.NewInputFlags(packet.InputFlagCount)}
		swings := tick*cps/20 - (tick-1)*cps/20
		for range swings {
			p.Serverbound(&packet.LevelSoundEvent{SoundType: packet.SoundEventAttackNoDamage}, at)
		}
		if swings > 0 {
			input.InputData.Set(packet.InputFlagMissedSwing)
		}
		p.Serverbound(input, at)
	}
}

func TestTimerLegit(t *testing.T) {
	p, r := newPlayer(player.InputModeMouse)
	play(p, 120, 1.0, 120, 0)
	if r.flags[TimerA] != 0 {
		t.Fatalf("legit player flagged %d times", r.flags[TimerA])
	}
}

func TestTimerFast(t *testing.T) {
	p, r := newPlayer(player.InputModeMouse)
	play(p, 60, 1.1, 40, 0)
	if r.flags[TimerA] == 0 {
		t.Fatal("1.1x timer was not flagged")
	}
}

func TestAutoclicker(t *testing.T) {
	p, r := newPlayer(player.InputModeMouse)
	play(p, 30, 1.0, 0, 14)
	if r.flags[AutoclickerA] != 0 {
		t.Fatalf("14 cps flagged %d times", r.flags[AutoclickerA])
	}
	p, r = newPlayer(player.InputModeTouch)
	play(p, 30, 1.0, 0, 19)
	if r.flags[AutoclickerA] == 0 {
		t.Fatal("19 cps on touch was not flagged")
	}
	p, r = newPlayer(player.InputModeMouse)
	play(p, 30, 1.0, 0, 28)
	if r.flags[AutoclickerA] == 0 {
		t.Fatal("28 cps on mouse was not flagged")
	}
}

func TestBadPitch(t *testing.T) {
	p, r := newPlayer(player.InputModeMouse)
	p.Serverbound(&packet.PlayerAuthInput{Tick: 1, Pitch: 120}, 0)
	if r.punished[BadPacketA] != wire.ActionKick {
		t.Fatal("pitch 120 was not punished")
	}
}
