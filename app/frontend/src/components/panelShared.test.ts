import {describe, expect, it} from 'vitest'
import {
  ensureVisiblePipConfig,
  formatBytes,
  formatTime,
  isRecordingPackagePath,
  joinDisplayPath,
  normalizeRecordingQuality,
  packageDisplayName,
} from './panelShared'

describe('formatBytes', () => {
  it('formats each unit tier', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(512)).toBe('512 B')
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatBytes(1024 * 1024)).toBe('1.0 MB')
    expect(formatBytes(3 * 1024 * 1024 * 1024)).toBe('3.0 GB')
  })

  it('returns empty for invalid input', () => {
    expect(formatBytes(-1)).toBe('')
    expect(formatBytes(Number.NaN)).toBe('')
  })
})

describe('formatTime', () => {
  it('renders hours, minutes and seconds', () => {
    expect(formatTime(0)).toBe('00:00')
    expect(formatTime(59)).toBe('00:59')
    expect(formatTime(60)).toBe('01:00')
    expect(formatTime(3661)).toBe('01:01:01')
  })
})

describe('joinDisplayPath', () => {
  it('returns the leaf alone for browser-preview roots', () => {
    expect(joinDisplayPath('browser-preview', 'screen.mp4')).toBe('screen.mp4')
  })

  it('keeps backslash roots on their own separator', () => {
    expect(packageDisplayName('C:/data/pkg.rfrec/')).toBe('pkg.rfrec')
  })

  it('trims trailing separators', () => {
    expect(joinDisplayPath('/tmp/data/', 'screen.mp4')).toBe('/tmp/data/screen.mp4')
  })
})

describe('packageDisplayName', () => {
  it('returns the last path segment', () => {
    expect(packageDisplayName('/tmp/recording-2026-09-16.rfrec')).toBe('recording-2026-09-16.rfrec')
    expect(packageDisplayName('C:/data/pkg.rfrec/')).toBe('pkg.rfrec')
    expect(packageDisplayName('plain-name')).toBe('plain-name')
  })
})

describe('isRecordingPackagePath', () => {
  it('accepts only .rfrec paths', () => {
    expect(isRecordingPackagePath('/tmp/pkg.rfrec')).toBe(true)
    expect(isRecordingPackagePath('/tmp/pkg.mp4')).toBe(false)
    expect(isRecordingPackagePath('')).toBe(false)
  })
})

describe('normalizeRecordingQuality', () => {
  it('keeps known qualities and falls back to balanced', () => {
    expect(normalizeRecordingQuality('standard')).toBe('standard')
    expect(normalizeRecordingQuality('high')).toBe('high')
    expect(normalizeRecordingQuality('ultra')).toBe('balanced')
  })
})

describe('ensureVisiblePipConfig', () => {
  it('keeps a valid config unchanged', () => {
    const config = {preset: 'free' as const, shape: 'circle' as const, mirror: true, position: {x: 0.5, y: 0.5}, scale: 0.3, edgeFeather: 0}
    expect(ensureVisiblePipConfig(config)).toEqual(config)
  })
})
