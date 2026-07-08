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

test('whiteboard selected image OCR queues its own exported image as whiteboard-selection', async ({page}) => {
  await openWhiteboardWithSettings(page, 'en', 'mountain-green', undefined, selectedImageWhiteboardScene())

  const selectedImageButton = page.getByRole('button', {name: 'Recognize selected image'})
  await expect(selectedImageButton).toBeEnabled()
  await selectedImageButton.click()

  await expect.poll(async () => page.evaluate(() => {
    const state = window as Window & {
      __RF_LAST_WHITEBOARD_OCR_QUEUE__?: {
        status?: string
        request?: {
          imagePath?: string
          sourceKind?: string
          sourceId?: string
          language?: string
          priority?: string
        }
      }
      __RF_LAST_WHITEBOARD_OCR_REQUEST__?: {
        imagePath?: string
        sceneId?: string
        elementId?: string
        language?: string
      }
    }
    return {
      status: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.status ?? '',
      imagePath: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.imagePath ?? '',
      sourceKind: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.sourceKind ?? '',
      sourceId: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.sourceId ?? '',
      language: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.language ?? '',
      priority: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.priority ?? '',
      elementId: state.__RF_LAST_WHITEBOARD_OCR_REQUEST__?.elementId ?? '',
      rawImagePath: state.__RF_LAST_WHITEBOARD_OCR_REQUEST__?.imagePath ?? '',
    }
  })).toEqual({
    status: 'queued',
    imagePath: 'browser-preview/data/whiteboards/exports/whiteboard.png',
    sourceKind: 'whiteboard-selection',
    sourceId: 'browser-preview/data/whiteboards/board-current.excalidraw',
    language: 'zh-en',
    priority: 'interactive',
    elementId: 'selected-image-element',
    rawImagePath: 'browser-preview/data/whiteboards/exports/whiteboard.png',
  })
  await expect(page.locator('.whiteboard-status')).toContainText('OCR queued')
})

test('whiteboard OCR blocks map to the selected image and recognized text inserts as scene text', async ({page}) => {
  await openWhiteboardWithSettings(page, 'en', 'mountain-green', undefined, selectedImageWhiteboardScene())

  const selectedImageButton = page.getByRole('button', {name: 'Recognize selected image'})
  await expect(selectedImageButton).toBeEnabled()
  await selectedImageButton.click()
  await expect(page.locator('.whiteboard-status')).toContainText('OCR queued')

  await page.evaluate((eventDetail) => {
    window.dispatchEvent(new CustomEvent('rf-ocr-job', {detail: eventDetail}))
  }, readyWhiteboardSelectionOcrEvent())

  const toggleBlocksButton = page.getByRole('button', {name: 'Show / hide OCR blocks'})
  await expect(toggleBlocksButton).toBeEnabled()
  await toggleBlocksButton.click()

  await expect.poll(async () => readWhiteboardOcrSceneState(page)).toMatchObject({
    blockCount: 2,
    textCount: 0,
    firstBlock: {
      x: 32,
      y: 18,
      width: 128,
      height: 36,
      text: 'RecordingFreedom',
    },
  })

  const insertTextButton = page.getByRole('button', {name: 'Insert recognized text'})
  await expect(insertTextButton).toBeEnabled()
  await insertTextButton.click()

  await expect.poll(async () => readWhiteboardOcrSceneState(page)).toMatchObject({
    blockCount: 2,
    textCount: 1,
    insertedText: 'RecordingFreedom\n文字识别',
  })

  await toggleBlocksButton.click()
  await expect.poll(async () => readWhiteboardOcrSceneState(page)).toMatchObject({
    blockCount: 0,
    textCount: 1,
    insertedText: 'RecordingFreedom\n文字识别',
  })
})

