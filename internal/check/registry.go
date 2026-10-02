package check

import (
	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
)

type factory func(settings config.Check) player.Check

var factories = map[string]factory{
	TimerA:       func(s config.Check) player.Check { return newTimer(s) },
	AutoclickerA: func(s config.Check) player.Check { return newAutoclicker(s) },
	BadPacketA:   func(s config.Check) player.Check { return newBadPacket(s) },
	ReachA:       func(s config.Check) player.Check { return newCombat(ReachA, s, false) },
	KillAuraA:    func(s config.Check) player.Check { return newCombat(KillAuraA, s, true) },
	SpeedA:       func(s config.Check) player.Check { return newMovement(SpeedA, s, false) },
	FlyA:         func(s config.Check) player.Check { return newMovement(FlyA, s, true) },
}

func Build(cfg config.Config) []player.Check {
	checks := make([]player.Check, 0, len(factories))
	for _, name := range []string{TimerA, AutoclickerA, BadPacketA, ReachA, KillAuraA, SpeedA, FlyA} {
		settings := cfg.Check(name)
		if settings.Enabled {
			checks = append(checks, factories[name](settings))
		}
	}
	return checks
}
