> Rewritten in Go from the original Python project [`posture-guard`](https://github.com/MatheusDSantossi/posture-guard) by [Matheus D. Santos](https://github.com/MatheusDSantossi).

`posture-guard` is a lightweight, real-time posture monitoring tool that runs silently in the background and alerts you when you start slouching, hunching, tilting your head, or straining your neck. All video frame processing is computed locally on your machine with zero cloud connectivity, zero recording, and zero tracking.

# What It Does

- **Monitors Posture Continuously**: Analyzes your upper body posture in real time using your webcam or video feed.
- **Calibrates to Your Baseline**: Establishes a personalized posture reference profile during a brief startup calibration step instead of enforcing fixed body proportions.
- **Detects Multiple Posture Defects**: Identifies shoulder drops (slouching), shoulder narrowing (hunching), uneven shoulder tilts, lateral leaning, head tilt angles, and craniovertebral angle neck strain (forward head protrusion).
- **Smooths Frame Jitter**: Applies rolling window metric smoothing to prevent spurious alerts while detecting sustained slouching.
- **Dispatches Native Desktop Notifications**: Sends native alerts via Windows notifications, macOS Notification Center, or Linux desktop notifications when bad posture is sustained.
- **Supports Extreme Alerts**: Optionally triggers embedded audio playback and visual screen alerts during prolonged poor posture.
- **Provides Dual-Mode Terminal Output**: Delivers clean visual feedback for humans and token-conservative machine format when `AGENT=1` is set.

# How It Works

- The user launches `posture-guard monitor` and sits straight looking forward for a brief calibration period.
- `posture-guard` captures camera frames, extracts key facial and upper-torso pose landmarks (nose, ears, shoulders), and averages them into a baseline posture profile.
- During active monitoring, each video frame computes posture deltas against your calibrated baseline across a rolling history window.
- If posture defects exceed configured sensitivity thresholds for longer than the sustained posture duration, `posture-guard` dispatches a desktop notification and optional audio alerts.
- When good posture is restored, alert timers reset immediately.

# How it Really Works

- Reads and writes persistent configuration to `$XDG_CONFIG_HOME/posture-guard/config.json` (falling back to `~/.config/posture-guard/config.json` or local `posture_guard_settings.json`).
- Executes local OS notification dispatchers: `osascript` on macOS, `notify-send` on Linux, and PowerShell WinRT toast notifications on Windows.
- Plays alert audio using embedded WAV assets and native system players (`afplay` on macOS, `aplay`/`paplay`/`pw-play` on Linux, `[System.Media.SoundPlayer]` on Windows).
- Uses rolling window averaging (8 frames) to absorb momentary movement and head turns without false positive alert triggers.
- Sends user-requested `--help` output to stdout and execution errors or usage diagnostics to stderr.
- Detects the `AGENT=1` environment variable to switch from styled terminal banners to compact key-value outputs.
- Exits with status code `0` on successful completion and `1` on error or interrupt.

# Prerequisites

- Linux desktop users require `libnotify-bin` for desktop notifications and `alsa-utils` or `pulseaudio-utils` for audio playback.

# Installation

Download the precompiled binary for your architecture from the [GitHub Releases](https://github.com/MatheusDSantossi/posture-guard/releases/latest) page.

```bash
curl -sL https://github.com/MatheusDSantossi/posture-guard/releases/download/v0.3.3/posture-guard_0.3.3_darwin_arm64.tar.gz | tar -xz -C /usr/local/bin posture-guard
```

# Quick Start

Start posture monitoring:

```bash
posture-guard monitor
```

Sample Output:

```
[INFO] Calibrating baseline posture — sit straight and look forward...
[OK] Baseline calibrated successfully (shoulder_mid_y: 0.450, width: 0.200)
[INFO] PostureGuard active. Press Ctrl+C to stop.
```

Calibrate baseline and output JSON:

```bash
posture-guard calibrate --frames 30
```

Sample Output:

```
[OK] Calibration complete:
{
  "shoulder_mid_y": 0.45,
  "shoulder_width": 0.2,
  "shoulder_tilt": 0,
  "lean_offset": 0,
  "neck_y_gap": -0.2,
  "cva": 90,
  "head_tilt": 0,
  "valid": true
}
```

Inspect and update configuration:

```bash
posture-guard config get sensitivity
posture-guard config set sensitivity high
posture-guard config show
```

Test desktop notifications and alert sound:

```bash
posture-guard alert test --sound
```

# Options & Flags

| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--help` | `-h` | `false` | Display hierarchical help tree |
| `--version` | `-v` | `false` | Print raw version string |

### `posture-guard monitor`

| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--sensitivity <level>` | | `""` | Detection sensitivity: `low`, `medium`, or `high` |
| `--cooldown <seconds>` | | `0` | Minimum seconds between alert notifications |
| `--duration <seconds>` | | `0` | Seconds of sustained bad posture required before alerting |
| `--extreme` | | `false` | Enable extreme alerts (audio sound + visual flash) |
| `--debug` | | `false` | Output real-time posture metrics to terminal |
| `--simulated` | | `false` | Run with simulated camera and posture feed |

### `posture-guard calibrate`

| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--frames <count>` | | `30` | Number of video frames to sample during calibration |
| `--simulated` | | `true` | Use simulated camera feed during calibration |

### `posture-guard alert test`

| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--sound` | | `true` | Play embedded audio alert sound |
| `--flash` | | `false` | Trigger screen flash visual alert |

# License

[MIT](LICENSE)
