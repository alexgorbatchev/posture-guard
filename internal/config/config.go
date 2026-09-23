package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Sensitivity multipliers mapping to threshold scale factors.
var SensitivityMultipliers = map[string]float64{
	"low":    1.6,
	"medium": 1.0,
	"high":   0.6,
}

// Settings represents user-tunable persistent configuration.
type Settings struct {
	AlertCooldown      float64 `json:"alert_cooldown"`
	BadPostureDuration float64 `json:"bad_posture_duration"`
	ExtremeAlert       bool    `json:"extreme_alert"`
	ExtremeSound       bool    `json:"extreme_sound"`
	ExtremeFlash       bool    `json:"extreme_flash"`
	Sensitivity        string  `json:"sensitivity"`
}

// Thresholds contains posture geometry detection parameters.
type Thresholds struct {
	ShoulderDropThreshold  float64
	ShoulderWidthThreshold float64
	ShoulderTiltThreshold  float64
	HeadRiseThreshold      float64
	ForwardLeanThreshold   float64
	HeadTiltThreshold      float64
	ForwardHeadThreshold   float64
	VisibilityMin          float64
	CalibrationFrames      int
	LoopSleepSeconds       float64
	ModelPath              string
	Debug                  bool
}

// DefaultSettings returns standard configuration defaults.
func DefaultSettings() Settings {
	return Settings{
		AlertCooldown:      30.0,
		BadPostureDuration: 2.0,
		ExtremeAlert:       false,
		ExtremeSound:       true,
		ExtremeFlash:       true,
		Sensitivity:        "medium",
	}
}

// DefaultThresholds returns default thresholds with environment variable overrides applied.
func DefaultThresholds() Thresholds {
	return Thresholds{
		ShoulderDropThreshold:  envFloat("PG_SHOULDER_DROP", 0.04),
		ShoulderWidthThreshold: envFloat("PG_SHOULDER_WIDTH", 0.06),
		ShoulderTiltThreshold:  envFloat("PG_SHOULDER_TILT", 8.0),
		HeadRiseThreshold:      envFloat("PG_HEAD_RISE", 0.05),
		ForwardLeanThreshold:   envFloat("PG_FORWARD_LEAN", 0.04),
		HeadTiltThreshold:      envFloat("PG_HEAD_TILT", 10.0),
		ForwardHeadThreshold:   envFloat("PG_FORWARD_HEAD", 4.0),
		VisibilityMin:          envFloat("PG_VISIBILITY_MIN", 0.5),
		CalibrationFrames:      envInt("PG_CALIB_FRAMES", 90),
		LoopSleepSeconds:       envFloat("PG_LOOP_SLEEP", 0.067),
		ModelPath:              envString("PG_MODEL_PATH", "pose_landmarker_full.task"),
		Debug:                  envBool("DEBUG", false),
	}
}

// GetMultiplier returns the sensitivity multiplier for the given setting.
func GetMultiplier(sensitivity string) float64 {
	if m, ok := SensitivityMultipliers[strings.ToLower(sensitivity)]; ok {
		return m
	}
	return 1.0
}

// GetConfigPath returns the standard XDG-compliant config file path.
func GetConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "posture-guard", "config.json")
	}
	baseDir, err := os.UserConfigDir()
	if err == nil && baseDir != "" {
		return filepath.Join(baseDir, "posture-guard", "config.json")
	}
	return "posture_guard_settings.json"
}

// LoadSettings loads settings from the given path or default location.
func LoadSettings(path string) (Settings, error) {
	s := DefaultSettings()
	if path == "" {
		path = GetConfigPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, fmt.Errorf("reading settings file %q: %w", path, err)
	}

	if err := json.Unmarshal(data, &s); err != nil {
		return s, fmt.Errorf("parsing settings JSON %q: %w", path, err)
	}

	return s, nil
}

// SaveSettings writes settings to disk at the given path or default location.
func SaveSettings(path string, s Settings) error {
	if path == "" {
		path = GetConfigPath()
	}

	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating config directory %q: %w", dir, err)
		}
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("writing settings file %q: %w", path, err)
	}

	return nil
}

func envString(key string, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func envFloat(key string, fallback float64) float64 {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return fallback
	}
	return f
}

func envInt(key string, fallback int) int {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}

func envBool(key string, fallback bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	lower := strings.ToLower(val)
	return lower == "1" || lower == "true" || lower == "yes"
}
