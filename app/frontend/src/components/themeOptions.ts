import type {RecorderCopy} from '../i18n'
import {themeGroups, themeSwatches, type ThemeCode, type ThemeGroupCode} from '../services/mockBackend'

export type SelectMenuOption = {
  value: string
  label: string
  disabled?: boolean
  swatch?: string
}

export function themeSelectOptions(copy: RecorderCopy) {
  const groups: ThemeGroupCode[] = ['dark', 'light']
  return groups.flatMap((group) => [
    {
      value: `theme-group-${group}`,
      label: copy.themeGroupNames[group],
      disabled: true },
    ...themeGroups[group].map((code): SelectMenuOption => ({
      value: code,
      label: copy.themeNames[code],
      swatch: themeSwatches[code] })),
  ])
}

export function isThemeGroupOption(option: SelectMenuOption) {
  return option.disabled === true && option.value.startsWith('theme-group-')
}

export const themeHintKey = 'recordingfreedom.theme-hint.v1'

// Applies the theme now and records it so the next window (and the next app
// launch, where webview storage persists) paints with it on the first frame.
export function applyTheme(theme: ThemeCode) {
  document.documentElement.dataset.theme = theme
  try {
    window.localStorage.setItem(themeHintKey, theme)
  } catch {
    // Storage may be unavailable; the URL parameter still covers secondary windows.
  }
}
