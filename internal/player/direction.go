package player

import "math"

func Direction(yaw, pitch float32) [3]float32 {
	y, p := float64(yaw)*math.Pi/180, float64(pitch)*math.Pi/180
	return [3]float32{float32(-math.Sin(y) * math.Cos(p)), float32(-math.Sin(p)), float32(math.Cos(y) * math.Cos(p))}
}