test('whiteboard OCR blocks and recognized text persist after reopen', async ({page}) => {
  await openWhiteboardWithSettings(page, 'en', 'mountain-green', undefined, selectedImageWhiteboardScene())

  const selectedImageButton = page.getByRole('button', {name: 'Recognize selected image'})
  await expect(selectedImageButton).toBeEnabled()
  await selectedImageButton.click()
  await expect(page.locator('.whiteboard-status')).toContainText('OCR queued')
  await page.evaluate((eventDetail) => {
    window.dispatchEvent(new CustomEvent('rf-ocr-job', {detail: eventDetail}))
  }, readyWhiteboardSelectionOcrEvent())

  await page.getByRole('button', {name: 'Show / hide OCR blocks'}).click()
  await page.getByRole('button', {name: 'Insert recognized text'}).click()
  await expect.poll(async () => readWhiteboardOcrSceneState(page)).toMatchObject({
    blockCount: 2,
    textCount: 1,
    firstBlock: {
      x: 32,
      y: 18,
      width: 128,
      height: 36,
      text: 'RecordingFreedom',
    },
    insertedText: 'RecordingFreedom\n文字识别',
  })
  const persistedScene = await page.evaluate((sceneKey) => window.localStorage.getItem(sceneKey) || '', browserWhiteboardSceneKey)
  expect(persistedScene).toContain('recordingFreedomOcr')
  const context = page.context()
  await page.close()

  const reopenedPage = await context.newPage()
  await openWhiteboardWithSettings(reopenedPage, 'en', 'mountain-green', undefined, persistedScene)
  await expect(reopenedPage.locator('.whiteboard-shell')).toBeVisible()
  await expect.poll(async () => readWhiteboardOcrSceneState(reopenedPage)).toMatchObject({
    blockCount: 2,
    textCount: 1,
    firstBlock: {
      x: 32,
      y: 18,
      width: 128,
      height: 36,
      text: 'RecordingFreedom',
    },
    insertedText: 'RecordingFreedom\n文字识别',
  })
  await reopenedPage.close()
})

test('whiteboard selected image OCR shows positioned text on its exported image', async ({page}) => {
  await openWhiteboardWithSettings(page, 'en', 'mountain-green', undefined, selectedImageWhiteboardScene())
  const readyEvent = readyWhiteboardSelectionOcrEvent()
  const result = readyEvent.result
  await page.evaluate(({ocrResult}) => {
    ;(window as Window & {__RF_OCR_RESULTS__?: Record<string, unknown>}).__RF_OCR_RESULTS__ = {
      [ocrResult.id]: ocrResult,
    }
  }, {ocrResult: result})

  const selectedImageButton = page.getByRole('button', {name: 'Recognize selected image'})
  await expect(selectedImageButton).toBeEnabled()
  await selectedImageButton.click()
  await expect(page.locator('.whiteboard-status')).toContainText('OCR queued')
  await page.evaluate((eventDetail) => {
    window.dispatchEvent(new CustomEvent('rf-ocr-job', {detail: eventDetail}))
  }, readyEvent)

  await page.getByRole('button', {name: 'View board OCR result'}).click()
  const ocrLayer = page.locator('.whiteboard-canvas .ocr-position-text-layer.whiteboard')
  await expect(ocrLayer.locator('.ocr-position-text-button')).toHaveCount(2)
  const firstText = ocrLayer.locator('.ocr-position-text-button').filter({hasText: 'RecordingFreedom'}).first()
  await expect(firstText).toBeVisible()
  await firstText.click()
  await expect(ocrLayer.locator('.ocr-position-text-button').filter({hasText: 'Text copied'})).toHaveCount(1)
  await expect.poll(async () => readWhiteboardOcrSceneState(page)).toMatchObject({
    positionTextCount: 0,
  })

  await page.getByRole('button', {name: 'View board OCR result'}).click()
  await expect(ocrLayer.locator('.ocr-position-text-button')).toHaveCount(0)
})

