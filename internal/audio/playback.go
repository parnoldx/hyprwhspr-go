package audio

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/vorbis"
)

var speakerInitialized = false

// simpleVolume is a straightforward volume control that directly multiplies samples
type simpleVolume struct {
	streamer beep.Streamer
	volume   float64
}

func (v *simpleVolume) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = v.streamer.Stream(samples)
	for i := range samples[:n] {
		samples[i][0] *= v.volume
		samples[i][1] *= v.volume
	}
	return n, ok
}

func (v *simpleVolume) Err() error {
	return v.streamer.Err()
}

// PlayerConfig contains configuration for the audio player
type PlayerConfig struct {
	AudioFeedback    bool
	StartSoundVolume float64
	StopSoundVolume  float64
	StartSoundPath   *string
	StopSoundPath    *string
}

// Player handles audio playback for notification sounds
type Player struct {
	config         PlayerConfig
	startSoundPath string
	stopSoundPath  string
	enabled        bool
}

// NewPlayer creates a new audio player
func NewPlayer(config PlayerConfig) (*Player, error) {
	player := &Player{
		config:  config,
		enabled: config.AudioFeedback,
	}

	// Validate and clamp volumes to 0.0-1.0
	if player.config.StartSoundVolume < 0.0 {
		player.config.StartSoundVolume = 0.0
	} else if player.config.StartSoundVolume > 1.0 {
		player.config.StartSoundVolume = 1.0
	}

	if player.config.StopSoundVolume < 0.0 {
		player.config.StopSoundVolume = 0.0
	} else if player.config.StopSoundVolume > 1.0 {
		player.config.StopSoundVolume = 1.0
	}

	// Resolve sound file paths
	if err := player.resolveSoundPaths(); err != nil {
		fmt.Printf("⚠️  Audio feedback disabled: %v\n", err)
		player.enabled = false
		return player, nil
	}

	return player, nil
}

// resolveSoundPaths finds the sound files
func (p *Player) resolveSoundPaths() error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = ""
	}

	// Get executable path to find assets
	execPath, err := os.Executable()
	if err != nil {
		execPath = "."
	}
	execDir := filepath.Dir(execPath)

	// Try multiple possible asset locations
	possiblePaths := []string{
		filepath.Join(homeDir, ".local", "share", "hyprwhspr", "assets"), // XDG user data dir with assets subdir
		filepath.Join(homeDir, ".local", "share", "hyprwhspr"),           // XDG user data dir (no subdir)
		filepath.Join(execDir, "share", "assets"),                        // Next to binary
		filepath.Join(execDir, "..", "share", "assets"),                  // Up one level from bin/
		"share/assets", // Relative to working directory
	}

	var assetsDir string
	for _, path := range possiblePaths {
		if _, err := os.Stat(filepath.Join(path, "start.ogg")); err == nil {
			assetsDir = path
			fmt.Printf("[audio] Found sound assets in: %s\n", path)
			break
		}
	}

	if assetsDir == "" {
		return fmt.Errorf("sound assets not found in any standard location:\n  - %s\n  - %s\n  - %s\n  - %s\n  - %s",
			possiblePaths[0], possiblePaths[1], possiblePaths[2], possiblePaths[3], possiblePaths[4])
	}

	// Resolve start sound path
	if p.config.StartSoundPath != nil && *p.config.StartSoundPath != "" {
		// Try custom path
		customPath := *p.config.StartSoundPath
		if filepath.IsAbs(customPath) {
			if _, err := os.Stat(customPath); err == nil {
				p.startSoundPath = customPath
			}
		} else {
			// Try relative to assets dir
			relPath := filepath.Join(assetsDir, customPath)
			if _, err := os.Stat(relPath); err == nil {
				p.startSoundPath = relPath
			}
		}
	}

	// Fallback to default start sound
	if p.startSoundPath == "" {
		p.startSoundPath = filepath.Join(assetsDir, "start.ogg")
	}

	// Resolve stop sound path
	if p.config.StopSoundPath != nil && *p.config.StopSoundPath != "" {
		// Try custom path
		customPath := *p.config.StopSoundPath
		if filepath.IsAbs(customPath) {
			if _, err := os.Stat(customPath); err == nil {
				p.stopSoundPath = customPath
			}
		} else {
			// Try relative to assets dir
			relPath := filepath.Join(assetsDir, customPath)
			if _, err := os.Stat(relPath); err == nil {
				p.stopSoundPath = relPath
			}
		}
	}

	// Fallback to default stop sound
	if p.stopSoundPath == "" {
		p.stopSoundPath = filepath.Join(assetsDir, "stop.ogg")
	}

	// Verify files exist
	if _, err := os.Stat(p.startSoundPath); err != nil {
		return fmt.Errorf("start sound not found: %s", p.startSoundPath)
	}
	if _, err := os.Stat(p.stopSoundPath); err != nil {
		return fmt.Errorf("stop sound not found: %s", p.stopSoundPath)
	}

	fmt.Printf("🔊 Audio feedback enabled:\n")
	fmt.Printf("   Start: %s (volume: %.0f%%)\n", p.startSoundPath, p.config.StartSoundVolume*100)
	fmt.Printf("   Stop: %s (volume: %.0f%%)\n", p.stopSoundPath, p.config.StopSoundVolume*100)

	return nil
}

// PlayStart plays the recording start sound
func (p *Player) PlayStart() {
	if !p.enabled || p.startSoundPath == "" {
		return
	}
	go p.playSound(p.startSoundPath, p.config.StartSoundVolume)
}

// PlayStop plays the recording stop sound
func (p *Player) PlayStop() {
	if !p.enabled || p.stopSoundPath == "" {
		return
	}
	go p.playSound(p.stopSoundPath, p.config.StopSoundVolume)
}

func (p *Player) playSound(path string, volume float64) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("⚠️  Failed to open sound file: %v\n", err)
		return
	}
	defer f.Close()

	streamer, format, err := vorbis.Decode(f)
	if err != nil {
		fmt.Printf("⚠️  Failed to decode sound file: %v\n", err)
		return
	}
	defer streamer.Close()

	// Initialize speaker if not already done
	if !speakerInitialized {
		err := speaker.Init(format.SampleRate, format.SampleRate.N(format.SampleRate.D(1)/10))
		if err != nil {
			fmt.Printf("⚠️  Failed to initialize audio speaker: %v\n", err)
			return
		}
		speakerInitialized = true
	}

	// Apply volume control by directly multiplying samples
	// Simple and transparent: 0.4 means 40% amplitude
	volumeCtrl := &simpleVolume{
		streamer: streamer,
		volume:   volume,
	}

	done := make(chan bool)
	speaker.Play(beep.Seq(volumeCtrl, beep.Callback(func() {
		done <- true
	})))

	<-done
}

// Close closes the player (currently no cleanup needed)
func (p *Player) Close() {
	// Future cleanup if needed
}
