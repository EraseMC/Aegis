package protocols

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const airName = "minecraft:air"

type blockState struct {
	Name       string         `nbt:"name"`
	Properties map[string]any `nbt:"states"`
	Version    int32          `nbt:"version"`
}

type blockRegistry struct {
	states []blockState
	air    uint32
}

func newBlockRegistry(raw []byte) (*blockRegistry, error) {
	registry := &blockRegistry{}
	buf := bytes.NewBuffer(raw)
	dec := nbt.NewDecoderWithEncoding(buf, nbt.NetworkLittleEndian)
	found := false

	for buf.Len() > 0 {
		var state blockState
		if err := dec.Decode(&state); err != nil {
			return nil, fmt.Errorf("decode block state %d: %w", len(registry.states), err)
		}

		if state.Name == airName {
			registry.air = uint32(len(registry.states))
			found = true
		}
		registry.states = append(registry.states, state)
	}

	if !found {
		return nil, fmt.Errorf("block palette of %d states has no %s", len(registry.states), airName)
	}

	return registry, nil
}

func (r *blockRegistry) BlockCount() int {
	return len(r.states)
}

func (r *blockRegistry) AirRuntimeID() uint32 {
	return r.air
}

func (r *blockRegistry) RuntimeIDToState(runtimeID uint32) (name string, properties map[string]any, found bool) {
	if int(runtimeID) >= len(r.states) {
		return "", nil, false
	}

	state := r.states[runtimeID]

	return state.Name, state.Properties, true
}

type itemEntry struct {
	RuntimeID      int16  `json:"runtime_id"`
	ComponentBased bool   `json:"component_based"`
	Version        int32  `json:"version"`
	ComponentNBT   string `json:"component_nbt"`
}

func decodeItems(raw []byte) ([]protocol.ItemEntry, error) {
	var entries map[string]itemEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, fmt.Errorf("decode item list: %w", err)
	}

	items := make([]protocol.ItemEntry, 0, len(entries))
	for name, entry := range entries {
		data, err := componentData(entry.ComponentNBT)
		if err != nil {
			return nil, fmt.Errorf("decode components of %s: %w", name, err)
		}

		items = append(items, protocol.ItemEntry{
			Name:           name,
			RuntimeID:      entry.RuntimeID,
			ComponentBased: entry.ComponentBased,
			Version:        entry.Version,
			Data:           data,
		})
	}

	return items, nil
}

func componentData(encoded string) (map[string]any, error) {
	if encoded == "" {
		return nil, nil
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := nbt.UnmarshalEncoding(raw, &data, nbt.LittleEndian); err != nil {
		return nil, err
	}

	return data, nil
}
