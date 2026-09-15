import {getFloatingPanelState, hideFloatingSelect, showFloatingSelect} from '../services/recorderBackend'
import {resolveFloatingSelectPlacement} from './floating/floatingPosition'
import {eventPathContains, sourceIcon} from './panelShared'
import {RecorderCopy} from '../i18n'
import {CaptureCapability, CaptureSource, clampNumber} from '../services/mockBackend'
import {FloatingSelectOption, isWailsDesktopRuntime, subscribeFloatingSelectChosen} from '../services/recorderBackend'
import {floatingSelectMaxHeight, floatingSelectMaxWidth, floatingSelectMinWidth, sourceMeta, sourceName} from './panelShared'
import {isThemeGroupOption} from './themeOptions'
import {useEffect, useRef, useState, type ReactNode} from 'react'
import {Check, ChevronDown} from 'lucide-react'
import type {SelectMenuOption} from './themeOptions'

export function SwitchRow({label, checked, disabled = false, onChange}: {label: string; checked: boolean; disabled?: boolean; onChange: (value: boolean) => void}) {
  return (
    <label className={`switch-row ${disabled ? 'is-disabled' : ''}`}>
      <span>{label}</span>
      <input type="checkbox" checked={checked} disabled={disabled} onChange={(event) => onChange(event.target.checked)} />
      <i aria-hidden="true" />
    </label>
  )
}

