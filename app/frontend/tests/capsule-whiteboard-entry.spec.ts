import {expect, test, type Page} from '@playwright/test'

const browserSettingsKey = 'recordingfreedom.settings.v1'

test('capsule whiteboard opens board before recording and annotation during video recording', async ({page}) => {
  await openRecorderShell(page)

  const whiteboardButton = page.getByRole('button', {name: 'Open whiteboard'})
  await expect(whiteboardButton).toBeVisible()

  await whiteboardButton.click()
  await expectWhiteboardLaunch(page, 'whiteboard', '/#/whiteboard')
  await expect(whiteboardButton).toHaveAttribute('aria-pressed', 'true')
  await emitWhiteboardVisibility(page, {visible: false, mode: 'whiteboard'})
  await expect(whiteboardButton).toHaveAttribute('aria-pressed', 'false')

  await page.getByRole('button', {name: 'Start recording'}).click()
  await expect(page.getByRole('button', {name: 'Stop recording'})).toBeVisible()
  await expect(page.locator('.rf-shell')).toHaveClass(/is-recording-compact/)
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

  const whiteboardButton = page.getByRole('button', {name: 'Open whiteboard'})
  await expect(whiteboardButton).toBeVisible()

  await page.getByRole('button', {name: 'Start recording'}).click()
  await expect(page.getByRole('button', {name: 'Stop recording'})).toBeVisible()
  await expect(page.locator('.rf-shell')).toHaveClass(/is-recording-compact/)
  await expect(whiteboardButton).toBeVisible()
  await expect(whiteboardButton).toBeEnabled()

  await whiteboardButton.click()
  await expectWhiteboardLaunch(page, 'whiteboard', '/#/whiteboard')
  await expect(whiteboardButton).toHaveAttribute('aria-pressed', 'true')
})

async function openRecorderShell(page: Page, options: {microphone?: boolean; systemAudio?: boolean} = {}) {
  await page.addInitScript(({settingsKey, microphone, systemAudio}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify({
      schemaVersion: 1,
      locale: 'en',
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
    }))
    window.open = ((url?: string | URL, target?: string, features?: string) => {
      ;(window as Window & {__RF_LAST_WINDOW_OPEN__?: {url: string; target?: string; features?: string}}).__RF_LAST_WINDOW_OPEN__ = {
        url: String(url ?? ''),
        target,
        features,
      }
      return {focus: () => undefined} as Window
    }) as typeof window.open
  }, {settingsKey: browserSettingsKey, microphone: options.microphone === true, systemAudio: options.systemAudio === true})
  await page.goto('/')
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
