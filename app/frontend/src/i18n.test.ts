import {describe, expect, it} from 'vitest'
import {copyByLocale} from './i18n'

function keyPaths(value: unknown, prefix = ''): string[] {
  if (value === null || typeof value !== 'object') {
    return [prefix]
  }
  return Object.entries(value as Record<string, unknown>).flatMap(([key, child]) => (
    keyPaths(child, prefix ? `${prefix}.${key}` : key)
  ))
}

describe('copyByLocale', () => {
  it('defines the same key structure for every locale', () => {
    const locales = Object.keys(copyByLocale)
    expect(locales).toContain('zh-CN')
    expect(locales).toContain('en')

    const reference = [...keyPaths(copyByLocale['zh-CN'])].sort()
    for (const locale of locales) {
      const paths = [...keyPaths(copyByLocale[locale as keyof typeof copyByLocale])].sort()
      expect(paths, `locale ${locale} key structure`).toEqual(reference)
    }
  })

  it('keeps every string entry non-empty', () => {
    for (const [locale, copy] of Object.entries(copyByLocale)) {
      for (const path of keyPaths(copy)) {
        const value = path.split('.').reduce<unknown>((node, key) => (
          node !== null && typeof node === 'object' ? (node as Record<string, unknown>)[key] : node
        ), copy)
        if (typeof value === 'string') {
          expect(value.trim() !== '', `${locale}:${path} is empty`).toBe(true)
        }
      }
    }
  })
})
