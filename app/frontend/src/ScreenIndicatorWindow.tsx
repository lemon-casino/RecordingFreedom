import {useEffect, useState} from 'react'

export default function ScreenIndicatorWindow() {
  const indicatorWindow = window as Window & {__RF_SCREEN_INDICATOR__?: {label?: string}}
  const [label, setLabel] = useState(indicatorWindow.__RF_SCREEN_INDICATOR__?.label ?? '')

  useEffect(() => {
    document.body.classList.add('rf-screen-indicator-window')
    return () => document.body.classList.remove('rf-screen-indicator-window')
  }, [])

  useEffect(() => {
    const onIndicator = (event: Event) => {
      const next = (event as CustomEvent<{label?: string}>).detail
      setLabel(next?.label ?? '')
    }
    window.addEventListener('rf-screen-indicator', onIndicator)
    return () => window.removeEventListener('rf-screen-indicator', onIndicator)
  }, [])

  return (
    <main className="screen-indicator-shell" aria-hidden="true">
      <span>{label || '1'}</span>
    </main>
  )
}
