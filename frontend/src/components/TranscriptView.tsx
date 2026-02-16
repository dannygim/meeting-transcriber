interface Props {
  text: string
  savedPath: string
  isTranscribing: boolean
  error: string
  onSave: () => void
}

export function TranscriptView({ text, savedPath, isTranscribing, error, onSave }: Props) {
  if (isTranscribing) {
    return (
      <div className="transcript-view">
        <div className="transcribing-indicator">
          <span className="spinner" />
          Transcribing...
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="transcript-view">
        <div className="transcript-error">{error}</div>
      </div>
    )
  }

  if (!text) return null

  return (
    <div className="transcript-view">
      <div className="transcript-text">{text}</div>
      <div className="transcript-actions">
        <button className="btn btn-save" onClick={onSave}>
          Save as Markdown
        </button>
        {savedPath && (
          <div className="saved-path">Saved to: {savedPath}</div>
        )}
      </div>
    </div>
  )
}
