package config

import (
	"testing"

	"github.com/EraseMC/Aegis/internal/wire"
)

func TestDefaultPunishments(t *testing.T) {
	for name, want := range map[string]wire.Action{
		"Reach_A": wire.ActionBan, "KillAura_A": wire.ActionBan, "Timer_A": wire.ActionBan,
		"Fly_A": wire.ActionKick, "Speed_A": wire.ActionKick,
		"BadPacket_A": wire.ActionKick, "Autoclicker_A": wire.ActionKick,
	} {
		check := Default().Check(name)
		if !check.Enabled || check.Punishment() != want {
			t.Fatalf("%s: got %+v, want action %v", name, check, want)
		}
	}
}

func TestNewHeuristicsOnlyReport(t *testing.T) {
	for _, name := range []string{"KillAura_B", "AutoCrit_A", "Velocity_A", "NoJumpDelay_A"} {
		check := Default().Check(name)
		if !check.Enabled || check.Punishment() != wire.ActionNone {
			t.Fatalf("%s %+v", name, check)
		}
	}
}
