package detector

import (
	"context"
	"sync"

	"github.com/MatheusDSantossi/posture-guard/internal/camera"
	"github.com/MatheusDSantossi/posture-guard/internal/posture"
)

// PoseDetector abstracts pose landmark estimation from camera frames.
type PoseDetector interface {
	Detect(ctx context.Context, frame *camera.Frame) ([]posture.Landmark, error)
	Close() error
}

// MockPoseDetector provides scripted landmarks for test suites.
type MockPoseDetector struct {
	mu        sync.Mutex
	Landmarks [][]posture.Landmark
	index     int
	closed    bool
}

// NewMockPoseDetector creates a mock detector that cycles through the provided landmark slices.
func NewMockPoseDetector(scripted [][]posture.Landmark) *MockPoseDetector {
	return &MockPoseDetector{
		Landmarks: scripted,
	}
}

// Detect returns the next scripted set of landmarks.
func (m *MockPoseDetector) Detect(_ context.Context, _ *camera.Frame) ([]posture.Landmark, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Landmarks) == 0 {
		return nil, nil
	}

	lm := m.Landmarks[m.index%len(m.Landmarks)]
	m.index++
	return lm, nil
}

// Close marks the mock detector closed.
func (m *MockPoseDetector) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// LandmarkGeneratorDetector produces synthetic posture landmarks for testing and simulation.
type LandmarkGeneratorDetector struct {
	mu           sync.Mutex
	ShoulderDrop float64
	ShoulderHunch float64
	LeanOffset   float64
	HeadTilt     float64
}

// NewLandmarkGeneratorDetector creates a generator for synthetic landmark streams.
func NewLandmarkGeneratorDetector() *LandmarkGeneratorDetector {
	return &LandmarkGeneratorDetector{}
}

// SetPostureOffset adjusts the generated synthetic landmark offsets.
func (g *LandmarkGeneratorDetector) SetPostureOffset(drop, hunch, lean, tilt float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.ShoulderDrop = drop
	g.ShoulderHunch = hunch
	g.LeanOffset = lean
	g.HeadTilt = tilt
}

// Detect generates a standard set of 33 landmarks with configured offsets.
func (g *LandmarkGeneratorDetector) Detect(_ context.Context, _ *camera.Frame) ([]posture.Landmark, error) {
	g.mu.Lock()
	drop := g.ShoulderDrop
	hunch := g.ShoulderHunch
	lean := g.LeanOffset
	tilt := g.HeadTilt
	g.mu.Unlock()

	lm := make([]posture.Landmark, 33)
	// Base landmarks
	lm[posture.LandmarkNose] = posture.Landmark{X: 0.50 + lean, Y: 0.25, Z: 0, Visibility: 0.99}
	lm[posture.LandmarkLeftEar] = posture.Landmark{X: 0.45, Y: 0.22 - tilt, Z: 0, Visibility: 0.99}
	lm[posture.LandmarkRightEar] = posture.Landmark{X: 0.55, Y: 0.22 + tilt, Z: 0, Visibility: 0.99}
	lm[posture.LandmarkLeftShoulder] = posture.Landmark{X: 0.40 + hunch, Y: 0.45 + drop, Z: 0, Visibility: 0.99}
	lm[posture.LandmarkRightShoulder] = posture.Landmark{X: 0.60 - hunch, Y: 0.45 + drop, Z: 0, Visibility: 0.99}

	return lm, nil
}

// Close closes the generator.
func (g *LandmarkGeneratorDetector) Close() error {
	return nil
}
