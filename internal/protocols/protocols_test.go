package protocols

import (
	"slices"
	"testing"
)

func TestLegacyAdaptersMatchNativeData(t *testing.T) {
	adapters, err := Legacy()
	if err != nil {
		t.Fatal(err)
	}

	ids := make([]int32, 0, len(adapters))
	for _, adapter := range adapters {
		ids = append(ids, adapter.ID())
	}
	slices.Sort(ids)

	want := []int32{748, 766, 827, 844, 898, 924, 944, 975, 1001, 2168, 2169}
	if !slices.Equal(ids, want) {
		t.Fatalf("protocols = %v, want %v", ids, want)
	}
}

func TestBlockRegistryResolvesAir(t *testing.T) {
	blocks, err := newBlockRegistry(nativeBlockStates)
	if err != nil {
		t.Fatal(err)
	}

	name, _, found := blocks.RuntimeIDToState(blocks.AirRuntimeID())
	if !found || name != airName {
		t.Fatalf("air runtime id %d resolves to %q", blocks.AirRuntimeID(), name)
	}
	if _, _, found := blocks.RuntimeIDToState(uint32(blocks.BlockCount())); found {
		t.Fatal("runtime id past the palette resolved to a state")
	}
}