export function SelectMenu({
  id,
  value,
  options,
  disabled = false,
  className = '',
  onChange }: {
  id?: string
  value: string
  options: SelectMenuOption[]
  disabled?: boolean
  className?: string
  onChange: (value: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [dropDirection, setDropDirection] = useState<'down' | 'up'>('down')
  const rootRef = useRef<HTMLDivElement | null>(null)
  const openedAtRef = useRef(0)
  const pointerInsideAtRef = useRef(0)
  const pointerInsideRef = useRef(false)
  const floatingSelectTokenRef = useRef(0)
  const selected = options.find((option) => option.value === value) ?? options.find((option) => !option.disabled) ?? options[0]
  const selectOption = (option: SelectMenuOption) => {
    if (option.disabled) return
    onChange(option.value)
    setOpen(false)
  }
  const updateDropDirection = () => {
    const rect = rootRef.current?.getBoundingClientRect()
    if (!rect) return
    const estimatedMenuHeight = Math.min(220, Math.max(44, options.length * 44 + 12))
    const spaceBelow = window.innerHeight - rect.bottom
    const spaceAbove = rect.top
    setDropDirection(spaceBelow < estimatedMenuHeight + 10 && spaceAbove > spaceBelow ? 'up' : 'down')
  }

  useEffect(() => {
    if (!open) return
    if (isWailsDesktopRuntime()) return
    openedAtRef.current = Date.now()
    updateDropDirection()
    const close = () => setOpen(false)
    const markPointerInside = (inside: boolean) => {
      pointerInsideRef.current = inside
      if (inside) pointerInsideAtRef.current = Date.now()
    }
    const onPointerDown = (event: PointerEvent) => {
      if (eventPathContains(event, rootRef.current)) {
        markPointerInside(true)
        return
      }
      markPointerInside(false)
      close()
    }
    const onPointerMove = (event: PointerEvent) => {
      markPointerInside(eventPathContains(event, rootRef.current))
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        close()
      }
    }
    const onVisibilityChange = () => {
      if (document.visibilityState !== 'visible' && !pointerInsideRef.current) close()
    }
    const onWindowBlur = () => {
      window.setTimeout(() => {
        const now = Date.now()
        if (
          pointerInsideRef.current ||
          now - pointerInsideAtRef.current < 900 ||
          now - openedAtRef.current < 360
        ) {
          return
        }
        close()
      }, 140)
    }
    document.addEventListener('pointerdown', onPointerDown, true)
    document.addEventListener('pointermove', onPointerMove, true)
    document.addEventListener('keydown', onKeyDown)
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('resize', updateDropDirection)
    window.addEventListener('blur', onWindowBlur)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown, true)
      document.removeEventListener('pointermove', onPointerMove, true)
      document.removeEventListener('keydown', onKeyDown)
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.removeEventListener('resize', updateDropDirection)
      window.removeEventListener('blur', onWindowBlur)
      pointerInsideRef.current = false
    }
  }, [open, options.length])

  useEffect(() => subscribeFloatingSelectChosen((event) => {
    if (!isWailsDesktopRuntime()) return
    if (event.token !== floatingSelectTokenRef.current) return
    const option = options.find((candidate) => candidate.value === event.value)
    if (!option || option.disabled) return
    onChange(option.value)
    setOpen(false)
  }), [onChange, options])

  useEffect(() => {
    if (!disabled) return
    setOpen(false)
    if (floatingSelectTokenRef.current) void hideFloatingSelect(floatingSelectTokenRef.current)
  }, [disabled])

  const toggleFloatingSelect = async () => {
    if (!isWailsDesktopRuntime()) {
      if (!open) updateDropDirection()
      setOpen((value) => !value)
      return
    }
    if (open) {
      await hideFloatingSelect(floatingSelectTokenRef.current)
      setOpen(false)
      return
    }
    const root = rootRef.current
    if (!root) return
    const token = floatingSelectTokenRef.current + 1
    floatingSelectTokenRef.current = token
    const anchorWidth = root.getBoundingClientRect().width
    const placement = await resolveFloatingSelectPlacement(root, {
      width: clampNumber(anchorWidth, floatingSelectMinWidth, floatingSelectMaxWidth),
      minWidth: floatingSelectMinWidth,
      maxWidth: floatingSelectMaxWidth,
      maxHeight: floatingSelectMaxHeight,
      optionCount: options.length })
    const parentPanel = await getFloatingPanelState().catch(() => undefined)
    await showFloatingSelect({
      id: id ?? `select-${token}`,
      anchor: placement.anchor,
      bounds: placement.bounds,
      value,
      options: options as FloatingSelectOption[],
      token,
      panelToken: parentPanel?.visible ? parentPanel.token : undefined,
      width: placement.bounds.width,
      maxHeight: placement.bounds.height,
      screenId: placement.screenId,
      direction: placement.direction })
    setOpen(true)
  }

  return (
    <div
      ref={rootRef}
      className={`select-menu ${open ? 'open' : ''} drop-${dropDirection} ${disabled ? 'disabled' : ''} ${className}`}
      onPointerDownCapture={() => {
        pointerInsideRef.current = true
        pointerInsideAtRef.current = Date.now()
      }}
      onPointerMoveCapture={() => {
        pointerInsideRef.current = true
        pointerInsideAtRef.current = Date.now()
      }}
    >
      <button
        id={id}
        type="button"
        className="select-menu-button"
        disabled={disabled || options.length === 0}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => void toggleFloatingSelect()}
      >
        <span className="select-menu-label">
          {selected?.swatch && <i className="select-menu-swatch" style={{background: selected.swatch}} aria-hidden="true" />}
          <span>{selected?.label ?? ''}</span>
        </span>
        <ChevronDown size={16} />
      </button>
      {open && !isWailsDesktopRuntime() && (
        <div className="select-menu-list" role="listbox" aria-labelledby={id}>
          {options.map((option) => {
            const groupHeader = isThemeGroupOption(option)
            return (
              <button
                key={option.value}
                type="button"
                role="option"
                className={`select-menu-option ${groupHeader ? 'select-menu-group-label' : ''} ${option.value === value ? 'selected' : ''}`}
                aria-selected={option.value === value}
                disabled={option.disabled}
                onPointerDown={(event) => {
                  event.preventDefault()
                  event.stopPropagation()
                  selectOption(option)
                }}
                onClick={() => {
                  selectOption(option)
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
                    {option.value === value && <Check size={15} />}
                  </>
                )}
              </button>
            )
          })}
        </div>
      )}
    </div>
  )
}

export function SourceGroup({title, children}: {title: string; children: ReactNode}) {
  return (
    <section className="source-group" aria-label={title}>
      <div className="source-group-label">{title}</div>
      {children}
    </section>
  )
}

