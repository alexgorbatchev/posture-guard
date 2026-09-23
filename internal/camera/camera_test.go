package camera

import (
	"context"
	"testing"
	"time"
)

func TestMockFrameSource(t *testing.T) {
	mock := NewMockFrameSource(640, 480, 5)
	ctx := context.Background()

	count := 0
	for {
		f, err := mock.ReadFrame(ctx)
		if err != nil {
			break
		}
		count++
		if f.Width != 640 || f.Height != 480 {
			t.Fatalf("unexpected dimensions: %dx%d", f.Width, f.Height)
		}
	}

	if count != 5 {
		t.Fatalf("expected 5 frames, got %d", count)
	}

	if err := mock.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	_, err := mock.ReadFrame(ctx)
	if err != ErrCameraClosed {
		t.Fatalf("expected ErrCameraClosed after Close(), got %v", err)
	}
}

func TestSyntheticDeviceContextCancel(t *testing.T) {
	dev := NewSyntheticDevice(320, 240, 30)
	defer dev.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	// Device interval is 33ms, timeout is 5ms -> should return context deadline exceeded
	_, err := dev.ReadFrame(ctx)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
}
