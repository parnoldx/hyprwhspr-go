package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pa/hyprwhspr/internal/audio"
	"github.com/pa/hyprwhspr/internal/command"
	"github.com/pa/hyprwhspr/internal/config"
	"github.com/pa/hyprwhspr/internal/inject"
	"github.com/pa/hyprwhspr/internal/ipc"
	"github.com/pa/hyprwhspr/internal/models"
	"github.com/pa/hyprwhspr/internal/whisper"
)

type App struct {
	cfg         *config.Config
	ipcServer   *ipc.Server
	recorder    *audio.Recorder
	loopbackRec *audio.LoopbackRecorder
	aecProc     *audio.AECProcessor
	vadProc     *audio.VADProcessor
	transcriber *whisper.Transcriber
	injector    *inject.Injector
	player      *audio.Player
	cmdExecutor *command.Executor

	isRecording  bool
	isProcessing bool
}

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		command := os.Args[1]

		switch command {
		case "start", "stop", "toggle", "status":
			// Control command - send to daemon
			runControl(command)
			return
		case "daemon":
			// Explicit daemon mode
			runDaemon()
			return
		case "download":
			// Download model command
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: hyprwhspr download <model>\n")
				os.Exit(1)
			}
			runDownloadModel(os.Args[2])
			return
		case "models":
			// List models command
			runListModels()
			return
		case "delete":
			// Delete model command
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: hyprwhspr delete <model>\n")
				os.Exit(1)
			}
			runDeleteModel(os.Args[2])
			return
		case "model":
			// Set model command
			if len(os.Args) < 3 {
				fmt.Fprintf(os.Stderr, "Usage: hyprwhspr model <model>\n")
				os.Exit(1)
			}
			runSetModel(os.Args[2])
			return
		case "help", "-h", "--help":
			printUsage()
			return
		case "version", "-v", "--version":
			printVersion()
			return
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
			printUsage()
			os.Exit(1)
		}
	}

	// No arguments - run daemon by default
	runDaemon()
}

func printUsage() {
	fmt.Println("hyprwhspr - Speech-to-text daemon for Hyprland")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  hyprwhspr [command] [options]")
	fmt.Println("")
	fmt.Println("Daemon Commands:")
	fmt.Println("  (none)         Start daemon (default)")
	fmt.Println("  daemon         Start daemon explicitly")
	fmt.Println("")
	fmt.Println("Recording Commands:")
	fmt.Println("  start          Start recording")
	fmt.Println("  stop           Stop recording")
	fmt.Println("  toggle         Toggle recording on/off")
	fmt.Println("  status         Get current status")
	fmt.Println("")
	fmt.Println("Model Management:")
	fmt.Println("  models         List available and downloaded models")
	fmt.Println("  download <model> Download a whisper model")
	fmt.Println("  delete <model>  Delete a downloaded model")
	fmt.Println("  model <model>  Set the active whisper model")
	fmt.Println("")
	fmt.Println("Other:")
	fmt.Println("  help           Show this help")
	fmt.Println("  version        Show version")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  hyprwhspr              # Start daemon")
	fmt.Println("  hyprwhspr toggle       # Toggle recording")
	fmt.Println("  hyprwhspr models       # List models")
	fmt.Println("  hyprwhspr download base # Download base model")
	fmt.Println("  hyprwhspr model small # Switch to small model")
	fmt.Println("")
	fmt.Println("Hyprland config:")
	fmt.Println("  bind = SUPER D, exec, hyprwhspr toggle")
}

func runDownloadModel(modelName string) {
	cfgPath := config.GetConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	modelManager := models.NewManager(cfg.WhisperModelDir)

	if err := modelManager.DownloadModelWithProgress(modelName); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to download model: %v\n", err)
		os.Exit(1)
	}
}

func runListModels() {
	// Load actual config to get the current model setting
	cfgPath := config.GetConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	modelManager := models.NewManager(cfg.WhisperModelDir)
	modelManager.PrintModelInfo(cfg.Model)
}

func runDeleteModel(modelName string) {
	cfgPath := config.GetConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	modelManager := models.NewManager(cfg.WhisperModelDir)

	if err := modelManager.DeleteModel(modelName); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to delete model: %v\n", err)
		os.Exit(1)
	}
}

