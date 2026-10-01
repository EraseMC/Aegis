package protocols

import (
	_ "embed"
	"fmt"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/shawtymarco/go-multiversion/mapping"
	"github.com/shawtymarco/go-multiversion/protocols/v1_21_100"
	"github.com/shawtymarco/go-multiversion/protocols/v1_21_110"
	"github.com/shawtymarco/go-multiversion/protocols/v1_21_130"
	"github.com/shawtymarco/go-multiversion/protocols/v1_21_40"
	"github.com/shawtymarco/go-multiversion/protocols/v1_21_50"
	"github.com/shawtymarco/go-multiversion/protocols/v1_26_0"
	"github.com/shawtymarco/go-multiversion/protocols/v1_26_10"
	"github.com/shawtymarco/go-multiversion/protocols/v1_26_20"
	"github.com/shawtymarco/go-multiversion/protocols/v1_26_30"
	"github.com/shawtymarco/go-multiversion/protocols/v1_26_44"
	"github.com/shawtymarco/go-multiversion/protocols/v1_26_45"
)

var (
	//go:embed data/canonical_block_states.nbt
	nativeBlockStates []byte
	//go:embed data/required_item_list.json
	nativeItemList []byte
)

type constructor func(native mapping.BlockRegistry, nativeItems []protocol.ItemEntry) (minecraft.Protocol, error)

var constructors = []constructor{
	v1_26_45.NewWithRegistries,
	v1_26_44.NewWithRegistries,
	v1_26_30.NewWithRegistries,
	v1_26_20.NewWithRegistries,
	v1_26_10.NewWithRegistries,
	v1_26_0.NewWithRegistries,
	v1_21_130.NewWithRegistries,
	v1_21_110.NewWithRegistries,
	v1_21_100.NewWithRegistries,
	v1_21_50.NewWithRegistries,
	v1_21_40.NewWithRegistries,
}

func Legacy() ([]minecraft.Protocol, error) {
	blocks, err := newBlockRegistry(nativeBlockStates)
	if err != nil {
		return nil, err
	}

	items, err := decodeItems(nativeItemList)
	if err != nil {
		return nil, err
	}

	adapters := make([]minecraft.Protocol, 0, len(constructors))
	for _, construct := range constructors {
		adapter, err := construct(blocks, items)
		if err != nil {
			return nil, fmt.Errorf("build protocol adapter: %w", err)
		}
		adapters = append(adapters, adapter)
	}

	return adapters, nil
}
