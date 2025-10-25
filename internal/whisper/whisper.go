package whisper

/*
#include <whisper.h>
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"os"
	"unsafe"
)

// Transcriber handles audio transcription using whisper.cpp
type Transcriber struct {
	ctx              *C.struct_whisper_context
	modelPath        string
	threads          int
	prompt           string
	allowedLanguages []string // Restrict detection to these languages (e.g. ["de", "en"])
}

// IsCudaEnabled returns whether CUDA support is enabled
func IsCudaEnabled() bool {
	return cudaEnabled
}

// New creates a new transcriber
func New(modelPath string, threads int, prompt string, allowedLanguages []string) (*Transcriber, error) {
	// Check if model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("model file not found: %s", modelPath)
	}

	fmt.Printf("[whisper] Model: %s\n", modelPath)
	fmt.Printf("[whisper] Threads: %d\n", threads)

	// Display language mode
	if len(allowedLanguages) > 0 {
		fmt.Printf("[whisper] Language mode: AUTO-DETECT (restricted to: %v)\n", allowedLanguages)
	} else {
		fmt.Println("[whisper] Language mode: AUTO-DETECT (all languages)")
	}

	if cudaEnabled {
		fmt.Println("[whisper] Acceleration: CUDA (GPU)")
	} else {
		fmt.Println("[whisper] Acceleration: CPU only")
	}
	if prompt != "" {
		fmt.Printf("[whisper] Initial prompt: %s\n", prompt)
	}

	// Initialize whisper context
	cModelPath := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cModelPath))

	ctx := C.whisper_init_from_file(cModelPath)
	if ctx == nil {
		return nil, fmt.Errorf("failed to initialize whisper model: %s", modelPath)
	}

	fmt.Println("[whisper] Model loaded successfully")

	return &Transcriber{
		ctx:              ctx,
		modelPath:        modelPath,
		threads:          threads,
		prompt:           prompt,
		allowedLanguages: allowedLanguages,
	}, nil
}

// Transcribe transcribes audio data to text
func (t *Transcriber) Transcribe(samples []float32) (string, error) {
	if len(samples) == 0 {
		return "", fmt.Errorf("no audio data")
	}

	if t.ctx == nil {
		return "", fmt.Errorf("whisper context not initialized")
	}

	fmt.Printf("🧠 Processing audio with Whisper (auto-detect language)...\n")
	fmt.Printf("   Samples: %d\n", len(samples))

	// Get default parameters
	params := C.whisper_full_default_params(C.WHISPER_SAMPLING_GREEDY)

	// Configure parameters
	params.n_threads = C.int(t.threads)
	params.print_realtime = C.bool(false)
	params.print_progress = C.bool(false)
	params.print_timestamps = C.bool(false)
	params.print_special = C.bool(false)
	params.translate = C.bool(false) // No translation, transcribe in detected language
	params.n_max_text_ctx = 16384
	params.offset_ms = 0
	params.duration_ms = 0
	params.single_segment = C.bool(false)

	// Set initial prompt if provided
	var cPrompt *C.char
	if t.prompt != "" {
		cPrompt = C.CString(t.prompt)
		defer C.free(unsafe.Pointer(cPrompt))
		params.initial_prompt = cPrompt
	}

	// Pre-detect language if allowed_languages is set
	if len(t.allowedLanguages) > 0 {
		// First, process audio to get mel spectrogram for language detection
		// We need to encode the audio first
		if C.whisper_pcm_to_mel(t.ctx, (*C.float)(unsafe.Pointer(&samples[0])), C.int(len(samples)), C.int(t.threads)) != 0 {
			fmt.Printf("[WARN] Failed to encode audio for language detection, using auto-detect\n")
			params.language = nil
		} else {
			// Get language probabilities
			maxLangID := int(C.whisper_lang_max_id())
			probs := make([]float32, maxLangID+1)

			langID := C.whisper_lang_auto_detect(
				t.ctx,
				0, // offset_ms
				C.int(t.threads),
				(*C.float)(unsafe.Pointer(&probs[0])),
			)

			if langID < 0 {
				fmt.Printf("[WARN] Language detection failed, using auto-detect\n")
				params.language = nil
			} else {
				// Find best language from allowed list
				bestLang := ""
				bestProb := float32(-1.0)

				for _, lang := range t.allowedLanguages {
					cLangTemp := C.CString(lang)
					id := int(C.whisper_lang_id(cLangTemp))
					C.free(unsafe.Pointer(cLangTemp))

					if id >= 0 && id < len(probs) {
						prob := probs[id]
						fmt.Printf("[DETECT] %s: %.2f%%\n", lang, prob*100)
						if prob > bestProb {
							bestProb = prob
							bestLang = lang
						}
					}
				}

				if bestLang != "" {
					fmt.Printf("[SELECTED] Using language: %s (%.2f%% confidence)\n", bestLang, bestProb*100)
					cLang := C.CString(bestLang)
					defer C.free(unsafe.Pointer(cLang))
					params.language = cLang
				} else {
					fmt.Printf("[WARN] No allowed language detected, using auto-detect\n")
					params.language = nil
				}
			}
		}
	} else {
		// No restriction, auto-detect from all languages
		params.language = nil
	}

	// Run transcription
	ret := C.whisper_full(
		t.ctx,
		params,
		(*C.float)(unsafe.Pointer(&samples[0])),
		C.int(len(samples)),
	)

	if ret != 0 {
		return "", fmt.Errorf("whisper_full failed with code: %d", ret)
	}

	// Get number of segments
	nSegments := int(C.whisper_full_n_segments(t.ctx))
	if nSegments == 0 {
		return "", fmt.Errorf("no segments transcribed")
	}

	// Concatenate all segments
	var result string
	for i := 0; i < nSegments; i++ {
		text := C.whisper_full_get_segment_text(t.ctx, C.int(i))
		if text != nil {
			result += C.GoString(text)
		}
	}

	// Show final language used for transcription
	langID := C.whisper_full_lang_id(t.ctx)
	if langID >= 0 {
		langStr := C.whisper_lang_str(langID)
		if langStr != nil {
			detectedLang := C.GoString(langStr)
			fmt.Printf("[TRANSCRIBED] Language: %s\n", detectedLang)
		}
	}

	return result, nil
}

// Close releases resources
func (t *Transcriber) Close() {
	if t.ctx != nil {
		C.whisper_free(t.ctx)
		t.ctx = nil
	}
}