test('whiteboard selected image OCR positions real worker smoke evidence text on selected image', async ({page}) => {
  await openWhiteboardWithSettings(page, 'en', 'mountain-green', undefined, selectedImageWhiteboardScene())
  const resultId = 'whiteboard-selection-smoke-result'
  const smokeResult = {
    id: resultId,
    sourceKind: 'whiteboard-selection',
    sourceId: 'browser-preview/data/whiteboards/board-current.excalidraw',
    imagePath: 'release-out/ocr-smoke-evidence/region.png',
    imageSha256: 'browser-smoke-evidence-whiteboard-selection',
    modelId: 'ppocrv5-mobile-zh-en',
    language: 'zh-en',
    width: 900,
    height: 280,
    plainText: 'RecordingFreedom\n文字识别',
    createdAt: '2026-07-05T20:37:46.0866316Z',
    durationMs: 126,
    blocks: [
      {
        id: 'b1',
        text: 'RecordingFreedom',
        confidence: 0.9995158016681671,
        lineIndex: 0,
        languageHint: 'en',
        box: [
          {x: 30.133928571428573, y: 32.083333333333336},
          {x: 669.9776785714286, y: 32.083333333333336},
          {x: 669.9776785714286, y: 109.86111111111113},
          {x: 30.133928571428573, y: 109.86111111111113},
        ],
      },
      {
        id: 'b2',
        text: '文字识别',
        confidence: 0.9980595409870148,
        lineIndex: 1,
        languageHint: 'zh',
        box: [
          {x: 41.34375, y: 137.08333333333334},
          {x: 330.3080357142857, y: 137.08333333333334},
          {x: 330.3080357142857, y: 234.30555555555557},
          {x: 41.34375, y: 234.30555555555557},
        ],
      },
    ],
  }
  const readyEvent = {
    jobId: 'whiteboard-selection-smoke-job',
    sourceKind: 'whiteboard-selection',
    sourceId: smokeResult.sourceId,
    status: 'ready',
    result: smokeResult,
  }
  await page.evaluate(({ocrResult}) => {
    ;(window as Window & {__RF_OCR_RESULTS__?: Record<string, unknown>}).__RF_OCR_RESULTS__ = {
      [ocrResult.id]: ocrResult,
    }
  }, {ocrResult: smokeResult})

  const selectedImageButton = page.getByRole('button', {name: 'Recognize selected image'})
  await expect(selectedImageButton).toBeEnabled()
  await selectedImageButton.click()
  await expect(page.locator('.whiteboard-status')).toContainText('OCR queued')
  await page.evaluate((eventDetail) => {
    window.dispatchEvent(new CustomEvent('rf-ocr-job', {detail: eventDetail}))
  }, readyEvent)

  await page.getByRole('button', {name: 'View board OCR result'}).click()
  const ocrLayer = page.locator('.whiteboard-canvas .ocr-position-text-layer.whiteboard')
  await expect(ocrLayer.locator('.ocr-position-text-button')).toHaveCount(2)
  await expect(ocrLayer.locator('.ocr-position-text-button').first()).toContainText('RecordingFreedom')
  await expect.poll(async () => readWhiteboardOcrSceneState(page)).toMatchObject({
    positionTextCount: 0,
  })
})

test('whiteboard translated OCR text inserts as scene text after explicit provider configuration', async ({page}) => {
  await openWhiteboardWithSettings(page, 'en', 'mountain-green', undefined, selectedImageWhiteboardScene(), true)

  const selectedImageButton = page.getByRole('button', {name: 'Recognize selected image'})
  await expect(selectedImageButton).toBeEnabled()
  await selectedImageButton.click()
  await expect(page.locator('.whiteboard-status')).toContainText('OCR queued')

  await page.evaluate((eventDetail) => {
    window.dispatchEvent(new CustomEvent('rf-ocr-job', {detail: eventDetail}))
  }, readyWhiteboardSelectionOcrEvent())

  const translateButton = page.getByRole('button', {name: 'Translate recognized text'})
  await expect(translateButton).toBeEnabled()
  await translateButton.click()

  const insertTranslationButton = page.getByRole('button', {name: 'Insert translated text'})
  await expect(insertTranslationButton).toBeEnabled()
  await expect.poll(async () => readWhiteboardOcrSceneState(page)).toMatchObject({
    blockCount: 0,
    textCount: 0,
    translationCount: 2,
    firstTranslation: {
      x: 32,
      y: 18,
      width: 128,
      height: 36,
      text: 'RecordingFreedom translated',
    },
    insertedText: '',
  })
})

test('whiteboard OCR failure during an audio recording does not stop the recorder shell', async ({page}) => {
  await openRecorderShellForAudioRecording(page)
  await page.locator('.source-pill').click()
  await page.getByRole('group', {name: 'Recording mode'}).getByRole('button', {name: 'Audio'}).click()
  await page.getByRole('button', {name: 'Start recording'}).click()

  await expect(page.getByRole('button', {name: 'Stop recording'})).toBeVisible()
  await expect(page.locator('.rf-shell')).toHaveClass(/is-recording-compact/)
  const whiteboardButton = page.getByRole('button', {name: 'Open whiteboard'})
  await expect(whiteboardButton).toBeEnabled()
  await whiteboardButton.click()
  await expect.poll(async () => page.evaluate(() => {
    const launch = (window as Window & {__RF_LAST_WHITEBOARD_LAUNCH__?: {mode?: string; url?: string}}).__RF_LAST_WHITEBOARD_LAUNCH__
    return {mode: launch?.mode ?? '', url: launch?.url ?? ''}
  })).toEqual({mode: 'whiteboard', url: '/#/whiteboard'})

  const boardPage = await page.context().newPage()
  await openWhiteboardWithSettings(boardPage, 'en', 'mountain-green', undefined, selectedImageWhiteboardScene())
  const selectedImageButton = boardPage.getByRole('button', {name: 'Recognize selected image'})
  await expect(selectedImageButton).toBeEnabled()
  await selectedImageButton.click()
  await expect(boardPage.locator('.whiteboard-status')).toContainText('OCR queued')

  await boardPage.evaluate((eventDetail) => {
    window.dispatchEvent(new CustomEvent('rf-ocr-job', {detail: eventDetail}))
  }, failedWhiteboardSelectionOcrEvent())

  await expect(boardPage.locator('.whiteboard-status')).toContainText('worker unavailable during recording')
  await expect(selectedImageButton).toBeEnabled()
  await expect(page.getByRole('button', {name: 'Stop recording'})).toBeVisible()
  await expect(page.locator('.rf-shell')).toHaveClass(/is-recording-compact/)

  await page.getByRole('button', {name: 'Stop recording'}).click()
  await expect(page.getByRole('button', {name: 'Start recording'})).toBeVisible()
  await boardPage.close()
})

