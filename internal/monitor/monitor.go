package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MatheusDSantossi/posture-guard/internal/alert"
	"github.com/MatheusDSantossi/posture-guard/internal/camera"
	"github.com/MatheusDSantossi/posture-guard/internal/config"
	"github.com/MatheusDSantossi/posture-guard/internal/detector"
	"github.com/MatheusDSantossi/posture-guard/internal/notifier"
	"github.com/MatheusDSantossi/posture-guard/internal/posture"
)

// Monitor coordinates camera capture, pose detection, posture analysis, and alerts.
type Monitor struct {
	mu           sync.RWMutex
	Camera       camera.FrameSource
	Detector     detector.PoseDetector
	Notifier     notifier.Notifier
	AlertPlayer  alert.Player
	Settings     config.Settings
	Thresholds   config.Thresholds
	Baseline     posture.Baseline
	History      *posture.RollingHistory
	running      bool
	paused       bool
	recalibrate  bool
	badStartTime *time.Time
	lastAlert    time.Time
	lastResult   posture.PostureResult

	// OnPostureChange callback for UI/CLI event listeners
	OnPostureChange func(res posture.PostureResult)
}

// New creates a new Monitor instance.
func New(
	cam camera.FrameSource,
	det detector.PoseDetector,
	notif notifier.Notifier,
	alertPlayer alert.Player,
	settings config.Settings,
	thresh config.Thresholds,
) *Monitor {
	return &Monitor{
		Camera:      cam,
		Detector:    det,
		Notifier:    notif,
		AlertPlayer: alertPlayer,
		Settings:    settings,
		Thresholds:  thresh,
		History:     posture.NewRollingHistory(8),
	}
}

// Calibrate runs the calibration routine to establish the baseline good posture.
func (m *Monitor) Calibrate(ctx context.Context, frameCount int) (posture.Baseline, error) {
	if frameCount <= 0 {
		frameCount = m.Thresholds.CalibrationFrames
	}
	if frameCount <= 0 {
		frameCount = 90
	}

	var samples []posture.Sample
	m.History.Reset()

	for len(samples) < frameCount {
		select {
		case <-ctx.Done():
			return posture.Baseline{}, ctx.Err()
		default:
		}

		frame, err := m.Camera.ReadFrame(ctx)
		if err != nil {
			return posture.Baseline{}, fmt.Errorf("reading calibration frame: %w", err)
		}

		lm, err := m.Detector.Detect(ctx, frame)
		if err != nil {
			return posture.Baseline{}, fmt.Errorf("detecting landmarks during calibration: %w", err)
		}

		if len(lm) > 0 {
			if s, ok := posture.CollectSample(lm, frame.Width, frame.Height, m.Thresholds.VisibilityMin); ok {
				samples = append(samples, s)
			}
		}
	}

	base := posture.BuildBaseline(samples)
	m.mu.Lock()
	m.Baseline = base
	m.mu.Unlock()

	return base, nil
}

// Step processes a single video frame and evaluates posture state.
func (m *Monitor) Step(ctx context.Context) (posture.PostureResult, error) {
	m.mu.Lock()
	if m.paused {
		m.mu.Unlock()
		return posture.PostureResult{IsGood: true}, nil
	}
	baseline := m.Baseline
	thresh := m.Thresholds
	settings := m.Settings
	m.mu.Unlock()

	frame, err := m.Camera.ReadFrame(ctx)
	if err != nil {
		return posture.PostureResult{}, fmt.Errorf("reading camera frame: %w", err)
	}

	lm, err := m.Detector.Detect(ctx, frame)
	if err != nil {
		return posture.PostureResult{}, fmt.Errorf("detecting pose: %w", err)
	}

	now := time.Now()
	res := posture.CheckPosture(lm, frame.Width, frame.Height, baseline, m.History, thresh, settings)

	m.mu.Lock()
	m.lastResult = res
	if !res.IsGood {
		if m.badStartTime == nil {
			m.badStartTime = &now
		} else if now.Sub(*m.badStartTime).Seconds() >= settings.BadPostureDuration {
			if now.Sub(m.lastAlert).Seconds() >= settings.AlertCooldown {
				m.lastAlert = now
				m.triggerAlert(ctx, res.ReasonString(), settings)
			}
		}
	} else {
		m.badStartTime = nil
	}
	callback := m.OnPostureChange
	m.mu.Unlock()

	if callback != nil {
		callback(res)
	}

	return res, nil
}

func (m *Monitor) triggerAlert(ctx context.Context, reason string, s config.Settings) {
	if m.Notifier != nil {
		go func() {
			_ = m.Notifier.Notify(ctx, "PostureGuard", reason)
		}()
	}

	if s.ExtremeAlert && m.AlertPlayer != nil {
		if s.ExtremeSound {
			go func() {
				_ = m.AlertPlayer.PlayRandomSound(ctx)
			}()
		}
		if s.ExtremeFlash {
			go func() {
				_ = m.AlertPlayer.FlashScreen(ctx)
			}()
		}
	}
}

// Run executes the continuous background monitoring loop.
func (m *Monitor) Run(ctx context.Context) error {
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	sleepDuration := time.Duration(m.Thresholds.LoopSleepSeconds * float64(time.Second))
	if sleepDuration <= 0 {
		sleepDuration = 67 * time.Millisecond
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		m.mu.RLock()
		running := m.running
		recalib := m.recalibrate
		paused := m.paused
		m.mu.RUnlock()

		if !running {
			return nil
		}

		if recalib {
			m.mu.Lock()
			m.recalibrate = false
			m.badStartTime = nil
			m.mu.Unlock()
			if _, err := m.Calibrate(ctx, m.Thresholds.CalibrationFrames); err != nil {
				return fmt.Errorf("recalibration failed: %w", err)
			}
			continue
		}

		if paused {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
			}
			continue
		}

		if _, err := m.Step(ctx); err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sleepDuration):
		}
	}
}

// Pause pauses posture monitoring.
func (m *Monitor) Pause() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paused = true
}

// Resume resumes posture monitoring.
func (m *Monitor) Resume() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.paused = false
	m.badStartTime = nil
	m.History.Reset()
}

// RequestRecalibrate flags the monitor to run calibration on the next cycle.
func (m *Monitor) RequestRecalibrate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.recalibrate = true
}

// Stop stops the monitor.
func (m *Monitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.running = false
}

// LastResult returns the most recent posture evaluation result.
func (m *Monitor) LastResult() posture.PostureResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastResult
}
