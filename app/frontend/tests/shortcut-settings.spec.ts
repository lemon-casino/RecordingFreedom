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

test('settings persists start at login toggle', async ({page}) => {
  await openRecorderShell(page)

  await page.getByRole('button', {name: 'Open settings'}).click()
  const startAtLoginRow = page.locator('.setting-control').filter({hasText: 'Start at login'})
  const startAtLogin = startAtLoginRow.getByLabel('Start at login')
  await expect(startAtLogin).not.toBeChecked()

  await startAtLoginRow.locator('.switch-row').click()
  await expect(startAtLogin).toBeChecked()
  await expect.poll(async () => page.evaluate((settingsKey) => {
    const raw = window.localStorage.getItem(settingsKey)
    return raw ? JSON.parse(raw).window?.startAtLogin : null
  }, browserSettingsKey)).toBe(true)

  await startAtLoginRow.locator('.switch-row').click()
  await expect(startAtLogin).not.toBeChecked()
  await expect.poll(async () => page.evaluate((settingsKey) => {
    const raw = window.localStorage.getItem(settingsKey)
    return raw ? JSON.parse(raw).window?.startAtLogin : null
  }, browserSettingsKey)).toBe(false)
})

test('settings exposes local OCR model package management', async ({page}) => {
  await openRecorderShell(page)

  await page.getByRole('button', {name: 'Open settings'}).click()
  const ocrModels = page.locator('.ocr-model-settings')
  await expect(ocrModels.getByText('OCR models', {exact: true})).toBeVisible()
  await expect(ocrModels).toContainText('No model')
  await expect(ocrModels).toContainText('Stable')
  await expect(ocrModels).toContainText('Latest')
  await expect(ocrModels).toContainText('Quality')
  await expect(ocrModels).toContainText('ppocrv5-mobile-zh-en')
  await expect(ocrModels).toContainText('ppocrv6-mobile-zh-en')
  await expect(ocrModels).toContainText('ppocrv6-medium-zh-en')
  await expect(ocrModels).toContainText('No RecordingFreedom-verified download package yet')
  await expect(ocrModels.getByPlaceholder('Model package .zip or extracted folder path')).toBeVisible()

  await ocrModels.getByPlaceholder('Model package .zip or extracted folder path').fill('C:/tmp/missing-ocr-model.zip')
  await ocrModels.getByRole('button', {name: 'Import'}).click()
  await expect(ocrModels).toContainText('OCR model package import is local-only')
})

test('settings downloads a verified OCR model package without auto-switching active model', async ({page}) => {
  await page.addInitScript((ocrStatus) => {
    ;(window as Window & {__RF_OCR_STATUS__?: unknown}).__RF_OCR_STATUS__ = ocrStatus
  }, downloadableOcrStatus())
  await openRecorderShell(page)

  await page.getByRole('button', {name: 'Open settings'}).click()
  const ocrModels = page.locator('.ocr-model-settings')
  const stableModel = ocrModels.locator('.ocr-model-row').filter({hasText: 'ppocrv5-mobile-zh-en'})
  const latestModel = ocrModels.locator('.ocr-model-row').filter({hasText: 'ppocrv6-mobile-zh-en'})

  await expect(stableModel).toContainText('Active')
  await expect(latestModel).toContainText('Download size')
  await latestModel.getByRole('button', {name: 'Download'}).click()

  await expect.poll(async () => page.evaluate(() => {
    return (window as Window & {__RF_LAST_OCR_MODEL_DOWNLOAD__?: {modelId: string}}).__RF_LAST_OCR_MODEL_DOWNLOAD__?.modelId ?? ''
  })).toBe('ppocrv6-mobile-zh-en')
  await expect(ocrModels).toContainText('Model downloaded and verified')
  await expect(latestModel).toContainText('Verified')
  await expect(stableModel).toContainText('Active')
  await expect.poll(async () => page.evaluate(() => {
    return (window as Window & {__RF_OCR_STATUS__?: {activeModelId?: string}}).__RF_OCR_STATUS__?.activeModelId ?? ''
  })).toBe('ppocrv5-mobile-zh-en')

  await latestModel.getByRole('button', {name: 'Use'}).click()
  await latestModel.getByRole('button', {name: 'Confirm switch'}).click()
  await expect.poll(async () => page.evaluate(() => {
    return (window as Window & {__RF_LAST_SET_ACTIVE_OCR_MODEL__?: {modelId: string}}).__RF_LAST_SET_ACTIVE_OCR_MODEL__?.modelId ?? ''
  })).toBe('ppocrv6-mobile-zh-en')
})

