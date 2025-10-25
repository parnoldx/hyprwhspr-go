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
	Language         *string           `json:"language"`         // nil = auto-detect
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
}

// Default returns default configuration
func Default() *Config {
	homeDir, _ := os.UserHomeDir()
	socketPath := filepath.Join(homeDir, ".config", "hyprwhspr", "hyprwhspr.sock")
	modelDir := filepath.Join(homeDir, ".local", "share", "hyprwhspr")

	return &Config{
		Model:            "base",
		Threads:          4,
		Language:         nil,           // auto-detect
		AllowedLanguages: []string{},    // empty = all languages allowed
		AudioDevice:      nil,           // default device
		SampleRate:       16000,
		SocketPath:       socketPath,
		WhisperModelDir:  modelDir,
		AudioFeedback:    true,  // Enable audio feedback by default
		StartSoundVolume: 0.4,   // 40% volume for start sound
		StopSoundVolume:  0.4,   // 40% volume for stop sound
		StartSoundPath:   nil,   // Use default
		StopSoundPath:    nil,   // Use default
		CommandMode:      false, // Disabled by default
		Commands:         make(map[string]string), // Empty by default
		WhisperPrompt:    "Transcribe with proper capitalization, including sentence beginnings, proper nouns, titles, and standard English capitalization rules.",
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
