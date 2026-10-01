package session

import (
	"fmt"
	"strings"

	"github.com/oomph-ac/oomph/anticheat/player"
)

func ParseDebugModes(names []string) ([]int, error) {
	modes := make([]int, 0, len(names))
	for _, name := range names {
		mode, ok := player.DebugModeMap[strings.TrimSpace(name)]
		if !ok {
			return nil, fmt.Errorf("unknown debug mode %q, expected one of %s", name, strings.Join(player.DebugModeList, ", "))
		}
		modes = append(modes, mode)
	}

	return modes, nil
}
