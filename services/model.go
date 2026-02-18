package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type ModelInfo struct {
	Name     string `json:"name"`
	FileName string `json:"fileName"`
	Size     string `json:"size"`
	URL      string `json:"url"`
	Exists   bool   `json:"exists"`
}

type DownloadProgress struct {
	ModelName   string  `json:"modelName"`
	BytesLoaded int64   `json:"bytesLoaded"`
	BytesTotal  int64   `json:"bytesTotal"`
	Percent     float64 `json:"percent"`
	Done        bool    `json:"done"`
	Error       string  `json:"error,omitempty"`
}

type ModelService struct {
	mu          sync.Mutex
	cancelFunc  context.CancelFunc
	downloading bool
}

var modelDefinitions = []ModelInfo{
	{
		Name:     "base",
		FileName: "ggml-base.bin",
		Size:     "142 MB",
		URL:      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin",
	},
	{
		Name:     "small",
		FileName: "ggml-small.bin",
		Size:     "466 MB",
		URL:      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin",
	},
	{
		Name:     "medium",
		FileName: "ggml-medium.bin",
		Size:     "1.5 GB",
		URL:      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin",
	},
	{
		Name:     "large-v3",
		FileName: "ggml-large-v3.bin",
		Size:     "3.1 GB",
		URL:      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin",
	},
}

func (m *ModelService) ServiceName() string {
	return "ModelService"
}

func (m *ModelService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	return nil
}

func (m *ModelService) ServiceShutdown() error {
	m.CancelDownload()
	return nil
}

func (m *ModelService) GetModelsDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share", "whisper-cpp", "models")
}

func (m *ModelService) ListModels() []ModelInfo {
	dir := m.GetModelsDir()
	models := make([]ModelInfo, len(modelDefinitions))
	for i, def := range modelDefinitions {
		models[i] = def
		if dir != "" {
			p := filepath.Join(dir, def.FileName)
			if _, err := os.Stat(p); err == nil {
				models[i].Exists = true
			}
		}
	}
	return models
}

func (m *ModelService) DownloadModel(name string) error {
	m.mu.Lock()
	if m.downloading {
		m.mu.Unlock()
		return fmt.Errorf("a download is already in progress")
	}

	var model *ModelInfo
	for _, def := range modelDefinitions {
		if def.Name == name {
			model = &def
			break
		}
	}
	if model == nil {
		m.mu.Unlock()
		return fmt.Errorf("unknown model: %s", name)
	}

	dir := m.GetModelsDir()
	if dir == "" {
		m.mu.Unlock()
		return fmt.Errorf("cannot determine models directory")
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFunc = cancel
	m.downloading = true
	m.mu.Unlock()

	go m.doDownload(ctx, *model, dir)
	return nil
}

func (m *ModelService) CancelDownload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}
	return nil
}

func (m *ModelService) IsDownloading() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.downloading
}

func (m *ModelService) doDownload(ctx context.Context, model ModelInfo, dir string) {
	defer func() {
		m.mu.Lock()
		m.downloading = false
		m.cancelFunc = nil
		m.mu.Unlock()
	}()

	emit := func(p DownloadProgress) {
		application.Get().Event.Emit("model:download-progress", p)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		emit(DownloadProgress{ModelName: model.Name, Error: fmt.Sprintf("failed to create directory: %v", err)})
		return
	}

	// Verify the directory is writable before starting the download
	testFile, err := os.CreateTemp(dir, ".model-download-writetest-*")
	if err != nil {
		emit(DownloadProgress{ModelName: model.Name, Error: fmt.Sprintf("directory is not writable: %v", err)})
		return
	}
	testFile.Close()
	os.Remove(testFile.Name())

	finalPath := filepath.Join(dir, model.FileName)
	partPath := finalPath + ".part"

	req, err := http.NewRequestWithContext(ctx, "GET", model.URL, nil)
	if err != nil {
		emit(DownloadProgress{ModelName: model.Name, Error: fmt.Sprintf("failed to create request: %v", err)})
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			os.Remove(partPath)
			emit(DownloadProgress{ModelName: model.Name, Error: "cancelled"})
			return
		}
		emit(DownloadProgress{ModelName: model.Name, Error: fmt.Sprintf("download failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		emit(DownloadProgress{ModelName: model.Name, Error: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)})
		return
	}

	total := resp.ContentLength

	f, err := os.Create(partPath)
	if err != nil {
		emit(DownloadProgress{ModelName: model.Name, Error: fmt.Sprintf("failed to create file: %v", err)})
		return
	}

	buf := make([]byte, 32*1024)
	var loaded int64
	lastEmit := time.Time{}
	var downloadErr error

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := f.Write(buf[:n]); writeErr != nil {
				downloadErr = fmt.Errorf("write failed: %v", writeErr)
				break
			}
			loaded += int64(n)

			now := time.Now()
			if now.Sub(lastEmit) >= 200*time.Millisecond || readErr != nil {
				var pct float64
				if total > 0 {
					pct = float64(loaded) / float64(total) * 100
				}
				emit(DownloadProgress{
					ModelName:   model.Name,
					BytesLoaded: loaded,
					BytesTotal:  total,
					Percent:     pct,
				})
				lastEmit = now
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			if ctx.Err() == context.Canceled {
				downloadErr = fmt.Errorf("cancelled")
			} else {
				downloadErr = fmt.Errorf("download failed: %v", readErr)
			}
			break
		}
	}

	f.Close()

	if downloadErr != nil {
		os.Remove(partPath)
		emit(DownloadProgress{ModelName: model.Name, Error: downloadErr.Error()})
		return
	}

	if err := os.Rename(partPath, finalPath); err != nil {
		os.Remove(partPath)
		emit(DownloadProgress{ModelName: model.Name, Error: fmt.Sprintf("failed to finalize file: %v", err)})
		return
	}

	emit(DownloadProgress{
		ModelName:   model.Name,
		BytesLoaded: loaded,
		BytesTotal:  total,
		Percent:     100,
		Done:        true,
	})
}
