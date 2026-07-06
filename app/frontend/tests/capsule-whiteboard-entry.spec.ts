import {expect, test, type Page} from '@playwright/test'
import {Buffer} from 'node:buffer'

const browserSettingsKey = 'recordingfreedom.settings.v1'
const browserScreenshotHistoryKey = 'recordingfreedom.screenshots.history.v1'
const browserScreenshotPinStateKey = 'recordingfreedom.screenshots.pin.v1'

test('capsule whiteboard opens board before recording and annotation during video recording', async ({page}) => {
  await openRecorderShell(page)

  const toolsButton = page.getByRole('button', {name: 'Screenshot / board'})
  await expect(toolsButton).toBeVisible()
  await expect(toolsButton).not.toContainText('Tools')

  await toolsButton.click()
  await expect(page.getByRole('dialog', {name: 'board menu'})).toBeVisible()
  await expect(page.getByRole('button', {name: /Region screenshot/})).toBeVisible()
  await expect(page.getByRole('button', {name: /Window screenshot/})).toHaveCount(0)
  await expect(page.getByRole('button', {name: /Focused window/})).toHaveCount(0)
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

test('screenshot tools hide window and focused-window capture modes', async ({page}) => {
  await openRecorderShell(page)

  await page.getByRole('button', {name: 'Screenshot / board'}).click()
  await expect(page.getByRole('button', {name: /Region screenshot/})).toBeVisible()
  await expect(page.getByRole('button', {name: /Full screenshot/})).toBeVisible()
  await expect(page.getByRole('button', {name: /Scrolling screenshot/})).toBeVisible()
  await expect(page.getByRole('button', {name: /Window screenshot/})).toHaveCount(0)
  await expect(page.getByRole('button', {name: /Focused window/})).toHaveCount(0)
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

test('capsule shows an icon-only whiteboard entry before recording in Chinese', async ({page}) => {
  await openRecorderShell(page, {locale: 'zh-CN'})

  const toolsButton = page.getByRole('button', {name: '截图 / 画板'})
  await expect(toolsButton).toBeVisible()
  await expect(toolsButton).not.toContainText('工具')
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

test('screenshot history does not show stale pinned state before the user pins a screenshot', async ({page}) => {
  await openRecorderShell(page, {
    screenshotHistory: [{
      id: 'stale-pin-shot',
      path: 'browser-preview/data/screenshots/stale-pin-shot.png',
      thumbnailPath: 'browser-preview/data/screenshots/thumbnails/stale-pin-shot.png',
      createdAt: '2026-07-04T12:00:00Z',
      width: 420,
      height: 260,
      mode: 'region',
      pinned: true,
      fixed: false,
    }],
  })

  await page.getByRole('button', {name: 'Screenshot / board'}).click()
  await expect(page.getByText('Pinned')).toHaveCount(0)
})

test('screenshot history pin action opens a real pinned image window state', async ({page}) => {
  await openRecorderShell(page, {
    screenshotHistory: [{
      id: 'pin-shot',
      path: 'browser-preview/data/screenshots/pin-shot.png',
      thumbnailPath: 'browser-preview/data/screenshots/thumbnails/pin-shot.png',
      createdAt: '2026-07-04T12:00:00Z',
      width: 420,
      height: 260,
      mode: 'region',
      pinned: false,
      fixed: false,
    }],
  })

  await page.getByRole('button', {name: 'Screenshot / board'}).click()
  await page.getByRole('button', {name: 'Pin image'}).click()

  await expect.poll(async () => page.evaluate((key) => {
    const state = JSON.parse(window.localStorage.getItem(key) || '{}')
    return {
      visible: state.visible === true,
      itemId: state.item?.id ?? '',
      hasImage: typeof state.dataUrl === 'string' && state.dataUrl.startsWith('data:image/'),
    }
  }, browserScreenshotPinStateKey)).toEqual({
    visible: true,
    itemId: 'pin-shot',
    hasImage: true,
  })

  const pinPage = await page.context().newPage()
  await pinPage.goto('/#/screenshot-pin')
  await expect(pinPage.locator('.screenshot-pin-shell.empty')).toHaveCount(0)
  await expect(pinPage.locator('.screenshot-pin-shell img')).toHaveAttribute('src', /data:image\//)
  await pinPage.close()
})

test('floating OCR result panel shows screenshot image blocks and translation guard', async ({page}) => {
  const screenshotHistory = [{
    id: 'ocr-ready-shot',
    path: 'browser-preview/data/screenshots/ocr-ready-shot.png',
    thumbnailPath: 'browser-preview/data/screenshots/thumbnails/ocr-ready-shot.png',
    createdAt: '2026-07-04T12:00:00Z',
    width: 520,
    height: 320,
    mode: 'region',
    pinned: false,
    fixed: false,
    ocrStatus: 'ready',
    ocrResultId: 'ocr-result-ready-shot',
    ocrModelId: 'ppocrv5-mobile-zh-en',
    ocrLanguage: 'zh-en',
    ocrUpdatedAt: '2026-07-04T12:01:00Z',
  }]
  await page.addInitScript(({settingsKey, screenshotHistoryKey, settings, history}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    window.localStorage.setItem(screenshotHistoryKey, JSON.stringify(history))
    ;(window as Window & {__RF_FLOATING_PANEL__?: unknown}).__RF_FLOATING_PANEL__ = {
      visible: true,
      kind: 'ocr-result',
      anchor: {x: 120, y: 120, width: 44, height: 44},
      bounds: {x: 120, y: 174, width: 380, height: 420},
      dockSide: 'none',
      token: 9,
      direction: 'down',
      contextId: 'ocr-result-ready-shot',
    }
  }, {
    settingsKey: browserSettingsKey,
    screenshotHistoryKey: browserScreenshotHistoryKey,
    settings: baseBrowserSettings('en', false, false),
    history: screenshotHistory,
  })

  await page.goto('/#/floating-panel')

  const panel = page.locator('.floating-panel-shell.panel-ocr-result')
  await expect(panel).toBeVisible()
  await expect(panel.getByText('OCR result')).toBeVisible()
  await expect(panel.locator('.ocr-result-summary')).toContainText('ppocrv5-mobile-zh-en')
  await expect(panel.locator('.ocr-result-summary')).toContainText('2 text blocks')
  await expect(panel.locator('.ocr-result-text')).toContainText('RecordingFreedom')
  await expect(panel.locator('.ocr-result-text')).toContainText('文字识别')
  await expect(panel.locator('.ocr-preview-frame img')).toHaveAttribute('src', /^data:image\/svg\+xml;base64,/)
  await expect(panel.locator('.ocr-preview-frame polygon')).toHaveCount(2)
  await expect(panel.locator('.ocr-preview-frame .ocr-position-text-button')).toHaveCount(2)
  await expect(panel.locator('.ocr-preview-frame .ocr-position-text-button').first()).toContainText('RecordingFreedom')
  await expect(panel.locator('.ocr-block-row')).toHaveCount(2)

  const firstBlock = panel.locator('.ocr-block-row').filter({hasText: 'RecordingFreedom'})
  await firstBlock.hover()
  await expect(firstBlock).toHaveClass(/active/)
  await expect(panel.locator('.ocr-preview-frame polygon.active')).toHaveCount(1)
  await expect(panel.locator('.ocr-preview-frame .ocr-position-text-button.active')).toHaveCount(1)

  await panel.getByRole('button', {name: 'Translate text'}).click()
  await expect(panel.locator('.ocr-translation-note')).toContainText('Translation provider is not configured')
})

test('floating OCR result panel renders real worker smoke evidence coordinates', async ({page}) => {
  const resultId = 'ocr-smoke-region-result'
  const imageDataUrl = screenshotSvgDataUrl(900, 280, 'RecordingFreedom', '文字识别')
  const smokeResult = {
    id: resultId,
    sourceKind: 'region-screenshot',
    sourceId: 'screenshot-ocr-smoke-region',
    imagePath: 'release-out/ocr-smoke-evidence/region.png',
    imageSha256: 'browser-smoke-evidence-region',
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
  const screenshotHistory = [{
    id: smokeResult.sourceId,
    path: smokeResult.imagePath,
    thumbnailPath: 'release-out/ocr-smoke-evidence/region.png',
    createdAt: smokeResult.createdAt,
    width: smokeResult.width,
    height: smokeResult.height,
    mode: 'region',
    pinned: false,
    fixed: false,
    ocrStatus: 'ready',
    ocrResultId: resultId,
    ocrModelId: smokeResult.modelId,
    ocrLanguage: smokeResult.language,
    ocrUpdatedAt: smokeResult.createdAt,
  }]
  await page.addInitScript(({settingsKey, screenshotHistoryKey, settings, history, result, image}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    window.localStorage.setItem(screenshotHistoryKey, JSON.stringify(history))
    ;(window as Window & {__RF_OCR_RESULTS__?: Record<string, unknown>}).__RF_OCR_RESULTS__ = {
      [result.id]: result,
    }
    ;(window as Window & {__RF_OCR_IMAGES__?: Record<string, unknown>}).__RF_OCR_IMAGES__ = {
      [result.id]: {
        available: true,
        dataUrl: image,
        path: result.imagePath,
        bytes: image.length,
      },
    }
    ;(window as Window & {__RF_FLOATING_PANEL__?: unknown}).__RF_FLOATING_PANEL__ = {
      visible: true,
      kind: 'ocr-result',
      anchor: {x: 120, y: 120, width: 44, height: 44},
      bounds: {x: 120, y: 174, width: 380, height: 420},
      dockSide: 'none',
      token: 10,
      direction: 'down',
      contextId: result.id,
    }
  }, {
    settingsKey: browserSettingsKey,
    screenshotHistoryKey: browserScreenshotHistoryKey,
    settings: baseBrowserSettings('en', false, false),
    history: screenshotHistory,
    result: smokeResult,
    image: imageDataUrl,
  })

  await page.goto('/#/floating-panel')

  const panel = page.locator('.floating-panel-shell.panel-ocr-result')
  await expect(panel).toBeVisible()
  await expect(panel.locator('.ocr-result-summary')).toContainText('ppocrv5-mobile-zh-en')
  await expect(panel.locator('.ocr-result-text')).toContainText('RecordingFreedom')
  await expect(panel.locator('.ocr-result-text')).toContainText('文字识别')
  await expect(panel.locator('.ocr-preview-frame img')).toHaveAttribute('src', /^data:image\/svg\+xml;base64,/)
  await expect(panel.locator('.ocr-preview-frame svg')).toHaveAttribute('viewBox', '0 0 900 280')
  await expect(panel.locator('.ocr-preview-frame polygon')).toHaveCount(2)
  await expect(panel.locator('.ocr-preview-frame .ocr-position-text-button')).toHaveCount(2)
  await expect(panel.locator('.ocr-preview-frame .ocr-position-text-button').nth(1)).toContainText('文字识别')
  await expect(panel.locator('.ocr-preview-frame polygon').first()).toHaveAttribute(
    'points',
    '30.133928571428573,32.083333333333336 669.9776785714286,32.083333333333336 669.9776785714286,109.86111111111113 30.133928571428573,109.86111111111113',
  )
  await expect(panel.locator('.ocr-preview-frame polygon').nth(1)).toHaveAttribute(
    'points',
    '41.34375,137.08333333333334 330.3080357142857,137.08333333333334 330.3080357142857,234.30555555555557 41.34375,234.30555555555557',
  )
})

test('screenshot history ready item opens OCR result floating panel with real worker evidence', async ({page}) => {
  const resultId = 'ocr-history-smoke-region-result'
  const imageDataUrl = screenshotSvgDataUrl(900, 280, 'RecordingFreedom', '文字识别')
  const smokeResult = {
    id: resultId,
    sourceKind: 'region-screenshot',
    sourceId: 'screenshot-history-smoke-region',
    imagePath: 'release-out/ocr-smoke-evidence/region.png',
    imageSha256: 'browser-history-smoke-evidence-region',
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
  const screenshotHistory = [{
    id: smokeResult.sourceId,
    path: smokeResult.imagePath,
    thumbnailPath: 'release-out/ocr-smoke-evidence/region.png',
    createdAt: smokeResult.createdAt,
    width: smokeResult.width,
    height: smokeResult.height,
    mode: 'region',
    pinned: false,
    fixed: false,
    ocrStatus: 'ready',
    ocrResultId: resultId,
    ocrModelId: smokeResult.modelId,
    ocrLanguage: smokeResult.language,
    ocrUpdatedAt: smokeResult.createdAt,
  }]
  const setup = {
    settingsKey: browserSettingsKey,
    screenshotHistoryKey: browserScreenshotHistoryKey,
    settings: baseBrowserSettings('en', false, false),
    history: screenshotHistory,
    result: smokeResult,
    image: imageDataUrl,
  }
  await page.addInitScript(({settingsKey, screenshotHistoryKey, settings, history, result, image}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    window.localStorage.setItem(screenshotHistoryKey, JSON.stringify(history))
    ;(window as Window & {__RF_FORCE_FLOATING_PANEL_WINDOWS__?: boolean}).__RF_FORCE_FLOATING_PANEL_WINDOWS__ = true
    ;(window as Window & {__RF_OCR_RESULTS__?: Record<string, unknown>}).__RF_OCR_RESULTS__ = {
      [result.id]: result,
    }
    ;(window as Window & {__RF_OCR_IMAGES__?: Record<string, unknown>}).__RF_OCR_IMAGES__ = {
      [result.id]: {
        available: true,
        dataUrl: image,
        path: result.imagePath,
        bytes: image.length,
      },
    }
  }, setup)
  await page.goto('/')

  await page.getByRole('button', {name: 'Screenshot / board'}).click()
  const row = page.locator('.screenshot-history-row').filter({hasText: 'Recognized'})
  await expect(row).toContainText('Recognized')
  await row.getByRole('button', {name: 'View OCR result'}).click()

  await expect.poll(async () => page.evaluate(() => {
    const panel = (window as Window & {
      __RF_FLOATING_PANEL__?: {visible?: boolean; kind?: string; contextId?: string; bounds?: {width?: number; height?: number}}
    }).__RF_FLOATING_PANEL__
    return {
      visible: panel?.visible === true,
      kind: panel?.kind ?? '',
      contextId: panel?.contextId ?? '',
      width: panel?.bounds?.width ?? 0,
      height: panel?.bounds?.height ?? 0,
    }
  })).toEqual({
    visible: true,
    kind: 'ocr-result',
    contextId: resultId,
    width: 380,
    height: 420,
  })

  const floatingPage = await page.context().newPage()
  await floatingPage.addInitScript(({settingsKey, screenshotHistoryKey, settings, history, result, image, panel}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    window.localStorage.setItem(screenshotHistoryKey, JSON.stringify(history))
    ;(window as Window & {__RF_OCR_RESULTS__?: Record<string, unknown>}).__RF_OCR_RESULTS__ = {
      [result.id]: result,
    }
    ;(window as Window & {__RF_OCR_IMAGES__?: Record<string, unknown>}).__RF_OCR_IMAGES__ = {
      [result.id]: {
        available: true,
        dataUrl: image,
        path: result.imagePath,
        bytes: image.length,
      },
    }
    ;(window as Window & {__RF_FLOATING_PANEL__?: unknown}).__RF_FLOATING_PANEL__ = panel
  }, {...setup, panel: await page.evaluate(() => (window as Window & {__RF_FLOATING_PANEL__?: unknown}).__RF_FLOATING_PANEL__)})
  await floatingPage.goto('/#/floating-panel')
  const panel = floatingPage.locator('.floating-panel-shell.panel-ocr-result')
  await expect(panel).toBeVisible()
  await expect(panel.locator('.ocr-result-summary')).toContainText('ppocrv5-mobile-zh-en')
  await expect(panel.locator('.ocr-result-text')).toContainText('RecordingFreedom')
  await expect(panel.locator('.ocr-preview-frame svg')).toHaveAttribute('viewBox', '0 0 900 280')
  await expect(panel.locator('.ocr-preview-frame polygon')).toHaveCount(2)
  await floatingPage.close()
})

test('screenshot history translates ready OCR text through the configured provider', async ({page}) => {
  await openRecorderShell(page, {
    ocrTranslation: true,
    screenshotHistory: [{
      id: 'history-translate-shot',
      path: 'browser-preview/data/screenshots/history-translate-shot.png',
      thumbnailPath: 'browser-preview/data/screenshots/thumbnails/history-translate-shot.png',
      createdAt: '2026-07-04T12:00:00Z',
      width: 520,
      height: 320,
      mode: 'region',
      pinned: false,
      fixed: false,
      ocrStatus: 'ready',
      ocrResultId: 'ocr-result-history-translate-shot',
      ocrModelId: 'ppocrv5-mobile-zh-en',
      ocrLanguage: 'zh-en',
      ocrUpdatedAt: '2026-07-04T12:01:00Z',
    }],
  })

  await page.getByRole('button', {name: 'Screenshot / board'}).click()
  await page.getByRole('button', {name: 'Translate text'}).click()
  await expect(page.locator('.screenshot-history-header')).toContainText('Translation copied')
})

test('pinned screenshot window restores OCR highlight and result floating panel after resize', async ({page}) => {
  await openRecorderShell(page, {
    ocrTranslation: true,
    screenshotHistory: [{
      id: 'pin-ocr-ready-shot',
      path: 'browser-preview/data/screenshots/pin-ocr-ready-shot.png',
      thumbnailPath: 'browser-preview/data/screenshots/thumbnails/pin-ocr-ready-shot.png',
      createdAt: '2026-07-04T12:00:00Z',
      width: 640,
      height: 360,
      mode: 'region',
      pinned: false,
      fixed: false,
      ocrStatus: 'ready',
      ocrResultId: 'ocr-result-pin-ready-shot',
      ocrModelId: 'ppocrv5-mobile-zh-en',
      ocrLanguage: 'zh-en',
      ocrUpdatedAt: '2026-07-04T12:01:00Z',
    }],
  })

  await page.getByRole('button', {name: 'Screenshot / board'}).click()
  await page.getByRole('button', {name: 'Pin image'}).click()
  await expect.poll(async () => page.evaluate((key) => {
    const state = JSON.parse(window.localStorage.getItem(key) || '{}')
    return {
      visible: state.visible === true,
      itemId: state.item?.id ?? '',
      ocrStatus: state.item?.ocrStatus ?? '',
    }
  }, browserScreenshotPinStateKey)).toEqual({
    visible: true,
    itemId: 'pin-ocr-ready-shot',
    ocrStatus: 'ready',
  })

  const pinPage = await page.context().newPage()
  await pinPage.setViewportSize({width: 520, height: 340})
  await pinPage.goto('/#/screenshot-pin')
  const pinShell = pinPage.locator('.screenshot-pin-shell')
  await expect(pinShell).toBeVisible()
  await expect(pinPage.locator('.screenshot-pin-title')).toContainText('2 text blocks')
  await expect(pinPage.locator('.screenshot-pin-canvas img')).toHaveAttribute('src', /^data:image\//)

  await pinPage.getByRole('button', {name: 'View OCR result'}).nth(1).click()
  const overlay = pinPage.locator('.screenshot-pin-canvas svg')
  await expect(overlay).toBeVisible()
  await expect(overlay).toHaveAttribute('viewBox', '0 0 640 360')
  await expect(overlay.locator('polygon')).toHaveCount(2)
  await expect(pinPage.locator('.screenshot-pin-canvas .ocr-position-text-button')).toHaveCount(2)

  const firstPolygon = overlay.locator('polygon').first()
  const firstPositionText = pinPage.locator('.screenshot-pin-canvas .ocr-position-text-button').filter({hasText: 'RecordingFreedom'}).first()
  await firstPositionText.hover()
  await expect(firstPolygon).toHaveClass(/active/)
  await firstPositionText.click()
  await expect(pinPage.locator('.screenshot-pin-title')).toContainText('Text copied')

  await pinPage.getByRole('button', {name: 'Translate text'}).click()
  await expect(pinPage.locator('.screenshot-pin-title')).toContainText('Translation copied')

  const beforeResize = await pinPage.locator('.screenshot-pin-canvas').boundingBox()
  await pinPage.setViewportSize({width: 760, height: 460})
  const afterResize = await pinPage.locator('.screenshot-pin-canvas').boundingBox()
  expect(beforeResize).not.toBeNull()
  expect(afterResize).not.toBeNull()
  expect(afterResize!.width).toBeGreaterThan(beforeResize!.width)
  await expect(overlay).toHaveAttribute('viewBox', '0 0 640 360')
  await expect(overlay.locator('polygon')).toHaveCount(2)
  await overlay.locator('polygon').nth(1).hover()
  await expect(overlay.locator('polygon').nth(1)).toHaveClass(/active/)

  await pinPage.getByRole('button', {name: 'View OCR result'}).first().click()
  await expect.poll(async () => pinPage.evaluate(() => {
    const panel = (window as Window & {
      __RF_FLOATING_PANEL__?: {visible?: boolean; kind?: string; contextId?: string}
    }).__RF_FLOATING_PANEL__
    return {
      visible: panel?.visible === true,
      kind: panel?.kind ?? '',
      contextId: panel?.contextId ?? '',
    }
  })).toEqual({
    visible: true,
    kind: 'ocr-result',
    contextId: 'ocr-result-pin-ready-shot',
  })
  await pinPage.close()
})

test('pinned screenshot window renders real worker smoke evidence after resize', async ({page}) => {
  const resultId = 'ocr-result-pin-smoke-region'
  const imageDataUrl = screenshotSvgDataUrl(900, 280, 'RecordingFreedom', '文字识别')
  const smokeResult = {
    id: resultId,
    sourceKind: 'pinned-screenshot',
    sourceId: 'pin-smoke-ready-shot',
    imagePath: 'release-out/ocr-smoke-evidence/region.png',
    imageSha256: 'browser-pin-smoke-evidence-region',
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
  const screenshotHistory = [{
    id: smokeResult.sourceId,
    path: smokeResult.imagePath,
    thumbnailPath: 'release-out/ocr-smoke-evidence/region.png',
    createdAt: smokeResult.createdAt,
    width: smokeResult.width,
    height: smokeResult.height,
    mode: 'region',
    pinned: false,
    fixed: false,
    ocrStatus: 'ready',
    ocrResultId: resultId,
    ocrModelId: smokeResult.modelId,
    ocrLanguage: smokeResult.language,
    ocrUpdatedAt: smokeResult.createdAt,
  }]
  await page.addInitScript(({settingsKey, screenshotHistoryKey, settings, history, result, image}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    window.localStorage.setItem(screenshotHistoryKey, JSON.stringify(history))
    ;(window as Window & {__RF_OCR_RESULTS__?: Record<string, unknown>}).__RF_OCR_RESULTS__ = {
      [result.id]: result,
    }
    ;(window as Window & {__RF_OCR_IMAGES__?: Record<string, unknown>}).__RF_OCR_IMAGES__ = {
      [result.id]: {
        available: true,
        dataUrl: image,
        path: result.imagePath,
        bytes: image.length,
      },
    }
  }, {
    settingsKey: browserSettingsKey,
    screenshotHistoryKey: browserScreenshotHistoryKey,
    settings: baseBrowserSettings('en', false, false),
    history: screenshotHistory,
    result: smokeResult,
    image: imageDataUrl,
  })
  await page.goto('/')

  await page.getByRole('button', {name: 'Screenshot / board'}).click()
  await page.getByRole('button', {name: 'Pin image'}).click()
  await expect.poll(async () => page.evaluate((key) => {
    const state = JSON.parse(window.localStorage.getItem(key) || '{}')
    return {
      visible: state.visible === true,
      itemId: state.item?.id ?? '',
      ocrResultId: state.item?.ocrResultId ?? '',
    }
  }, browserScreenshotPinStateKey)).toEqual({
    visible: true,
    itemId: 'pin-smoke-ready-shot',
    ocrResultId: resultId,
  })

  const pinPage = await page.context().newPage()
  await pinPage.setViewportSize({width: 520, height: 340})
  await pinPage.addInitScript(({settingsKey, screenshotHistoryKey, settings, history, result, image}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    window.localStorage.setItem(screenshotHistoryKey, JSON.stringify(history))
    ;(window as Window & {__RF_OCR_RESULTS__?: Record<string, unknown>}).__RF_OCR_RESULTS__ = {
      [result.id]: result,
    }
    ;(window as Window & {__RF_OCR_IMAGES__?: Record<string, unknown>}).__RF_OCR_IMAGES__ = {
      [result.id]: {
        available: true,
        dataUrl: image,
        path: result.imagePath,
        bytes: image.length,
      },
    }
  }, {
    settingsKey: browserSettingsKey,
    screenshotHistoryKey: browserScreenshotHistoryKey,
    settings: baseBrowserSettings('en', false, false),
    history: screenshotHistory,
    result: smokeResult,
    image: imageDataUrl,
  })
  await pinPage.goto('/#/screenshot-pin')
  await expect(pinPage.locator('.screenshot-pin-title')).toContainText('2 text blocks')
  await pinPage.getByRole('button', {name: 'View OCR result'}).nth(1).click()

  const overlay = pinPage.locator('.screenshot-pin-canvas svg')
  await expect(overlay).toHaveAttribute('viewBox', '0 0 900 280')
  await expect(overlay.locator('polygon')).toHaveCount(2)
  await expect(pinPage.locator('.screenshot-pin-canvas .ocr-position-text-button')).toHaveCount(2)
  await expect(pinPage.locator('.screenshot-pin-canvas .ocr-position-text-button').first()).toContainText('RecordingFreedom')
  await expect(overlay.locator('polygon').first()).toHaveAttribute(
    'points',
    '30.133928571428573,32.083333333333336 669.9776785714286,32.083333333333336 669.9776785714286,109.86111111111113 30.133928571428573,109.86111111111113',
  )

  const beforeResize = await pinPage.locator('.screenshot-pin-canvas').boundingBox()
  await pinPage.setViewportSize({width: 760, height: 460})
  const afterResize = await pinPage.locator('.screenshot-pin-canvas').boundingBox()
  expect(beforeResize).not.toBeNull()
  expect(afterResize).not.toBeNull()
  expect(afterResize!.width).toBeGreaterThan(beforeResize!.width)
  await expect(overlay).toHaveAttribute('viewBox', '0 0 900 280')
  await expect(overlay.locator('polygon')).toHaveCount(2)
  await expect(pinPage.locator('.screenshot-pin-canvas .ocr-position-text-button')).toHaveCount(2)
  await pinPage.close()
})

test('region screenshot reuses the annotation overlay before saving', async ({page}) => {
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

  await expect.poll(async () => page.evaluate(() => {
    const launch = (window as Window & {
      __RF_LAST_WHITEBOARD_LAUNCH__?: {mode: string; url: string}
      __RF_ANNOTATION_OVERLAY__?: {mode?: string; target?: {type?: string}}
    }).__RF_LAST_WHITEBOARD_LAUNCH__
    const overlay = (window as Window & {__RF_ANNOTATION_OVERLAY__?: {mode?: string; target?: {type?: string}}}).__RF_ANNOTATION_OVERLAY__
    return {
      mode: launch?.mode ?? '',
      url: launch?.url ?? '',
      overlayMode: overlay?.mode ?? '',
      targetType: overlay?.target?.type ?? '',
    }
  })).toEqual({
    mode: 'screenshot',
    url: '/#/annotation-overlay',
    overlayMode: 'screenshot',
    targetType: 'screenshot-region',
  })
  await expect.poll(async () => page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) || '[]').length, browserScreenshotHistoryKey)).toBe(0)
})

test('region screenshot recognizes element hover candidates and cycles parent levels', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'screenshot-region-hover-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'screenshot',
      candidates: [
        {
          id: 'screen:test-primary',
          kind: 'screen',
          label: 'Screen',
          bounds: {x: 0, y: 0, width: 900, height: 620},
          score: 0.99,
        },
        {
          id: 'window:test-app',
          kind: 'window',
          label: 'App window',
          bounds: {x: 120, y: 80, width: 680, height: 470},
          score: 0.98,
        },
        {
          id: 'element:test-sidebar-item',
          kind: 'element',
          label: 'Sidebar item',
          bounds: {x: 210, y: 140, width: 130, height: 84},
          score: 0.95,
        },
        {
          id: 'element:test-sidebar-pane',
          kind: 'element',
          label: 'Sidebar pane',
          bounds: {x: 180, y: 110, width: 280, height: 360},
          score: 0.9,
        },
        {
          id: 'edge:test-sidebar-panel',
          kind: 'edge',
          label: 'Smart panel',
          bounds: {x: 170, y: 100, width: 310, height: 390},
          score: 0.86,
        },
      ],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(260, 180)
  await expect(page.locator('.region-smart-candidate.element')).toBeVisible()
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Sidebar item')
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: {source?: string; best?: {id?: string}}}).__RF_LAST_REGION_ASSIST__
  ))).toMatchObject({
    source: 'element',
    best: {id: 'element:test-sidebar-item'},
  })

  await page.mouse.wheel(0, -120)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Sidebar pane')
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: {source?: string; best?: {id?: string}}}).__RF_LAST_REGION_ASSIST__
  ))).toMatchObject({
    source: 'element',
    best: {id: 'element:test-sidebar-pane'},
  })

  await page.mouse.wheel(0, -120)
  await expect(page.locator('.region-smart-candidate.edge')).toContainText('Smart panel')
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: {source?: string; best?: {id?: string}}}).__RF_LAST_REGION_ASSIST__
  ))).toMatchObject({
    source: 'image-hover',
    best: {id: 'edge:test-sidebar-panel'},
  })

  await page.mouse.click(260, 180)

  await expect.poll(async () => page.evaluate(() => {
    const overlay = (window as Window & {
      __RF_ANNOTATION_OVERLAY__?: {mode?: string; target?: {type?: string; geometry?: {x: number; y: number; width: number; height: number}}}
    }).__RF_ANNOTATION_OVERLAY__
    return {
      mode: overlay?.mode ?? '',
      targetType: overlay?.target?.type ?? '',
      geometry: overlay?.target?.geometry ?? null,
    }
  })).toEqual({
    mode: 'screenshot',
    targetType: 'screenshot-region',
    geometry: {x: 170, y: 100, width: 310, height: 390},
  })
})

