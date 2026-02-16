interface Props {
  isPaused: boolean
  onPause: () => void
  onResume: () => void
}

export function PauseResumeButton({ isPaused, onPause, onResume }: Props) {
  return (
    <button
      className={`btn ${isPaused ? 'btn-resume' : 'btn-pause'}`}
      onClick={isPaused ? onResume : onPause}
    >
      {isPaused ? 'Resume' : 'Pause'}
    </button>
  )
}
