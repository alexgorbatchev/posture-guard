package posture

import (
	"fmt"
	"strings"

	"github.com/MatheusDSantossi/posture-guard/internal/config"
)

// PostureResult contains the outcome of a posture evaluation frame.
type PostureResult struct {
	IsGood  bool     `json:"is_good"`
	Reasons []string `json:"reasons"`
	Metrics Metrics  `json:"metrics"`
}

// ReasonString returns a combined human-readable string of detected posture issues.
func (p PostureResult) ReasonString() string {
	if p.IsGood || len(p.Reasons) == 0 {
		return "Good posture"
	}
	return strings.Join(p.Reasons, " | ")
}

// CheckPosture evaluates the current landmarks against baseline thresholds and smoothed history.
func CheckPosture(
	lm []Landmark,
	width, height int,
	baseline Baseline,
	history *RollingHistory,
	thresh config.Thresholds,
	settings config.Settings,
) PostureResult {
	if !baseline.Valid {
		return PostureResult{IsGood: true, Reasons: nil}
	}

	raw, ok := ExtractMetrics(lm, width, height, baseline, thresh.VisibilityMin)
	if !ok {
		return PostureResult{IsGood: true, Reasons: nil}
	}

	if history != nil {
		history.Add(raw)
		raw = history.Average()
	}

	mult := config.GetMultiplier(settings.Sensitivity)
	var reasons []string

	if raw.ShoulderDrop > thresh.ShoulderDropThreshold*mult {
		reasons = append(reasons, fmt.Sprintf("Shoulders dropped (%+.3f)", raw.ShoulderDrop))
	}
	if raw.ShoulderWidth > thresh.ShoulderWidthThreshold*mult {
		reasons = append(reasons, fmt.Sprintf("Hunching forward (%.3f)", raw.ShoulderWidth))
	}
	if raw.ShoulderTilt > thresh.ShoulderTiltThreshold*mult {
		reasons = append(reasons, fmt.Sprintf("Shoulders uneven (%+.1f°)", raw.ShoulderTilt))
	}
	if raw.NeckGap > thresh.HeadRiseThreshold*mult {
		reasons = append(reasons, fmt.Sprintf("Head forward (%+.3f)", raw.NeckGap))
	}
	if raw.Lean > thresh.ForwardLeanThreshold*mult {
		reasons = append(reasons, fmt.Sprintf("Leaning (%.3f)", raw.Lean))
	}
	if raw.HeadTilt > thresh.HeadTiltThreshold*mult {
		reasons = append(reasons, fmt.Sprintf("Head tilted (%+.1f°)", raw.HeadTilt))
	}
	if raw.ForwardHead > thresh.ForwardHeadThreshold*mult {
		reasons = append(reasons, fmt.Sprintf("Head pushed forward (%+.1f°)", raw.ForwardHead))
	}

	return PostureResult{
		IsGood:  len(reasons) == 0,
		Reasons: reasons,
		Metrics: raw,
	}
}