test('settings refreshes the verified OCR model catalog before exposing downloads', async ({page}) => {
  await page.addInitScript(({initialStatus, catalogStatus}) => {
    ;(window as Window & {
      __RF_OCR_STATUS__?: unknown
      __RF_OCR_MODEL_CATALOG_STATUS__?: unknown
    }).__RF_OCR_STATUS__ = initialStatus
    ;(window as Window & {
      __RF_OCR_MODEL_CATALOG_STATUS__?: unknown
    }).__RF_OCR_MODEL_CATALOG_STATUS__ = catalogStatus
  }, {
    initialStatus: catalogRefreshInitialOcrStatus(),
    catalogStatus: downloadableOcrStatus(),
  })
  await openRecorderShell(page)

  await page.getByRole('button', {name: 'Open settings'}).click()
  const ocrModels = page.locator('.ocr-model-settings')
  const latestModel = ocrModels.locator('.ocr-model-row').filter({hasText: 'ppocrv6-mobile-zh-en'})

  await expect(latestModel).toContainText('No RecordingFreedom-verified download package yet')
  await expect(latestModel.getByRole('button', {name: 'Download'})).toHaveCount(0)

  await ocrModels.getByRole('button', {name: 'Refresh catalog'}).click()
  await expect.poll(async () => page.evaluate(() => {
    return (window as Window & {__RF_LAST_OCR_MODEL_CATALOG_REFRESH__?: {catalogUrl: string}}).__RF_LAST_OCR_MODEL_CATALOG_REFRESH__?.catalogUrl ?? 'missing'
  })).toBe('')
  await expect(ocrModels).toContainText('Model catalog refreshed')
  await expect(latestModel).toContainText('Download size')
  await expect(latestModel.getByRole('button', {name: 'Download'})).toBeVisible()
})

test('settings confirms before switching the active OCR model', async ({page}) => {
  await page.addInitScript((ocrStatus) => {
    ;(window as Window & {__RF_OCR_STATUS__?: unknown}).__RF_OCR_STATUS__ = ocrStatus
  }, installedOcrStatus())
  await openRecorderShell(page)

  await page.getByRole('button', {name: 'Open settings'}).click()
  const ocrModels = page.locator('.ocr-model-settings')
  const latestModel = ocrModels.locator('.ocr-model-row').filter({hasText: 'ppocrv6-mobile-zh-en'})

  await latestModel.getByRole('button', {name: 'Use'}).click()
  await expect(latestModel.getByRole('alert')).toContainText('Switching runs only after the model is installed')
  await expect.poll(async () => page.evaluate(() => {
    return (window as Window & {__RF_LAST_SET_ACTIVE_OCR_MODEL__?: {modelId: string}}).__RF_LAST_SET_ACTIVE_OCR_MODEL__?.modelId ?? ''
  })).toBe('')

  await latestModel.getByRole('button', {name: 'Confirm switch'}).click()
  await expect.poll(async () => page.evaluate(() => {
    return (window as Window & {__RF_LAST_SET_ACTIVE_OCR_MODEL__?: {modelId: string}}).__RF_LAST_SET_ACTIVE_OCR_MODEL__?.modelId ?? ''
  })).toBe('ppocrv6-mobile-zh-en')
  await expect.poll(async () => page.evaluate(() => {
    return (window as Window & {__RF_OCR_STATUS__?: {activeModelId?: string}}).__RF_OCR_STATUS__?.activeModelId ?? ''
  })).toBe('ppocrv6-mobile-zh-en')
  await expect(latestModel).toContainText('Active')
})

test('settings keeps the current OCR model when a confirmed switch fails', async ({page}) => {
  await page.addInitScript((ocrStatus) => {
    ;(window as Window & {
      __RF_OCR_STATUS__?: unknown
      __RF_SET_ACTIVE_OCR_MODEL_ERROR__?: string
    }).__RF_OCR_STATUS__ = ocrStatus
    ;(window as Window & {
      __RF_SET_ACTIVE_OCR_MODEL_ERROR__?: string
    }).__RF_SET_ACTIVE_OCR_MODEL_ERROR__ = 'OCR model smoke failed'
  }, installedOcrStatus())
  await openRecorderShell(page)

  await page.getByRole('button', {name: 'Open settings'}).click()
  const ocrModels = page.locator('.ocr-model-settings')
  const stableModel = ocrModels.locator('.ocr-model-row').filter({hasText: 'ppocrv5-mobile-zh-en'})
  const latestModel = ocrModels.locator('.ocr-model-row').filter({hasText: 'ppocrv6-mobile-zh-en'})

  await latestModel.getByRole('button', {name: 'Use'}).click()
  await latestModel.getByRole('button', {name: 'Confirm switch'}).click()

  await expect(ocrModels).toContainText('OCR model smoke failed')
  await expect.poll(async () => page.evaluate(() => {
    return (window as Window & {__RF_OCR_STATUS__?: {activeModelId?: string}}).__RF_OCR_STATUS__?.activeModelId ?? ''
  })).toBe('ppocrv5-mobile-zh-en')
  await expect(stableModel).toContainText('Active')
  await expect(latestModel).toContainText('Verified')
})

