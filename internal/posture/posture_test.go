package posture

import (
	"math"
	"testing"

	"github.com/MatheusDSantossi/posture-guard/internal/config"
)

func TestAngleFromHorizontal(t *testing.T) {
	tests := []struct {
		name string
		p1   Point
		p2   Point
		want float64
	}{
		{"horizontal", Point{0, 10}, Point{10, 10}, 0.0},
		{"vertical", Point{10, 0}, Point{10, 10}, 90.0},
		{"45 degrees", Point{0, 0}, Point{10, 10}, 45.0},
		{"negative 45 degrees", Point{0, 10}, Point{10, 0}, 45.0},
		{"reversed horizontal", Point{20, 10}, Point{10, 10}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AngleFromHorizontal(tt.p1, tt.p2)
			if math.Abs(got-tt.want) > 0.001 {
				t.Fatalf("AngleFromHorizontal(%v, %v) = %v, want %v", tt.p1, tt.p2, got, tt.want)
			}
		})
	}
}

func TestMidPoint(t *testing.T) {
	p1 := Point{X: 10, Y: 20}
	p2 := Point{X: 30, Y: 40}
	mid := MidPoint(p1, p2)
	if mid.X != 20 || mid.Y != 30 {
		t.Fatalf("MidPoint(%v, %v) = %v, want (20, 30)", p1, p2, mid)
	}
}

func createStandardLandmarks() []Landmark {
	lm := make([]Landmark, 33)
	// Nose
	lm[LandmarkNose] = Landmark{X: 0.50, Y: 0.25, Z: 0, Visibility: 0.95}
	// Ears
	lm[LandmarkLeftEar] = Landmark{X: 0.45, Y: 0.22, Z: 0, Visibility: 0.95}
	lm[LandmarkRightEar] = Landmark{X: 0.55, Y: 0.22, Z: 0, Visibility: 0.95}
	// Shoulders
	lm[LandmarkLeftShoulder] = Landmark{X: 0.40, Y: 0.45, Z: 0, Visibility: 0.95}
	lm[LandmarkRightShoulder] = Landmark{X: 0.60, Y: 0.45, Z: 0, Visibility: 0.95}
	return lm
}

func TestCalibrationAndBaseline(t *testing.T) {
	lm := createStandardLandmarks()
	sample, ok := CollectSample(lm, 640, 480, 0.5)
	if !ok {
		t.Fatalf("CollectSample failed on valid landmarks")
	}

	if sample.ShoulderMidY != 0.45 {
		t.Fatalf("expected ShoulderMidY 0.45, got %v", sample.ShoulderMidY)
	}
	if math.Abs(sample.ShoulderWidth-0.20) > 0.001 {
		t.Fatalf("expected ShoulderWidth 0.20, got %v", sample.ShoulderWidth)
	}

	samples := []Sample{sample, sample}
	baseline := BuildBaseline(samples)
	if !baseline.Valid {
		t.Fatalf("expected baseline to be valid")
	}
	if baseline.ShoulderMidY != 0.45 {
		t.Fatalf("expected baseline ShoulderMidY 0.45, got %v", baseline.ShoulderMidY)
	}
}

func TestRollingHistory(t *testing.T) {
	rh := NewRollingHistory(3)
	rh.Add(Metrics{ShoulderDrop: 0.02})
	rh.Add(Metrics{ShoulderDrop: 0.04})
	rh.Add(Metrics{ShoulderDrop: 0.06})

	avg := rh.Average()
	if math.Abs(avg.ShoulderDrop-0.04) > 0.0001 {
		t.Fatalf("expected average 0.04, got %v", avg.ShoulderDrop)
	}

	// Add a 4th metric, window of 3 drops the first (0.02)
	rh.Add(Metrics{ShoulderDrop: 0.08})
	avg = rh.Average()
	if math.Abs(avg.ShoulderDrop-0.06) > 0.0001 {
		t.Fatalf("expected average 0.06, got %v", avg.ShoulderDrop)
	}

	rh.Reset()
	avg = rh.Average()
	if avg.ShoulderDrop != 0 {
		t.Fatalf("expected 0 after reset, got %v", avg.ShoulderDrop)
	}
}

func TestCheckPostureConditions(t *testing.T) {
	thresh := config.DefaultThresholds()
	settings := config.DefaultSettings()

	baseLM := createStandardLandmarks()
	sample, _ := CollectSample(baseLM, 640, 480, 0.5)
	baseline := BuildBaseline([]Sample{sample})

	t.Run("Good posture", func(t *testing.T) {
		history := NewRollingHistory(8)
		res := CheckPosture(baseLM, 640, 480, baseline, history, thresh, settings)
		if !res.IsGood {
			t.Fatalf("expected good posture, got: %v", res.Reasons)
		}
		if res.ReasonString() != "Good posture" {
			t.Fatalf("expected 'Good posture', got %q", res.ReasonString())
		}
	})

	t.Run("Shoulder drop (slouching)", func(t *testing.T) {
		slouchedLM := createStandardLandmarks()
		// Drop shoulders down by Y +0.07 (threshold is 0.04)
		slouchedLM[LandmarkLeftShoulder].Y += 0.07
		slouchedLM[LandmarkRightShoulder].Y += 0.07

		history := NewRollingHistory(1)
		res := CheckPosture(slouchedLM, 640, 480, baseline, history, thresh, settings)
		if res.IsGood {
			t.Fatalf("expected bad posture for shoulder drop, got good posture")
		}
		found := false
		for _, r := range res.Reasons {
			if len(r) > 0 && r[:8] == "Shoulder" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected reason for shoulders dropped, got: %v", res.Reasons)
		}
	})

	t.Run("Hunching forward (shoulder narrowing)", func(t *testing.T) {
		hunchedLM := createStandardLandmarks()
		// Narrow shoulders (width shrinks from 0.20 to 0.10, delta = 0.10 > 0.06)
		hunchedLM[LandmarkLeftShoulder].X = 0.45
		hunchedLM[LandmarkRightShoulder].X = 0.55

		history := NewRollingHistory(1)
		res := CheckPosture(hunchedLM, 640, 480, baseline, history, thresh, settings)
		if res.IsGood {
			t.Fatalf("expected bad posture for hunching, got good posture")
		}
	})

	t.Run("Leaning laterally", func(t *testing.T) {
		leaningLM := createStandardLandmarks()
		// Move nose far to the right
		leaningLM[LandmarkNose].X += 0.08

		history := NewRollingHistory(1)
		res := CheckPosture(leaningLM, 640, 480, baseline, history, thresh, settings)
		if res.IsGood {
			t.Fatalf("expected bad posture for leaning, got good posture")
		}
	})
}
