package posture

import (
	"math"
)

// Sample represents a set of posture metrics extracted from a single frame during calibration.
type Sample struct {
	ShoulderMidY  float64
	ShoulderWidth float64
	ShoulderTilt  float64
	LeanOffset    float64
	NeckYGap      float64
	CVA           float64
	HeadTilt      float64

	HasShoulders bool
	HasHead      bool
}

// Baseline represents the average baseline metrics computed during good-posture calibration.
type Baseline struct {
	ShoulderMidY  float64 `json:"shoulder_mid_y"`
	ShoulderWidth float64 `json:"shoulder_width"`
	ShoulderTilt  float64 `json:"shoulder_tilt"`
	LeanOffset    float64 `json:"lean_offset"`
	NeckYGap      float64 `json:"neck_y_gap"`
	CVA           float64 `json:"cva"`
	HeadTilt      float64 `json:"head_tilt"`
	Valid         bool    `json:"valid"`
}

// CollectSample extracts a calibration sample from normalized landmarks.
func CollectSample(lm []Landmark, width, height int, minVis float64) (Sample, bool) {
	var s Sample
	if len(lm) <= LandmarkRightShoulder {
		return s, false
	}

	w := float64(width)
	h := float64(height)

	if AreVisible(lm, minVis, LandmarkLeftShoulder, LandmarkRightShoulder) {
		ls := lm[LandmarkLeftShoulder].PointInPixels(width, height)
		rs := lm[LandmarkRightShoulder].PointInPixels(width, height)

		s.ShoulderMidY = (lm[LandmarkLeftShoulder].Y + lm[LandmarkRightShoulder].Y) / 2.0
		s.ShoulderWidth = math.Abs(lm[LandmarkLeftShoulder].X - lm[LandmarkRightShoulder].X)
		s.ShoulderTilt = AngleFromHorizontal(ls, rs)

		if AreVisible(lm, minVis, LandmarkNose) {
			midX := (lm[LandmarkLeftShoulder].X + lm[LandmarkRightShoulder].X) / 2.0
			s.LeanOffset = lm[LandmarkNose].X - midX
			s.NeckYGap = lm[LandmarkNose].Y - lm[LandmarkLeftShoulder].Y
		}

		if AreVisible(lm, minVis, LandmarkLeftEar, LandmarkRightEar) {
			earMid := MidPoint(
				lm[LandmarkLeftEar].PointInPixels(width, height),
				lm[LandmarkRightEar].PointInPixels(width, height),
			)
			shoulderMid := MidPoint(ls, rs)
			s.CVA = AngleFromHorizontal(shoulderMid, earMid)
		}

		s.HasShoulders = true
	}

	if AreVisible(lm, minVis, LandmarkLeftEar, LandmarkRightEar) {
		le := Point{X: lm[LandmarkLeftEar].X * w, Y: lm[LandmarkLeftEar].Y * h}
		re := Point{X: lm[LandmarkRightEar].X * w, Y: lm[LandmarkRightEar].Y * h}
		s.HeadTilt = AngleFromHorizontal(le, re)
		s.HasHead = true
	}

	if !s.HasShoulders && !s.HasHead {
		return s, false
	}

	return s, true
}

// BuildBaseline averages collected samples into a calibrated baseline.
func BuildBaseline(samples []Sample) Baseline {
	if len(samples) == 0 {
		return Baseline{Valid: false}
	}

	var sumShoulderMidY, sumShoulderWidth, sumShoulderTilt float64
	var sumLeanOffset, sumNeckYGap, sumCVA, sumHeadTilt float64
	var countShoulders, countHead, countCVA, countNose float64

	for _, s := range samples {
		if s.HasShoulders {
			sumShoulderMidY += s.ShoulderMidY
			sumShoulderWidth += s.ShoulderWidth
			sumShoulderTilt += s.ShoulderTilt
			countShoulders++

			if s.LeanOffset != 0 || s.NeckYGap != 0 {
				sumLeanOffset += s.LeanOffset
				sumNeckYGap += s.NeckYGap
				countNose++
			}
			if s.CVA != 0 {
				sumCVA += s.CVA
				countCVA++
			}
		}
		if s.HasHead {
			sumHeadTilt += s.HeadTilt
			countHead++
		}
	}

	var b Baseline
	if countShoulders > 0 {
		b.ShoulderMidY = sumShoulderMidY / countShoulders
		b.ShoulderWidth = sumShoulderWidth / countShoulders
		b.ShoulderTilt = sumShoulderTilt / countShoulders
	}
	if countNose > 0 {
		b.LeanOffset = sumLeanOffset / countNose
		b.NeckYGap = sumNeckYGap / countNose
	}
	if countCVA > 0 {
		b.CVA = sumCVA / countCVA
	}
	if countHead > 0 {
		b.HeadTilt = sumHeadTilt / countHead
	}

	b.Valid = countShoulders > 0
	return b
}
