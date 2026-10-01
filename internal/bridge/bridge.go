package bridge

import (
	"encoding/json"
	"fmt"

	"github.com/oomph-ac/oomph/anticheat/oconfig"
	"github.com/oomph-ac/oomph/anticheat/player"
	"github.com/oomph-ac/oomph/anticheat/utils"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

const (
	flagID       = "oomph:aegis.flag"
	punishmentID = "oomph:aegis.punishment"
)

type flag struct {
	Detection     string  `json:"detection"`
	Violations    float64 `json:"violations"`
	MaxViolations float64 `json:"max_violations"`
	Data          string  `json:"data"`
}

type punishment struct {
	Detection  string  `json:"detection"`
	Violations float64 `json:"violations"`
	Action     string  `json:"action"`
}

func Key(d player.Detection) string {
	return d.Type() + "_" + d.SubType()
}

func SendFlag(p *player.Player, d player.Detection, extraData []any) error {
	m := d.Metadata()

	return send(p, flagID, flag{
		Detection:     Key(d),
		Violations:    m.Violations,
		MaxViolations: m.MaxViolations,
		Data:          utils.KeyValsToString(extraData),
	})
}

func SendPunishment(p *player.Player, d player.Detection) error {
	return send(p, punishmentID, punishment{
		Detection:  Key(d),
		Violations: d.Metadata().Violations,
		Action:     oconfig.DtcOpts(Key(d)).Punishment,
	})
}

func send(p *player.Player, id string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode %s: %w", id, err)
	}

	return p.SendPacketToServer(&packet.ScriptMessage{Identifier: id, Data: data})
}
