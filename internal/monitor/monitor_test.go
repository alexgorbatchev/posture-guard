package monitor

import (
	"context"
	"testing"
	"time"

	"github.com/MatheusDSantossi/posture-guard/internal/alert"
	"github.com/MatheusDSantossi/posture-guard/internal/camera"
	"github.com/MatheusDSantossi/posture-guard/internal/config"
	"github.com/MatheusDSantossi/posture-guard/internal/detector"
	"github.com/MatheusDSantossi/posture-guard/internal/notifier"
)

func TestMonitorCalibrateAndStep(t *testing.T) {
	cam := camera.NewMockFrameSource(640, 480, 100)
	defer cam.Close()

	gen := detector.NewLandmarkGeneratorDetector()
	defer gen.Close()

	notif := &notifier.MockNotifier{}
	alertPl := &alert.MockAlertPlayer{}

	settings := config.DefaultSettings()
	settings.BadPostureDuration = 0.05
	settings.AlertCooldown = 0.1
	settings.ExtremeAlert = true

	thresh := config.DefaultThresholds()
	thresh.CalibrationFrames = 5

	m := New(cam, gen, notif, alertPl, settings, thresh)

	ctx := context.Background()

	// 1. Calibration
	baseline, err := m.Calibrate(ctx, 5)
	if err != nil {
		t.Fatalf("Calibrate failed: %v", err)
	}
	if !baseline.Valid {
		t.Fatalf("expected valid baseline")
	}

	// 2. Step with good posture
	res, err := m.Step(ctx)
	if err != nil {
		t.Fatalf("Step failed: %v", err)
	}
	if !res.IsGood {
		t.Fatalf("expected good posture, got: %v", res.Reasons)
	}

	// 3. Step with bad posture (slouching)
	gen.SetPostureOffset(0.08, 0, 0, 0)
	time.Sleep(60 * time.Millisecond)

	// Step multiple times to trigger bad posture duration threshold
	for i := 0; i < 5; i++ {
		_, err := m.Step(ctx)
		if err != nil {
			t.Fatalf("Step %d failed: %v", i, err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Verify notification received
	time.Sleep(50 * time.Millisecond)
	if len(notif.Calls()) == 0 {
		t.Fatalf("expected notification to be sent for bad posture")
	}
}

func TestMonitorPauseResume(t *testing.T) {
	cam := camera.NewMockFrameSource(640, 480, 50)
	gen := detector.NewLandmarkGeneratorDetector()
	m := New(cam, gen, &notifier.MockNotifier{}, &alert.MockAlertPlayer{}, config.DefaultSettings(), config.DefaultThresholds())

	m.Pause()
	ctx := context.Background()
	res, err := m.Step(ctx)
	if err != nil {
		t.Fatalf("Step during pause failed: %v", err)
	}
	if !res.IsGood {
		t.Fatalf("expected step to return good while paused")
	}

	m.Resume()
}