test('region overlay recognizes the initial pointer before the first mouse move', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-initial-pointer-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      initialPointer: {x: 260, y: 180},
      candidates: [{
        id: 'element:initial-pointer',
        kind: 'element',
        label: 'Initial pointer target',
        bounds: {x: 210, y: 140, width: 180, height: 120},
        score: 0.95,
      }],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await expect(page.locator('.region-smart-candidate.element')).toContainText('Initial pointer target')
  await expect(page.locator('.region-crosshair')).toHaveCount(0)
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: {source?: string; best?: {id?: string}}}).__RF_LAST_REGION_ASSIST__
  ))).toMatchObject({
    source: 'element',
    best: {id: 'element:initial-pointer'},
  })
})

test('region overlay does not keep an element candidate on its right or bottom edge', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-half-open-edge-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [{
        id: 'element:half-open',
        kind: 'element',
        label: 'Half-open target',
        bounds: {x: 210, y: 140, width: 180, height: 120},
        score: 0.95,
      }],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(389, 259)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Half-open target')

  await page.mouse.move(390, 259)
  await expect(page.locator('.region-smart-candidate.element')).toHaveCount(0)
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: {best?: unknown}}).__RF_LAST_REGION_ASSIST__?.best ?? null
  ))).toBeNull()

  await page.mouse.move(389, 260)
  await expect(page.locator('.region-smart-candidate.element')).toHaveCount(0)
})

