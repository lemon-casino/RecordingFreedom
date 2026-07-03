import {expect, test, type Page} from '@playwright/test'

const browserSettingsKey = 'recordingfreedom.settings.v1'

type LocaleCode = 'zh-CN' | 'en'
type ThemeCode = 'night-teal' | 'mountain-green'

const cases: Array<{
  name: string
  locale: LocaleCode
  theme: ThemeCode
  title: string
  subtitle: string
  penLabel: string
  saveLabel: string
  exportLabel: string
}> = [
  {
    name: 'Chinese Night Teal',
    locale: 'zh-CN',
    theme: 'night-teal',
    title: '画板',
    subtitle: '录制前和录制中都可使用',
    penLabel: '画笔',
    saveLabel: '保存',
    exportLabel: '导出 PNG',
  },
  {
    name: 'English Mountain Green',
    locale: 'en',
    theme: 'mountain-green',
    title: 'Whiteboard',
    subtitle: 'Available before and during recording',
    penLabel: 'Pen',
    saveLabel: 'Save',
    exportLabel: 'Export PNG',
  },
]

test.describe('whiteboard capsule localization and themes', () => {
  for (const scenario of cases) {
    test(`${scenario.name} renders the complete capsule`, async ({page}) => {
      await openWhiteboardWithSettings(page, scenario.locale, scenario.theme)

      await expect(page.locator('.whiteboard-shell')).toBeVisible()
      await expect(page.locator('.whiteboard-capsule')).toBeVisible()
      await expect(page.locator('.whiteboard-title strong')).toHaveText(scenario.title)
      await expect(page.locator('.whiteboard-title span')).toHaveText(scenario.subtitle)
      await expect(page.locator('.whiteboard-tools button')).toHaveCount(10)
      await expect(page.getByRole('button', {name: scenario.penLabel})).toHaveClass(/selected/)
      await expect(page.getByRole('button', {name: scenario.saveLabel})).toBeVisible()
      await expect(page.getByRole('button', {name: scenario.exportLabel})).toBeVisible()
      await expect(page.locator('.whiteboard-status')).toBeVisible()

      await expect(page.locator('html')).toHaveAttribute('lang', scenario.locale)
      await expect(page.locator('html')).toHaveAttribute('data-theme', scenario.theme)
      await expect(page.locator('.whiteboard-shell')).toHaveAttribute('data-theme', scenario.theme)
      await expectNoHorizontalOverflow(page)

      const screenshot = await page.locator('.whiteboard-capsule').screenshot()
      expect(screenshot.length).toBeGreaterThan(10_000)
    })
  }
})

async function openWhiteboardWithSettings(page: Page, locale: LocaleCode, theme: ThemeCode) {
  await page.addInitScript(({settingsKey, localeValue, themeValue}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify({
      schemaVersion: 1,
      locale: localeValue,
      source: {lastSourceType: 'screen'},
      storage: {dataRootDir: 'browser-preview'},
      recording: {
        quality: 'balanced',
        fps: 30,
        captureCursor: true,
        countdownSeconds: 0,
      },
      audio: {
        system: false,
        systemDeviceId: 'system-audio:default',
        microphone: false,
        microphoneDeviceId: 'microphone:browser-preview',
        noiseSuppression: false,
        microphoneGain: 1,
      },
      camera: {
        enabled: false,
        deviceId: 'camera:default',
        pipPreset: 'bottom-right',
        pip: {
          preset: 'bottom-right',
          shape: 'circle',
          mirror: true,
          position: {x: 1, y: 1},
          scale: 0.08,
          edgeFeather: 0.16,
        },
      },
      whiteboard: {
        enabled: true,
        lastMode: 'board',
        lastTool: 'freedraw',
        lastStrokeColor: '#ef4444',
        lastStrokeWidth: 'medium',
        lastOpacity: 100,
        capturePolicy: 'export-compose',
      },
      window: {
        minimizeToTray: true,
        theme: themeValue,
      },
    }))
  }, {settingsKey: browserSettingsKey, localeValue: locale, themeValue: theme})
  await page.goto('/#/whiteboard')
}

async function expectNoHorizontalOverflow(page: Page) {
  const overflow = await page.evaluate(() => ({
    body: document.body.scrollWidth - document.body.clientWidth,
    root: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    capsule: (() => {
      const capsule = document.querySelector('.whiteboard-capsule')
      if (!capsule) return 0
      const rect = capsule.getBoundingClientRect()
      return Math.max(0, rect.right - window.innerWidth, -rect.left)
    })(),
  }))
  expect(overflow.body).toBeLessThanOrEqual(1)
  expect(overflow.root).toBeLessThanOrEqual(1)
  expect(overflow.capsule).toBeLessThanOrEqual(1)
}
