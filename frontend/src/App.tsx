import { useEffect, useRef, useState } from 'react'
import { useRecorder } from './hooks/useRecorder'
import { useTranscription } from './hooks/useTranscription'
import { useModelDownload } from './hooks/useModelDownload'
import { RecordButton } from './components/RecordButton'
import { PauseResumeButton } from './components/PauseResumeButton'
import { StopButton } from './components/StopButton'
import { Timer } from './components/Timer'
import { TranscriptView } from './components/TranscriptView'
import { Spectrum } from './components/Spectrum'
import { ModelDownload } from './components/ModelDownload'
import { TranscribeService } from '../bindings/github.com/dannygim/meeting-transcriber/services'

function App() {
  const recorder = useRecorder()
  const transcription = useTranscription()
  const modelDownload = useModelDownload()
  const wavPathRef = useRef('')
  const [modelPath, setModelPath] = useState<string | null>(null)

  useEffect(() => {
    transcription.checkWhisper()
    modelDownload.loadModels()
    TranscribeService.GetModelPath().then(setModelPath)
  }, [])

  // Re-check model path when models list changes (after download completes)
  useEffect(() => {
    if (modelDownload.models.some((m) => m.exists)) {
      TranscribeService.GetModelPath().then(setModelPath)
    }
  }, [modelDownload.models])

  const handleStart = async () => {
    try {
      await recorder.startRecording()
    } catch (err: any) {
      console.error('Failed to start recording:', err)
    }
  }

  const handleStop = async () => {
    try {
      const path = await recorder.stopRecording()
      wavPathRef.current = path
      recorder.setTranscribing()
      await transcription.transcribe(path)
      recorder.setIdle()
    } catch (err: any) {
      console.error('Failed to stop/transcribe:', err)
      recorder.setIdle()
    }
  }

  const handleSave = async () => {
    if (!wavPathRef.current) return
    try {
      await transcription.saveToFile(wavPathRef.current)
    } catch (err: any) {
      console.error('Failed to save:', err)
    }
  }

  const isIdle = recorder.state === 'idle'
  const isRecording = recorder.state === 'recording'
  const isPaused = recorder.state === 'paused'
  const isTranscribing = recorder.state === 'transcribing'
  const isActive = isRecording || isPaused

  const whisperMissing = transcription.whisperAvailable === false
  const needsModel = transcription.whisperAvailable === true && modelPath === ''

  return (
    <div className="container">
      <header className="app-header">
        <h1>Meeting Transcriber</h1>
        <p className="subtitle">On-device audio transcription</p>
      </header>

      {whisperMissing && (
        <div className="warning">
          whisper-cpp not found. Please install: <code>brew install whisper-cpp</code>
        </div>
      )}

      {needsModel && (
        <ModelDownload
          models={modelDownload.models}
          downloading={modelDownload.downloading}
          progress={modelDownload.progress}
          error={modelDownload.error}
          onDownload={modelDownload.startDownload}
          onCancel={modelDownload.cancelDownload}
        />
      )}

      {!whisperMissing && !needsModel && (
        <>
          <div className="controls">
            {isIdle && !isTranscribing && (
              <RecordButton
                onClick={handleStart}
                disabled={false}
              />
            )}

            {isActive && (
              <div className="active-controls">
                <PauseResumeButton
                  isPaused={isPaused}
                  onPause={recorder.pauseRecording}
                  onResume={recorder.resumeRecording}
                />
                <StopButton onClick={handleStop} />
              </div>
            )}

            {isActive && (
              <div className="recording-status">
                <span className={`status-dot ${isPaused ? 'paused' : 'recording'}`} />
                {isPaused ? 'Paused' : 'Recording'}
              </div>
            )}
          </div>

          {isActive && (
            <Spectrum active={isActive} />
          )}

          {(isActive || isTranscribing) && (
            <Timer seconds={recorder.elapsed} />
          )}

          <TranscriptView
            text={transcription.text}
            savedPath={transcription.savedPath}
            isTranscribing={transcription.isTranscribing}
            error={transcription.error}
            onSave={handleSave}
          />
        </>
      )}
    </div>
  )
}

export default App