func runSetModel(modelName string) {
	// Get socket path from config
	cfgPath := config.GetConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	socketPath := cfg.SocketPath

	// Create IPC client
	client := ipc.NewClient(socketPath)

	// Send model command
	response, err := client.SendCommand("model " + modelName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Print response
	fmt.Println(response)

	// Exit with appropriate code
	if len(response) >= 5 && response[:5] == "ERROR" {
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Println("hyprwhspr v1.0.0-go")
	fmt.Println("Speech-to-text daemon for Hyprland")
}

func runControl(command string) {
	// Get socket path from config
	cfgPath := config.GetConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}
	socketPath := cfg.SocketPath

	// Create IPC client
	client := ipc.NewClient(socketPath)

	// Send command
	response, err := client.SendCommand(command)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Print response
	fmt.Println(response)

	// Exit with appropriate code
	if len(response) >= 5 && response[:5] == "ERROR" {
		os.Exit(1)
	}
}

func runDaemon() {
	fmt.Println("🚀 HYPRWHSPR STARTING UP!")
	fmt.Println(strings.Repeat("=", 50))

	// Load configuration
	cfgPath := config.GetConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create application
	app := &App{
		cfg: cfg,
	}

	// Initialize components
	if err := app.initialize(); err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Start IPC server
	if err := app.ipcServer.Start(); err != nil {
		log.Fatalf("Failed to start IPC server: %v", err)
	}

	fmt.Println("✅ hyprwhspr initialized successfully")
	fmt.Println("🎧 Running in daemon mode - use hyprwhspr to control recording")

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n🛑 Shutting down hyprwhspr...")
	app.cleanup()
}

func (app *App) initialize() error {
	// Initialize audio recorder
	var err error
	app.recorder, err = audio.NewRecorder(app.cfg.SampleRate, app.cfg.AudioDevice)
	if err != nil {
		return fmt.Errorf("failed to initialize audio recorder: %w", err)
	}

	// Initialize AEC and VAD if enabled
	if app.cfg.EchoCancellation {
		app.loopbackRec, err = audio.NewLoopbackRecorder(app.cfg.SampleRate)
		if err != nil {
			fmt.Printf("⚠️  Failed to initialize loopback recorder: %v\n", err)
		} else {
			aecConfig := audio.AECConfig{
				FilterLength:    app.cfg.AECFilterLength,
				StepSize:        app.cfg.AECStepSize,
				LeakageFactor:   0.999,
				EchoSuppression: app.cfg.AECEchoSuppression,
			}
			app.aecProc = audio.NewAECProcessor(aecConfig)
			fmt.Println("✅ Echo cancellation enabled")
		}
	}

	if app.cfg.VoiceActivityDetection {
		vadConfig := audio.VADConfig{
			FrameSize:       512,
			Overlap:         256,
			EnergyThreshold: app.cfg.VADEnergyThreshold,
			ZcrThreshold:    0.1,
			VoiceThreshold:  app.cfg.VADVoiceThreshold,
		}
		app.vadProc = audio.NewVADProcessor(vadConfig)
		fmt.Println("✅ Voice activity detection enabled")
	}

	// Initialize AEC and VAD if enabled
	fmt.Printf("🔧 Initializing AEC/VAD - EchoCancellation: %v, VAD: %v\n", app.cfg.EchoCancellation, app.cfg.VoiceActivityDetection)

	if app.cfg.EchoCancellation {
		fmt.Println("🔧 Creating loopback recorder...")
		app.loopbackRec, err = audio.NewLoopbackRecorder(app.cfg.SampleRate)
		if err != nil {
			fmt.Printf("❌ Failed to initialize loopback recorder: %v\n", err)
			fmt.Println("❌ Echo cancellation disabled")
		} else {
			fmt.Println("✅ Loopback recorder created")
			aecConfig := audio.AECConfig{
				FilterLength:    app.cfg.AECFilterLength,
				StepSize:        app.cfg.AECStepSize,
				LeakageFactor:   0.999,
				EchoSuppression: app.cfg.AECEchoSuppression,
			}
			app.aecProc = audio.NewAECProcessor(aecConfig)
			fmt.Println("✅ Echo cancellation enabled")
		}
	}

	if app.cfg.VoiceActivityDetection {
		fmt.Println("🔧 Creating VAD processor...")
		vadConfig := audio.VADConfig{
			FrameSize:       512,
			Overlap:         256,
			EnergyThreshold: app.cfg.VADEnergyThreshold,
			ZcrThreshold:    0.1,
			VoiceThreshold:  app.cfg.VADVoiceThreshold,
		}
		app.vadProc = audio.NewVADProcessor(vadConfig)
		fmt.Println("✅ Voice activity detection enabled")
	}

	// Initialize audio player for notifications
	app.player, err = audio.NewPlayer(audio.PlayerConfig{
		AudioFeedback:    app.cfg.AudioFeedback,
		StartSoundVolume: app.cfg.StartSoundVolume,
		StopSoundVolume:  app.cfg.StopSoundVolume,
		StartSoundPath:   app.cfg.StartSoundPath,
		StopSoundPath:    app.cfg.StopSoundPath,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize audio player: %w", err)
	}

	// Initialize whisper transcriber
	modelPath := filepath.Join(app.cfg.WhisperModelDir, fmt.Sprintf("ggml-%s.bin", app.cfg.Model))
	app.transcriber, err = whisper.New(modelPath, app.cfg.Threads, app.cfg.WhisperPrompt, app.cfg.AllowedLanguages)
	if err != nil {
		return fmt.Errorf("failed to initialize whisper: %w", err)
	}

	// Initialize text injector
	app.injector = inject.New()
	fmt.Println(app.injector.GetStatus())

	// Initialize command executor
	app.cmdExecutor = command.NewExecutor(app.cfg.CommandMode, app.cfg.Commands)
	fmt.Println(app.cmdExecutor.GetStatus())

	// Create IPC server
	app.ipcServer = ipc.NewServer(app.cfg.SocketPath, app.handleCommand)

	return nil
}

func (app *App) handleCommand(command string) string {
	// Parse command with arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "ERROR: Empty command"
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "start":
		if app.isRecording {
			return "ERROR: Already recording"
		}
		if err := app.startRecording(); err != nil {
			return fmt.Sprintf("ERROR: %v", err)
		}
		return "OK: Recording started"

	case "stop":
		if !app.isRecording {
			return "ERROR: Not recording"
		}
		if err := app.stopRecording(); err != nil {
			return fmt.Sprintf("ERROR: %v", err)
		}
		return "OK: Recording stopped"

	case "toggle":
		if app.isRecording {
			if err := app.stopRecording(); err != nil {
				return fmt.Sprintf("ERROR: %v", err)
			}
			return "OK: Recording stopped"
		} else {
			if err := app.startRecording(); err != nil {
				return fmt.Sprintf("ERROR: %v", err)
			}
			return "OK: Recording started"
		}

	case "status":
		if app.isRecording {
			return "1"
		} else {
			return "0"
		}

	case "model":
		if len(args) < 1 {
			return "ERROR: model requires a model name"
		}
		modelName := args[0]
		if err := app.setModel(modelName); err != nil {
			return fmt.Sprintf("ERROR: %v", err)
		}
		return fmt.Sprintf("OK: Model set to %s", modelName)

	default:
		return fmt.Sprintf("ERROR: Unknown command '%s'", cmd)
	}
}

func (app *App) startRecording() error {
	if app.isRecording {
		return fmt.Errorf("already recording")
	}

	app.isRecording = true

	// Start loopback recording if AEC is enabled
	if app.loopbackRec != nil {
		if err := app.loopbackRec.Start(); err != nil {
			fmt.Printf("⚠️  Failed to start loopback recording: %v\n", err)
			app.loopbackRec = nil
			app.aecProc = nil
		}
	}

	// Play start sound
	if app.player != nil {
		app.player.PlayStart()
	}

	// Notify waybar of recording state change
	exec.Command("pkill", "-RTMIN+9", "waybar").Run()

	return app.recorder.Start()
}

func (app *App) stopRecording() error {
	app.isRecording = false

	// Play stop sound
	if app.player != nil {
		app.player.PlayStop()
	}

	// Notify waybar of recording state change
	exec.Command("pkill", "-RTMIN+9", "waybar").Run()

	// Get recorded audio
	samples, err := app.recorder.Stop()
	if err != nil {
		return err
	}

	// Get loopback audio if available
	var loopbackSamples []float32
	if app.loopbackRec != nil {
		loopbackSamples, err = app.loopbackRec.Stop()
		if err != nil {
			fmt.Printf("⚠️  Failed to stop loopback recording: %v\n", err)
		}
	}

	// Process audio in background
	go app.processAudio(samples, loopbackSamples)

	return nil
}

func (app *App) processAudio(samples []float32, loopbackSamples []float32) {
	app.isProcessing = true
	defer func() {
		app.isProcessing = false
	}()

	// Debug: Print sample counts
	fmt.Printf("🔍 DEBUG: Mic samples: %d, Loopback samples: %d\n", len(samples), len(loopbackSamples))

	// Apply AEC if available
	processedSamples := samples
	if app.aecProc != nil && len(loopbackSamples) > 0 {
		fmt.Println("🔊 AEC: Processing with echo cancellation...")
		// Ensure both samples have same length
		minLen := len(samples)
		if len(loopbackSamples) < minLen {
			minLen = len(loopbackSamples)
		}

		if minLen > 0 {
			micSamples := samples[:minLen]
			farEndSamples := loopbackSamples[:minLen]
			processedSamples = app.aecProc.ProcessFrame(micSamples, farEndSamples)
			fmt.Printf("✅ AEC: Processed %d samples\n", minLen)
		}
	} else if app.aecProc == nil {
		fmt.Println("⚠️  AEC: Disabled (aecProc is nil)")
	} else if len(loopbackSamples) == 0 {
		fmt.Println("⚠️  AEC: No loopback samples captured!")
	}

	// Apply VAD if available
	samplesToTranscribe := processedSamples
	if app.vadProc != nil {
		voiceSegments := app.vadProc.GetVoiceSegments(processedSamples)
		if len(voiceSegments) == 0 {
			fmt.Println("⚠️  VAD: No voice detected - skipping transcription (only background/output audio)")
			return
		}
		fmt.Printf("✅ VAD: Detected %d voice segment(s)\n", len(voiceSegments))

		// Instead of extracting segments, mute non-voice parts in-place
		// This preserves timing and structure for Whisper
		sampleRate := float64(app.cfg.SampleRate)
		paddingMs := 200.0 // Add 200ms padding before/after each segment
		paddingSamples := int(paddingMs * sampleRate / 1000.0)

		// Create a copy to modify
		mutedSamples := make([]float32, len(processedSamples))
		copy(mutedSamples, processedSamples)

		// Create a mask: true = keep audio, false = mute
		keepMask := make([]bool, len(processedSamples))

		// Mark voice segments (with padding) to keep
		for i, seg := range voiceSegments {
			startSample := int(seg.Start*sampleRate/1000.0) - paddingSamples
			endSample := int(seg.End*sampleRate/1000.0) + paddingSamples

			// Bounds check
			if startSample < 0 {
				startSample = 0
			}
			if endSample > len(processedSamples) {
				endSample = len(processedSamples)
			}

			// Mark this range to keep
			for j := startSample; j < endSample; j++ {
				keepMask[j] = true
			}

			fmt.Printf("   Segment %d: %.1fms-%.1fms (%.1fms duration, keeping with %.0fms padding)\n",
				i+1, seg.Start, seg.End, seg.Duration, paddingMs*2)
		}

		// Mute (zero out) all non-voice parts
		mutedCount := 0
		for i := range mutedSamples {
			if !keepMask[i] {
				mutedSamples[i] = 0.0
				mutedCount++
			}
		}

		keptSamples := len(mutedSamples) - mutedCount
		fmt.Printf("📊 VAD: Keeping %d samples, muted %d samples (%.1f%% voice)\n",
			keptSamples, mutedCount, float64(keptSamples)/float64(len(mutedSamples))*100)

		samplesToTranscribe = mutedSamples
	}

	// Transcribe
	text, err := app.transcriber.Transcribe(samplesToTranscribe)
	if err != nil {
		fmt.Printf("❌ Transcription failed: %v\n", err)
		return
	}

	if text == "" {
		fmt.Println("⚠️  No transcription generated")
		return
	}

	fmt.Printf("📝 Transcription: %s\n", text)

	// Check if it's a command
	wasCommand, err := app.cmdExecutor.Execute(text)
	if err != nil {
		fmt.Printf("❌ Command execution failed: %v\n", err)
		// Fall through to text injection on error
	}

	if wasCommand {
		fmt.Println("✅ Command executed successfully")
		return
	}

	// Not a command, inject text normally
	if err := app.injector.Inject(text); err != nil {
		fmt.Printf("❌ Text injection failed: %v\n", err)
	}
}

func (app *App) setModel(modelName string) error {
	// Validate model name
	modelManager := models.NewManager(app.cfg.WhisperModelDir)
	if !modelManager.IsModelDownloaded(modelName) {
		return fmt.Errorf("model '%s' is not downloaded. Use 'hyprwhspr download %s' first", modelName, modelName)
	}

	// Close existing transcriber
	if app.transcriber != nil {
		app.transcriber.Close()
	}

	// Initialize new transcriber with the specified model
	modelPath := filepath.Join(app.cfg.WhisperModelDir, fmt.Sprintf("ggml-%s.bin", modelName))
	transcriber, err := whisper.New(modelPath, app.cfg.Threads, app.cfg.WhisperPrompt, app.cfg.AllowedLanguages)
	if err != nil {
		return fmt.Errorf("failed to initialize whisper with model '%s': %w", modelName, err)
	}

	app.transcriber = transcriber
	app.cfg.Model = modelName

	// Save the updated model to config
	if err := app.cfg.Save(config.GetConfigPath()); err != nil {
		fmt.Printf("⚠️  Failed to save model to config: %v\n", err)
	}

	fmt.Printf("✅ Model switched to '%s'\n", modelName)
	return nil
}

func (app *App) cleanup() {
	if app.ipcServer != nil {
		app.ipcServer.Stop()
	}
	if app.recorder != nil {
		app.recorder.Close()
	}
	if app.loopbackRec != nil {
		app.loopbackRec.Close()
	}
	if app.player != nil {
		app.player.Close()
	}
	if app.transcriber != nil {
		app.transcriber.Close()
	}
	fmt.Println("✅ Cleanup completed")
}
