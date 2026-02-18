package services

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type TranscribeService struct {
	language   string
	modelPath  string
	whisperBin string
}

func (t *TranscribeService) ServiceName() string {
	return "TranscribeService"
}

func (t *TranscribeService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	t.language = "ja"
	t.modelPath = t.findModelPath()
	t.whisperBin = t.findWhisperBin()
	return nil
}

func (t *TranscribeService) ServiceShutdown() error {
	return nil
}

func (t *TranscribeService) Transcribe(wavPath string) (string, error) {
	if !t.IsWhisperAvailable() {
		return "", fmt.Errorf("whisper-cpp is not installed. Please install it with: brew install whisper-cpp")
	}

	modelPath := t.modelPath
	if modelPath == "" {
		return "", fmt.Errorf("whisper model not found. Please download a model file")
	}

	args := []string{
		"--model", modelPath,
		"--language", t.language,
		"--output-txt",
		"--no-prints",
		wavPath,
	}

	cmd := exec.Command(t.whisperBin, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("whisper-cpp failed: %w\nOutput: %s", err, string(output))
	}

	// whisper-cpp with --output-txt writes to <input>.txt
	txtPath := wavPath + ".txt"
	text, err := os.ReadFile(txtPath)
	if err != nil {
		// Fallback: try to use stdout
		return strings.TrimSpace(string(output)), nil
	}
	defer os.Remove(txtPath)

	return strings.TrimSpace(string(text)), nil
}

func (t *TranscribeService) TranscribeToFile(wavPath string) (string, error) {
	text, err := t.Transcribe(wavPath)
	if err != nil {
		return "", err
	}

	saveDir := filepath.Join(os.Getenv("HOME"), "Documents", "Transcriptions")
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create save directory: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_150405")
	mdPath := filepath.Join(saveDir, timestamp+".md")

	content := fmt.Sprintf("# Meeting Transcription\n\n**Date:** %s\n\n---\n\n%s\n",
		time.Now().Format("2006-01-02 15:04:05"),
		text,
	)

	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write transcription file: %w", err)
	}

	// Copy WAV file to the same directory for verification
	wavDst := filepath.Join(saveDir, timestamp+".wav")
	if wavData, err := os.ReadFile(wavPath); err == nil {
		os.WriteFile(wavDst, wavData, 0644)
	}

	return mdPath, nil
}

func (t *TranscribeService) IsWhisperAvailable() bool {
	return t.whisperBin != ""
}

func (t *TranscribeService) findWhisperBin() string {
	// Try PATH first
	for _, name := range []string{"whisper-cli", "whisper-cpp"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}

	// macOS GUI apps don't inherit shell PATH, so check Homebrew paths directly
	homebrewBins := []string{
		"/opt/homebrew/bin", // Apple Silicon
		"/usr/local/bin",   // Intel
	}
	binNames := []string{"whisper-cli", "whisper-cpp"}

	for _, dir := range homebrewBins {
		for _, name := range binNames {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	return ""
}

func (t *TranscribeService) GetModelPath() string {
	return t.modelPath
}

func (t *TranscribeService) RefreshModelPath() string {
	t.modelPath = t.findModelPath()
	return t.modelPath
}

func (t *TranscribeService) SetLanguage(lang string) error {
	if lang == "" {
		return fmt.Errorf("language cannot be empty")
	}
	t.language = lang
	return nil
}

func (t *TranscribeService) findModelPath() string {
	// Check common locations for whisper models
	candidates := []string{
		"models/ggml-large-v3.bin",
		"models/ggml-medium.bin",
		"models/ggml-base.bin",
		"models/ggml-small.bin",
	}

	// Check project-local models directory
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			abs, _ := filepath.Abs(p)
			return abs
		}
	}

	// Check Homebrew whisper-cpp model locations
	homebrewPaths := []string{
		"/opt/homebrew/share/whisper-cpp/models",
		"/usr/local/share/whisper-cpp/models",
	}
	modelNames := []string{
		"ggml-large-v3.bin",
		"ggml-medium.bin",
		"ggml-base.bin",
		"ggml-small.bin",
	}

	for _, dir := range homebrewPaths {
		for _, model := range modelNames {
			p := filepath.Join(dir, model)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	// Check home directory models
	home := os.Getenv("HOME")
	if home != "" {
		for _, model := range modelNames {
			p := filepath.Join(home, ".local", "share", "whisper-cpp", "models", model)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	return ""
}
