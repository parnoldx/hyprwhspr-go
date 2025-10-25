package models

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	ModelBaseURL = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main"
)

var AvailableModels = []string{
	"tiny",
	"tiny.en",
	"base",
	"base.en",
	"small",
	"small.en",
	"medium",
	"medium.en",
	"large-v1",
	"large-v2",
	"large-v3",
	"large",
}

type Manager struct {
	modelDir string
}

func NewManager(modelDir string) *Manager {
	return &Manager{
		modelDir: modelDir,
	}
}

func (m *Manager) GetModelDir() string {
	return m.modelDir
}

func (m *Manager) EnsureModelDir() error {
	return os.MkdirAll(m.modelDir, 0755)
}

func (m *Manager) ListAvailableModels() []string {
	return AvailableModels
}

func (m *Manager) ListDownloadedModels() ([]string, error) {
	files, err := os.ReadDir(m.modelDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var models []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".bin") && strings.HasPrefix(file.Name(), "ggml-") {
			modelName := strings.TrimPrefix(file.Name(), "ggml-")
			modelName = strings.TrimSuffix(modelName, ".bin")
			models = append(models, modelName)
		}
	}

	return models, nil
}

func (m *Manager) IsModelDownloaded(model string) bool {
	modelPath := filepath.Join(m.modelDir, fmt.Sprintf("ggml-%s.bin", model))
	_, err := os.Stat(modelPath)
	return err == nil
}

func (m *Manager) GetModelPath(model string) string {
	return filepath.Join(m.modelDir, fmt.Sprintf("ggml-%s.bin", model))
}

func (m *Manager) DownloadModel(model string, progressCallback func(float64)) error {
	// Validate model name
	if !m.isValidModel(model) {
		return fmt.Errorf("invalid model name: %s", model)
	}

	// Check if already downloaded
	if m.IsModelDownloaded(model) {
		return fmt.Errorf("model %s is already downloaded", model)
	}

	// Ensure model directory exists
	if err := m.EnsureModelDir(); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Download URL
	url := fmt.Sprintf("%s/ggml-%s.bin", ModelBaseURL, model)
	outputPath := m.GetModelPath(model)

	fmt.Printf("📥 Downloading model '%s' from %s\n", model, url)

	// Download with progress tracking
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create model file: %w", err)
	}
	defer file.Close()

	// Get content length for progress tracking
	contentLength := resp.ContentLength
	if contentLength <= 0 {
		contentLength = 1 // Avoid division by zero
	}

	// Copy with progress tracking
	var downloaded int64
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			written, err := file.Write(buffer[:n])
			if err != nil {
				return fmt.Errorf("failed to write model file: %w", err)
			}
			downloaded += int64(written)

			// Report progress
			if progressCallback != nil {
				progress := float64(downloaded) / float64(contentLength)
				if progress > 1.0 {
					progress = 1.0
				}
				progressCallback(progress)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("download interrupted: %w", err)
		}
	}

	fmt.Printf("✅ Model '%s' downloaded successfully to %s\n", model, outputPath)
	return nil
}

func (m *Manager) DownloadModelWithProgress(model string) error {
	return m.DownloadModel(model, func(progress float64) {
		// Simple progress bar
		percentage := int(progress * 100)
		bar := strings.Repeat("=", percentage/5) + strings.Repeat(" ", 20-percentage/5)
		fmt.Printf("\r📥 [%s] %d%%", bar, percentage)
		if progress >= 1.0 {
			fmt.Println() // New line when complete
		}
	})
}

func (m *Manager) DeleteModel(model string) error {
	if !m.isValidModel(model) {
		return fmt.Errorf("invalid model name: %s", model)
	}

	modelPath := m.GetModelPath(model)
	if !m.IsModelDownloaded(model) {
		return fmt.Errorf("model %s is not downloaded", model)
	}

	if err := os.Remove(modelPath); err != nil {
		return fmt.Errorf("failed to delete model: %w", err)
	}

	fmt.Printf("🗑️  Model '%s' deleted successfully\n", model)
	return nil
}

func (m *Manager) GetModelSize(model string) (int64, error) {
	if !m.IsModelDownloaded(model) {
		return 0, fmt.Errorf("model %s is not downloaded", model)
	}

	modelPath := m.GetModelPath(model)
	info, err := os.Stat(modelPath)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

func (m *Manager) isValidModel(model string) bool {
	for _, availableModel := range AvailableModels {
		if availableModel == model {
			return true
		}
	}
	return false
}

func (m *Manager) PrintModelInfo(activeModel string) {
	fmt.Println("🤖 Whisper Models:")
	fmt.Printf("🎯 Active model: %s\n\n", activeModel)

	// Downloaded models
	downloaded, err := m.ListDownloadedModels()
	if err != nil {
		fmt.Printf("❌ Failed to list downloaded models: %v\n", err)
		return
	}

	if len(downloaded) > 0 {
		fmt.Println("📁 Downloaded models:")
		for _, model := range downloaded {
			size, _ := m.GetModelSize(model)
			sizeMB := float64(size) / (1024 * 1024)
			if model == activeModel {
				fmt.Printf("  ★ %s (%.1f MB) [ACTIVE]\n", model, sizeMB)
			} else {
				fmt.Printf("  ✓ %s (%.1f MB)\n", model, sizeMB)
			}
		}
		fmt.Println()
	}

	// Available models
	fmt.Println("📥 Available models:")
	for _, model := range AvailableModels {
		if m.IsModelDownloaded(model) {
			if model == activeModel {
				fmt.Printf("  ★ %s (downloaded) [ACTIVE]\n", model)
			} else {
				fmt.Printf("  ✓ %s (downloaded)\n", model)
			}
		} else {
			if model == activeModel {
				fmt.Printf("  ★ %s [ACTIVE - not downloaded]\n", model)
			} else {
				fmt.Printf("  ○ %s\n", model)
			}
		}
	}
	fmt.Println()

	// Model recommendations
	fmt.Println("💡 Recommendations:")
	fmt.Println("  • tiny    - Fastest, lowest accuracy (~39MB)")
	fmt.Println("  • base    - Good balance of speed and accuracy (~142MB)")
	fmt.Println("  • small   - Better accuracy, slower (~466MB)")
	fmt.Println("  • medium  - High accuracy, much slower (~1.5GB)")
	fmt.Println("  • large   - Best accuracy, slowest (~2.9GB)")
	fmt.Println()
	fmt.Println("  Models ending with '.en' are English-only and slightly faster.")
	fmt.Println("  Use 'hyprwhspr download <model>' to download a model.")
	fmt.Println("  Use 'hyprwhspr model <model>' to set the active model.")
}
