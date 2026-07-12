import {expect, test, type Page} from '@playwright/test'

const browserSettingsKey = 'recordingfreedom.settings.v1'
const browserAnnotationSceneKey = 'recordingfreedom.annotation.scene.v1'
const browserScreenshotAnnotationKey = 'recordingfreedom.screenshots.annotation.v1'
const browserScreenshotHistoryKey = 'recordingfreedom.screenshots.history.v1'
const annotationFrameInset = 10

test('annotation overlay narrows hit regions in pass-through mode', async ({page}) => {
  await openAnnotationOverlay(page)

  const shell = page.locator('.annotation-overlay-shell')
  const canvas = page.locator('.annotation-overlay-canvas')
  await expect(shell).toBeVisible()
  await expect(shell).toHaveClass(/is-drawing/)
  await expect(canvas).toHaveCSS('pointer-events', 'auto')
  await expectAnnotationFrame(page)
  await expectAnnotationHitRegions(page, 'drawing')

  await page.locator('.annotation-tools').getByRole('button', {name: 'Select'}).click()
  await expect(shell).toHaveClass(/is-pass-through/)
  await expect(canvas).toHaveCSS('pointer-events', 'none')
  await expectAnnotationHitRegions(page, 'pass-through')

  await page.locator('.annotation-tools').getByRole('button', {name: 'Pen'}).click()
  await expect(shell).toHaveClass(/is-drawing/)
  await expect(canvas).toHaveCSS('pointer-events', 'auto')
  await expectAnnotationHitRegions(page, 'drawing')
})

test('annotation overlay exposes undo and region reselect controls', async ({page}) => {
  await openAnnotationOverlay(page)

  await expect(page.getByRole('button', {name: 'Undo'})).toBeVisible()
  await expect(page.getByRole('button', {name: 'Reselect board area'})).toBeVisible()
  await page.evaluate((sceneKey) => {
    window.localStorage.setItem(sceneKey, '{"type":"excalidraw","elements":[{"id":"old"}],"appState":{},"files":{}}')
  }, browserAnnotationSceneKey)

  await page.getByRole('button', {name: 'Reselect board area'}).click()

  await expect.poll(async () => page.evaluate((sceneKey) => {
    const session = (window as Window & {
      __RF_LAST_ANNOTATION_REGION_RESELECT__?: {purpose?: string}
    }).__RF_LAST_ANNOTATION_REGION_RESELECT__
    return {
      scene: window.localStorage.getItem(sceneKey),
      purpose: session?.purpose,
    }
  }, browserAnnotationSceneKey)).toEqual({
    scene: null,
    purpose: 'annotation',
  })
})

test('recording annotation overlay queues background OCR and shows positioned text on board', async ({page}) => {
  await openAnnotationOverlay(page)

  await page.getByRole('button', {name: 'Recognize board text'}).click()

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
        priority?: string
      }
    }
    return {
      status: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.status ?? '',
      imagePath: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.imagePath ?? '',
      sourceKind: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.sourceKind ?? '',
      sourceId: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.sourceId ?? '',
      language: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.language ?? '',
      priority: state.__RF_LAST_WHITEBOARD_OCR_QUEUE__?.request?.priority ?? '',
      rawSceneId: state.__RF_LAST_WHITEBOARD_OCR_REQUEST__?.sceneId ?? '',
      rawPriority: state.__RF_LAST_WHITEBOARD_OCR_REQUEST__?.priority ?? '',
    }
  })).toEqual({
    status: 'queued',
    imagePath: 'browser-preview/data/video/recording-preview.rfrec/annotations/snapshots/annotation-000001.png',
    sourceKind: 'whiteboard',
    sourceId: 'browser-preview/data/video/recording-preview.rfrec',
    language: 'zh-en',
    priority: 'background',
    rawSceneId: 'browser-preview/data/video/recording-preview.rfrec',
    rawPriority: 'background',
  })
  await expect(page.locator('.annotation-ocr-status')).toContainText('Board OCR queued')

  const readyEvent = readyAnnotationOcrEvent()
  const imageDataUrl = annotationOcrImageDataUrl()
  await page.evaluate(({eventDetail, image}) => {
    const result = eventDetail.result
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
    window.dispatchEvent(new CustomEvent('rf-ocr-job', {detail: eventDetail}))
  }, {eventDetail: readyEvent, image: imageDataUrl})

  await expect(page.locator('.annotation-ocr-status')).toContainText('Board text recognized')
  await page.getByRole('button', {name: 'View board OCR result'}).click()

  const ocrLayer = page.locator('.annotation-overlay-canvas .ocr-position-text-layer.annotation')
  await expect(ocrLayer.locator('.ocr-position-text-button')).toHaveCount(1)
  const firstText = ocrLayer.locator('.ocr-position-text-button').filter({hasText: 'RecordingFreedom'}).first()
  await expect(firstText).toBeVisible()
  await firstText.click()
  await expect(ocrLayer.locator('.ocr-position-text-button').filter({hasText: 'Text copied'})).toHaveCount(1)
  await expect.poll(async () => readAnnotationOcrSceneState(page)).toMatchObject({
    positionTextCount: 0,
  })

  await page.getByRole('button', {name: 'View board OCR result'}).click()
  await expect(ocrLayer.locator('.ocr-position-text-button')).toHaveCount(0)
})

