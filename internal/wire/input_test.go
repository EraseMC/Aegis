package wire

import (
	"bytes"
	"testing"
)

func TestInputControlsAndLegacyFrame(t *testing.T) {
	for _, control := range []uint8{0, ControlKnown | ControlJump | ControlJumpStart | ControlSprint | ControlJumpPressed, ControlKnown | ControlHorizontalCollision | ControlVerticalCollision | ControlSneak} {
		want := Input{Session: 7, Time: 1000, Tick: 42, Position: [3]float32{1, 2, 3}, Pitch: 4, Yaw: 5, HeadYaw: 6, Mode: 1, Missed: true, Control: control}
		_, body, err := ReadFrame(bytes.NewBuffer(want.Encode()), nil)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeInput(body)
		if err != nil || got != want {
			t.Fatalf("%+v %v", got, err)
		}
		if _, err := DecodeInput(append(body, 0, 0)); err == nil {
			t.Fatal("accepted unexpected fields")
		}
	}
}
