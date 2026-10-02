package wire

const (
	EntityUpdate uint8 = iota
	EntityRemove
	Teleport
	Velocity
	BlockChange
)

const (
	StateFlying uint32 = 1 << iota
	StateSpecial
	StateGround
	StateLagging
	StateDead
	StateFrozen
	StateCeiling
)

type Observation struct {
	Session, Time, Target uint64
	Kind                  uint8
	Box                   [6]float32
}

func DecodeObservation(body []byte) (Observation, error) {
	r := NewReader(body)
	m := Observation{Session: r.U64(), Time: r.U64(), Target: r.U64(), Kind: r.U8()}
	for i := range m.Box {
		m.Box[i] = r.F32()
	}
	return m, r.Err()
}

func (m Observation) Encode() []byte {
	return frame(TypeObservation, func(w *Writer) {
		w.U64(m.Session)
		w.U64(m.Time)
		w.U64(m.Target)
		w.U8(m.Kind)
		for _, v := range m.Box {
			w.F32(v)
		}
	})
}

type State struct {
	Session, Time       uint64
	Flags               uint32
	Speed, Jump, Offset float32
	Position            [3]float32
}

func DecodeState(body []byte) (State, error) {
	r := NewReader(body)
	m := State{Session: r.U64(), Time: r.U64(), Flags: r.U32(), Speed: r.F32(), Jump: r.F32(), Offset: r.F32()}
	for i := range m.Position {
		m.Position[i] = r.F32()
	}
	return m, r.Err()
}

func (m State) Encode() []byte {
	return frame(TypeState, func(w *Writer) {
		w.U64(m.Session)
		w.U64(m.Time)
		w.U32(m.Flags)
		w.F32(m.Speed)
		w.F32(m.Jump)
		w.F32(m.Offset)
		for _, v := range m.Position {
			w.F32(v)
		}
	})
}