test('screenshot annotation overlay saves into screenshot history on explicit save', async ({page}) => {
  await openScreenshotAnnotationOverlay(page)

  await expect(page.locator('.annotation-overlay-shell')).toBeVisible()
  await expect(page.locator('.annotation-capsule-title')).toHaveText('Region screenshot')
  await expect(page.locator('.annotation-save-status')).toContainText('Unsaved')
  await expect.poll(async () => page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) || '[]').length, browserScreenshotHistoryKey)).toBe(0)

  await page.getByRole('button', {name: 'Save'}).click()

  await expect.poll(async () => page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) || '[]').length, browserScreenshotHistoryKey)).toBe(1)
  await expect(page.locator('.annotation-save-status')).toContainText('Saved')
  await page.waitForTimeout(350)
  await expect(page.locator('.annotation-save-status')).toContainText('Saved')
})

test('small screenshot region keeps the complete toolbar outside the capture canvas', async ({page}) => {
  await openScreenshotAnnotationOverlay(page)
  await page.setViewportSize({width: 740, height: 214})
  await page.evaluate(() => {
    window.dispatchEvent(new CustomEvent('rf-annotation-overlay', {detail: {
      mode: 'screenshot',
      windowBounds: {x: 0, y: 0, width: 740, height: 214},
      canvasBounds: {x: 310, y: 114, width: 120, height: 80},
      toolbarBounds: {x: 10, y: 10, width: 720, height: 96},
      toolbarPlacement: 'top',
      target: {
        type: 'screenshot-region',
        id: 'screenshot-small-region-test',
        geometry: {x: 600, y: 320, width: 120, height: 80},
      },
      captureExcluded: false,
    }}))
  })

  const capsule = page.locator('.annotation-capsule')
  const canvas = page.locator('.annotation-overlay-canvas')
  await expect(capsule).toBeVisible()
  await expect(capsule.locator('button')).toHaveCount(14)
  await expect.poll(async () => {
    const [toolbarBox, canvasBox] = await Promise.all([capsule.boundingBox(), canvas.boundingBox()])
    const buttons = await capsule.locator('button').evaluateAll((elements) => elements.map((element) => {
      const box = element.getBoundingClientRect()
      return {left: box.left, right: box.right, top: box.top, bottom: box.bottom}
    }))
    if (!toolbarBox || !canvasBox || toolbarBox.bottom > canvasBox.top) return false
    return buttons.every((button) => button.left >= 0 && button.right <= 740 && button.top >= 0 && button.bottom <= 214)
  }).toBe(true)
})

async function openAnnotationOverlay(page: Page) {
  await page.setViewportSize({width: 1280 + annotationFrameInset * 2, height: 720 + annotationFrameInset * 2})
  await page.addInitScript(({settingsKey, frameInset}) => {
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
        lastMode: 'annotation',
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
    ;(window as Window & {__RF_ANNOTATION_OVERLAY__?: unknown}).__RF_ANNOTATION_OVERLAY__ = {
      packageDir: 'browser-preview/data/video/recording-preview.rfrec',
      manifestPath: 'browser-preview/data/video/recording-preview.rfrec/manifest.json',
      windowBounds: {x: -frameInset, y: -frameInset, width: 1280 + frameInset * 2, height: 720 + frameInset * 2},
      canvasBounds: {x: frameInset, y: frameInset, width: 1280, height: 720},
      target: {
        type: 'screen',
        id: 'screen:primary',
        geometry: {x: 0, y: 0, width: 1280, height: 720, displayIndex: 1, nativeId: 'display-1'},
      },
      captureExcluded: false,
    }
  }, {settingsKey: browserSettingsKey, frameInset: annotationFrameInset})
  await page.goto('/#/annotation-overlay')
}

