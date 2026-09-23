package alert

import (
	"context"
	"testing"
	"time"
)

func TestMockAlertPlayer(t *testing.T) {
	player := &MockAlertPlayer{}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := player.PlayRandomSound(ctx); err != nil {
		t.Fatalf("PlayRandomSound failed: %v", err)
	}
	if err := player.FlashScreen(ctx); err != nil {
		t.Fatalf("FlashScreen failed: %v", err)
	}

	sounds, flashes := player.Stats()
	if sounds != 1 {
		t.Fatalf("expected sounds 1, got %d", sounds)
	}
	if flashes != 1 {
		t.Fatalf("expected flashes 1, got %d", flashes)
	}
}

func TestEmbeddedAudioFiles(t *testing.T) {
	entries, err := audioFS.ReadDir("assets/audio")
	if err != nil {
		t.Fatalf("failed to read embedded audio dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected embedded audio files, found 0")
	}

	foundWAV := false
	for _, e := range entries {
		if !e.IsDir() && len(e.Name()) > 4 && e.Name()[len(e.Name())-4:] == ".wav" {
			foundWAV = true
			data, err := audioFS.ReadFile("assets/audio/" + e.Name())
			if err != nil {
				t.Fatalf("failed to read %s: %v", e.Name(), err)
			}
			if len(data) == 0 {
				t.Fatalf("file %s is empty", e.Name())
			}
		}
	}

	if !foundWAV {
		t.Fatalf("expected at least one embedded .wav file")
	}
}
