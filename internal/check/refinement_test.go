package check

import (
	"math"
	"testing"

	"github.com/EraseMC/Aegis/internal/player"
	"github.com/EraseMC/Aegis/internal/wire"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func input(tick uint64, x, y float32) *packet.PlayerAuthInput {
	return &packet.PlayerAuthInput{Tick: tick, Position: [3]float32{x, y + 1.621, 0}, InputMode: player.InputModeMouse, InputData: protocol.NewInputFlags(packet.InputFlagCount)}
}

func TestGroundPhysicsEnvelope(t *testing.T) {
	for _, scenario := range []struct {
		name         string
		speed, scale float32
		want         bool
	}{
		{"walk", .1, 1, false}, {"sprint", .13, 1, false}, {"speed_effect", .182, 1, false},
		{"slowness", .07, 1, false}, {"small_sprint_speed", .13, 1.05, true}, {"small_walk_speed", .1, 1.1, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			p, r := newPlayer(player.InputModeMouse)
			var x, v float32
			for tick := uint64(1); tick <= 120; tick++ {
				v = v*.546 + scenario.speed*.98
				x += v * scenario.scale
				at := 10000 + tick*50
				ready(p, at, true)
				p.State.Flags |= wire.StatePhysics
				p.State.Speed = scenario.speed
				p.Serverbound(input(tick, x, 0), at)
			}
			if (r.flags[SpeedA] > 0) != scenario.want {
				t.Fatal(r.flags)
			}
			if !scenario.want && len(r.punished) > 0 {
				t.Fatal(r.punished)
			}
		})
	}
}

func TestFlyPunishmentWithinFourSeconds(t *testing.T) {
	for _, step := range []float32{0, .08, .36, .42} {
		p, r := newPlayer(player.InputModeTouch)
		punishedAt := uint64(0)
		for tick := uint64(1); tick <= 100; tick++ {
			at := 10000 + tick*50
			ready(p, at, false)
			p.Serverbound(input(tick, 0, 3+float32(tick)*step), at)
			if r.punished[FlyA] != wire.ActionNone {
				punishedAt = tick * 50
				break
			}
		}
		if punishedAt == 0 || punishedAt > 4000 {
			t.Fatalf("step=%v punishment=%d flags=%v", step, punishedAt, r.flags)
		}
	}
}

func TestPhysicsSprintJump(t *testing.T) {
	p, r := newPlayer(player.InputModeMouse)
	var x, y, vx, vy float32
	for tick := uint64(1); tick <= 600; tick++ {
		ground := y <= 0
		if ground {
			y = 0
			vx = vx*.546 + .13*.98
			if tick%15 == 1 {
				vy = .42
				vx += .2
			}
		} else {
			vx = vx*.91 + .026
		}
		x += vx
		y += vy
		vy = (vy - .08) * .98
		if y < 0 {
			y = 0
			vy = 0
		}
		at := 10000 + tick*50
		ready(p, at, y == 0)
		p.State.Flags |= wire.StatePhysics
		p.State.Speed = .13
		p.State.Position[1] = y
		p.Serverbound(input(tick, x, y), at)
	}
	if r.flags[SpeedA] > 0 || r.flags[FlyA] > 0 {
		t.Fatal(r.flags)
	}
}

func TestJumpWithPeriodicServerSnapshots(t *testing.T) {
	p, r := newPlayer(player.InputModeMouse)
	var y, vy float32
	var state wire.State
	for tick := uint64(1); tick <= 600; tick++ {
		if y <= 0 {
			y = 0
			vy = .42
		}
		y += vy
		vy = (vy - .08) * .98
		if y < 0 {
			y, vy = 0, 0
		}
		at := 10000 + tick*50
		ready(p, at, y == 0)
		if tick%5 == 1 {
			p.State.Position[1] = y
			p.State.Flags |= wire.StatePhysics
			state = p.State
		}
		p.State = state
		p.Serverbound(input(tick, 0, y), at)
	}
	if r.flags[FlyA] > 0 {
		t.Fatal(r.flags)
	}
}