test('region overlay keeps auto recognition during small pointer jitter', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-auto-jitter-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [{
        id: 'element:auto-jitter',
        kind: 'element',
        label: 'Auto jitter target',
        bounds: {x: 210, y: 140, width: 180, height: 120},
        score: 0.95,
      }],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(250, 180)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Auto jitter target')
  await page.mouse.down({button: 'left'})
  await page.mouse.move(254, 183)
  await expect(page.locator('.region-selection-rect')).toHaveCount(0)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Auto jitter target')
  await page.mouse.up({button: 'left'})
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_SELECTION__?: {geometry?: {x: number; y: number; width: number; height: number}}}).__RF_LAST_REGION_SELECTION__?.geometry
  ))).toEqual({x: 210, y: 140, width: 180, height: 120})
})

test('region overlay falls back to the topmost window candidate when no element is available', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-window-fallback-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [
        {
          id: 'window:front',
          kind: 'window',
          label: 'Front window',
          bounds: {x: 180, y: 120, width: 340, height: 240},
          score: 0.82,
        },
        {
          id: 'window:back',
          kind: 'window',
          label: 'Back window',
          bounds: {x: 140, y: 90, width: 460, height: 330},
          score: 0.72,
        },
      ],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(240, 180)
  await expect(page.locator('.region-smart-candidate.window')).toContainText('Front window')
  await expect(page.locator('.region-crosshair')).toHaveCount(0)
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: {source?: string; best?: {id?: string}}}).__RF_LAST_REGION_ASSIST__
  ))).toMatchObject({
    source: 'static',
    best: {id: 'window:front'},
  })

  await page.mouse.click(240, 180)
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_SELECTION__?: {geometry?: {x: number; y: number; width: number; height: number}}}).__RF_LAST_REGION_SELECTION__?.geometry ?? null
  ))).toEqual({x: 180, y: 120, width: 340, height: 240})
})

