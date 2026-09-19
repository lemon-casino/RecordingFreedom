import {useEffect, useLayoutEffect, useRef, useState} from 'react'
import {Check} from 'lucide-react'
import {normalizeTheme, type ThemeCode} from './services/mockBackend'
import {
  completeFloatingSelect,
  getFloatingSelectState,
  hideFloatingSelect,
  loadSettings,
  setFloatingSelectHitRegions,
  subscribeFloatingSelectChanged,
  subscribeSettingsChanged,
  type FloatingSelectState } from './services/recorderBackend'
import {elementHitRegion} from './components/hitRegion'
import {applyTheme, isThemeGroupOption} from './components/themeOptions'

export default function FloatingSelectWindow() {
  const rootRef = useRef<HTMLElement | null>(null)
  const [theme, setTheme] = useState<ThemeCode>('night-teal')
  const [selectState, setSelectState] = useState<FloatingSelectState>(() => ({
    visible: false,
    anchor: {x: 0, y: 0, width: 0, height: 0},
    bounds: {x: 0, y: 0, width: 0, height: 0},
    options: [],
    token: 0 }))

  useEffect(() => {
    document.body.classList.add('rf-floating-select-window')
    return () => document.body.classList.remove('rf-floating-select-window')
  }, [])

  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  useEffect(() => {
    let cancelled = false
    let receivedSettingsEvent = false
    void loadSettings()
      .then((settings) => {
        if (!cancelled && !receivedSettingsEvent) {
          setTheme(normalizeTheme(settings.window.theme))
        }
      })
      .catch((error) => console.info('Floating select settings unavailable:', error))
    const unsubscribe = subscribeSettingsChanged((settings) => {
      receivedSettingsEvent = true
      if (!cancelled) setTheme(normalizeTheme(settings.window.theme))
    })
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  useEffect(() => {
    let cancelled = false
    void getFloatingSelectState()
      .then((state) => {
        if (!cancelled) setSelectState(state)
      })
      .catch((error) => console.info('Floating select state unavailable:', error))
    const unsubscribe = subscribeFloatingSelectChanged(setSelectState)
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  useLayoutEffect(() => {
    const viewportWidth = window.innerWidth || document.documentElement.clientWidth || 0
    const viewportHeight = window.innerHeight || document.documentElement.clientHeight || 0
    const region = elementHitRegion(rootRef.current, viewportWidth, viewportHeight, 'round-rect', 16)
    void setFloatingSelectHitRegions({
      enabled: Boolean(region),
      force: true,
      viewportWidth,
      viewportHeight,
      devicePixelRatio: window.devicePixelRatio || 1,
      regions: region ? [region] : [] })
  }, [selectState.visible, selectState.options.length])

  useEffect(() => {
    if (!selectState.visible) return
    const token = selectState.token
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        void hideFloatingSelect(token)
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [selectState.token, selectState.visible])

  if (!selectState.visible) {
    return <main className="floating-select-shell empty" aria-hidden="true" />
  }

  return (
    <main ref={rootRef} className={`floating-select-shell select-menu-list drop-${selectState.direction ?? 'down'}`} role="listbox">
      {selectState.options.map((option) => {
        const groupHeader = isThemeGroupOption(option)
        return (
          <button
            key={option.value}
            type="button"
            role="option"
            className={`select-menu-option ${groupHeader ? 'select-menu-group-label' : ''} ${option.value === selectState.value ? 'selected' : ''}`}
            aria-selected={option.value === selectState.value}
            disabled={option.disabled}
            onPointerDown={(event) => {
              event.preventDefault()
              event.stopPropagation()
            }}
            onClick={() => {
              if (option.disabled) return
              void completeFloatingSelect({
                id: selectState.id ?? '',
                value: option.value,
                token: selectState.token,
                panelToken: selectState.panelToken })
            }}
          >
            {groupHeader ? (
              <span className="select-menu-group-title">{option.label}</span>
            ) : (
              <>
                <span className="select-menu-label">
                  {option.swatch && <i className="select-menu-swatch" style={{background: option.swatch}} aria-hidden="true" />}
                  <span>{option.label}</span>
                </span>
                {option.value === selectState.value && <Check size={15} />}
              </>
            )}
          </button>
        )
      })}
    </main>
  )
}

