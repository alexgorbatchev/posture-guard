package detector

import (
	"context"
	"testing"

	"github.com/MatheusDSantossi/posture-guard/internal/camera"
	"github.com/MatheusDSantossi/posture-guard/internal/posture"
)

func TestMockPoseDetector(t *testing.T) {
	scripted := [][]posture.Landmark{
		{{X: 0.1, Y: 0.1, Visibility: 0.9}},
		{{X: 0.2, Y: 0.2, Visibility: 0.9}},
	}

	det := NewMockPoseDetector(scripted)
	defer det.Close()

	ctx := context.Background()
	frame := &camera.Frame{Width: 640, Height: 480}

	lm1, err := det.Detect(ctx, frame)
	if err != nil || len(lm1) != 1 || lm1[0].X != 0.1 {
		t.Fatalf("first detect unexpected: %v, err=%v", lm1, err)
	}

	lm2, err := det.Detect(ctx, frame)
	if err != nil || len(lm2) != 1 || lm2[0].X != 0.2 {
		t.Fatalf("second detect unexpected: %v, err=%v", lm2, err)
	}
}

func TestLandmarkGeneratorDetector(t *testing.T) {
	gen := NewLandmarkGeneratorDetector()
	defer gen.Close()

	gen.SetPostureOffset(0.05, 0.02, 0.03, 0.01)

	ctx := context.Background()
	frame := &camera.Frame{Width: 640, Height: 480}

	lm, err := gen.Detect(ctx, frame)
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if len(lm) != 33 {
		t.Fatalf("expected 33 landmarks, got %d", len(lm))
	}
	if lm[posture.LandmarkLeftShoulder].Y != 0.45+0.05 {
		t.Fatalf("expected left shoulder Y %v, got %v", 0.50, lm[posture.LandmarkLeftShoulder].Y)
	}
}