test('region smart candidate keeps its center transparent and dims only outside the target', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-candidate-visual-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [{
        id: 'element:visual-target',
        kind: 'element',
        label: 'Visual target',
        bounds: {x: 210, y: 140, width: 180, height: 120},
        score: 0.95,
      }],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(260, 180)
  const candidate = page.locator('.region-smart-candidate.element')
  await expect(candidate).toContainText('Visual target')
  await expect(page.locator('.region-crosshair')).toHaveCount(0)
  await expect(candidate).toHaveCSS('background-color', 'rgba(0, 0, 0, 0)')
  await expect.poll(async () => candidate.evaluate((element) => getComputedStyle(element).boxShadow)).toContain('9999px')
  await expect.poll(async () => candidate.evaluate((element) => {
    const box = element.getBoundingClientRect()
    return {
      left: Math.round(box.left),
      top: Math.round(box.top),
      width: Math.round(box.width),
      height: Math.round(box.height),
    }
  })).toEqual({left: 210, top: 140, width: 180, height: 120})
})

test('region overlay right click returns from manual selection to auto detection, then cancels', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-right-click-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [{
        id: 'element:test-auto',
        kind: 'element',
        label: 'Auto target',
        bounds: {x: 210, y: 140, width: 180, height: 120},
        score: 0.95,
      }],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(260, 180)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Auto target')
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: {source?: string}}).__RF_LAST_REGION_ASSIST__?.source
  ))).toBe('element')

  await page.mouse.move(120, 120)
  await page.mouse.down()
  await page.mouse.move(420, 300)
  await expect(page.locator('.region-selection-rect')).toBeVisible()
  await page.locator('.region-overlay-shell').dispatchEvent('contextmenu', {
    clientX: 260,
    clientY: 180,
    button: 2,
    bubbles: true,
    cancelable: true,
  })

  await expect(page.locator('.region-selection-rect')).toHaveCount(0)
  await page.mouse.up()
  await page.mouse.move(260, 180)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Auto target')

  await page.locator('.region-overlay-shell').dispatchEvent('contextmenu', {
    clientX: 260,
    clientY: 180,
    button: 2,
    bubbles: true,
    cancelable: true,
  })
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_CANCEL__?: {cancelled?: boolean}}).__RF_LAST_REGION_CANCEL__?.cancelled === true
  ))).toBe(true)
})

