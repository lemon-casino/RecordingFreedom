import type {RecorderCopy} from '../i18n'
import {themeGroups, themeSwatches, type ThemeGroupCode} from '../services/mockBackend'

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
