package services

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	outputSampleRate = 16000 // whisper.cpp expects 16kHz
	channels         = 1
	bitDepth         = 16
	bufferSize       = 1024
	spectrumBands    = 32
)

type recordingState int

const (
	stateIdle recordingState = iota
	stateRecording
	statePaused
)

func (s recordingState) String() string {
	switch s {
	case stateRecording:
		return "recording"
	case statePaused:
		return "paused"
	default:
		return "idle"
	}
}

type AudioService struct {
	mu          sync.Mutex
	state       recordingState
	stream      *portaudio.Stream
	nativeSR    float64 // device's native sample rate
	samples     []int16 // recorded at native sample rate
	startTime   time.Time
	elapsed     time.Duration
	pauseStart  time.Time
	totalPaused time.Duration

	// Ring buffer for spectrum visualization (latest callback data)
	specBuf []int16
}

func (a *AudioService) ServiceName() string {
	return "AudioService"
}

func (a *AudioService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	return portaudio.Initialize()
}

func (a *AudioService) ServiceShutdown() error {
	return portaudio.Terminate()
}

func (a *AudioService) StartRecording() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != stateIdle {
		return fmt.Errorf("cannot start recording: current state is %s", a.state)
	}

	// Detect native sample rate
	host, err := portaudio.DefaultHostApi()
	if err != nil {
		return fmt.Errorf("failed to get default host API: %w", err)
	}
	dev := host.DefaultInputDevice
	if dev == nil {
		return fmt.Errorf("no default input device found")
	}
	a.nativeSR = dev.DefaultSampleRate

	a.samples = nil
	a.totalPaused = 0
	a.specBuf = nil

	stream, err := portaudio.OpenDefaultStream(channels, 0, a.nativeSR, bufferSize, func(in []int16) {
		a.mu.Lock()
		defer a.mu.Unlock()
		// Always update spectrum buffer for visualization
		a.specBuf = make([]int16, len(in))
		copy(a.specBuf, in)
		if a.state == stateRecording {
			a.samples = append(a.samples, in...)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to open audio stream: %w", err)
	}

	if err := stream.Start(); err != nil {
		stream.Close()
		return fmt.Errorf("failed to start audio stream: %w", err)
	}

	a.stream = stream
	a.state = stateRecording
	a.startTime = time.Now()

	return nil
}

func (a *AudioService) PauseRecording() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != stateRecording {
		return fmt.Errorf("cannot pause: current state is %s", a.state)
	}

	a.state = statePaused
	a.pauseStart = time.Now()
	return nil
}

func (a *AudioService) ResumeRecording() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != statePaused {
		return fmt.Errorf("cannot resume: current state is %s", a.state)
	}

	a.totalPaused += time.Since(a.pauseStart)
	a.state = stateRecording
	return nil
}

func (a *AudioService) StopRecording() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state == stateIdle {
		return "", fmt.Errorf("not recording")
	}

	if a.state == statePaused {
		a.totalPaused += time.Since(a.pauseStart)
	}

	a.elapsed = time.Since(a.startTime) - a.totalPaused

	if err := a.stream.Stop(); err != nil {
		a.stream.Close()
		a.state = stateIdle
		return "", fmt.Errorf("failed to stop stream: %w", err)
	}
	a.stream.Close()
	a.state = stateIdle

	wavPath, err := a.writeWAV()
	if err != nil {
		return "", fmt.Errorf("failed to write WAV: %w", err)
	}

	return wavPath, nil
}

func (a *AudioService) GetElapsedTime() float64 {
	a.mu.Lock()
	defer a.mu.Unlock()

	switch a.state {
	case stateRecording:
		return (time.Since(a.startTime) - a.totalPaused).Seconds()
	case statePaused:
		return (time.Since(a.startTime) - a.totalPaused - time.Since(a.pauseStart)).Seconds()
	default:
		return a.elapsed.Seconds()
	}
}

func (a *AudioService) GetRecordingState() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state.String()
}