test('region overlay native right pointer cancels an active manual drag before the left button is released', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-right-pointer-drag-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [{
        id: 'element:right-pointer-auto',
        kind: 'element',
        label: 'Right pointer target',
        bounds: {x: 210, y: 140, width: 180, height: 120},
        score: 0.95,
      }],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(120, 120)
  await page.mouse.down({button: 'left'})
  await page.mouse.move(420, 300)
  await expect(page.locator('.region-selection-rect')).toBeVisible()

  await page.mouse.move(260, 180)
  await page.mouse.down({button: 'right'})
  await expect(page.locator('.region-selection-rect')).toHaveCount(0)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Right pointer target')
  await page.mouse.up({button: 'right'})
  await page.mouse.up({button: 'left'})

  await page.mouse.click(260, 180, {button: 'right'})
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_CANCEL__?: {cancelled?: boolean}}).__RF_LAST_REGION_CANCEL__?.cancelled === true
  ))).toBe(true)
})

test('region overlay right click returns from selected edit frame to auto detection before cancelling', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    const session = {
      id: 'region-edit-right-click-test',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [{
        id: 'element:edit-auto',
        kind: 'element',
        label: 'Edit auto target',
        bounds: {x: 210, y: 140, width: 180, height: 120},
        score: 0.95,
      }],
    }
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = session
    ;(window as Window & {__RF_REGION_FRAME__?: unknown}).__RF_REGION_FRAME__ = {
      bounds: {x: 180, y: 110, width: 280, height: 220},
      overlayBounds: {x: 0, y: 0, width: 900, height: 620},
      mode: 'edit',
      purpose: 'capture',
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await expect(page.locator('.region-edit-rect')).toBeVisible()
  await page.mouse.click(260, 180, {button: 'right'})

  await expect(page.locator('.region-edit-rect')).toHaveCount(0)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Edit auto target')
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: {source?: string; best?: {id?: string}}}).__RF_LAST_REGION_ASSIST__
  ))).toMatchObject({
    source: 'element',
    best: {id: 'element:edit-auto'},
  })

  await page.locator('.region-overlay-shell').dispatchEvent('contextmenu', {
    clientX: 260,
    clientY: 180,
    button: 2,
    bubbles: true,
    cancelable: true,
  })
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_CANCEL__?: {cancelled?: boolean}}).__RF_LAST_REGION_CANCEL__?.cancelled === true
  ))).toBe(true)
})

