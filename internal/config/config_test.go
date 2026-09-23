package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.AlertCooldown != 30.0 {
		t.Fatalf("expected AlertCooldown 30.0, got %v", s.AlertCooldown)
	}
	if s.BadPostureDuration != 2.0 {
		t.Fatalf("expected BadPostureDuration 2.0, got %v", s.BadPostureDuration)
	}
	if s.Sensitivity != "medium" {
		t.Fatalf("expected Sensitivity 'medium', got %q", s.Sensitivity)
	}
}

func TestSensitivityMultiplier(t *testing.T) {
	tests := []struct {
		name        string
		sensitivity string
		want        float64
	}{
		{"low", "low", 1.6},
		{"medium", "medium", 1.0},
		{"high", "high", 0.6},
		{"case-insensitive", "HIGH", 0.6},
		{"unknown fallback", "unknown", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMultiplier(tt.sensitivity)
			if got != tt.want {
				t.Fatalf("GetMultiplier(%q) = %v, want %v", tt.sensitivity, got, tt.want)
			}
		})
	}
}

func TestSaveAndLoadSettings(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "sub", "config.json")

	custom := Settings{
		AlertCooldown:      45.0,
		BadPostureDuration: 3.5,
		ExtremeAlert:       true,
		ExtremeSound:       false,
		ExtremeFlash:       true,
		Sensitivity:        "high",
	}

	if err := SaveSettings(cfgPath, custom); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	loaded, err := LoadSettings(cfgPath)
	if err != nil {
		t.Fatalf("LoadSettings failed: %v", err)
	}

	if loaded != custom {
		t.Fatalf("loaded %+v does not match saved %+v", loaded, custom)
	}
}

func TestDefaultThresholdsEnv(t *testing.T) {
	t.Setenv("PG_SHOULDER_DROP", "0.08")
	t.Setenv("PG_CALIB_FRAMES", "120")
	t.Setenv("DEBUG", "1")

	thresh := DefaultThresholds()
	if thresh.ShoulderDropThreshold != 0.08 {
		t.Fatalf("expected ShoulderDropThreshold 0.08, got %v", thresh.ShoulderDropThreshold)
	}
	if thresh.CalibrationFrames != 120 {
		t.Fatalf("expected CalibrationFrames 120, got %d", thresh.CalibrationFrames)
	}
	if !thresh.Debug {
		t.Fatalf("expected Debug true, got false")
	}
}

func TestLoadNonExistentReturnsDefault(t *testing.T) {
	loaded, err := LoadSettings(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded.AlertCooldown != 30.0 {
		t.Fatalf("expected default AlertCooldown 30.0, got %v", loaded.AlertCooldown)
	}
}
