interface Props {
  onClick: () => void
  disabled: boolean
}

export function RecordButton({ onClick, disabled }: Props) {
  return (
    <button
      className="btn btn-record"
      onClick={onClick}
      disabled={disabled}
    >
      <span className="record-icon" />
      Start Recording
    </button>
  )
}
