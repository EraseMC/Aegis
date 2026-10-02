package wire

import (
	"bytes"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	join := Join{Session: 42, Name: "Steve", XUID: "2535", Protocol: 2193, Version: "1.26.50", DeviceOS: 7, InputMode: 1}
	pk := Packet{Session: 42, Time: 1234, Payload: []byte{0x90, 0x01, 0xff}}
	stream := bytes.NewBuffer(nil)
	stream.Write(join.Encode())
	stream.Write(pk.Encode(TypeServerbound))

	typ, body, err := ReadFrame(stream, nil)
	if err != nil || typ != TypeJoin {
		t.Fatalf("join frame: %v %v", typ, err)
	}
	if got, err := DecodeJoin(body); err != nil || got != join {
		t.Fatalf("join decode: %+v %v", got, err)
	}

	typ, body, err = ReadFrame(stream, nil)
	if err != nil || typ != TypeServerbound {
		t.Fatalf("packet frame: %v %v", typ, err)
	}
	got, err := DecodePacket(body)
	if err != nil || got.Session != 42 || got.Time != 1234 || !bytes.Equal(got.Payload, pk.Payload) {
		t.Fatalf("packet decode: %+v %v", got, err)
	}
}

func TestShortFrame(t *testing.T) {
	if _, err := DecodeJoin([]byte{1, 2, 3}); err == nil {
		t.Fatal("short join decoded without an error")
	}
}
