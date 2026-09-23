package posture

// MediaPipe Pose Landmark indices.
const (
	LandmarkNose          = 0
	LandmarkLeftEyeInner  = 1
	LandmarkLeftEye       = 2
	LandmarkLeftEyeOuter  = 3
	LandmarkRightEyeInner = 4
	LandmarkRightEye      = 5
	LandmarkRightEyeOuter = 6
	LandmarkLeftEar       = 7
	LandmarkRightEar      = 8
	LandmarkMouthLeft     = 9
	LandmarkMouthRight    = 10
	LandmarkLeftShoulder  = 11
	LandmarkRightShoulder = 12
	LandmarkLeftElbow     = 13
	LandmarkRightElbow    = 14
	LandmarkLeftWrist     = 15
	LandmarkRightWrist    = 16
	LandmarkLeftPinky     = 17
	LandmarkRightPinky    = 18
	LandmarkLeftIndex     = 19
	LandmarkRightIndex    = 20
	LandmarkLeftThumb     = 21
	LandmarkRightThumb    = 22
	LandmarkLeftHip       = 23
	LandmarkRightHip      = 24
)

// Landmark represents a normalized 3D pose landmark with visibility score.
type Landmark struct {
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Z          float64 `json:"z"`
	Visibility float64 `json:"visibility"`
}

// PointInPixels converts normalized landmark coordinates (0..1) to pixel space.
func (l Landmark) PointInPixels(width, height int) Point {
	return Point{
		X: l.X * float64(width),
		Y: l.Y * float64(height),
	}
}

// AreVisible checks if all specified landmark indices meet the minimum visibility threshold.
func AreVisible(landmarks []Landmark, minVis float64, indices ...int) bool {
	if len(landmarks) == 0 {
		return false
	}
	for _, idx := range indices {
		if idx < 0 || idx >= len(landmarks) {
			return false
		}
		if landmarks[idx].Visibility < minVis {
			return false
		}
	}
	return true
}
