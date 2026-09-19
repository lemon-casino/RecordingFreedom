import {useEffect, useState} from 'react'
import {formatTime} from './panelShared'

type RecordingClockProps = {
  active: boolean
  // Bump to reset the clock to 00:00 (new session, stop, failure).
  epoch: number
  // When non-null the countdown value is shown instead of the elapsed time.
  countdown: number | null
}

// Self-contained recording timer: the 1 Hz tick re-renders only this component
// instead of the whole window shell.
export function RecordingClock({active, epoch, countdown}: RecordingClockProps) {
  const [elapsed, setElapsed] = useState(0)

  useEffect(() => {
    setElapsed(0)
  }, [epoch])

  useEffect(() => {
    if (!active) return
    const timer = window.setInterval(() => setElapsed((value) => value + 1), 1000)
    return () => window.clearInterval(timer)
  }, [active])

  return <>{countdown !== null ? formatTime(countdown) : formatTime(elapsed)}</>
}
