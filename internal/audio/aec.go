package audio

import (
	"fmt"
	"math"
	"sync"
)

// AECConfig contains configuration for acoustic echo cancellation
type AECConfig struct {
	FilterLength    int     // Length of adaptive filter (typically 512-2048)
	StepSize        float64 // Adaptation step size (0.01-0.1)
	LeakageFactor   float64 // Leakage factor to prevent filter windup (0.99-1.0)
	EchoSuppression float64 // Echo suppression gain (0.0-1.0)
}

// DefaultAECConfig returns default AEC configuration
func DefaultAECConfig() AECConfig {
	return AECConfig{
		FilterLength:    1024,
		StepSize:        0.05,
		LeakageFactor:   0.999,
		EchoSuppression: 0.7,
	}
}

// AECProcessor implements acoustic echo cancellation using NLMS algorithm
type AECProcessor struct {
	config AECConfig

	// Adaptive filter coefficients
	filter []float64

	// Buffer for far-end (reference) signal
	farEndBuffer []float64
	farEndIndex  int

	mu sync.Mutex
}

// NewAECProcessor creates a new AEC processor
func NewAECProcessor(config AECConfig) *AECProcessor {
	return &AECProcessor{
		config:       config,
		filter:       make([]float64, config.FilterLength),
		farEndBuffer: make([]float64, config.FilterLength),
		farEndIndex:  0,
	}
}

// ProcessFrame processes a single audio frame with echo cancellation
func (aec *AECProcessor) ProcessFrame(micSignal, farEndSignal []float32) []float32 {
	aec.mu.Lock()
	defer aec.mu.Unlock()

	if len(micSignal) != len(farEndSignal) {
		fmt.Printf("[WARN] AEC: signal length mismatch: mic=%d, farend=%d\n", len(micSignal), len(farEndSignal))
		return micSignal
	}

	output := make([]float32, len(micSignal))

	for i := 0; i < len(micSignal); i++ {
		// Update far-end buffer
		aec.farEndBuffer[aec.farEndIndex] = float64(farEndSignal[i])
		aec.farEndIndex = (aec.farEndIndex + 1) % aec.config.FilterLength

		// Compute estimated echo
		echoEstimate := 0.0
		for j := 0; j < aec.config.FilterLength; j++ {
			bufferIndex := (aec.farEndIndex - 1 - j + aec.config.FilterLength) % aec.config.FilterLength
			echoEstimate += aec.filter[j] * aec.farEndBuffer[bufferIndex]
		}

		// Error signal (mic signal - estimated echo)
		errorSignal := float64(micSignal[i]) - echoEstimate

		// Update filter coefficients using NLMS
		power := 0.0
		for j := 0; j < aec.config.FilterLength; j++ {
			bufferIndex := (aec.farEndIndex - 1 - j + aec.config.FilterLength) % aec.config.FilterLength
			power += aec.farEndBuffer[bufferIndex] * aec.farEndBuffer[bufferIndex]
		}

		if power > 1e-10 { // Avoid division by zero
			normalizedStepSize := aec.config.StepSize / (power + 1e-10)
			for j := 0; j < aec.config.FilterLength; j++ {
				bufferIndex := (aec.farEndIndex - 1 - j + aec.config.FilterLength) % aec.config.FilterLength
				aec.filter[j] = aec.config.LeakageFactor*aec.filter[j] +
					normalizedStepSize*errorSignal*aec.farEndBuffer[bufferIndex]
			}
		}

		// Apply echo suppression
		suppressedSignal := errorSignal * aec.config.EchoSuppression

		// Soft clipping to prevent distortion
		if suppressedSignal > 1.0 {
			suppressedSignal = 1.0
		} else if suppressedSignal < -1.0 {
			suppressedSignal = -1.0
		}

		output[i] = float32(suppressedSignal)
	}

	return output
}

// Reset resets the AEC processor state
func (aec *AECProcessor) Reset() {
	aec.mu.Lock()
	defer aec.mu.Unlock()

	for i := range aec.filter {
		aec.filter[i] = 0.0
	}
	for i := range aec.farEndBuffer {
		aec.farEndBuffer[i] = 0.0
	}
	aec.farEndIndex = 0
}

// GetEchoReturnLossEnhancement calculates ERLE in dB
func (aec *AECProcessor) GetEchoReturnLossEnhancement(micSignal, farendSignal, outputSignal []float32) float64 {
	if len(micSignal) == 0 || len(outputSignal) == 0 {
		return 0.0
	}

	micPower := 0.0
	outputPower := 0.0

	for i := 0; i < len(micSignal) && i < len(outputSignal); i++ {
		micPower += float64(micSignal[i] * micSignal[i])
		outputPower += float64(outputSignal[i] * outputSignal[i])
	}

	if outputPower < 1e-10 {
		return 60.0 // Cap at 60dB
	}

	erle := 10.0 * math.Log10(micPower/outputPower)
	if erle > 60.0 {
		erle = 60.0
	}
	if erle < 0.0 {
		erle = 0.0
	}

	return erle
}
