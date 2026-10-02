package check

import (
	"github.com/EraseMC/Aegis/internal/config"
	"github.com/EraseMC/Aegis/internal/player"
)

type violations struct {
	name     string
	settings config.Check
	level    float64
	punished bool
}

func newViolations(name string, settings config.Check) violations {
	return violations{name: name, settings: settings}
}

func (v *violations) fail(p *player.Player, amount float64, data string) {
	v.level += amount
	p.Flag(v.name, v.level, v.settings.Max, data)
	if v.settings.Max > 0 && v.level >= v.settings.Max && !v.punished {
		v.punished = true
		p.Punish(v.name, v.level, v.settings.Punishment(), data)
	}
}

func (v *violations) pass(amount float64) {
	v.level = max(0, v.level-amount)
}