async function openWhiteboardWithSettings(page: Page, locale: LocaleCode, theme: ThemeCode, screenshotContext?: unknown, sceneJson?: string, translationEnabled = false) {
  await page.addInitScript(({settingsKey, screenshotWhiteboardKey, whiteboardSceneKey, localeValue, themeValue, context, scene, enableTranslation}) => {
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
      ocr: {
        autoRecognizeScreenshots: false,
        translation: enableTranslation ? {
          provider: 'openai-compatible',
          baseUrl: 'https://translator.example/v1',
          apiKey: 'browser-test-key',
          model: 'rf-translator',
          sourceLanguage: 'auto',
          targetLanguage: 'en',
          privacyConfirmed: true,
          privacyConfirmedAt: '2026-07-06T10:00:00.000Z',
        } : {
          provider: 'disabled',
          sourceLanguage: 'auto',
          targetLanguage: 'zh-CN',
          privacyConfirmed: false,
        },
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
    if (scene) {
      window.localStorage.setItem(whiteboardSceneKey, scene)
    }
  }, {
    settingsKey: browserSettingsKey,
    screenshotWhiteboardKey: browserScreenshotWhiteboardKey,
    whiteboardSceneKey: browserWhiteboardSceneKey,
    localeValue: locale,
    themeValue: theme,
    context: screenshotContext,
    scene: sceneJson,
    enableTranslation: translationEnabled,
  })
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

function selectedImageWhiteboardScene() {
  return JSON.stringify({
    type: 'excalidraw',
    version: 2,
    source: 'recordingfreedom-test',
    elements: [{
      id: 'selected-image-element',
      type: 'image',
      x: 0,
      y: 0,
      width: 320,
      height: 180,
      angle: 0,
      strokeColor: 'transparent',
      backgroundColor: 'transparent',
      fillStyle: 'solid',
      strokeWidth: 1,
      strokeStyle: 'solid',
      roughness: 0,
      opacity: 100,
      groupIds: [],
      frameId: null,
      roundness: null,
      seed: 1,
      version: 1,
      versionNonce: 1,
      isDeleted: false,
      boundElements: null,
      updated: 1783250000000,
      link: null,
      locked: false,
      status: 'saved',
      fileId: 'selected-image-file',
      scale: [1, 1],
    }],
    appState: {
      viewBackgroundColor: '#111827',
      currentItemStrokeColor: '#ef4444',
      currentItemStrokeWidthKey: 'medium',
      currentItemOpacity: 100,
      scrollX: 80,
      scrollY: 80,
      selectedElementIds: {'selected-image-element': true},
    },
    files: {
      'selected-image-file': {
        id: 'selected-image-file',
        dataURL: 'data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=',
        mimeType: 'image/png',
        created: 1783250000000,
      },
    },
  })
}

function readyWhiteboardSelectionOcrEvent() {
  const sourceId = 'browser-preview/data/whiteboards/board-current.excalidraw'
  return {
    jobId: 'whiteboard-selection-job',
    sourceKind: 'whiteboard-selection',
    sourceId,
    status: 'ready',
    result: {
      id: 'whiteboard-selection-result',
      sourceKind: 'whiteboard-selection',
      sourceId,
      imagePath: 'browser-preview/data/whiteboards/exports/whiteboard.png',
      imageSha256: 'browser-e2e-selection-image',
      modelId: 'ppocrv5-mobile-zh-en',
      language: 'zh-en',
      width: 320,
      height: 180,
      blocks: [
        {
          id: 'block-recordingfreedom',
          text: 'RecordingFreedom',
          confidence: 0.97,
          lineIndex: 0,
          languageHint: 'en',
          box: [
            {x: 32, y: 18},
            {x: 160, y: 18},
            {x: 160, y: 54},
            {x: 32, y: 54},
          ],
        },
        {
          id: 'block-chinese',
          text: '文字识别',
          confidence: 0.94,
          lineIndex: 1,
          languageHint: 'zh',
          box: [
            {x: 32, y: 72},
            {x: 128, y: 72},
            {x: 128, y: 108},
            {x: 32, y: 108},
          ],
        },
      ],
      plainText: 'RecordingFreedom\n文字识别',
      createdAt: '2026-07-06T10:00:00.000Z',
      durationMs: 126,
    },
  }
}

function failedWhiteboardSelectionOcrEvent() {
  return {
    jobId: 'whiteboard-selection-failed-job',
    sourceKind: 'whiteboard-selection',
    sourceId: 'browser-preview/data/whiteboards/board-current.excalidraw',
    status: 'failed',
    error: 'worker unavailable during recording',
  }
}

async function openRecorderShellForAudioRecording(page: Page) {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
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
    settings: recorderAudioRecordingSettings(),
  })
  await page.goto('/')
}

