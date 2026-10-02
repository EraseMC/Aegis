package wire

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const Version uint16 = 1

const MaxFrameSize = 8 << 20

type Type uint8

const (
	TypeHello       Type = 0x01
	TypeJoin        Type = 0x02
	TypeQuit        Type = 0x03
	TypeServerbound Type = 0x04
	TypeClientbound Type = 0x05
	TypeTick        Type = 0x06

	TypeWelcome Type = 0x81
	TypeFlag    Type = 0x82
	TypePunish  Type = 0x83
)

type Action uint8

const (
	ActionNone Action = iota
	ActionKick
	ActionBan
)

type Hello struct {
	Version  uint16
	ShieldID int32
	Server   string
}

type Join struct {
	Session   uint64
	Name      string
	XUID      string
	Protocol  uint32
	Version   string
	DeviceOS  int32
	InputMode uint32
}

type Quit struct {
	Session uint64
}

type Packet struct {
	Session uint64
	Time    uint64
	Payload []byte
}

type Tick struct {
	Tick uint64
	Time uint64
}

type Welcome struct {
	Version uint16
}

type Flag struct {
	Session    uint64
	Check      string
	Violations float32
	Max        float32
	Data       string
}

type Punish struct {
	Session    uint64
	Check      string
	Violations float32
	Action     Action
	Data       string
}

var ErrShort = errors.New("frame is shorter than its fields")

type Reader struct {
	buf []byte
	off int
	err error
}

func NewReader(buf []byte) *Reader {
	return &Reader{buf: buf}
}

func (r *Reader) Err() error {
	if r.err == nil && r.off != len(r.buf) {
		return fmt.Errorf("%d trailing bytes", len(r.buf)-r.off)
	}
	return r.err
}

func (r *Reader) take(n int) []byte {
	if r.err != nil || n < 0 || r.off+n > len(r.buf) {
		r.err = ErrShort
		return nil
	}
	b := r.buf[r.off : r.off+n]
	r.off += n
	return b
}

func (r *Reader) U8() uint8 {
	if b := r.take(1); b != nil {
		return b[0]
	}
	return 0
}

func (r *Reader) U16() uint16 {
	if b := r.take(2); b != nil {
		return binary.LittleEndian.Uint16(b)
	}
	return 0
}

func (r *Reader) U32() uint32 {
	if b := r.take(4); b != nil {
		return binary.LittleEndian.Uint32(b)
	}
	return 0
}

func (r *Reader) U64() uint64 {
	if b := r.take(8); b != nil {
		return binary.LittleEndian.Uint64(b)
	}
	return 0
}

func (r *Reader) String() string {
	return string(r.take(int(r.U16())))
}

func (r *Reader) Bytes() []byte {
	return r.take(int(r.U32()))
}

type Writer struct {
	buf []byte
}

func (w *Writer) U8(v uint8) {
	w.buf = append(w.buf, v)
}

func (w *Writer) U16(v uint16) {
	w.buf = binary.LittleEndian.AppendUint16(w.buf, v)
}

func (w *Writer) U32(v uint32) {
	w.buf = binary.LittleEndian.AppendUint32(w.buf, v)
}

func (w *Writer) U64(v uint64) {
	w.buf = binary.LittleEndian.AppendUint64(w.buf, v)
}

func (w *Writer) F32(v float32) {
	w.U32(math.Float32bits(v))
}

func (w *Writer) String(v string) {
	if len(v) > math.MaxUint16 {
		v = v[:math.MaxUint16]
	}
	w.U16(uint16(len(v)))
	w.buf = append(w.buf, v...)
}

func (w *Writer) Bytes(v []byte) {
	w.U32(uint32(len(v)))
	w.buf = append(w.buf, v...)
}

func ReadFrame(r io.Reader, scratch []byte) (Type, []byte, error) {
	var head [5]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		return 0, nil, err
	}
	size := binary.LittleEndian.Uint32(head[:4])
	if size == 0 || size > MaxFrameSize {
		return 0, nil, fmt.Errorf("frame of %d bytes is out of bounds", size)
	}
	if cap(scratch) < int(size)-1 {
		scratch = make([]byte, size-1)
	}
	body := scratch[:size-1]
	if _, err := io.ReadFull(r, body); err != nil {
		return 0, nil, err
	}
	return Type(head[4]), body, nil
}

func frame(t Type, fill func(w *Writer)) []byte {
	w := &Writer{buf: make([]byte, 5, 64)}
	fill(w)
	binary.LittleEndian.PutUint32(w.buf[:4], uint32(len(w.buf)-4))
	w.buf[4] = byte(t)
	return w.buf
}

func (m Hello) Encode() []byte {
	return frame(TypeHello, func(w *Writer) {
		w.U16(m.Version)
		w.U32(uint32(m.ShieldID))
		w.String(m.Server)
	})
}

func (m Join) Encode() []byte {
	return frame(TypeJoin, func(w *Writer) {
		w.U64(m.Session)
		w.String(m.Name)
		w.String(m.XUID)
		w.U32(m.Protocol)
		w.String(m.Version)
		w.U32(uint32(m.DeviceOS))
		w.U32(m.InputMode)
	})
}

func (m Quit) Encode() []byte {
	return frame(TypeQuit, func(w *Writer) { w.U64(m.Session) })
}

func (m Packet) Encode(t Type) []byte {
	return frame(t, func(w *Writer) {
		w.U64(m.Session)
		w.U64(m.Time)
		w.Bytes(m.Payload)
	})
}

func (m Tick) Encode() []byte {
	return frame(TypeTick, func(w *Writer) {
		w.U64(m.Tick)
		w.U64(m.Time)
	})
}

func (m Welcome) Encode() []byte {
	return frame(TypeWelcome, func(w *Writer) { w.U16(m.Version) })
}

func (m Flag) Encode() []byte {
	return frame(TypeFlag, func(w *Writer) {
		w.U64(m.Session)
		w.String(m.Check)
		w.F32(m.Violations)
		w.F32(m.Max)
		w.String(m.Data)
	})
}

func (m Punish) Encode() []byte {
	return frame(TypePunish, func(w *Writer) {
		w.U64(m.Session)
		w.String(m.Check)
		w.F32(m.Violations)
		w.U8(uint8(m.Action))
		w.String(m.Data)
	})
}

func DecodeHello(b []byte) (Hello, error) {
	r := NewReader(b)
	m := Hello{Version: r.U16(), ShieldID: int32(r.U32()), Server: r.String()}
	return m, r.Err()
}

func DecodeJoin(b []byte) (Join, error) {
	r := NewReader(b)
	m := Join{Session: r.U64(), Name: r.String(), XUID: r.String(), Protocol: r.U32(), Version: r.String(), DeviceOS: int32(r.U32()), InputMode: r.U32()}
	return m, r.Err()
}

func DecodeQuit(b []byte) (Quit, error) {
	r := NewReader(b)
	m := Quit{Session: r.U64()}
	return m, r.Err()
}

func DecodePacket(b []byte) (Packet, error) {
	r := NewReader(b)
	m := Packet{Session: r.U64(), Time: r.U64(), Payload: r.Bytes()}
	return m, r.Err()
}

func DecodeTick(b []byte) (Tick, error) {
	r := NewReader(b)
	m := Tick{Tick: r.U64(), Time: r.U64()}
	return m, r.Err()
}