export function SourceMenuRow({
  source,
  copy,
  selected,
  actionLabel,
  disabled = false,
  onSelect,
  onPreviewStart,
  onPreviewEnd }: {
  source: CaptureSource
  copy: RecorderCopy
  selected: boolean
  actionLabel?: string
  disabled?: boolean
  onSelect: () => void
  onPreviewStart?: () => void
  onPreviewEnd?: () => void
}) {
  const Icon = sourceIcon[source.type]
  return (
    <button
      className={`menu-row ${selected ? 'selected' : ''} ${source.available === false ? 'queued' : ''}`}
      type="button"
      disabled={disabled}
      onClick={onSelect}
      onPointerEnter={disabled ? undefined : onPreviewStart}
      onPointerLeave={disabled ? undefined : onPreviewEnd}
      onFocus={disabled ? undefined : onPreviewStart}
      onBlur={onPreviewEnd}
    >
      <Icon size={18} />
      <span>
        <strong>{sourceName(source, copy)}</strong>
        <small>{sourceMeta(source, copy)}</small>
      </span>
      {actionLabel && <b className="row-action-label">{actionLabel}</b>}
      {selected && <Check size={16} />}
    </button>
  )
}

export function SettingLine({
  title,
  value,
  status,
  statusLabel,
  detail,
  actionLabel,
  actionDisabled,
  onAction }: {
  title: string
  value: string
  status?: CaptureCapability['status']
  statusLabel?: string
  detail?: string
  actionLabel?: string
  actionDisabled?: boolean
  onAction?: () => void
}) {
  return (
    <div className={`setting-line ${status ? `status-${status}` : ''}`}>
      <span>{title}</span>
      <div className="setting-value">
        <strong>{value}</strong>
        {status && <b className={`status-badge ${status}`}>{statusLabel ?? status}</b>}
        {actionLabel && (
          <button className="setting-action" type="button" disabled={actionDisabled} onClick={onAction}>
            {actionLabel}
          </button>
        )}
      </div>
      {detail && <small>{detail}</small>}
    </div>
  )
}

export function SettingSelect({
  title,
  value,
  options,
  detail,
  onChange }: {
  title: string
  value: string
  options: SelectMenuOption[]
  detail?: string
  onChange: (value: string) => void
}) {
  return (
    <div className="setting-line setting-control">
      <span>{title}</span>
      <SelectMenu className="setting-control-select" value={value} options={options} onChange={onChange} />
      {detail && <small>{detail}</small>}
    </div>
  )
}

export function SettingTextAction({
  title,
  value,
  detail,
  actionLabel,
  actionDisabled,
  onChange,
  onAction }: {
  title: string
  value: string
  detail?: string
  actionLabel: string
  actionDisabled?: boolean
  onChange: (value: string) => void
  onAction: () => void
}) {
  return (
    <label className="setting-line setting-control">
      <span>{title}</span>
      <div className="setting-input-row">
        <input
          className="setting-control-input"
          value={value}
          onChange={(event) => onChange(event.target.value)}
        />
        <button className="setting-action" type="button" disabled={actionDisabled} onClick={onAction}>
          {actionLabel}
        </button>
      </div>
      {detail && <small>{detail}</small>}
    </label>
  )
}

export function SettingTextInput({
  title,
  value,
  detail,
  placeholder,
  inputType = 'text',
  onCommit }: {
  title: string
  value: string
  detail?: string
  placeholder?: string
  inputType?: 'text' | 'password'
  onCommit: (value: string) => void
}) {
  const [draft, setDraft] = useState(value)
  useEffect(() => setDraft(value), [value])
  const commit = () => {
    if (draft.trim() !== value.trim()) onCommit(draft.trim())
  }
  return (
    <label className="setting-line setting-control">
      <span>{title}</span>
      <input
        className="setting-control-input"
        value={draft}
        type={inputType}
        placeholder={placeholder}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={commit}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.preventDefault()
            event.currentTarget.blur()
          }
        }}
      />
      {detail && <small>{detail}</small>}
    </label>
  )
}

