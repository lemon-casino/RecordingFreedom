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

test('screenshot annotation overlay saves into screenshot history on explicit save', async ({page}) => {
  await openScreenshotAnnotationOverlay(page)

  await expect(page.locator('.annotation-overlay-shell')).toBeVisible()
  await expect(page.locator('.annotation-capsule-title')).toHaveText('Region screenshot')
  await expect.poll(async () => page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) || '[]').length, browserScreenshotHistoryKey)).toBe(0)

  await page.getByRole('button', {name: 'Save'}).click()

  await expect.poll(async () => page.evaluate((key) => JSON.parse(window.localStorage.getItem(key) || '[]').length, browserScreenshotHistoryKey)).toBe(1)
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
  await expect(frame).toHaveCSS('border-top-color', 'rgba(255, 59, 59, 0.98)')
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
