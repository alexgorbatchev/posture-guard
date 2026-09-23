package posture

import (
	"math"
	"sync"
)

// Metrics contains calculated deltas relative to baseline.
type Metrics struct {
	ShoulderDrop float64 `json:"shoulder_drop"`
	ShoulderWidth float64 `json:"shoulder_width"`
	ShoulderTilt float64 `json:"shoulder_tilt"`
	NeckGap      float64 `json:"neck_gap"`
	Lean         float64 `json:"lean"`
	ForwardHead  float64 `json:"forward_head"`
	HeadTilt     float64 `json:"head_tilt"`
}

// ExtractMetrics calculates raw posture deltas for a frame against baseline.
func ExtractMetrics(lm []Landmark, width, height int, baseline Baseline, minVis float64) (Metrics, bool) {
	var m Metrics
	if len(lm) <= LandmarkRightShoulder {
		return m, false
	}

	w := float64(width)
	h := float64(height)
	hasData := false

	if AreVisible(lm, minVis, LandmarkLeftShoulder, LandmarkRightShoulder) {
		ls := lm[LandmarkLeftShoulder].PointInPixels(width, height)
		rs := lm[LandmarkRightShoulder].PointInPixels(width, height)

		midY := (lm[LandmarkLeftShoulder].Y + lm[LandmarkRightShoulder].Y) / 2.0
		midX := (lm[LandmarkLeftShoulder].X + lm[LandmarkRightShoulder].X) / 2.0
		shWidth := math.Abs(lm[LandmarkLeftShoulder].X - lm[LandmarkRightShoulder].X)
		tilt := AngleFromHorizontal(ls, rs)

		m.ShoulderDrop = midY - baseline.ShoulderMidY
		m.ShoulderWidth = baseline.ShoulderWidth - shWidth
		m.ShoulderTilt = tilt - baseline.ShoulderTilt
		hasData = true

		if AreVisible(lm, minVis, LandmarkNose) {
			neckGap := lm[LandmarkNose].Y - lm[LandmarkLeftShoulder].Y
			m.NeckGap = neckGap - baseline.NeckYGap
			m.Lean = math.Abs((lm[LandmarkNose].X - midX) - baseline.LeanOffset)
		}

		if AreVisible(lm, minVis, LandmarkLeftEar, LandmarkRightEar) {
			earMid := MidPoint(
				lm[LandmarkLeftEar].PointInPixels(width, height),
				lm[LandmarkRightEar].PointInPixels(width, height),
			)
			shoulderMid := MidPoint(ls, rs)
			cva := AngleFromHorizontal(shoulderMid, earMid)
			m.ForwardHead = baseline.CVA - cva
		}
	}

	if AreVisible(lm, minVis, LandmarkLeftEar, LandmarkRightEar) {
		le := Point{X: lm[LandmarkLeftEar].X * w, Y: lm[LandmarkLeftEar].Y * h}
		re := Point{X: lm[LandmarkRightEar].X * w, Y: lm[LandmarkRightEar].Y * h}
		tilt := AngleFromHorizontal(le, re)
		m.HeadTilt = tilt - baseline.HeadTilt
		hasData = true
	}

	return m, hasData
}

// RollingHistory maintains a sliding window of metrics to smooth jitter.
type RollingHistory struct {
	mu      sync.Mutex
	window  int
	history []Metrics
}

// NewRollingHistory creates a new history smoother with the specified window size.
func NewRollingHistory(window int) *RollingHistory {
	if window <= 0 {
		window = 8
	}
	return &RollingHistory{
		window:  window,
		history: make([]Metrics, 0, window),
	}
}

// Add appends a new metric sample to the history window.
func (r *RollingHistory) Add(m Metrics) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.history) >= r.window {
		r.history = r.history[1:]
	}
	r.history = append(r.history, m)
}

// Reset clears the metrics history buffer.
func (r *RollingHistory) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.history = r.history[:0]
}

// Average computes the mean delta values across the active history window.
func (r *RollingHistory) Average() Metrics {
	r.mu.Lock()
	defer r.mu.Unlock()

	n := len(r.history)
	if n == 0 {
		return Metrics{}
	}

	var sum Metrics
	for _, m := range r.history {
		sum.ShoulderDrop += m.ShoulderDrop
		sum.ShoulderWidth += m.ShoulderWidth
		sum.ShoulderTilt += m.ShoulderTilt
		sum.NeckGap += m.NeckGap
		sum.Lean += m.Lean
		sum.ForwardHead += m.ForwardHead
		sum.HeadTilt += m.HeadTilt
	}

	count := float64(n)
	return Metrics{
		ShoulderDrop:  sum.ShoulderDrop / count,
		ShoulderWidth: sum.ShoulderWidth / count,
		ShoulderTilt:  sum.ShoulderTilt / count,
		NeckGap:       sum.NeckGap / count,
		Lean:          sum.Lean / count,
		ForwardHead:   sum.ForwardHead / count,
		HeadTilt:      sum.HeadTilt / count,
	}
}
