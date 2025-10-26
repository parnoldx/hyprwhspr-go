package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	Model            string            `json:"model"`
	Threads          int               `json:"threads"`
	Language         *string           `json:"language"`          // nil = auto-detect
	AllowedLanguages []string          `json:"allowed_languages"` // Restrict auto-detect to these languages (e.g. ["de", "en"])
	AudioDevice      *string           `json:"audio_device"`
	SampleRate       int               `json:"sample_rate"`
	SocketPath       string            `json:"socket_path"`
	WhisperModelDir  string            `json:"whisper_model_dir"`
	AudioFeedback    bool              `json:"audio_feedback"`
	StartSoundVolume float64           `json:"start_sound_volume"`
	StopSoundVolume  float64           `json:"stop_sound_volume"`
	StartSoundPath   *string           `json:"start_sound_path"` // nil = default
	StopSoundPath    *string           `json:"stop_sound_path"`  // nil = default
	CommandMode      bool              `json:"command_mode"`     // Enable command mode
	Commands         map[string]string `json:"commands"`         // command_word -> script_path
	WhisperPrompt    string            `json:"whisper_prompt"`   // Initial prompt for whisper transcription

	// Echo Cancellation settings
	EchoCancellation   bool    `json:"echo_cancellation"`    // Enable acoustic echo cancellation
	AECFilterLength    int     `json:"aec_filter_length"`    // AEC filter length (512-2048)
	AECStepSize        float64 `json:"aec_step_size"`        // AEC adaptation step size (0.01-0.1)
	AECEchoSuppression float64 `json:"aec_echo_suppression"` // Echo suppression gain (0.0-1.0)

	// Voice Activity Detection settings
	VoiceActivityDetection bool    `json:"voice_activity_detection"` // Enable VAD
	VADEnergyThreshold     float64 `json:"vad_energy_threshold"`     // Energy threshold for VAD
	VADVoiceThreshold      float64 `json:"vad_voice_threshold"`      // Voice probability threshold
}

// Default returns default configuration
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	socketPath := filepath.Join(homeDir, ".config", "hyprwhspr", "hyprwhspr.sock")
	modelDir := filepath.Join(homeDir, ".local", "share", "hyprwhspr")

	return &Config{
		Model:            "base",
		Threads:          4,
		Language:         nil,        // auto-detect
		AllowedLanguages: []string{}, // empty = all languages allowed
		AudioDevice:      nil,        // default device
		SampleRate:       16000,
		SocketPath:       socketPath,
		WhisperModelDir:  modelDir,
		AudioFeedback:    true,                    // Enable audio feedback by default
		StartSoundVolume: 0.4,                     // 40% volume for start sound
		StopSoundVolume:  0.4,                     // 40% volume for stop sound
		StartSoundPath:   nil,                     // Use default
		StopSoundPath:    nil,                     // Use default
		CommandMode:      false,                   // Disabled by default
		Commands:         make(map[string]string), // Empty by default
		WhisperPrompt:    "Transcribe with proper capitalization, including sentence beginnings, proper nouns, titles, and standard English capitalization rules.",

		// Echo Cancellation defaults
		EchoCancellation:   true, // Enable AEC by default
		AECFilterLength:    1024, // Default filter length
		AECStepSize:        0.05, // Default step size
		AECEchoSuppression: 0.7,  // Default echo suppression

		// VAD defaults
		VoiceActivityDetection: true, // Enable VAD by default
		VADEnergyThreshold:     0.01, // Default energy threshold
		VADVoiceThreshold:      0.5,  // Default voice probability threshold
	}
}

// Load loads configuration from file
func Load(configPath string) (*Config, error) {
	// Start with defaults
	cfg := Default()

	// Try to read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config doesn't exist, return defaults
			return cfg, nil
		}
		return nil, err
	}

	// Parse JSON
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save saves configuration to file
func (c *Config) Save(configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(configPath, data, 0644)
}

// GetConfigPath returns the default config path
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "hyprwhspr", "config.json")
}