test('region overlay uses the same selected-edit right click rule for every selector purpose', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  for (const purpose of ['capture', 'screenshot', 'scrolling-screenshot', 'annotation'] as const) {
    const label = `Edit ${purpose} target`
    await page.evaluate(({nextPurpose, nextLabel}) => {
      const session = {
        id: `region-edit-purpose-${nextPurpose}`,
        bounds: {x: 0, y: 0, width: 900, height: 620},
        captureBounds: {x: 0, y: 0, width: 900, height: 620},
        minimumWidth: nextPurpose === 'screenshot' ? 12 : 64,
        minimumHeight: nextPurpose === 'screenshot' ? 12 : 64,
        displayCount: 1,
        purpose: nextPurpose,
        candidates: [{
          id: `element:edit-${nextPurpose}`,
          kind: 'element',
          label: nextLabel,
          bounds: {x: 210, y: 140, width: 180, height: 120},
          score: 0.95,
        }],
      }
      const frame = {
        bounds: {x: 180, y: 110, width: 280, height: 220},
        overlayBounds: {x: 0, y: 0, width: 900, height: 620},
        mode: 'edit',
        purpose: nextPurpose,
      }
      delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
      delete (window as Window & {__RF_LAST_REGION_CANCEL__?: unknown}).__RF_LAST_REGION_CANCEL__
      ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = session
      ;(window as Window & {__RF_REGION_FRAME__?: unknown}).__RF_REGION_FRAME__ = frame
      window.dispatchEvent(new CustomEvent('rf-region-session', {detail: session}))
      window.dispatchEvent(new CustomEvent('rf-region-frame', {detail: frame}))
    }, {nextPurpose: purpose, nextLabel: label})

    await expect(page.locator('.region-edit-rect')).toBeVisible()
    await page.mouse.click(260, 180, {button: 'right'})
    await expect(page.locator('.region-edit-rect')).toHaveCount(0)
    await expect(page.locator('.region-smart-candidate.element')).toContainText(label)
    await expect.poll(async () => page.evaluate(() => {
      const state = (window as Window & {
        __RF_REGION_SESSION__?: {purpose?: string}
        __RF_LAST_REGION_ASSIST__?: {source?: string; best?: {id?: string}}
      })
      return {
        purpose: state.__RF_REGION_SESSION__?.purpose,
        source: state.__RF_LAST_REGION_ASSIST__?.source,
        best: state.__RF_LAST_REGION_ASSIST__?.best?.id,
      }
    })).toEqual({
      purpose,
      source: 'element',
      best: `element:edit-${purpose}`,
    })

    await page.mouse.click(260, 180, {button: 'right'})
    await expect.poll(async () => page.evaluate(() => (
      (window as Window & {__RF_LAST_REGION_CANCEL__?: {cancelled?: boolean}}).__RF_LAST_REGION_CANCEL__?.cancelled === true
    ))).toBe(true)
  }
})

