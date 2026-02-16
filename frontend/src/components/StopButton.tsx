interface Props {
  onClick: () => void
}

export function StopButton({ onClick }: Props) {
  return (
    <button className="btn btn-stop" onClick={onClick}>
      Stop & Transcribe
    </button>
  )
}
