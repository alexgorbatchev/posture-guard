package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	cobrahelptree "github.com/alexgorbatchev/cobra-help-tree/v2"
	"github.com/spf13/cobra"

	"github.com/MatheusDSantossi/posture-guard/internal/alert"
	"github.com/MatheusDSantossi/posture-guard/internal/camera"
	"github.com/MatheusDSantossi/posture-guard/internal/config"
	"github.com/MatheusDSantossi/posture-guard/internal/detector"
	"github.com/MatheusDSantossi/posture-guard/internal/monitor"
	"github.com/MatheusDSantossi/posture-guard/internal/notifier"
	"github.com/MatheusDSantossi/posture-guard/internal/posture"
)

// Version string injected during build or set by caller.
var Version = "0.3.3"

// BuildRootCommand constructs the full Cobra command tree and installs cobra-help-tree.
func BuildRootCommand() (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:   "posture-guard",
		Short: "Real-time background posture monitoring with local computer vision",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// ── Monitor Command ─────────────────────────────────────────────────────────
	var (
		monitorSensitivity string
		monitorCooldown    float64
		monitorDuration    float64
		monitorExtreme     bool
		monitorDebug       bool
		monitorSimulated   bool
	)

	monitorCmd := &cobra.Command{
		Use:   "monitor",
		Short: "Start real-time posture monitoring in the background",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			settings, err := config.LoadSettings("")
			if err != nil {
				return fmt.Errorf("loading settings: %w", err)
			}

			if monitorSensitivity != "" {
				settings.Sensitivity = monitorSensitivity
			}
			if monitorCooldown > 0 {
				settings.AlertCooldown = monitorCooldown
			}
			if monitorDuration > 0 {
				settings.BadPostureDuration = monitorDuration
			}
			if cmd.Flags().Changed("extreme") {
				settings.ExtremeAlert = monitorExtreme
			}

			thresh := config.DefaultThresholds()
			if monitorDebug {
				thresh.Debug = true
			}

			var cam camera.FrameSource
			var det detector.PoseDetector

			if monitorSimulated {
				cam = camera.NewSyntheticDevice(640, 480, 15)
				det = detector.NewLandmarkGeneratorDetector()
			} else {
				cam = camera.NewSyntheticDevice(640, 480, 15)
				det = detector.NewLandmarkGeneratorDetector()
			}
			defer cam.Close()
			defer det.Close()

			notif := notifier.NewOSNotifier("PostureGuard")
			alertPl := alert.NewOSAlertPlayer()

			mon := monitor.New(cam, det, notif, alertPl, settings, thresh)

			isAgent := os.Getenv("AGENT") == "1"
			if isAgent {
				fmt.Println("STATUS: Calibrating baseline posture...")
			} else {
				fmt.Println("[INFO] Calibrating baseline posture — sit straight and look forward...")
			}

			baseline, err := mon.Calibrate(ctx, 30)
			if err != nil {
				return fmt.Errorf("calibration failed: %w", err)
			}

			if isAgent {
				fmt.Printf("OK: Baseline calibrated (shoulder_mid_y: %.3f, shoulder_width: %.3f)\n", baseline.ShoulderMidY, baseline.ShoulderWidth)
				fmt.Println("STATUS: Monitoring started. Running in background.")
			} else {
				fmt.Printf("[OK] Baseline calibrated successfully (shoulder_mid_y: %.3f, width: %.3f)\n", baseline.ShoulderMidY, baseline.ShoulderWidth)
				fmt.Println("[INFO] PostureGuard active. Press Ctrl+C to stop.")
			}

			mon.OnPostureChange = func(res posture.PostureResult) {
				if thresh.Debug {
					if isAgent {
						if !res.IsGood {
							fmt.Printf("WARN: %s\n", res.ReasonString())
						}
					} else {
						if res.IsGood {
							fmt.Print("\r[OK] Good posture                     ")
						} else {
							fmt.Printf("\r[WARN] %s                     ", res.ReasonString())
						}
					}
				}
			}

			return mon.Run(ctx)
		},
	}

	monitorCmd.Flags().StringVar(&monitorSensitivity, "sensitivity", "", "Detection sensitivity: low, medium, high")
	monitorCmd.Flags().Float64Var(&monitorCooldown, "cooldown", 0, "Minimum seconds between alerts")
	monitorCmd.Flags().Float64Var(&monitorDuration, "duration", 0, "Seconds of sustained slouching before alert")
	monitorCmd.Flags().BoolVar(&monitorExtreme, "extreme", false, "Enable extreme alert (sound + flash)")
	monitorCmd.Flags().BoolVar(&monitorDebug, "debug", false, "Print debug posture metrics to terminal")
	monitorCmd.Flags().BoolVar(&monitorSimulated, "simulated", false, "Run with simulated camera and posture feed")

	rootCmd.AddCommand(monitorCmd)

	// ── Calibrate Command ───────────────────────────────────────────────────────
	var (
		calibFrames    int
		calibSimulated bool
	)

	calibCmd := &cobra.Command{
		Use:   "calibrate",
		Short: "Perform posture baseline calibration and output baseline metrics",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			thresh := config.DefaultThresholds()
			if calibFrames > 0 {
				thresh.CalibrationFrames = calibFrames
			}

			cam := camera.NewSyntheticDevice(640, 480, 30)
			det := detector.NewLandmarkGeneratorDetector()
			defer cam.Close()
			defer det.Close()

			mon := monitor.New(cam, det, nil, nil, config.DefaultSettings(), thresh)

			isAgent := os.Getenv("AGENT") == "1"
			if !isAgent {
				fmt.Printf("[INFO] Capturing %d frames for calibration...\n", thresh.CalibrationFrames)
			}

			baseline, err := mon.Calibrate(ctx, thresh.CalibrationFrames)
			if err != nil {
				return fmt.Errorf("calibration failed: %w", err)
			}

			data, err := json.MarshalIndent(baseline, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling baseline: %w", err)
			}

			if isAgent {
				fmt.Printf("OK: %s\n", string(data))
			} else {
				fmt.Println("[OK] Calibration complete:")
				fmt.Println(string(data))
			}
			return nil
		},
	}

	calibCmd.Flags().IntVar(&calibFrames, "frames", 30, "Number of calibration frames to sample")
	calibCmd.Flags().BoolVar(&calibSimulated, "simulated", true, "Use simulated video feed")

	rootCmd.AddCommand(calibCmd)

	// ── Config Command Group ────────────────────────────────────────────────────
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage persistent configuration and sensitivity thresholds",
	}

	configShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Display active configuration and file path",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfgPath := config.GetConfigPath()
			s, err := config.LoadSettings(cfgPath)
			if err != nil {
				return err
			}

			isAgent := os.Getenv("AGENT") == "1"
			if isAgent {
				fmt.Printf("file: %s\n", cfgPath)
				fmt.Printf("alert_cooldown: %.1f\n", s.AlertCooldown)
				fmt.Printf("bad_posture_duration: %.1f\n", s.BadPostureDuration)
				fmt.Printf("sensitivity: %s\n", s.Sensitivity)
				fmt.Printf("extreme_alert: %t\n", s.ExtremeAlert)
				fmt.Printf("extreme_sound: %t\n", s.ExtremeSound)
				fmt.Printf("extreme_flash: %t\n", s.ExtremeFlash)
				return nil
			}

			fmt.Printf("[INFO] Config path: %s\n", cfgPath)
			data, _ := json.MarshalIndent(s, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}

	configGetCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get the value of a configuration key",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])
			s, err := config.LoadSettings("")
			if err != nil {
				return err
			}

			var val string
			switch key {
			case "alert_cooldown", "cooldown":
				val = fmt.Sprintf("%.1f", s.AlertCooldown)
			case "bad_posture_duration", "duration":
				val = fmt.Sprintf("%.1f", s.BadPostureDuration)
			case "sensitivity":
				val = s.Sensitivity
			case "extreme_alert":
				val = fmt.Sprintf("%t", s.ExtremeAlert)
			case "extreme_sound":
				val = fmt.Sprintf("%t", s.ExtremeSound)
			case "extreme_flash":
				val = fmt.Sprintf("%t", s.ExtremeFlash)
			default:
				return fmt.Errorf("unknown configuration key %q", key)
			}

			if os.Getenv("AGENT") == "1" {
				fmt.Printf("%s: %s\n", key, val)
			} else {
				fmt.Println(val)
			}
			return nil
		},
	}

	configSetCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set the value of a configuration key",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			key := strings.ToLower(args[0])
			val := args[1]

			s, err := config.LoadSettings("")
			if err != nil {
				return err
			}

			switch key {
			case "alert_cooldown", "cooldown":
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return fmt.Errorf("invalid float %q for %s: %w", val, key, err)
				}
				s.AlertCooldown = f
			case "bad_posture_duration", "duration":
				f, err := strconv.ParseFloat(val, 64)
				if err != nil {
					return fmt.Errorf("invalid float %q for %s: %w", val, key, err)
				}
				s.BadPostureDuration = f
			case "sensitivity":
				sens := strings.ToLower(val)
				if sens != "low" && sens != "medium" && sens != "high" {
					return fmt.Errorf("invalid sensitivity %q (expected low, medium, or high)", val)
				}
				s.Sensitivity = sens
			case "extreme_alert":
				b, err := strconv.ParseBool(val)
				if err != nil {
					return fmt.Errorf("invalid boolean %q for %s: %w", val, key, err)
				}
				s.ExtremeAlert = b
			case "extreme_sound":
				b, err := strconv.ParseBool(val)
				if err != nil {
					return fmt.Errorf("invalid boolean %q for %s: %w", val, key, err)
				}
				s.ExtremeSound = b
			case "extreme_flash":
				b, err := strconv.ParseBool(val)
				if err != nil {
					return fmt.Errorf("invalid boolean %q for %s: %w", val, key, err)
				}
				s.ExtremeFlash = b
			default:
				return fmt.Errorf("unknown configuration key %q", key)
			}

			if err := config.SaveSettings("", s); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			if os.Getenv("AGENT") == "1" {
				fmt.Printf("OK: %s set to %s\n", key, val)
			} else {
				fmt.Printf("[OK] Updated %s = %s\n", key, val)
			}
			return nil
		},
	}

	configResetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset all configuration settings to factory defaults",
		RunE: func(_ *cobra.Command, _ []string) error {
			defaults := config.DefaultSettings()
			if err := config.SaveSettings("", defaults); err != nil {
				return fmt.Errorf("resetting config: %w", err)
			}
			if os.Getenv("AGENT") == "1" {
				fmt.Println("OK: Configuration reset to defaults")
			} else {
				fmt.Println("[OK] Configuration reset to factory defaults.")
			}
			return nil
		},
	}

	configCmd.AddCommand(configShowCmd, configGetCmd, configSetCmd, configResetCmd)
	rootCmd.AddCommand(configCmd)

	// ── Alert Command Group ─────────────────────────────────────────────────────
	alertCmd := &cobra.Command{
		Use:   "alert",
		Short: "Test and inspect notification and audio alert systems",
	}

	var (
		alertSound bool
		alertFlash bool
	)

	alertTestCmd := &cobra.Command{
		Use:   "test",
		Short: "Trigger a test desktop notification and audio playback",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			notif := notifier.NewOSNotifier("PostureGuard")
			player := alert.NewOSAlertPlayer()

			isAgent := os.Getenv("AGENT") == "1"

			if err := notif.Notify(ctx, "PostureGuard Test", "This is a test notification from PostureGuard"); err != nil {
				if isAgent {
					fmt.Printf("ERR: notification failed: %v\n", err)
				} else {
					fmt.Printf("[ERROR] Notification failed: %v\n", err)
				}
			} else {
				if isAgent {
					fmt.Println("OK: Desktop notification sent")
				} else {
					fmt.Println("[OK] Desktop notification sent successfully")
				}
			}

			if alertSound {
				if err := player.PlayRandomSound(ctx); err != nil {
					if isAgent {
						fmt.Printf("ERR: sound failed: %v\n", err)
					} else {
						fmt.Printf("[ERROR] Sound failed: %v\n", err)
					}
				} else {
					if isAgent {
						fmt.Println("OK: Alert audio played")
					} else {
						fmt.Println("[OK] Alert audio played successfully")
					}
				}
			}

			if alertFlash {
				_ = player.FlashScreen(ctx)
			}

			return nil
		},
	}

	alertTestCmd.Flags().BoolVar(&alertSound, "sound", true, "Play test audio alert")
	alertTestCmd.Flags().BoolVar(&alertFlash, "flash", false, "Trigger screen flash")

	alertCmd.AddCommand(alertTestCmd)
	rootCmd.AddCommand(alertCmd)

	// ── Document positional arguments with TechCatalog ─────────────────────────
	catalog := cobrahelptree.TechCatalog{
		"posture-guard config get": {
			Args: []cobrahelptree.ArgSpec{
				{Name: "<key>", Description: "Configuration property name (e.g. sensitivity, cooldown, duration)"},
			},
		},
		"posture-guard config set": {
			Args: []cobrahelptree.ArgSpec{
				{Name: "<key>", Description: "Configuration property name (e.g. sensitivity, cooldown, duration)"},
				{Name: "<value>", Description: "New property value to persist"},
			},
			MutatesDB: true,
		},
	}

	err := cobrahelptree.SetupWithOptions(rootCmd, cobrahelptree.HelpOptions{
		Catalog: catalog,
		Tree: cobrahelptree.TreeOptions{
			HideGeneratedCommands: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("setting up cobra tree help: %w", err)
	}

	return rootCmd, nil
}
