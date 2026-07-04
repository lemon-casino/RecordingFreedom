import {expect, test, type Page} from '@playwright/test'

const browserSettingsKey = 'recordingfreedom.settings.v1'
const browserScreenshotHistoryKey = 'recordingfreedom.screenshots.history.v1'

test('capsule whiteboard opens board before recording and annotation during video recording', async ({page}) => {
  await openRecorderShell(page)

  const toolsButton = page.getByRole('button', {name: 'Screenshot / board'})
  await expect(toolsButton).toBeVisible()
  await expect(toolsButton.getByText('Tools')).toBeVisible()

  await toolsButton.click()
  await expect(page.getByRole('dialog', {name: 'board menu'})).toBeVisible()
  await expect(page.getByRole('button', {name: /Region screenshot/})).toBeVisible()
  await page.getByRole('button', {name: /Board/}).click()
  await expectWhiteboardLaunch(page, 'whiteboard', '/#/whiteboard')
  await expect(toolsButton).toHaveAttribute('aria-pressed', 'true')
  await emitWhiteboardVisibility(page, {visible: false, mode: 'whiteboard'})
  await expect(toolsButton).toHaveAttribute('aria-pressed', 'false')

  await page.getByRole('button', {name: 'Start recording'}).click()
  await expect(page.getByRole('button', {name: 'Stop recording'})).toBeVisible()
  await expect(page.locator('.rf-shell')).toHaveClass(/is-recording-compact/)
  const whiteboardButton = page.getByRole('button', {name: 'Open whiteboard'})
  await expect(whiteboardButton).toBeVisible()
  await expect(whiteboardButton).toBeEnabled()

  await whiteboardButton.click()
  await expectWhiteboardLaunch(page, 'annotation', '/#/annotation-overlay')
  await expect(whiteboardButton).toHaveAttribute('aria-pressed', 'true')
  await emitWhiteboardVisibility(page, {visible: false, mode: 'annotation'})
  await expect(whiteboardButton).toHaveAttribute('aria-pressed', 'false')

  await page.getByRole('button', {name: 'Pause recording'}).click()
  await expect(page.getByRole('button', {name: 'Resume recording'})).toBeVisible()
  await expect(whiteboardButton).toBeVisible()
  await expect(whiteboardButton).toBeEnabled()

  await whiteboardButton.click()
  await expectWhiteboardLaunch(page, 'annotation', '/#/annotation-overlay')
  await expect(whiteboardButton).toHaveAttribute('aria-pressed', 'true')
})

test('capsule whiteboard remains available as a board during audio recording', async ({page}) => {
  await openRecorderShell(page, {systemAudio: true})

  await page.locator('.source-pill').click()
  await page.getByRole('group', {name: 'Recording mode'}).getByRole('button', {name: 'Audio'}).click()

  await expect(page.getByRole('button', {name: 'Screenshot / board'})).toBeVisible()

  await page.getByRole('button', {name: 'Start recording'}).click()
  await expect(page.getByRole('button', {name: 'Stop recording'})).toBeVisible()
  await expect(page.locator('.rf-shell')).toHaveClass(/is-recording-compact/)
  const whiteboardButton = page.getByRole('button', {name: 'Open whiteboard'})
  await expect(whiteboardButton).toBeVisible()
  await expect(whiteboardButton).toBeEnabled()

  await whiteboardButton.click()
  await expectWhiteboardLaunch(page, 'whiteboard', '/#/whiteboard')
  await expect(whiteboardButton).toHaveAttribute('aria-pressed', 'true')
})

test('capsule shows a labeled whiteboard entry before recording in Chinese', async ({page}) => {
  await openRecorderShell(page, {locale: 'zh-CN'})

  const toolsButton = page.getByRole('button', {name: '截图 / 画板'})
  await expect(toolsButton).toBeVisible()
  await expect(toolsButton.getByText('工具')).toBeVisible()
})