export function SettingShortcut({
  title,
  value,
  detail,
  actionLabel,
  capturing,
  onStart,
  onCancel,
  onCapture }: {
  title: string
  value: string
  detail?: string
  actionLabel: string
  capturing: boolean
  onStart: () => void
  onCancel: () => void
  onCapture: (accelerator: string) => void
}) {
  const buttonRef = useRef<HTMLButtonElement | null>(null)
  useEffect(() => {
    if (capturing) buttonRef.current?.focus()
  }, [capturing])
  return (
    <div className={`setting-line setting-shortcut ${capturing ? 'is-capturing' : ''}`}>
      <span>{title}</span>
      <div className="setting-shortcut-row">
        <kbd>{capturing ? '...' : value}</kbd>
        <button
          ref={buttonRef}
          className="setting-action shortcut-capture-button"
          type="button"
          onClick={capturing ? onCancel : onStart}
          onKeyDown={(event) => {
            if (!capturing) return
            event.preventDefault()
            event.stopPropagation()
            if (event.key === 'Escape') {
              onCancel()
              return
            }
            const accelerator = shortcutFromKeyboardEvent(event.nativeEvent)
            if (accelerator) onCapture(accelerator)
          }}
        >
          {actionLabel}
        </button>
      </div>
      {detail && <small>{detail}</small>}
    </div>
  )
}

export function SettingToggle({title, checked, detail, onChange}: {title: string; checked: boolean; detail?: string; onChange: (value: boolean) => void}) {
  return (
    <div className="setting-line setting-control">
      <SwitchRow label={title} checked={checked} onChange={onChange} />
      {detail && <small>{detail}</small>}
    </div>
  )
}

export function shortcutFromKeyboardEvent(event: KeyboardEvent): string | null {
  const key = normalizeKeyboardShortcutKey(event.key)
  if (!key) return null
  const modifiers: string[] = []
  const macLike = isMacLikePlatform()
  if (event.metaKey || (!macLike && event.ctrlKey)) modifiers.push('CmdOrCtrl')
  if (macLike && event.ctrlKey) modifiers.push('Ctrl')
  if (event.altKey) modifiers.push('OptionOrAlt')
  if (event.shiftKey) modifiers.push('Shift')
  const uniqueModifiers = Array.from(new Set(modifiers))
  if (uniqueModifiers.length === 0 && !/^F\d{1,2}$/i.test(key)) return null
  if (uniqueModifiers.length === 1 && uniqueModifiers[0] === 'Shift' && isPrintableShortcutKey(key)) return null
  return [...uniqueModifiers, key].join('+')
}

export function normalizeKeyboardShortcutKey(key: string): string | null {
  if (!key || key === 'Control' || key === 'Shift' || key === 'Alt' || key === 'Meta') return null
  if (key === ' ') return 'Space'
  if (key === '+') return 'Plus'
  if (key.length === 1) return key.toUpperCase()
  if (key.startsWith('Arrow')) return key.replace('Arrow', '')
  if (/^F\d{1,2}$/i.test(key)) return key.toUpperCase()
  const aliases: Record<string, string> = {
    Escape: 'Escape',
    Esc: 'Escape',
    Enter: 'Enter',
    Return: 'Enter',
    Backspace: 'Backspace',
    Delete: 'Delete',
    Tab: 'Tab',
    Home: 'Home',
    End: 'End',
    PageUp: 'Page Up',
    PageDown: 'Page Down' }
  return aliases[key] ?? null
}

export function isMacLikePlatform(): boolean {
  const platform = navigator.platform?.toLowerCase() ?? ''
  return platform.includes('mac') || /mac os|iphone|ipad/.test(navigator.userAgent.toLowerCase())
}

export function isPrintableShortcutKey(key: string): boolean {
  return key.length === 1 || key === 'Space' || key === 'Plus'
}

export function formatShortcutForDisplay(shortcut: string): string {
  return shortcut
    .split('+')
    .filter(Boolean)
    .map((part) => {
      if (part === 'CmdOrCtrl') return isMacLikePlatform() ? 'Cmd' : 'Ctrl'
      if (part === 'OptionOrAlt') return isMacLikePlatform() ? 'Option' : 'Alt'
      return part
    })
    .join(' + ')
}

export function shortcutIdentity(shortcut: string): string {
  return shortcut
    .split('+')
    .filter(Boolean)
    .map((part, index, parts) => {
      if (index === parts.length - 1) return part.toLowerCase()
      if (part === 'CmdOrCtrl') return isMacLikePlatform() ? 'cmd' : 'ctrl'
      if (part === 'OptionOrAlt') return 'alt'
      return part.toLowerCase()
    })
    .sort()
    .join('+')
}