func TestTouchAuraAndLegitimateTap(t *testing.T) {
	for _, targets := range []int{1, 2, 3} {
		p, r := combatPlayer(2, 180)
		for id := uint64(3); id <= 4; id++ {
			p.View.Observe(wire.Observation{Target: id, Box: [6]float32{-.3, 0, 1.7, .3, 1.8, 2.3}})
		}
		p.View.Mark(99, 10050)
		p.View.Ack(99, 10100)
		for tick := uint64(2); tick <= 100; tick++ {
			at := 10000 + tick*50
			ready(p, at, true)
			for id := uint64(2); id < uint64(2+targets); id++ {
				p.Serverbound(&packet.InventoryTransaction{TransactionData: &protocol.UseItemOnEntityTransactionData{TargetEntityRuntimeID: id, ActionType: protocol.UseItemOnEntityActionAttack}}, at)
			}
			pk := input(tick, 0, 0)
			pk.InputMode = player.InputModeTouch
			pk.Yaw = 180
			p.Serverbound(pk, at)
		}
		if r.flags[KillAuraA] > 0 {
			t.Fatal("tap aim treated as crosshair", r.flags)
		}
		if (r.flags[KillAuraB] > 0) != (targets == 3) {
			t.Fatalf("targets=%d flags=%v", targets, r.flags)
		}
		if r.punished[KillAuraB] != wire.ActionNone {
			t.Fatal("new aura heuristic punished")
		}
	}
}

func TestLegitimateAttackAnimations(t *testing.T) {
	p, r := combatPlayer(2, 0)
	for tick := uint64(2); tick <= 300; tick++ {
		at := 10000 + tick*50
		ready(p, at, true)
		p.Serverbound(&packet.Animate{ActionType: packet.AnimateActionSwingArm}, at)
		p.Serverbound(&packet.InventoryTransaction{TransactionData: &protocol.UseItemOnEntityTransactionData{TargetEntityRuntimeID: 2, ActionType: protocol.UseItemOnEntityActionAttack}}, at)
		p.Serverbound(input(tick, 0, 0), at)
	}
	if r.flags[AutoclickerA] > 0 || r.flags[KillAuraB] > 0 {
		t.Fatal(r.flags)
	}
}

func TestMissingSwingAfterObservedAnimations(t *testing.T) {
	p, r := combatPlayer(2, 0)
	p.ControlsKnown = true
	p.Serverbound(&packet.Animate{ActionType: packet.AnimateActionSwingArm}, 10050)
	for tick := uint64(2); tick <= 180; tick++ {
		at := 10000 + tick*50
		ready(p, at, true)
		p.Serverbound(&packet.InventoryTransaction{TransactionData: &protocol.UseItemOnEntityTransactionData{TargetEntityRuntimeID: 2, ActionType: protocol.UseItemOnEntityActionAttack}}, at)
		pk := input(tick, 0, 0)
		pk.InputMode = player.InputModeTouch
		p.Serverbound(pk, at)
	}
	if r.flags[KillAuraB] == 0 {
		t.Fatal("missing swing was ignored")
	}
}

func TestLegacyBridgeWithoutAnimationTelemetry(t *testing.T) {
	p, r := combatPlayer(2, 0)
	p.Serverbound(&packet.LevelSoundEvent{SoundType: packet.SoundEventAttackNoDamage}, 10050)
	for tick := uint64(2); tick <= 180; tick++ {
		at := 10000 + tick*50
		ready(p, at, true)
		p.Serverbound(&packet.InventoryTransaction{TransactionData: &protocol.UseItemOnEntityTransactionData{TargetEntityRuntimeID: 2, ActionType: protocol.UseItemOnEntityActionAttack}}, at)
		p.Serverbound(input(tick, 0, 0), at)
	}
	if r.flags[KillAuraB] > 0 {
		t.Fatal("old bridge lacks animation telemetry", r.flags)
	}
}

func TestLegacyClientWithoutAttackAnimations(t *testing.T) {
	p, r := combatPlayer(2, 0)
	p.Protocol, p.ControlsKnown = 419, true
	p.Serverbound(&packet.LevelSoundEvent{SoundType: packet.SoundEventAttackNoDamage}, 10050)
	for tick := uint64(2); tick <= 180; tick++ {
		at := 10000 + tick*50
		ready(p, at, true)
		p.Serverbound(&packet.InventoryTransaction{TransactionData: &protocol.UseItemOnEntityTransactionData{TargetEntityRuntimeID: 2, ActionType: protocol.UseItemOnEntityActionAttack}}, at)
		p.Serverbound(input(tick, 0, 0), at)
	}
	if r.flags[KillAuraB] > 0 {
		t.Fatal("miss sound does not confirm attack animation support", r.flags)
	}
}

