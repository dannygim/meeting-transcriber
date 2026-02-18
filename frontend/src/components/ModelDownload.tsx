import type { ModelInfo } from '../../bindings/github.com/dannygim/meeting-transcriber/services'

interface DownloadProgress {
  modelName: string
  bytesLoaded: number
  bytesTotal: number
  percent: number
  done: boolean
  error?: string
}

interface Props {
  models: ModelInfo[]
  downloading: boolean
  progress: DownloadProgress | null
  error: string
  onDownload: (name: string) => void
  onCancel: () => void
}

function formatBytes(bytes: number): string {
  if (bytes < 1000 * 1000) return `${(bytes / 1000).toFixed(0)} KB`
  if (bytes < 1000 * 1000 * 1000) return `${(bytes / (1000 * 1000)).toFixed(0)} MB`
  return `${(bytes / (1000 * 1000 * 1000)).toFixed(1)} GB`
}

export function ModelDownload({ models, downloading, progress, error, onDownload, onCancel }: Props) {
  return (
    <div className="model-download">
      <p className="model-description">
        A whisper model is required for transcription. Choose a model to download:
      </p>

      {error && <div className="model-error">{error}</div>}

      <div className="model-list">
        {models.map((m) => (
          <div key={m.name} className="model-item">
            <div className="model-info">
              <span className="model-name">
                {m.name}
                {m.name === 'base' && <span className="model-recommended">Recommended</span>}
              </span>
              <span className="model-size">{m.size}</span>
            </div>
            <div className="model-action">
              {m.exists ? (
                <span className="model-check">Downloaded</span>
              ) : downloading && progress && progress.modelName === m.name ? (
                <div className="model-progress">
                  <div
                    className="progress-bar"
                    role="progressbar"
                    aria-valuenow={Math.round(progress.percent)}
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-label={`Downloading ${m.name} model`}
                  >
                    <div
                      className="progress-bar-fill"
                      style={{ width: `${progress.percent}%` }}
                    />
                  </div>
                  <div className="progress-info">
                    {formatBytes(progress.bytesLoaded)}
                    {progress.bytesTotal > 0 && ` / ${formatBytes(progress.bytesTotal)}`}
                  </div>
                  <button className="btn btn-cancel" onClick={onCancel}>Cancel</button>
                </div>
              ) : (
                <button
                  className="btn btn-download"
                  onClick={() => onDownload(m.name)}
                  disabled={downloading}
                >
                  Download
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
