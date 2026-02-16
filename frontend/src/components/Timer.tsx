interface Props {
  seconds: number
}

export function Timer({ seconds }: Props) {
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  const display = `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`

  return <div className="timer">{display}</div>
}
