import { useEffect, useRef } from 'react'
import { useRecorder } from './hooks/useRecorder'
import { useTranscription } from './hooks/useTranscription'
import { RecordButton } from './components/RecordButton'
import { PauseResumeButton } from './components/PauseResumeButton'
import { StopButton } from './components/StopButton'
import { Timer } from './components/Timer'
import { TranscriptView } from './components/TranscriptView'
import { Spectrum } from './components/Spectrum'

function App() {
  const recorder = useRecorder()
  const transcription = useTranscription()
  const wavPathRef = useRef('')

  useEffect(() => {
    transcription.checkWhisper()
  }, [])

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

  return (
    <div className="container">
      <header className="app-header">
        <h1>Meeting Transcriber</h1>
        <p className="subtitle">On-device audio transcription</p>
      </header>

      {transcription.whisperAvailable === false && (
        <div className="warning">
          whisper-cpp not found. Please install: <code>brew install whisper-cpp</code>
        </div>
      )}

      <div className="controls">
        {isIdle && !isTranscribing && (
          <RecordButton
            onClick={handleStart}
            disabled={transcription.whisperAvailable === false}
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
    </div>
  )
}

export default App
