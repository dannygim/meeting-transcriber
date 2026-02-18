import { useState, useEffect, useCallback, useRef } from 'react'
import { Events } from '@wailsio/runtime'

import {
  ModelService,
  TranscribeService,
} from '../../bindings/github.com/dannygim/meeting-transcriber/services'
import type { ModelInfo } from '../../bindings/github.com/dannygim/meeting-transcriber/services'

interface DownloadProgress {
  modelName: string
  bytesLoaded: number
  bytesTotal: number
  percent: number
  done: boolean
  error?: string
}

export function useModelDownload() {
  const [models, setModels] = useState<ModelInfo[]>([])
  const [downloading, setDownloading] = useState(false)
  const [progress, setProgress] = useState<DownloadProgress | null>(null)
  const [error, setError] = useState('')
  const unsubRef = useRef<(() => void) | null>(null)

  useEffect(() => {
    unsubRef.current = Events.On('model:download-progress', (event) => {
      const p = event.data as DownloadProgress
      if (p.error) {
        setError(p.error === 'cancelled' ? '' : p.error)
        setDownloading(false)
        setProgress(null)
        return
      }
      if (p.done) {
        setProgress(null)
        setDownloading(false)
        setError('')
        // Refresh model path so TranscribeService picks it up
        TranscribeService.RefreshModelPath()
        // Refresh model list to update Exists flags
        ModelService.ListModels().then(setModels)
        return
      }
      setProgress(p)
    })

    return () => {
      if (unsubRef.current) unsubRef.current()
    }
  }, [])

  const loadModels = useCallback(async () => {
    const list = await ModelService.ListModels()
    setModels(list)
    return list
  }, [])

  const startDownload = useCallback(async (name: string) => {
    setError('')
    setProgress(null)
    setDownloading(true)
    try {
      await ModelService.DownloadModel(name)
    } catch (err: any) {
      setError(err?.message || String(err))
      setDownloading(false)
    }
  }, [])

  const cancelDownload = useCallback(async () => {
    try {
      await ModelService.CancelDownload()
    } catch {
      // ignore
    }
  }, [])

  return {
    models,
    downloading,
    progress,
    error,
    loadModels,
    startDownload,
    cancelDownload,
  }
}
