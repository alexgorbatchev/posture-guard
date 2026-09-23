package camera

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCameraClosed indicates that the camera frame source is closed.
var ErrCameraClosed = errors.New("camera source closed")

// Frame represents a single video capture frame.
type Frame struct {
	Width       int
	Height      int
	TimestampMs int64
	Data        []byte
}

// FrameSource is the interface for acquiring video frames from a camera or stream.
type FrameSource interface {
	ReadFrame(ctx context.Context) (*Frame, error)
	Close() error
}

// MockFrameSource provides deterministic frames for testing and synthetic simulations.
type MockFrameSource struct {
	mu          sync.Mutex
	Width       int
	Height      int
	FrameCount  int
	MaxFrames   int
	TimestampMs int64
	closed      bool
}

// NewMockFrameSource creates a mock frame source producing frames with the given dimensions.
func NewMockFrameSource(width, height, maxFrames int) *MockFrameSource {
	return &MockFrameSource{
		Width:     width,
		Height:    height,
		MaxFrames: maxFrames,
	}
}

// ReadFrame returns the next simulated video frame.
func (m *MockFrameSource) ReadFrame(ctx context.Context) (*Frame, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, ErrCameraClosed
	}

	if m.MaxFrames > 0 && m.FrameCount >= m.MaxFrames {
		return nil, ErrCameraClosed
	}

	m.FrameCount++
	m.TimestampMs += 33 // ~30 fps step

	f := &Frame{
		Width:       m.Width,
		Height:      m.Height,
		TimestampMs: m.TimestampMs,
		Data:        make([]byte, m.Width*m.Height*3),
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return f, nil
	}
}

// Close closes the mock frame source.
func (m *MockFrameSource) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// SyntheticDevice represents a simulated physical camera stream.
type SyntheticDevice struct {
	Width       int
	Height      int
	Interval    time.Duration
	TimestampMs int64
	closed      bool
	mu          sync.Mutex
}

// NewSyntheticDevice creates a synthetic camera device.
func NewSyntheticDevice(width, height int, fps int) *SyntheticDevice {
	if fps <= 0 {
		fps = 30
	}
	return &SyntheticDevice{
		Width:    width,
		Height:   height,
		Interval: time.Duration(1000/fps) * time.Millisecond,
	}
}

// ReadFrame blocks until the next frame interval and returns a frame.
func (s *SyntheticDevice) ReadFrame(ctx context.Context) (*Frame, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, ErrCameraClosed
	}
	s.TimestampMs += 33
	ts := s.TimestampMs
	w, h := s.Width, s.Height
	s.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(s.Interval):
		return &Frame{
			Width:       w,
			Height:      h,
			TimestampMs: ts,
			Data:        make([]byte, w*h*3),
		}, nil
	}
}

// Close closes the synthetic device.
func (s *SyntheticDevice) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}
