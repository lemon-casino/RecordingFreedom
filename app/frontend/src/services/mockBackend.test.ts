import {describe, expect, it} from 'vitest'
import {
  isDarkTheme,
  normalizeLocale,
  normalizeShortcutSettings,
  normalizeTheme,
  themeGroups,
  themeOptions,
  themeSwatches,
} from './mockBackend'

describe('normalizeTheme', () => {
  it('keeps known themes', () => {
    expect(normalizeTheme('night-teal')).toBe('night-teal')
    expect(normalizeTheme('cloud-white')).toBe('cloud-white')
  })

  it('falls back to the default theme for unknown values', () => {
    expect(normalizeTheme('neon-rainbow')).toBe('night-teal')
    expect(normalizeTheme(undefined)).toBe('night-teal')
    expect(normalizeTheme(42)).toBe('night-teal')
  })
})

describe('isDarkTheme', () => {
  it('classifies the dark group as dark', () => {
    for (const theme of themeGroups.dark) {
      expect(isDarkTheme(theme)).toBe(true)
    }
  })

  it('classifies the light group as light', () => {
    for (const theme of themeGroups.light) {
      expect(isDarkTheme(theme)).toBe(false)
    }
  })
})

describe('theme catalog consistency', () => {
  it('lists every theme exactly once across the two groups', () => {
    const all = [...themeGroups.dark, ...themeGroups.light]
    expect(new Set(all).size).toBe(all.length)
    expect(new Set(themeOptions).size).toBe(themeOptions.length)
    expect(new Set(all)).toEqual(new Set(themeOptions))
  })

  it('defines a swatch for every theme', () => {
    for (const theme of themeOptions) {
      expect(themeSwatches[theme]).toMatch(/^#[0-9a-f]{6}$/i)
    }
  })
})

describe('normalizeLocale', () => {
  it('keeps supported locales and falls back to zh-CN', () => {
    expect(normalizeLocale('en')).toBe('en')
    expect(normalizeLocale('zh-CN')).toBe('zh-CN')
    expect(normalizeLocale('fr')).toBe('zh-CN')
  })
})

describe('normalizeShortcutSettings', () => {
  it('fills missing shortcuts with defaults and trims values', () => {
    const normalized = normalizeShortcutSettings({toggleRecording: '  Ctrl+R  '})
    expect(normalized.toggleRecording).toBe('Ctrl+R')
    expect(normalized.togglePause).toBeTruthy()
    expect(normalized.openWhiteboard).toBeTruthy()
  })

  it('replaces blank values with defaults', () => {
    const normalized = normalizeShortcutSettings({pasteImage: '   '})
    expect(normalized.pasteImage).not.toBe('')
  })
})
