# Meeting Transcriber

On-device meeting audio transcription app for macOS. Records audio via PortAudio and transcribes using whisper.cpp — everything runs locally, no cloud services needed.

Built with [Wails v3](https://v3.wails.io/) (Go + React + TypeScript).

## Features

- Record meeting audio with pause/resume support
- On-device transcription via whisper.cpp (no data leaves your machine)
- Save transcriptions as Markdown files
- Dark theme macOS-native UI

## Install

```bash
brew tap dannygim/tap
brew install --cask meeting-transcriber
```

This installs the app to `/Applications/` and automatically pulls in dependencies (`portaudio`, `whisper-cpp`).

After installing, download a Whisper model from the app's settings, or manually:

```bash
mkdir -p ~/models && curl -L -o ~/models/ggml-large-v3.bin \
  https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin
```

## Development

```bash
# Install frontend dependencies
cd frontend && pnpm install && cd ..

# Run in development mode (hot-reload)
wails3 dev
```

## Build

```bash
wails3 task build
```

The built `.app` bundle will be in the `bin/` directory.

## Project Structure

```
├── main.go                  # Application entry point
├── services/
│   ├── audio.go             # AudioService (PortAudio recording, WAV output)
│   └── transcriber.go       # TranscribeService (whisper-cpp CLI integration)
├── frontend/
│   ├── src/
│   │   ├── App.tsx          # Main app component
│   │   ├── hooks/           # useRecorder, useTranscription
│   │   └── components/      # RecordButton, Timer, TranscriptView, etc.
│   └── public/style.css     # Dark theme styles
├── build/
│   ├── config.yml           # Wails build config
│   └── darwin/              # macOS-specific config (Info.plist)
└── models/                  # Whisper model files (gitignored)
```

## How It Works

1. **Record**: Click "Start Recording" to capture audio via PortAudio (16kHz, mono, 16-bit PCM)
2. **Stop & Transcribe**: Click "Stop & Transcribe" to save a WAV file and run whisper-cpp
3. **Save**: Save the transcription as a Markdown file to `~/Documents/Transcriptions/`

## License

MIT