func TestVelocityConfirmationAndWalls(t *testing.T) {
	for _, scenario := range []struct {
		name     string
		response bool
		flags    uint32
		ack      bool
		want     bool
	}{
		{"normal", true, 0, true, false}, {"cancel", false, 0, true, true},
		{"wall", false, wire.StateObstructed, true, false}, {"ceiling", false, wire.StateCeiling, true, false},
		{"water", false, wire.StateSpecial, true, false}, {"unconfirmed", false, 0, false, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			p, r := newPlayer(player.InputModeMouse)
			ready(p, 10000, true)
			p.Serverbound(input(1, 0, 0), 10000)
			p.Observe(wire.Observation{Target: 1, Time: 10000, Kind: wire.Velocity, Box: [6]float32{.4, .4, 0}})
			p.View.Mark(900, 10000)
			if scenario.ack {
				p.View.Ack(900, 10050)
			}
			for tick := uint64(2); tick <= 12; tick++ {
				at := 10000 + tick*50
				if scenario.ack {
					ready(p, at, false)
				} else {
					p.State.Time = at
					p.State.Flags = 0
				}
				p.State.Flags |= scenario.flags
				x, y := float32(0), float32(0)
				if scenario.response {
					x = float32(tick-1) * .2
					y = .4
				}
				p.Serverbound(input(tick, x, y), at)
			}
			if (r.flags[VelocityA] > 0) != scenario.want {
				t.Fatal(r.flags)
			}
		})
	}
}

func TestSmallReachOldProtocol(t *testing.T) {
	for _, dist := range []float32{2.9, 3, 3.05, 3.1, 3.2} {
		p, r := combatPlayer(dist+.3, 0)
		p.Protocol = 419
		for tick := uint64(1); tick <= 180; tick++ {
			at := 10050 + tick*50
			ready(p, at, true)
			p.Serverbound(input(tick, 0, 0), at)
			attack(p, at)
		}
		if (r.flags[ReachA] > 0) != (dist > 3.035) {
			t.Fatalf("distance=%v flags=%v", dist, r.flags)
		}
	}
}

func TestLegitimatePacketBatches(t *testing.T) {
	p, r := newPlayer(player.InputModeMouse)
	var x, y, vy float32
	for tick := uint64(1); tick <= 400; tick++ {
		ground := y <= 0
		if ground {
			y = 0
			vy = .42
		}
		y += vy
		vy = (vy - .08) * .98
		if y < 0 {
			y = 0
		}
		x += .36
		at := 10000 + ((tick+2)/3)*150
		ready(p, at, ground)
		p.State.Position[1] = y
		p.Serverbound(input(tick, x, y), at)
	}
	if r.flags[SpeedA] > 0 || r.flags[FlyA] > 0 {
		t.Fatal(r.flags)
	}
}

func TestNoJumpDelayReleaseAndContinuous(t *testing.T) {
	for _, held := range []bool{false, true} {
		p, r := newPlayer(player.InputModeMouse)
		p.ControlsKnown = true
		for tick := uint64(1); tick <= 200; tick++ {
			at := 10000 + tick*50
			ready(p, at, true)
			y := float32(0)
			if tick%5 == 2 {
				y = .42
			}
			pk := input(tick, 0, y)
			if held || tick%5 == 2 {
				pk.InputData.Set(packet.InputFlagJumping)
			}
			p.Serverbound(pk, at)
		}
		if (r.flags[NoJumpDelayA] > 0) != held {
			t.Fatal(held, r.flags)
		}
	}
}

func TestMicroCritAndVanillaJump(t *testing.T) {
	for _, hop := range []float32{.06, .42} {
		p, r := combatPlayer(2, 0)
		p.ControlsKnown = true
		for tick := uint64(2); tick <= 180; tick++ {
			at := 10000 + tick*50
			ready(p, at, true)
			y := float32(0)
			if tick%8 == 2 {
				y = hop
			}
			if tick%8 == 3 {
				y = hop * .5
			}
			pk := input(tick, 0, y)
			if hop > .15 {
				pk.InputData.Set(packet.InputFlagJumping)
			}
			p.Serverbound(pk, at)
			if tick%8 == 3 {
				attack(p, at)
			}
		}
		if (r.flags[AutoCritA] > 0) != (math.Abs(float64(hop)) < .15) {
			t.Fatal(hop, r.flags)
		}
	}
}
