package posture

import (
	"math"
)

// Point represents a 2D point in pixel or normalized space.
type Point struct {
	X float64
	Y float64
}

// AngleFromHorizontal calculates the angle of the line connecting p1 to p2 relative to the horizontal axis.
// Returns an angle in degrees in [0, 90] where 0 is perfectly horizontal.
func AngleFromHorizontal(p1, p2 Point) float64 {
	dx := p2.X - p1.X
	dy := p2.Y - p1.Y
	deg := math.Abs(math.Atan2(dy, dx) * 180.0 / math.Pi)
	if deg > 90.0 {
		return 180.0 - deg
	}
	return deg
}

// MidPoint computes the midpoint between two points.
func MidPoint(p1, p2 Point) Point {
	return Point{
		X: (p1.X + p2.X) / 2.0,
		Y: (p1.Y + p2.Y) / 2.0,
	}
}
