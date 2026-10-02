package check

import (
	"github.com/EraseMC/Aegis/internal/player"
	"github.com/EraseMC/Aegis/internal/wire"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"testing"
)

func ready(p *player.Player, at uint64, ground bool) {
	p.State = wire.State{Time: at, Offset: 1.621, Speed: 0.1, Jump: 0.42, Position: [3]float32{0, 0, 0}}
	if ground {
		p.State.Flags = wire.StateGround
	}
	p.View.Mark(int64(at), at-50)
	p.View.Ack(int64(at), at)
}

func combatPlayer(distance float32, yaw float32) (*player.Player, *recorder) {
	p, r := newPlayer(player.InputModeMouse)
	ready(p, 10000, true)
	p.View.Observe(wire.Observation{Target: 2, Box: [6]float32{-0.3, 0, distance - 0.3, 0.3, 1.8, distance + 0.3}})
	p.View.Mark(42, 10000)
	p.View.Ack(42, 10050)
	p.Serverbound(&packet.PlayerAuthInput{Tick: 1, Yaw: yaw, Position: [3]float32{0, 1.621, 0}, InputData: protocol.NewInputFlags(packet.InputFlagCount)}, 10050)
	return p, r
}

func attack(p *player.Player, at uint64) {
	p.Serverbound(&packet.InventoryTransaction{TransactionData: &protocol.UseItemOnEntityTransactionData{TargetEntityRuntimeID: 2, ActionType: protocol.UseItemOnEntityActionAttack}}, at)
}

func TestCombatLagAndReach(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		distance, yaw float32
		check         string
		want          bool
	}{
		{"legit", 3.3, 0, ReachA, false}, {"center315", 3.15, 0, ReachA, false}, {"reach315", 3.45, 0, ReachA, true}, {"reach320", 3.5, 0, ReachA, true}, {"reach", 4.5, 0, ReachA, true}, {"looking", 2.5, 0, KillAuraA, false}, {"backwards", 2.5, 180, KillAuraA, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			p, r := combatPlayer(scenario.distance, scenario.yaw)
			for tick := uint64(1); tick <= 450; tick++ {
				at := 10050 + tick*50
				ready(p, at, true)
				p.Serverbound(&packet.PlayerAuthInput{Tick: tick, Yaw: scenario.yaw, Position: [3]float32{0, 1.621, 0}, InputData: protocol.NewInputFlags(packet.InputFlagCount)}, at)
				attack(p, at)
			}
			if (r.flags[scenario.check] > 0) != scenario.want {
				t.Fatalf("flags %v", r.flags)
			}
			if scenario.want && r.punished[scenario.check] != wire.ActionBan {
				t.Fatalf("expected ban after persistent combat evidence, got %v", r.punished)
			}
			if !scenario.want && len(r.punished) != 0 {
				t.Fatalf("legitimate combat punished: %v", r.punished)
			}
		})
	}
	p, r := combatPlayer(5, 0)
	for i := 0; i < 20; i++ {
		attack(p, 13000)
	}
	if r.flags[ReachA] > 0 {
		t.Fatal("stale/lagged evidence was used")
	}
}

func TestJumpWithUnsampledLandings(t *testing.T) {
	for _, jump := range []float32{0.42, 0.62, 0.82} {
		p, r := newPlayer(player.InputModeMouse)
		y, velocity := float32(0), float32(0)
		for tick := uint64(1); tick <= 1000; tick++ {
			if y <= 0 {
				y = 0
				velocity = jump
			}
			y += velocity
			velocity = (velocity - .08) * .98
			at := 10000 + tick*50
			// Emulate 4Hz terrain sampling which never catches the short landing.
			ready(p, at, false)
			p.State.Jump = jump
			p.State.Position[1] = max(0, y)
			p.Serverbound(&packet.PlayerAuthInput{Tick: tick, Position: [3]float32{float32(tick) * .34, max(0, y) + 1.621, 0}, InputData: protocol.NewInputFlags(packet.InputFlagCount)}, at)
		}
		if r.flags[FlyA] > 0 {
			t.Fatalf("jump=%v flags=%v", jump, r.flags)
		}
	}
}