function recorderAudioRecordingSettings() {
  return {
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
      system: true,
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
      theme: 'night-teal',
    },
  }
}

async function readWhiteboardOcrSceneState(page: Page) {
  return page.evaluate((sceneKey) => {
    const scene = JSON.parse(window.localStorage.getItem(sceneKey) || '{}')
    const elements = Array.isArray(scene.elements) ? scene.elements : []
    const ocrElements = elements.filter((element: {customData?: {recordingFreedomOcr?: {kind?: string}}}) => element.customData?.recordingFreedomOcr)
    const blockElements = ocrElements.filter((element: {customData?: {recordingFreedomOcr?: {kind?: string}}}) => element.customData?.recordingFreedomOcr?.kind === 'block')
    const textElements = ocrElements.filter((element: {customData?: {recordingFreedomOcr?: {kind?: string}}; text?: string}) => element.customData?.recordingFreedomOcr?.kind === 'text')
    const positionTextElements = ocrElements.filter((element: {customData?: {recordingFreedomOcr?: {kind?: string}}; text?: string}) => element.customData?.recordingFreedomOcr?.kind === 'position-text')
    const translationElements = ocrElements.filter((element: {customData?: {recordingFreedomOcr?: {kind?: string}}; text?: string}) => element.customData?.recordingFreedomOcr?.kind === 'translation')
    const firstBlock = blockElements[0] as {x?: number; y?: number; width?: number; height?: number; customData?: {recordingFreedomOcr?: {text?: string}}} | undefined
    const firstPositionText = positionTextElements[0] as {x?: number; y?: number; width?: number; height?: number; text?: string} | undefined
    const firstTranslation = translationElements[0] as {x?: number; y?: number; width?: number; height?: number; text?: string} | undefined
    return {
      blockCount: blockElements.length,
      textCount: textElements.length,
      positionTextCount: positionTextElements.length,
      translationCount: translationElements.length,
      firstBlock: firstBlock ? {
        x: Math.round(firstBlock.x ?? 0),
        y: Math.round(firstBlock.y ?? 0),
        width: Math.round(firstBlock.width ?? 0),
        height: Math.round(firstBlock.height ?? 0),
        text: firstBlock.customData?.recordingFreedomOcr?.text ?? '',
      } : null,
      firstPositionText: firstPositionText ? {
        x: Math.round(firstPositionText.x ?? 0),
        y: Math.round(firstPositionText.y ?? 0),
        width: Math.round(firstPositionText.width ?? 0),
        height: Math.round(firstPositionText.height ?? 0),
        text: firstPositionText.text ?? '',
      } : null,
      firstTranslation: firstTranslation ? {
        x: Math.round(firstTranslation.x ?? 0),
        y: Math.round(firstTranslation.y ?? 0),
        width: Math.round(firstTranslation.width ?? 0),
        height: Math.round(firstTranslation.height ?? 0),
        text: firstTranslation.text ?? '',
      } : null,
      insertedText: (textElements[0] as {text?: string} | undefined)?.text ?? '',
    }
  }, browserWhiteboardSceneKey)
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
