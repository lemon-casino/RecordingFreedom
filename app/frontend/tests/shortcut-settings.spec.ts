import {expect, test, type Page} from '@playwright/test'

const browserSettingsKey = 'recordingfreedom.settings.v1'

test('settings captures and persists shortcut changes', async ({page}) => {
  await openRecorderShell(page)

  await page.getByRole('button', {name: 'Open settings'}).click()
  await expect(page.getByText('Shortcuts', {exact: true})).toBeVisible()

  const pauseShortcut = page.locator('.setting-shortcut').filter({hasText: 'Pause / resume'})
  await expect(pauseShortcut.locator('kbd')).toHaveText('Ctrl + Shift + P')

  await pauseShortcut.getByRole('button', {name: 'Change'}).click()
  await expect(pauseShortcut.getByRole('button', {name: 'Listening'})).toBeFocused()
  await pauseShortcut.getByRole('button', {name: 'Listening'}).press('Control+Alt+P')

  await expect(pauseShortcut.locator('kbd')).toHaveText('Ctrl + Alt + P')
  await expect.poll(async () => page.evaluate((settingsKey) => {
    const raw = window.localStorage.getItem(settingsKey)
    return raw ? JSON.parse(raw).shortcuts?.togglePause : ''
  }, browserSettingsKey)).toBe('CmdOrCtrl+OptionOrAlt+P')
})

async function openRecorderShell(page: Page) {
  await page.addInitScript((settingsKey) => {
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
      shortcuts: {
        toggleRecording: 'CmdOrCtrl+Shift+R',
        togglePause: 'CmdOrCtrl+Shift+P',
        toggleCamera: 'CmdOrCtrl+Shift+C',
        openWhiteboard: 'CmdOrCtrl+Shift+B',
      },
      window: {
        minimizeToTray: true,
        theme: 'night-teal',
      },
    }))
  }, browserSettingsKey)
  await page.goto('/')
}
