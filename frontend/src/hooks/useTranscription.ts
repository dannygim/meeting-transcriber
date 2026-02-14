import { useState, useCallback } from 'react'

import { TranscribeService } from '../../bindings/github.com/dannygim/meeting-transcriber/services'

export function useTranscription() {
  const [text, setText] = useState('')
  const [savedPath, setSavedPath] = useState('')
  const [isTranscribing, setIsTranscribing] = useState(false)
  const [error, setError] = useState('')
  const [whisperAvailable, setWhisperAvailable] = useState<boolean | null>(null)

  const checkWhisper = useCallback(async () => {
    const available = await TranscribeService.IsWhisperAvailable()
    setWhisperAvailable(available)
    return available
  }, [])

  const transcribe = useCallback(async (wavPath: string) => {
    setIsTranscribing(true)
    setError('')
    setText('')
    setSavedPath('')
    try {
      const result = await TranscribeService.Transcribe(wavPath)
      setText(result)
      return result
    } catch (err: any) {
      const msg = err?.message || String(err)
      setError(msg)
      throw err
    } finally {
      setIsTranscribing(false)
    }
  }, [])

  const saveToFile = useCallback(async (wavPath: string) => {
    try {
      const path = await TranscribeService.TranscribeToFile(wavPath)
      setSavedPath(path)
      return path
    } catch (err: any) {
      const msg = err?.message || String(err)
      setError(msg)
      throw err
    }
  }, [])

  return {
    text,
    savedPath,
    isTranscribing,
    error,
    whisperAvailable,
    checkWhisper,
    transcribe,
    saveToFile,
  }
}
