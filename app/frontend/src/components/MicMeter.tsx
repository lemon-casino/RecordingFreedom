import {useEffect, useState} from 'react'
import {
  readableError,
  startMicrophoneLevelMonitor,
  stopMicrophoneLevelMonitor,
  subscribeAudioLevel,
} from '../services/recorderBackend'
import type {RecorderCopy} from '../i18n'

type MicMeterProps = {
  copy: RecorderCopy
  microphone: boolean
  deviceId: string
  deviceAvailable: boolean
  hasAvailableDevice: boolean
  isRecording: boolean
}

// Self-contained microphone level meter: owns the audio-level subscription and
// monitor lifecycle so 20 Hz level updates re-render only this component, not
// the whole window shell. Unmounting it (panel closed) stops the monitor.
export function MicMeter({copy, microphone, deviceId, deviceAvailable, hasAvailableDevice, isRecording}: MicMeterProps) {
  const [level, setLevel] = useState(0)
  const [peak, setPeak] = useState(0)
  const [active, setActive] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => subscribeAudioLevel((update) => {
    if (update.deviceId && deviceId && update.deviceId !== deviceId) return
    if (update.error) {
      setError(update.error)
      setActive(false)
      setLevel(0)
      setPeak(0)
      return
    }
    setError(null)
    setActive(update.active)
    setLevel(update.active ? update.level : 0)
    setPeak(update.active ? update.peak : 0)
  }), [deviceId])

  useEffect(() => {
    const shouldMonitor = microphone && deviceAvailable && hasAvailableDevice && deviceId !== '' && !isRecording
    if (!shouldMonitor) {
      setActive(false)
      setLevel(0)
      setPeak(0)
      void stopMicrophoneLevelMonitor()
      return
    }
    let cancelled = false
    setError(null)
    void startMicrophoneLevelMonitor(deviceId)
      .then(() => {
        if (!cancelled) setActive(true)
      })
      .catch((err) => {
        if (cancelled) return
        setError(readableError(err))
        setActive(false)
        setLevel(0)
        setPeak(0)
      })
    return () => {
      cancelled = true
      void stopMicrophoneLevelMonitor()
    }
  }, [deviceId, deviceAvailable, hasAvailableDevice, isRecording, microphone])

  const statusText = error
    ? copy.panels.microphoneLevelError
    : !microphone
      ? copy.panels.microphoneLevelOff
      : !deviceAvailable || !hasAvailableDevice
        ? copy.panels.microphoneLevelUnavailable
        : active
          ? copy.panels.microphoneLevelLive
          : copy.panels.microphoneLevelWaiting

  const meterLevel = microphone && active ? level : 0
  const bars = Array.from({length: 18}, (_, index) => {
    const threshold = (index + 1) / 18
    const barActive = meterLevel >= threshold
    const height = barActive ? Math.max(14, Math.min(100, meterLevel * 100 + index * 0.9)) : 8
    return {barActive, height: `${height}%`}
  })

  return (
    <>
      <div className={`meter ${active ? 'live' : ''} ${error ? 'error' : ''}`} aria-label={copy.aria.microphoneLevel} title={error ?? statusText}>
        {bars.map((bar, index) => (
          <span key={index} className={bar.barActive ? 'active' : ''} style={{height: bar.height}} />
        ))}
      </div>
      <div className="meter-status">
        <span>{statusText}</span>
        <b>{Math.round(peak * 100)}%</b>
      </div>
    </>
  )
}