async function openScreenshotAnnotationOverlay(page: Page) {
  await page.setViewportSize({width: 900 + annotationFrameInset * 2, height: 520 + annotationFrameInset * 2})
  await page.addInitScript(({settingsKey, screenshotAnnotationKey, frameInset}) => {
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
        lastMode: 'annotation',
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
    const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="900" height="520"><rect width="900" height="520" fill="#1f2937"/><text x="24" y="52" fill="#fff" font-family="Arial" font-size="24">Screenshot draft</text></svg>'
    window.localStorage.setItem(screenshotAnnotationKey, JSON.stringify({
      available: true,
      item: {
        id: 'screenshot-draft-test',
        path: 'browser-preview/data/screenshots/screenshot-draft-test.png',
        thumbnailPath: 'browser-preview/data/screenshots/thumbnails/screenshot-draft-test.png',
        createdAt: '2026-07-04T12:00:00Z',
        width: 900,
        height: 520,
        mode: 'region',
        region: {x: 0, y: 0, width: 900, height: 520},
        pinned: false,
        fixed: false,
      },
      dataUrl: `data:image/svg+xml;base64,${window.btoa(svg)}`,
    }))
    ;(window as Window & {__RF_ANNOTATION_OVERLAY__?: unknown}).__RF_ANNOTATION_OVERLAY__ = {
      mode: 'screenshot',
      windowBounds: {x: -frameInset, y: -frameInset, width: 900 + frameInset * 2, height: 520 + frameInset * 2},
      canvasBounds: {x: frameInset, y: frameInset, width: 900, height: 520},
      target: {
        type: 'screenshot-region',
        id: 'screenshot-draft-test',
        geometry: {x: 0, y: 0, width: 900, height: 520},
      },
      captureExcluded: false,
    }
  }, {settingsKey: browserSettingsKey, screenshotAnnotationKey: browserScreenshotAnnotationKey, frameInset: annotationFrameInset})
  await page.goto('/#/annotation-overlay')
}

async function readAnnotationOcrSceneState(page: Page) {
  return page.evaluate((sceneKey) => {
    const scene = JSON.parse(window.localStorage.getItem(sceneKey) || '{}')
    const elements = Array.isArray(scene.elements) ? scene.elements : []
    const positionTextElements = elements.filter((element: {customData?: {recordingFreedomOcr?: {kind?: string}}}) => {
      return element.customData?.recordingFreedomOcr?.kind === 'position-text'
    })
    const firstPositionText = positionTextElements[0] as {x?: number; y?: number; width?: number; height?: number; text?: string} | undefined
    return {
      positionTextCount: positionTextElements.length,
      firstPositionText: firstPositionText ? {
        x: Math.round(firstPositionText.x ?? 0),
        y: Math.round(firstPositionText.y ?? 0),
        width: Math.round(firstPositionText.width ?? 0),
        height: Math.round(firstPositionText.height ?? 0),
        text: firstPositionText.text ?? '',
      } : null,
    }
  }, browserAnnotationSceneKey)
}

function readyAnnotationOcrEvent() {
  return {
    jobId: 'annotation-ocr-job',
    sourceKind: 'whiteboard',
    sourceId: 'browser-preview/data/video/recording-preview.rfrec',
    status: 'ready',
    result: {
      id: 'annotation-ocr-result',
      sourceKind: 'whiteboard',
      sourceId: 'browser-preview/data/video/recording-preview.rfrec',
      imagePath: 'browser-preview/data/video/recording-preview.rfrec/annotations/snapshots/annotation-000001.png',
      imageSha256: 'browser-annotation-ocr',
      modelId: 'ppocrv5-mobile-zh-en',
      language: 'zh-en',
      width: 1280,
      height: 720,
      blocks: [
        {
          id: 'annotation-block-recordingfreedom',
          text: 'RecordingFreedom',
          confidence: 0.97,
          lineIndex: 0,
          languageHint: 'en',
          box: [
            {x: 120, y: 120},
            {x: 420, y: 120},
            {x: 420, y: 168},
            {x: 120, y: 168},
          ],
        },
      ],
      plainText: 'RecordingFreedom',
      createdAt: '2026-07-06T10:00:00.000Z',
      durationMs: 128,
    },
  }
}

