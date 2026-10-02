package player

import (
	"github.com/EraseMC/Aegis/internal/wire"
	"math"
)

const MaxTargets = 128
const maxMarkers = 16
const historySize = 16

type Box [6]float32

func (b Box) Valid() bool {
	for _, v := range b {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return false
		}
	}
	return b[0] <= b[3] && b[1] <= b[4] && b[2] <= b[5] && b[3]-b[0] < 16 && b[4]-b[1] < 16 && b[5]-b[2] < 16
}

func (b Box) Distance(pos [3]float32) float64 {
	var total float64
	for i, v := range pos {
		d := float64(max(b[i]-v, v-b[i+3], 0))
		total += d * d
	}
	return math.Sqrt(total)
}

func (b Box) Ray(pos, direction [3]float32, margin float32) bool {
	lo, hi := float64(0), float64(8)
	for i, v := range direction {
		if math.Abs(float64(v)) < 1e-6 {
			if pos[i] < b[i]-margin || pos[i] > b[i+3]+margin {
				return false
			}
			continue
		}
		a, c := float64((b[i]-margin-pos[i])/v), float64((b[i+3]+margin-pos[i])/v)
		if a > c {
			a, c = c, a
		}
		lo = max(lo, a)
		hi = min(hi, c)
		if hi < lo {
			return false
		}
	}
	return true
}

type sample struct {
	Box Box
	At  uint64
}
type target struct {
	Samples     [historySize]sample
	Next, Count int
	Removed     bool
	Last        uint64
}
type marker struct {
	ID      int64
	At      uint64
	Updates []wire.Observation
}

type View struct {
	Targets      map[uint64]*target
	pending      []wire.Observation
	markers      []marker
	LastAck, RTT uint64
	invalid      bool
}

func NewView() View { return View{Targets: make(map[uint64]*target)} }

func (v *View) Observe(update wire.Observation) {
	if len(v.pending) >= MaxTargets*2 {
		*v = NewView()
		v.invalid = true
		return
	}
	if update.Kind == wire.EntityUpdate && !Box(update.Box).Valid() {
		*v = NewView()
		v.invalid = true
		return
	}
	v.pending = append(v.pending, update)
}

func (v *View) Mark(id int64, at uint64) {
	if len(v.markers) >= maxMarkers {
		v.markers = nil
		v.pending = nil
		v.Targets = make(map[uint64]*target)
		v.invalid = true
	}
	v.markers = append(v.markers, marker{ID: id, At: at, Updates: v.pending})
	v.pending = nil
}

func (v *View) Ack(id int64, at uint64) bool {
	index := -1
	for i, m := range v.markers {
		if m.ID == id {
			index = i
			break
		}
	}
	if index < 0 || at < v.markers[index].At || at-v.markers[index].At > 2000 {
		return false
	}
	for _, m := range v.markers[:index+1] {
		for _, u := range m.Updates {
			if u.Kind != wire.EntityUpdate && u.Kind != wire.EntityRemove {
				continue
			}
			t := v.Targets[u.Target]
			if t == nil {
				if len(v.Targets) >= MaxTargets {
					var oldest uint64
					age := uint64(math.MaxUint64)
					for id, entry := range v.Targets {
						if entry.Last < age {
							oldest, age = id, entry.Last
						}
					}
					delete(v.Targets, oldest)
				}
				t = &target{}
				v.Targets[u.Target] = t
			}
			t.Removed = u.Kind == wire.EntityRemove
			t.Last = at
			if t.Removed {
				t.Count = 0
				continue
			}
			t.Samples[t.Next] = sample{Box: Box(u.Box), At: at}
			t.Next = (t.Next + 1) % historySize
			t.Count = min(t.Count+1, historySize)
		}
	}
	v.RTT = at - v.markers[index].At
	v.LastAck = at
	v.markers = append(v.markers[:0], v.markers[index+1:]...)
	v.invalid = false
	return true
}

func (v *View) Synced(at uint64) bool {
	return !v.invalid && v.LastAck > 0 && at >= v.LastAck && at-v.LastAck <= 1000 && v.RTT <= 500
}

func (v *View) Distance(id, at uint64, pos [3]float32, directions [][3]float32) (float64, bool, bool) {
	if !v.Synced(at) {
		return 0, false, false
	}
	t := v.Targets[id]
	if t == nil || t.Removed || t.Count == 0 || at < t.Last {
		return 0, false, false
	}
	distance, hit := math.Inf(1), false
	var previous Box
	for i := 0; i < t.Count; i++ {
		s := t.Samples[(t.Next-1-i+historySize)%historySize]
		if i > 0 && at-s.At > min(uint64(650), v.RTT+350) {
			break
		}
		box := s.Box
		if i > 0 {
			// Include the swept envelope: clients interpolate between acknowledged positions.
			for axis := 0; axis < 3; axis++ {
				box[axis] = min(box[axis], previous[axis])
				box[axis+3] = max(box[axis+3], previous[axis+3])
			}
		}
		previous = s.Box
		distance = min(distance, box.Distance(pos))
		for _, d := range directions {
			hit = hit || box.Ray(pos, d, 0.35)
		}
	}
	// Unacknowledged moves enlarge the acceptable envelope but never authorize a new entity.
	for _, m := range v.markers {
		for _, u := range m.Updates {
			if u.Target == id && u.Kind == wire.EntityUpdate {
				distance = min(distance, Box(u.Box).Distance(pos))
				for _, d := range directions {
					hit = hit || Box(u.Box).Ray(pos, d, 0.35)
				}
			}
		}
	}
	for _, u := range v.pending {
		if u.Target == id && u.Kind == wire.EntityUpdate {
			distance = min(distance, Box(u.Box).Distance(pos))
			for _, d := range directions {
				hit = hit || Box(u.Box).Ray(pos, d, 0.35)
			}
		}
	}
	return distance, hit, !math.IsInf(distance, 1)
}