test('dynamic region recognition is shared by capture screenshot scrolling and annotation selectors', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  for (const purpose of ['capture', 'screenshot', 'scrolling-screenshot', 'annotation'] as const) {
    await page.evaluate((nextPurpose) => {
      const session = {
        id: `region-purpose-${nextPurpose}`,
        bounds: {x: 0, y: 0, width: 900, height: 620},
        captureBounds: {x: 0, y: 0, width: 900, height: 620},
        minimumWidth: 64,
        minimumHeight: 64,
        displayCount: 1,
        purpose: nextPurpose,
        candidates: [{
          id: `element:${nextPurpose}`,
          kind: 'element',
          label: `Element ${nextPurpose}`,
          bounds: {x: 200, y: 130, width: 220, height: 150},
          score: 0.95,
        }],
      }
      ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = session
      window.dispatchEvent(new CustomEvent('rf-region-session', {detail: session}))
    }, purpose)
    await page.mouse.move(250, 180)
    await expect(page.locator('.region-smart-candidate.element')).toContainText(`Element ${purpose}`)
    await expect.poll(async () => page.evaluate(() => (
      (window as Window & {__RF_LAST_REGION_ASSIST__?: {source?: string; best?: {id?: string}}}).__RF_LAST_REGION_ASSIST__
    ))).toMatchObject({
      source: 'element',
      best: {id: `element:${purpose}`},
    })
    await page.mouse.move(252, 182)
  }
})

