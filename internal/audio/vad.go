package audio

import (
	"math"
	"sync"
)

// VADConfig contains configuration for voice activity detection
type VADConfig struct {
	FrameSize       int     // Analysis frame size in samples
	Overlap         int     // Overlap between frames
	EnergyThreshold float64 // Energy threshold for voice detection
	ZcrThreshold    float64 // Zero-crossing rate threshold
	VoiceThreshold  float64 // Probability threshold for voice (0.0-1.0)
}

// DefaultVADConfig returns default VAD configuration
func DefaultVADConfig() VADConfig {
	return VADConfig{
		FrameSize:       512,
		Overlap:         256,
		EnergyThreshold: 0.01,
		ZcrThreshold:    0.1,
		VoiceThreshold:  0.5,
	}
}

// VADProcessor implements voice activity detection
type VADProcessor struct {
	config VADConfig

	// Previous frame for overlap
	prevFrame []float32

	mu sync.Mutex
}

// NewVADProcessor creates a new VAD processor
func NewVADProcessor(config VADConfig) *VADProcessor {
	return &VADProcessor{
		config:    config,
		prevFrame: make([]float32, 0),
	}
}

// ProcessFrame detects voice activity in audio frame
func (vad *VADProcessor) ProcessFrame(audio []float32) bool {
	vad.mu.Lock()
	defer vad.mu.Unlock()

	if len(audio) < vad.config.FrameSize {
		return false
	}

	// Calculate energy
	energy := vad.calculateEnergy(audio)

	// Calculate zero-crossing rate
	zcr := vad.calculateZCR(audio)

	// Calculate spectral centroid (simplified)
	spectralCentroid := vad.calculateSpectralCentroid(audio)

	// Simple voice detection logic
	energyScore := 0.0
	if energy > vad.config.EnergyThreshold {
		energyScore = math.Min(energy/vad.config.EnergyThreshold, 2.0) / 2.0
	}

	zcrScore := 0.0
	if zcr < vad.config.ZcrThreshold {
		zcrScore = 1.0 - (zcr / vad.config.ZcrThreshold)
	}

	spectralScore := spectralCentroid / 2000.0 // Normalize around 2kHz
	if spectralScore > 1.0 {
		spectralScore = 1.0
	}

	// Combined voice probability
	voiceProbability := (energyScore * 0.5) + (zcrScore * 0.3) + (spectralScore * 0.2)

	return voiceProbability > vad.config.VoiceThreshold
}

// calculateEnergy calculates signal energy
func (vad *VADProcessor) calculateEnergy(audio []float32) float64 {
	energy := 0.0
	for _, sample := range audio {
		energy += float64(sample * sample)
	}
	return energy / float64(len(audio))
}

// calculateZCR calculates zero-crossing rate
func (vad *VADProcessor) calculateZCR(audio []float32) float64 {
	if len(audio) < 2 {
		return 0.0
	}

	crossings := 0
	for i := 1; i < len(audio); i++ {
		if (audio[i-1] >= 0 && audio[i] < 0) || (audio[i-1] < 0 && audio[i] >= 0) {
			crossings++
		}
	}

	return float64(crossings) / float64(len(audio)-1)
}

// calculateSpectralCentroid calculates spectral centroid (simplified)
func (vad *VADProcessor) calculateSpectralCentroid(audio []float32) float64 {
	if len(audio) == 0 {
		return 0.0
	}

	// Simple approximation using magnitude and position
	weightedSum := 0.0
	magnitudeSum := 0.0

	for i, sample := range audio {
		magnitude := math.Abs(float64(sample))
		weightedSum += magnitude * float64(i)
		magnitudeSum += magnitude
	}

	if magnitudeSum == 0.0 {
		return 0.0
	}

	return weightedSum / magnitudeSum
}

// IsVoiceDetected processes audio buffer and returns voice activity segments
func (vad *VADProcessor) IsVoiceDetected(audio []float32) []bool {
	if len(audio) < vad.config.FrameSize {
		return make([]bool, 0)
	}

	frameCount := (len(audio)-vad.config.FrameSize)/vad.config.Overlap + 1
	voiceActivity := make([]bool, frameCount)

	for i := 0; i < frameCount; i++ {
		start := i * vad.config.Overlap
		end := start + vad.config.FrameSize
		if end > len(audio) {
			end = len(audio)
		}

		frame := audio[start:end]
		if len(frame) == vad.config.FrameSize {
			voiceActivity[i] = vad.ProcessFrame(frame)
		}
	}

	return voiceActivity
}

// GetVoiceSegments returns continuous voice segments
func (vad *VADProcessor) GetVoiceSegments(audio []float32) []VoiceSegment {
	voiceActivity := vad.IsVoiceDetected(audio)
	if len(voiceActivity) == 0 {
		return nil
	}

	var segments []VoiceSegment
	inVoice := false
	segmentStart := 0

	frameDurationMs := float64(vad.config.Overlap) / 16000.0 * 1000.0 // Assuming 16kHz

	for i, isVoice := range voiceActivity {
		if isVoice && !inVoice {
			// Start of voice segment
			inVoice = true
			segmentStart = i
		} else if !isVoice && inVoice {
			// End of voice segment
			inVoice = false
			segments = append(segments, VoiceSegment{
				Start:    float64(segmentStart) * frameDurationMs,
				End:      float64(i) * frameDurationMs,
				Duration: float64(i-segmentStart) * frameDurationMs,
			})
		}
	}

	// Handle case where audio ends with voice
	if inVoice {
		segments = append(segments, VoiceSegment{
			Start:    float64(segmentStart) * frameDurationMs,
			End:      float64(len(voiceActivity)) * frameDurationMs,
			Duration: float64(len(voiceActivity)-segmentStart) * frameDurationMs,
		})
	}

	return segments
}

// VoiceSegment represents a continuous voice segment
type VoiceSegment struct {
	Start    float64 // Start time in milliseconds
	End      float64 // End time in milliseconds
	Duration float64 // Duration in milliseconds
}