// GetSpectrum returns frequency band magnitudes (0.0-1.0) for visualization.
// Uses logarithmic frequency scaling focused on the voice range (80Hz-12kHz).
func (a *AudioService) GetSpectrum() []float64 {
	a.mu.Lock()
	buf := a.specBuf
	sr := a.nativeSR
	a.mu.Unlock()

	result := make([]float64, spectrumBands)
	if len(buf) == 0 || sr == 0 {
		return result
	}

	n := len(buf)
	freqRes := sr / float64(n) // Hz per DFT bin

	// Logarithmic band edges from 80Hz to 12kHz
	const minFreq = 80.0
	const maxFreq = 12000.0
	logMin := math.Log2(minFreq)
	logMax := math.Log2(maxFreq)

	// Compute DFT magnitudes for all needed bins (up to maxFreq)
	maxBin := int(maxFreq/freqRes) + 1
	if maxBin > n/2 {
		maxBin = n / 2
	}
	mags := make([]float64, maxBin+1)
	for k := 1; k <= maxBin; k++ {
		re, im := 0.0, 0.0
		for i, s := range buf {
			angle := 2.0 * math.Pi * float64(k) * float64(i) / float64(n)
			re += float64(s) * math.Cos(angle)
			im -= float64(s) * math.Sin(angle)
		}
		mags[k] = math.Sqrt(re*re+im*im) / float64(n)
	}

	// Map DFT bins to logarithmic bands
	for band := 0; band < spectrumBands; band++ {
		fLow := math.Pow(2, logMin+(logMax-logMin)*float64(band)/float64(spectrumBands))
		fHigh := math.Pow(2, logMin+(logMax-logMin)*float64(band+1)/float64(spectrumBands))

		kLow := int(fLow / freqRes)
		kHigh := int(fHigh / freqRes)
		if kLow < 1 {
			kLow = 1
		}
		if kHigh > maxBin {
			kHigh = maxBin
		}
		if kHigh < kLow {
			kHigh = kLow
		}

		// Average magnitude within this band
		sum := 0.0
		count := 0
		for k := kLow; k <= kHigh; k++ {
			sum += mags[k]
			count++
		}
		if count > 0 {
			sum /= float64(count)
		}

		// Normalize with log scaling for better dynamic range
		// Apply gain boost for typical speech levels
		normalized := sum / 800.0
		if normalized > 0 {
			normalized = (math.Log10(normalized*9+1) / math.Log10(10))
		}
		if normalized > 1.0 {
			normalized = 1.0
		}
		result[band] = normalized
	}

	return result
}

// downsample converts from nativeSR to outputSampleRate using simple linear interpolation.
func (a *AudioService) downsample() []int16 {
	if a.nativeSR == float64(outputSampleRate) {
		return a.samples
	}

	ratio := a.nativeSR / float64(outputSampleRate)
	outLen := int(float64(len(a.samples)) / ratio)
	out := make([]int16, outLen)

	for i := range out {
		srcPos := float64(i) * ratio
		idx := int(srcPos)
		frac := srcPos - float64(idx)

		if idx+1 < len(a.samples) {
			out[i] = int16(float64(a.samples[idx])*(1-frac) + float64(a.samples[idx+1])*frac)
		} else if idx < len(a.samples) {
			out[i] = a.samples[idx]
		}
	}

	return out
}

func (a *AudioService) writeWAV() (string, error) {
	tmpDir := os.TempDir()
	filename := fmt.Sprintf("meeting_%s.wav", time.Now().Format("20060102_150405"))
	wavPath := filepath.Join(tmpDir, filename)

	// Downsample to 16kHz for whisper.cpp
	samples := a.downsample()

	f, err := os.Create(wavPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	dataSize := uint32(len(samples) * 2) // 16-bit = 2 bytes per sample
	fileSize := 36 + dataSize

	// RIFF header
	f.Write([]byte("RIFF"))
	binary.Write(f, binary.LittleEndian, fileSize)
	f.Write([]byte("WAVE"))

	// fmt sub-chunk
	f.Write([]byte("fmt "))
	binary.Write(f, binary.LittleEndian, uint32(16))                                      // sub-chunk size
	binary.Write(f, binary.LittleEndian, uint16(1))                                        // PCM format
	binary.Write(f, binary.LittleEndian, uint16(channels))                                 // channels
	binary.Write(f, binary.LittleEndian, uint32(outputSampleRate))                         // sample rate
	binary.Write(f, binary.LittleEndian, uint32(outputSampleRate*channels*bitDepth/8))     // byte rate
	binary.Write(f, binary.LittleEndian, uint16(channels*bitDepth/8))                      // block align
	binary.Write(f, binary.LittleEndian, uint16(bitDepth))                                 // bits per sample

	// data sub-chunk
	f.Write([]byte("data"))
	binary.Write(f, binary.LittleEndian, dataSize)
	binary.Write(f, binary.LittleEndian, samples)

	return wavPath, nil
}
