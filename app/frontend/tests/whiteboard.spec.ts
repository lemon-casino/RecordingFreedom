import {expect, test, type Page} from '@playwright/test'

const browserSettingsKey = 'recordingfreedom.settings.v1'
const browserWhiteboardSceneKey = 'recordingfreedom.whiteboard.scene.v1'
const browserScreenshotWhiteboardKey = 'recordingfreedom.screenshots.whiteboard.v1'

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
      await expectRoundedWhiteboardCorners(page)

      const screenshot = await page.locator('.whiteboard-capsule').screenshot()
      expect(screenshot.length).toBeGreaterThan(10_000)
    })
  }
})

test('whiteboard imports a screenshot context on initial load', async ({page}) => {
  await openWhiteboardWithSettings(page, 'en', 'mountain-green', screenshotWhiteboardContext('initial-shot'))

  await expect(page.locator('.whiteboard-shell')).toBeVisible()
  await expect.poll(async () => readSavedWhiteboardScene(page, 'initial-shot')).toEqual({
    hasImageElement: true,
    hasImageFile: true,
  })
})

test('whiteboard imports a screenshot context while the window is already open', async ({page}) => {
  await openWhiteboardWithSettings(page, 'en', 'mountain-green')
  await expect(page.locator('.whiteboard-shell')).toBeVisible()

  await page.evaluate((context) => {
    ;(window as Window & {__RF_SCREENSHOT_WHITEBOARD__?: unknown}).__RF_SCREENSHOT_WHITEBOARD__ = context
    window.dispatchEvent(new CustomEvent('rf-screenshot-whiteboard', {detail: context}))
  }, screenshotWhiteboardContext('live-shot'))

  await expect.poll(async () => readSavedWhiteboardScene(page, 'live-shot')).toEqual({
    hasImageElement: true,
    hasImageFile: true,
  })
})

async function openWhiteboardWithSettings(page: Page, locale: LocaleCode, theme: ThemeCode, screenshotContext?: unknown) {
  await page.addInitScript(({settingsKey, screenshotWhiteboardKey, localeValue, themeValue, context}) => {
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
    if (context) {
      window.localStorage.setItem(screenshotWhiteboardKey, JSON.stringify(context))
      ;(window as Window & {__RF_SCREENSHOT_WHITEBOARD__?: unknown}).__RF_SCREENSHOT_WHITEBOARD__ = context
    }
  }, {settingsKey: browserSettingsKey, screenshotWhiteboardKey: browserScreenshotWhiteboardKey, localeValue: locale, themeValue: theme, context: screenshotContext})
  await page.goto('/#/whiteboard')
}

function screenshotWhiteboardContext(id: string) {
  return {
    available: true,
    item: {
      id,
      path: `browser-preview/data/screenshots/${id}.png`,
      thumbnailPath: `browser-preview/data/screenshots/thumbnails/${id}.png`,
      createdAt: '2026-07-04T12:00:00Z',
      width: 640,
      height: 360,
      mode: 'region',
      pinned: false,
      fixed: false,
    },
    dataUrl: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=',
  }
}

async function readSavedWhiteboardScene(page: Page, screenshotId: string) {
  return page.evaluate(({sceneKey, id}) => {
    const raw = window.localStorage.getItem(sceneKey) || ''
    if (!raw) return {hasImageElement: false, hasImageFile: false}
    const scene = JSON.parse(raw)
    const fileId = `rf-screenshot-${id}`
    return {
      hasImageElement: Array.isArray(scene.elements) && scene.elements.some((element: {type?: string; fileId?: string}) => element.type === 'image' && element.fileId === fileId),
      hasImageFile: Boolean(scene.files?.[fileId]?.dataURL),
    }
  }, {sceneKey: browserWhiteboardSceneKey, id: screenshotId})
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

async function expectRoundedWhiteboardCorners(page: Page) {
  await expect(page.locator('.whiteboard-canvas .excalidraw')).toBeVisible()

  const corners = await page.evaluate(() => {
    const read = (selector: string) => {
      const element = document.querySelector(selector)
      if (!element) return null
      const style = window.getComputedStyle(element)
      return {
        background: style.backgroundColor,
        borderTopLeftRadius: style.borderTopLeftRadius,
        borderTopRightRadius: style.borderTopRightRadius,
        borderBottomRightRadius: style.borderBottomRightRadius,
        borderBottomLeftRadius: style.borderBottomLeftRadius,
        clipPath: style.clipPath,
        overflowX: style.overflowX,
        overflowY: style.overflowY,
      }
    }

    return {
      body: read('body'),
      shell: read('.whiteboard-shell'),
      canvas: read('.whiteboard-canvas'),
      excalidraw: read('.whiteboard-canvas .excalidraw'),
    }
  })

  expect(corners.body?.background).toBe('rgba(0, 0, 0, 0)')
  expect(corners.shell?.borderTopLeftRadius).toBe('24px')
  expect(corners.shell?.borderTopRightRadius).toBe('24px')
  expect(corners.shell?.borderBottomRightRadius).toBe('24px')
  expect(corners.shell?.borderBottomLeftRadius).toBe('24px')
  expect(corners.shell?.overflowX).toBe('hidden')
  expect(corners.canvas?.borderTopLeftRadius).toBe('22px')
  expect(corners.canvas?.borderTopRightRadius).toBe('22px')
  expect(corners.canvas?.borderBottomRightRadius).toBe('22px')
  expect(corners.canvas?.borderBottomLeftRadius).toBe('22px')
  expect(corners.canvas?.clipPath).not.toBe('none')
  expect(corners.canvas?.overflowX).toBe('hidden')
  expect(corners.excalidraw?.borderTopLeftRadius).toBe('22px')
  expect(corners.excalidraw?.overflowX).toBe('hidden')
}