test('screenshot history exposes folder and delete actions', async ({page}) => {
  await openRecorderShell(page, {
    screenshotHistory: [{
      id: 'history-shot',
      path: 'browser-preview/data/screenshots/history-shot.png',
      thumbnailPath: 'browser-preview/data/screenshots/thumbnails/history-shot.png',
      createdAt: '2026-07-04T12:00:00Z',
      width: 420,
      height: 260,
      mode: 'region',
      pinned: false,
      fixed: false,
    }],
  })

  await page.getByRole('button', {name: 'Screenshot / board'}).click()
  await expect(page.getByRole('button', {name: 'Open containing folder'})).toBeVisible()
  await page.getByRole('button', {name: 'Delete screenshot'}).click()
  await expect(page.getByText('No screenshot history')).toBeVisible()
  await expect.poll(async () => page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) || '[]').length, browserScreenshotHistoryKey)).toBe(0)
})

test('region screenshot enters editable selection before saving', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'screenshot-region-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'screenshot',
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(120, 120)
  await page.mouse.down()
  await page.mouse.move(420, 300)
  await page.mouse.up()

  await expect(page.locator('.region-edit-rect')).toBeVisible()
  await expect(page.getByRole('button', {name: 'Save screenshot'})).toBeVisible()
  await page.getByRole('button', {name: 'Save screenshot'}).click()
  await expect.poll(async () => page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) || '[]').length, browserScreenshotHistoryKey)).toBe(1)
})

async function openRecorderShell(page: Page, options: {locale?: 'zh-CN' | 'en'; microphone?: boolean; systemAudio?: boolean; screenshotHistory?: unknown[]} = {}) {
  await page.addInitScript(({settingsKey, screenshotHistoryKey, settings, screenshotHistory}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    if (screenshotHistory) {
      window.localStorage.setItem(screenshotHistoryKey, JSON.stringify(screenshotHistory))
    }
    window.open = ((url?: string | URL, target?: string, features?: string) => {
      ;(window as Window & {__RF_LAST_WINDOW_OPEN__?: {url: string; target?: string; features?: string}}).__RF_LAST_WINDOW_OPEN__ = {
        url: String(url ?? ''),
        target,
        features,
      }
      return {focus: () => undefined} as Window
    }) as typeof window.open
  }, {
    settingsKey: browserSettingsKey,
    screenshotHistoryKey: browserScreenshotHistoryKey,
    settings: baseBrowserSettings(
      options.locale ?? 'en',
      options.microphone === true,
      options.systemAudio === true,
    ),
    screenshotHistory: options.screenshotHistory,
  })
  await page.goto('/')
}

function baseBrowserSettings(locale: 'zh-CN' | 'en', microphone: boolean, systemAudio: boolean) {
  return {
    schemaVersion: 1,
    locale,
    source: {lastSourceType: 'screen'},
    storage: {dataRootDir: 'browser-preview'},
    recording: {
      quality: 'balanced',
      fps: 30,
      captureCursor: true,
      countdownSeconds: 0,
    },
    audio: {
      system: systemAudio,
      systemDeviceId: 'system-audio:default',
      microphone,
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
      theme: 'night-teal',
    },
  }
}

async function expectWhiteboardLaunch(page: Page, mode: 'whiteboard' | 'annotation', url: string) {
  await expect.poll(async () => page.evaluate(() => {
    const launch = (window as Window & {
      __RF_LAST_WHITEBOARD_LAUNCH__?: {mode: string; url: string}
      __RF_LAST_WINDOW_OPEN__?: {url: string}
    }).__RF_LAST_WHITEBOARD_LAUNCH__
    const popup = (window as Window & {__RF_LAST_WINDOW_OPEN__?: {url: string}}).__RF_LAST_WINDOW_OPEN__
    return {
      mode: launch?.mode ?? '',
      launchUrl: launch?.url ?? '',
      popupUrl: popup?.url ?? '',
    }
  })).toEqual({mode, launchUrl: url, popupUrl: url})
}

async function emitWhiteboardVisibility(page: Page, event: {visible: boolean; mode: 'whiteboard' | 'annotation'}) {
  await page.evaluate((detail) => {
    window.dispatchEvent(new CustomEvent('rf-whiteboard-visibility', {detail}))
  }, event)
}
