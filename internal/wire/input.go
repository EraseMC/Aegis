package wire

type Input struct {
	Session, Time, Tick uint64
	Position            [3]float32
	Pitch, Yaw, HeadYaw float32
	Mode                uint32
	Missed              bool
}

func DecodeInput(body []byte) (Input, error) {
	r := NewReader(body)
	m := Input{Session: r.U64(), Time: r.U64(), Tick: r.U64()}
	for i := range m.Position {
		m.Position[i] = r.F32()
	}
	m.Pitch, m.Yaw, m.HeadYaw = r.F32(), r.F32(), r.F32()
	m.Mode = r.U32()
	m.Missed = r.U8() != 0
	return m, r.Err()
}

func (m Input) Encode() []byte {
	return frame(TypeInput, func(w *Writer) {
		w.U64(m.Session)
		w.U64(m.Time)
		w.U64(m.Tick)
		for _, v := range m.Position {
			w.F32(v)
		}
		w.F32(m.Pitch)
		w.F32(m.Yaw)
		w.F32(m.HeadYaw)
		w.U32(m.Mode)
		if m.Missed {
			w.U8(1)
		} else {
			w.U8(0)
		}
	})
}

type Attack struct{ Session, Time, Target uint64 }

func DecodeAttack(body []byte) (Attack, error) {
	r := NewReader(body)
	m := Attack{Session: r.U64(), Time: r.U64(), Target: r.U64()}
	return m, r.Err()
}
func (m Attack) Encode() []byte {
	return frame(TypeAttack, func(w *Writer) { w.U64(m.Session); w.U64(m.Time); w.U64(m.Target) })
}

func DecodeSwing(body []byte) (Packet, error) {
	r := NewReader(body)
	m := Packet{Session: r.U64(), Time: r.U64()}
	return m, r.Err()
}
