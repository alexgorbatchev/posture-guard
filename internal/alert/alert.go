package alert

import (
	"context"
	"embed"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed assets/audio/*.wav
var audioFS embed.FS

// Player plays alert sounds and triggers visual alerts.
type Player interface {
	PlayRandomSound(ctx context.Context) error
	FlashScreen(ctx context.Context) error
}

// OSAlertPlayer implements Player with embedded audio files and platform audio commands.
type OSAlertPlayer struct {
	tempDir string
	mu      sync.Mutex
}

// NewOSAlertPlayer creates a new alert player.
func NewOSAlertPlayer() *OSAlertPlayer {
	return &OSAlertPlayer{}
}

// PlayRandomSound picks an embedded WAV file and plays it asynchronously.
func (p *OSAlertPlayer) PlayRandomSound(ctx context.Context) error {
	entries, err := audioFS.ReadDir("assets/audio")
	if err != nil || len(entries) == 0 {
		return p.playSystemBeep()
	}

	var wavFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".wav") {
			wavFiles = append(wavFiles, e.Name())
		}
	}

	if len(wavFiles) == 0 {
		return p.playSystemBeep()
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	chosen := wavFiles[r.Intn(len(wavFiles))]
	data, err := audioFS.ReadFile("assets/audio/" + chosen)
	if err != nil {
		return p.playSystemBeep()
	}

	return p.playWAVData(ctx, data, chosen)
}

func (p *OSAlertPlayer) playWAVData(ctx context.Context, data []byte, filename string) error {
	p.mu.Lock()
	if p.tempDir == "" {
		tmp, err := os.MkdirTemp("", "posture-guard-audio-*")
		if err == nil {
			p.tempDir = tmp
		}
	}
	tmpPath := filepath.Join(p.tempDir, filename)
	_ = os.WriteFile(tmpPath, data, 0644)
	p.mu.Unlock()

	switch runtime.GOOS {
	case "darwin":
		cmd := exec.CommandContext(ctx, "afplay", tmpPath)
		if err := cmd.Run(); err != nil {
			return p.playSystemBeep()
		}
		return nil
	case "linux":
		// Try aplay, paplay, or pw-play
		players := []string{"aplay", "paplay", "pw-play"}
		for _, prog := range players {
			if _, err := exec.LookPath(prog); err == nil {
				cmd := exec.CommandContext(ctx, prog, tmpPath)
				if err := cmd.Run(); err == nil {
					return nil
				}
			}
		}
		return p.playSystemBeep()
	case "windows":
		psScript := fmt.Sprintf(`(New-Object System.Media.SoundPlayer "%s").PlaySync()`, tmpPath)
		cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
		if err := cmd.Run(); err != nil {
			return p.playSystemBeep()
		}
		return nil
	default:
		return p.playSystemBeep()
	}
}

func (p *OSAlertPlayer) playSystemBeep() error {
	if os.Getenv("AGENT") != "1" {
		fmt.Print("\a")
	}
	return nil
}

// FlashScreen outputs a visual flash notification.
func (p *OSAlertPlayer) FlashScreen(ctx context.Context) error {
	if os.Getenv("AGENT") == "1" {
		fmt.Println("FLASH: [EXTREME POSTURE ALERT]")
		return nil
	}

	// ANSI flash sequence: reverse video, bell, pause, reset
	fmt.Print("\033[?5h\033[41m\033[37m\a  !!! POSTURE ALERT - SIT STRAIGHT !!!  \033[0m\033[?5l\n")
	select {
	case <-time.After(400 * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// MockAlertPlayer records sound and flash triggers for tests.
type MockAlertPlayer struct {
	mu           sync.Mutex
	soundPlayed  int
	screenFlashed int
}

// PlayRandomSound records sound playback call.
func (m *MockAlertPlayer) PlayRandomSound(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.soundPlayed++
	return nil
}

// FlashScreen records flash screen call.
func (m *MockAlertPlayer) FlashScreen(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.screenFlashed++
	return nil
}

// Stats returns playback counts safely.
func (m *MockAlertPlayer) Stats() (sounds int, flashes int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.soundPlayed, m.screenFlashed
}