function annotationOcrImageDataUrl() {
  const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="1280" height="720"><rect width="1280" height="720" fill="#101820"/><text x="120" y="160" fill="#fff" font-family="Arial" font-size="44">RecordingFreedom</text></svg>'
  return `data:image/svg+xml;base64,${Buffer.from(svg).toString('base64')}`
}

type HitRegionSnapshot = {
  kind: string
  x: number
  y: number
  width: number
  height: number
}

async function expectAnnotationHitRegions(page: Page, mode: 'drawing' | 'pass-through') {
  await expect.poll(async () => page.evaluate(() => {
    const request = (window as Window & {
      __RF_LAST_ANNOTATION_HIT_REGIONS__?: {
        enabled: boolean
        viewportWidth?: number
        viewportHeight?: number
        regions?: HitRegionSnapshot[]
      }
    }).__RF_LAST_ANNOTATION_HIT_REGIONS__
    const regions = request?.regions ?? []
    return {
      enabled: request?.enabled ?? false,
      viewportWidth: Math.round(request?.viewportWidth ?? 0),
      viewportHeight: Math.round(request?.viewportHeight ?? 0),
      regions: regions.map((region) => ({
        kind: region.kind,
        x: Math.round(region.x),
        y: Math.round(region.y),
        width: Math.round(region.width),
        height: Math.round(region.height),
      })),
    }
  })).toEqual(mode === 'drawing'
    ? {
      enabled: true,
      viewportWidth: 1280 + annotationFrameInset * 2,
      viewportHeight: 720 + annotationFrameInset * 2,
      regions: [
        expect.objectContaining({kind: 'pill'}),
        expect.objectContaining({kind: 'rect', x: annotationFrameInset, y: annotationFrameInset, width: 1280, height: 720}),
      ],
    }
    : {
      enabled: true,
      viewportWidth: 1280 + annotationFrameInset * 2,
      viewportHeight: 720 + annotationFrameInset * 2,
      regions: [
        expect.objectContaining({kind: 'pill'}),
      ],
    })

  const regions = await page.evaluate(() => (window as Window & {
    __RF_LAST_ANNOTATION_HIT_REGIONS__?: {regions?: HitRegionSnapshot[]}
  }).__RF_LAST_ANNOTATION_HIT_REGIONS__?.regions ?? [])
  const capsuleRegion = regions.find((region) => region.kind === 'pill')
  expect(capsuleRegion?.width).toBeLessThan(760)
  expect(capsuleRegion?.height).toBeLessThan(80)
  if (mode === 'pass-through') {
    expect(regions.some((region) => region.kind === 'rect')).toBe(false)
  }
}

async function expectAnnotationFrame(page: Page) {
  const frame = page.locator('.annotation-overlay-frame')
  const canvas = page.locator('.annotation-overlay-canvas')
  await expect(frame).toBeVisible()
  await expect(frame).toHaveCSS('pointer-events', 'none')
  await expect(frame).toHaveCSS('border-top-color', 'rgb(103, 232, 249)')
  await expect.poll(async () => {
    const [frameBox, canvasBox] = await Promise.all([frame.boundingBox(), canvas.boundingBox()])
    return {
      frame: frameBox && {
        x: Math.round(frameBox.x),
        y: Math.round(frameBox.y),
        width: Math.round(frameBox.width),
        height: Math.round(frameBox.height),
      },
      canvas: canvasBox && {
        x: Math.round(canvasBox.x),
        y: Math.round(canvasBox.y),
        width: Math.round(canvasBox.width),
        height: Math.round(canvasBox.height),
      },
    }
  }).toEqual({
    frame: {x: annotationFrameInset, y: annotationFrameInset, width: 1280, height: 720},
    canvas: {x: annotationFrameInset, y: annotationFrameInset, width: 1280, height: 720},
  })
}
