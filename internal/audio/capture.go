package audio

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/gen2brain/malgo"
)

// Recorder handles audio recording
type Recorder struct {
	ctx        *malgo.AllocatedContext
	device     *malgo.Device
	deviceName *string
	sampleRate uint32
	channels   uint32

	mu        sync.Mutex
	recording bool
	samples   []float32
}

// NewRecorder creates a new audio recorder
// deviceName: optional device name filter (e.g. "Mic1", "default", or nil for default)
func NewRecorder(sampleRate int, deviceName *string) (*Recorder, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize audio context: %w", err)
	}

	// List and display available devices
	if err := listAvailableDevices(ctx); err != nil {
		fmt.Printf("[WARN] Failed to list audio devices: %v\n", err)
	}

	return &Recorder{
		ctx:        ctx,
		deviceName: deviceName,
		sampleRate: uint32(sampleRate),
		channels:   1, // mono
		samples:    make([]float32, 0),
	}, nil
}

// listAvailableDevices prints all available capture devices
func listAvailableDevices(ctx *malgo.AllocatedContext) error {
	devices, err := ctx.Devices(malgo.Capture)
	if err != nil {
		return err
	}

	fmt.Println("[audio] Available capture devices:")
	for i, device := range devices {
		deviceType := "🎤 MICROPHONE"
		if strings.Contains(strings.ToLower(device.Name()), "monitor") {
			deviceType = "🔊 SYSTEM AUDIO (avoid this)"
		}
		fmt.Printf("  [%d] %s - %s\n", i, device.Name(), deviceType)
	}
	return nil
}

// Start starts recording audio
func (r *Recorder) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.recording {
		return fmt.Errorf("already recording")
	}

	// Reset samples buffer
	r.samples = make([]float32, 0, r.sampleRate*10) // pre-allocate for ~10 seconds

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = r.channels
	deviceConfig.SampleRate = r.sampleRate
	deviceConfig.Alsa.NoMMap = 1

	// Select specific device if deviceName is provided
	if r.deviceName != nil && *r.deviceName != "" {
		devices, err := r.ctx.Devices(malgo.Capture)
		if err != nil {
			return fmt.Errorf("failed to list devices: %w", err)
		}

		deviceFound := false
		for _, dev := range devices {
			if containsIgnoreCase(dev.Name(), *r.deviceName) {
				deviceConfig.Capture.DeviceID = dev.ID.Pointer()

				// Warn if selecting a monitor device
				if strings.Contains(strings.ToLower(dev.Name()), "monitor") {
					fmt.Printf("⚠️  WARNING: Selected device '%s' is a MONITOR (system audio)\n", dev.Name())
					fmt.Printf("⚠️  This will capture playing audio, not your microphone!\n")
				} else {
					fmt.Printf("✅ Using microphone: %s\n", dev.Name())
				}

				deviceFound = true
				break
			}
		}

		if !deviceFound {
			fmt.Printf("[WARN] Device '%s' not found, using default device\n", *r.deviceName)
			fmt.Println("[WARN] Check available devices list above")
		}
	} else {
		fmt.Println("[audio] Using default capture device")
	}

	// Callback to receive audio data
	onRecvFrames := func(pSample2, pSample []byte, framecount uint32) {
		r.mu.Lock()
		defer r.mu.Unlock()

		if !r.recording {
			return
		}

		// Convert bytes to float32 samples
		samples := make([]float32, framecount)
		for i := uint32(0); i < framecount; i++ {
			idx := i * 4 // 4 bytes per float32
			if idx+3 < uint32(len(pSample)) {
				// Convert bytes to float32 (little-endian)
				bits := uint32(pSample[idx]) |
					uint32(pSample[idx+1])<<8 |
					uint32(pSample[idx+2])<<16 |
					uint32(pSample[idx+3])<<24
				samples[i] = *(*float32)(unsafe.Pointer(&bits))
			}
		}

		r.samples = append(r.samples, samples...)
	}

	var err error
	r.device, err = malgo.InitDevice(r.ctx.Context, deviceConfig, malgo.DeviceCallbacks{
		Data: onRecvFrames,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize device: %w", err)
	}

	if err := r.device.Start(); err != nil {
		return fmt.Errorf("failed to start device: %w", err)
	}

	r.recording = true
	fmt.Println("🎤 Recording started")
	return nil
}

// Stop stops recording and returns the captured audio
func (r *Recorder) Stop() ([]float32, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.recording {
		return nil, fmt.Errorf("not recording")
	}

	r.recording = false

	if r.device != nil {
		r.device.Stop()
		r.device.Uninit()
		r.device = nil
	}

	fmt.Printf("🛑 Recording stopped (%d samples)\n", len(r.samples))
	return r.samples, nil
}

// IsRecording returns true if currently recording
func (r *Recorder) IsRecording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.recording
}

// Close closes the recorder and releases resources
func (r *Recorder) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.device != nil {
		r.device.Uninit()
		r.device = nil
	}

	if r.ctx != nil {
		_ = r.ctx.Uninit()
		r.ctx.Free()
		r.ctx = nil
	}
}

// containsIgnoreCase checks if haystack contains needle (case-insensitive)
func containsIgnoreCase(haystack, needle string) bool {
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(needle))
}