func TestCombatAttackBeforeRotation(t *testing.T) {
	p, r := combatPlayer(2.5, 180)
	for tick := uint64(1); tick <= 100; tick++ {
		at := 10050 + tick*50
		ready(p, at, true)
		attack(p, at)
		p.Serverbound(&packet.PlayerAuthInput{Tick: tick, Yaw: 0, Position: [3]float32{0, 1.621, 0}, InputData: protocol.NewInputFlags(packet.InputFlagCount)}, at)
	}
	if r.flags[KillAuraA] > 0 || r.flags[ReachA] > 0 {
		t.Fatal(r.flags)
	}
}

func TestCombatDuringKnockback(t *testing.T) {
	p, r := combatPlayer(4.5, 0)
	for tick := uint64(1); tick <= 100; tick++ {
		at := 10050 + tick*50
		ready(p, at, true)
		p.State.Flags |= wire.StateSpecial
		p.Observe(wire.Observation{Target: p.Session, Kind: wire.Velocity, Time: at})
		p.Serverbound(&packet.PlayerAuthInput{Tick: tick, Position: [3]float32{0, 1.621, 0}, InputMode: player.InputModeMouse, InputData: protocol.NewInputFlags(packet.InputFlagCount)}, at)
		attack(p, at)
		if p.Ready(at) {
			t.Fatal("movement lost knockback grace")
		}
	}
	if r.flags[ReachA] == 0 {
		t.Fatal("knockback disabled reach detection throughout combat")
	}
	p.Observe(wire.Observation{Target: p.Session, Kind: wire.Teleport, Time: 16000})
	ready(p, 16050, true)
	if p.CombatReady(16050) {
		t.Fatal("combat lost teleport grace")
	}
}

func TestMovementAndExemptions(t *testing.T) {
	for _, scenario := range []struct {
		name               string
		speed              float32
		hover              bool
		flags              uint32
		wantSpeed, wantFly bool
	}{
		{"sprint", 0.36, false, 0, false, false}, {"speed", 1.2, false, 0, true, false},
		{"hover", 0, true, 0, false, true}, {"flight", 1.2, true, wire.StateFlying, false, false},
		{"ice", 1.2, true, wire.StateSpecial, false, false}, {"lag", 1.2, true, wire.StateLagging, false, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			p, r := newPlayer(player.InputModeMouse)
			for tick := uint64(1); tick <= 150; tick++ {
				at := 10000 + tick*50
				ready(p, at, !scenario.hover)
				p.State.Flags |= scenario.flags
				y := float32(1.621)
				if scenario.hover {
					y += 3
				}
				p.Serverbound(&packet.PlayerAuthInput{Tick: tick, Position: [3]float32{float32(tick) * scenario.speed, y, 0}, InputData: protocol.NewInputFlags(packet.InputFlagCount)}, at)
			}
			if (r.flags[SpeedA] > 0) != scenario.wantSpeed || (r.flags[FlyA] > 0) != scenario.wantFly {
				t.Fatalf("flags %v", r.flags)
			}
		})
	}
	p, r := newPlayer(player.InputModeMouse)
	for tick := uint64(1); tick <= 30; tick++ {
		at := 10000 + tick*50
		ready(p, at, true)
		p.Observe(wire.Observation{Target: 1, Kind: wire.Velocity, Time: at})
		p.Serverbound(&packet.PlayerAuthInput{Tick: tick, Position: [3]float32{float32(tick), 4, 0}, InputData: protocol.NewInputFlags(packet.InputFlagCount)}, at)
	}
	if r.flags[FlyA] > 0 || r.flags[SpeedA] > 0 {
		t.Fatal("knockback was flagged")
	}
}

func TestVanillaJump(t *testing.T) {
	p, r := newPlayer(player.InputModeMouse)
	y, velocity := float32(0), float32(0)
	for tick := uint64(1); tick <= 500; tick++ {
		ground := y <= 0
		if ground {
			y = 0
			if tick%30 == 1 {
				velocity = 0.42
			} else {
				velocity = 0
			}
		}
		y += velocity
		velocity = (velocity - 0.08) * 0.98
		at := 10000 + tick*50
		ready(p, at, ground)
		p.State.Position[1] = max(0, y)
		p.Serverbound(&packet.PlayerAuthInput{Tick: tick, Position: [3]float32{float32(tick) * 0.36, max(0, y) + 1.621, 0}, InputData: protocol.NewInputFlags(packet.InputFlagCount)}, at)
	}
	if r.flags[FlyA] > 0 || r.flags[SpeedA] > 0 {
		t.Fatalf("jump flags %v", r.flags)
	}
}