test('region overlay waits for hover recognition before showing manual crosshair', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-hover-pending',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(440, 280)
  await expect(page.locator('.region-crosshair')).toHaveCount(0)
  await page.waitForTimeout(110)
  await expect(page.locator('.region-crosshair')).toHaveCount(2)
})

test('region overlay right click cancels while hover recognition is pending', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-pending-right-click',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(440, 280)
  await expect(page.locator('.region-crosshair')).toHaveCount(0)
  await page.mouse.click(440, 280, {button: 'right'})
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_CANCEL__?: {cancelled?: boolean}}).__RF_LAST_REGION_CANCEL__?.cancelled === true
  ))).toBe(true)
  await expect.poll(async () => page.evaluate(() => {
    const state = window as Window & {
      __RF_REGION_SESSION__?: unknown
      __RF_LAST_REGION_ASSIST__?: unknown
    }
    return {
      session: state.__RF_REGION_SESSION__ ?? null,
      assist: state.__RF_LAST_REGION_ASSIST__ ?? null,
    }
  })).toEqual({session: null, assist: null})
  await page.waitForTimeout(120)
  await expect(page.locator('.region-crosshair')).toHaveCount(0)
  await expect(page.locator('.region-smart-candidate')).toHaveCount(0)
})

test('region overlay drops stale hover candidates when the session changes or pointer leaves', async ({page}) => {
  await page.addInitScript(({settingsKey, settings}) => {
    window.localStorage.setItem(settingsKey, JSON.stringify(settings))
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = {
      id: 'region-stale-hover-old',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [{
        id: 'element:old-session',
        kind: 'element',
        label: 'Old session',
        bounds: {x: 210, y: 140, width: 180, height: 120},
        score: 0.95,
      }],
    }
  }, {settingsKey: browserSettingsKey, settings: baseBrowserSettings('en', false, false)})
  await page.goto('/#/region-overlay')

  await page.mouse.move(260, 180)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('Old session')

  await page.evaluate(() => {
    const session = {
      id: 'region-stale-hover-new',
      bounds: {x: 0, y: 0, width: 900, height: 620},
      captureBounds: {x: 0, y: 0, width: 900, height: 620},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'capture',
      candidates: [{
        id: 'element:new-session',
        kind: 'element',
        label: 'New session',
        bounds: {x: 500, y: 210, width: 180, height: 120},
        score: 0.95,
      }],
    }
    ;(window as Window & {__RF_REGION_SESSION__?: unknown}).__RF_REGION_SESSION__ = session
    window.dispatchEvent(new CustomEvent('rf-region-session', {detail: session}))
  })
  await page.waitForTimeout(120)
  await expect(page.locator('.region-smart-candidate')).toHaveCount(0)
  await expect(page.locator('.region-crosshair')).toHaveCount(0)
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__ ?? null
  ))).toBeNull()

  await page.mouse.move(540, 250)
  await expect(page.locator('.region-smart-candidate.element')).toContainText('New session')
  await expect(page.locator('.region-crosshair')).toHaveCount(0)
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: {best?: {id?: string}}}).__RF_LAST_REGION_ASSIST__?.best?.id
  ))).toBe('element:new-session')

  await page.evaluate(() => {
    const shell = document.querySelector('.region-overlay-shell')
    shell?.dispatchEvent(new PointerEvent('pointerout', {
      clientX: 900,
      clientY: 620,
      bubbles: true,
      cancelable: true,
      relatedTarget: document.body,
    }))
  })
  await expect(page.locator('.region-smart-candidate')).toHaveCount(0)
  await expect(page.locator('.region-crosshair')).toHaveCount(0)
  await expect.poll(async () => page.evaluate(() => (
    (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__ ?? null
  ))).toBeNull()
})

async function openRecorderShell(page: Page, options: {locale?: 'zh-CN' | 'en'; microphone?: boolean; systemAudio?: boolean; screenshotHistory?: unknown[]; ocrTranslation?: boolean} = {}) {
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
    settings: browserSettingsWithOcrTranslation(baseBrowserSettings(
      options.locale ?? 'en',
      options.microphone === true,
      options.systemAudio === true,
    ), options.ocrTranslation === true),
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

function browserSettingsWithOcrTranslation(settings: Record<string, any>, enabled: boolean) {
  return {
    ...settings,
    ocr: {
      autoRecognizeScreenshots: false,
      translation: enabled ? {
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
  }
}

function screenshotSvgDataUrl(width: number, height: number, title: string, subtitle: string) {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}"><rect width="100%" height="100%" fill="#ffffff"/><text x="30" y="92" fill="#111827" font-family="Arial, sans-serif" font-size="68" font-weight="700">${title}</text><text x="41" y="210" fill="#111827" font-family="Arial, sans-serif" font-size="78" font-weight="700">${subtitle}</text></svg>`
  return `data:image/svg+xml;base64,${Buffer.from(svg, 'utf-8').toString('base64')}`
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
