import {describe, expect, it} from 'vitest'
import {formatShortcutForDisplay, normalizeKeyboardShortcutKey, shortcutIdentity} from './controls'

describe('normalizeKeyboardShortcutKey', () => {
  it('maps bare modifiers to null', () => {
    expect(normalizeKeyboardShortcutKey('Control')).toBeNull()
    expect(normalizeKeyboardShortcutKey('Shift')).toBeNull()
    expect(normalizeKeyboardShortcutKey('Alt')).toBeNull()
    expect(normalizeKeyboardShortcutKey('Meta')).toBeNull()
  })

  it('normalizes single characters and special keys', () => {
    expect(normalizeKeyboardShortcutKey('r')).toBe('R')
    expect(normalizeKeyboardShortcutKey(' ')).toBe('Space')
    expect(normalizeKeyboardShortcutKey('+')).toBe('Plus')
    expect(normalizeKeyboardShortcutKey('ArrowLeft')).toBe('Left')
    expect(normalizeKeyboardShortcutKey('F5')).toBe('F5')
  })
})

describe('formatShortcutForDisplay', () => {
  it('expands CmdOrCtrl for the current platform', () => {
    const display = formatShortcutForDisplay('CmdOrCtrl+Shift+R')
    expect(display === 'Ctrl + Shift + R' || display === 'Cmd + Shift + R').toBe(true)
  })

  it('keeps literal parts', () => {
    expect(formatShortcutForDisplay('Ctrl+Alt+Delete')).toBe('Ctrl + Alt + Delete')
  })
})

describe('shortcutIdentity', () => {
  it('normalizes modifier case but keeps the key lowercase', () => {
    const identity = shortcutIdentity('CmdOrCtrl+Shift+R')
    expect(['ctrl+r+shift', 'cmd+r+shift']).toContain(identity)
  })
})