test('settings persists explicit OCR translation provider configuration', async ({page}) => {
  await openRecorderShell(page)

  await page.getByRole('button', {name: 'Open settings'}).click()
  const translation = page.locator('.ocr-translation-settings')
  await expect(translation.getByText('OCR translation', {exact: true})).toBeVisible()
  await expect(translation).toContainText('Off by default')

  await translation.getByRole('button', {name: 'Off'}).click()
  await page.getByRole('option', {name: 'OpenAI-compatible'}).click()
  await translation.getByPlaceholder('https://api.example.com/v1').fill('https://translator.example/v1')
  await translation.getByPlaceholder('https://api.example.com/v1').press('Enter')
  await translation.getByPlaceholder('sk-...').fill('local-test-key')
  await translation.getByPlaceholder('sk-...').press('Enter')
  await translation.getByPlaceholder('gpt-4o-mini').fill('gpt-test')
  await translation.getByPlaceholder('gpt-4o-mini').press('Enter')
  await translation.locator('.setting-control').filter({hasText: 'Allow sending OCR text for translation'}).locator('.switch-row').click()

  await expect.poll(async () => page.evaluate((settingsKey) => {
    const raw = window.localStorage.getItem(settingsKey)
    const translation = raw ? JSON.parse(raw).ocr?.translation : null
    return {
      provider: translation?.provider,
      baseUrl: translation?.baseUrl,
      apiKey: translation?.apiKey,
      model: translation?.model,
      privacyConfirmed: translation?.privacyConfirmed,
    }
  }, browserSettingsKey)).toEqual({
    provider: 'openai-compatible',
    baseUrl: 'https://translator.example/v1',
    apiKey: 'local-test-key',
    model: 'gpt-test',
    privacyConfirmed: true,
  })
})

function installedOcrStatus() {
  return {
    status: 'ready',
    activeModelId: 'ppocrv5-mobile-zh-en',
    models: [
      {
        id: 'ppocrv5-mobile-zh-en',
        name: 'PP-OCRv5 Mobile Chinese/English',
        channel: 'stable',
        engine: 'onnxruntime',
        language: ['zh', 'en'],
        downloadAvailable: false,
        downloadBytes: 0,
        installed: true,
        verified: true,
        active: true,
        smokeAssetReady: true,
        missingFiles: [],
      },
      {
        id: 'ppocrv6-mobile-zh-en',
        name: 'PP-OCRv6 Mobile Chinese/English',
        channel: 'latest',
        engine: 'onnxruntime',
        language: ['zh', 'en'],
        downloadAvailable: false,
        downloadBytes: 0,
        installed: true,
        verified: true,
        active: false,
        smokeAssetReady: true,
        missingFiles: [],
      },
      {
        id: 'ppocrv6-medium-zh-en',
        name: 'PP-OCRv6 Medium Chinese/English',
        channel: 'quality',
        engine: 'onnxruntime',
        language: ['zh', 'en'],
        downloadAvailable: false,
        downloadBytes: 0,
        installed: false,
        verified: false,
        active: false,
        smokeAssetReady: false,
        missingFiles: ['manifest.json', 'det.onnx', 'cls.onnx', 'rec.onnx', 'keys.txt'],
      },
    ],
    workerPath: 'browser-preview/rf-ocr-worker',
    runtimeDir: 'browser-preview/onnxruntime',
  }
}

function downloadableOcrStatus() {
  const status = installedOcrStatus()
  return {
    ...status,
    models: status.models.map((model) => model.id === 'ppocrv6-mobile-zh-en'
      ? {
          ...model,
          downloadAvailable: true,
          downloadBytes: 7340032,
          installed: false,
          verified: false,
          active: false,
          smokeAssetReady: false,
          missingFiles: ['manifest.json', 'det.onnx', 'cls.onnx', 'rec.onnx', 'keys.txt'],
        }
      : model),
  }
}

function catalogRefreshInitialOcrStatus() {
  const status = installedOcrStatus()
  return {
    ...status,
    models: status.models.map((model) => model.id === 'ppocrv6-mobile-zh-en'
      ? {
          ...model,
          downloadAvailable: false,
          downloadBytes: 0,
          installed: false,
          verified: false,
          active: false,
          smokeAssetReady: false,
          missingFiles: ['manifest.json', 'det.onnx', 'cls.onnx', 'rec.onnx', 'keys.txt'],
        }
      : model),
  }
}

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
      ocr: {
        autoRecognizeScreenshots: false,
        translation: {
          provider: 'disabled',
          sourceLanguage: 'auto',
          targetLanguage: 'zh-CN',
          privacyConfirmed: false,
        },
      },
      shortcuts: {
        toggleRecording: 'CmdOrCtrl+Shift+R',
        togglePause: 'CmdOrCtrl+Shift+P',
        toggleCamera: 'CmdOrCtrl+Shift+C',
        openWhiteboard: 'CmdOrCtrl+Shift+B',
        openScreenshot: 'CmdOrCtrl+Shift+S',
      },
      window: {
        minimizeToTray: true,
        theme: 'night-teal',
        startAtLogin: false,
      },
    }))
  }, browserSettingsKey)
  await page.goto('/')
}
