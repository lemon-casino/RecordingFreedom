import {expect, test, type Page} from '@playwright/test'

const browserSettingsKey = 'recordingfreedom.settings.v1'
const browserAnnotationSceneKey = 'recordingfreedom.annotation.scene.v1'

test('annotation overlay narrows hit regions in pass-through mode', async ({page}) => {
  await openAnnotationOverlay(page)

  const shell = page.locator('.annotation-overlay-shell')
  const canvas = page.locator('.annotation-overlay-canvas')
  await expect(shell).toBeVisible()
  await expect(shell).toHaveClass(/is-drawing/)
  await expect(canvas).toHaveCSS('pointer-events', 'auto')
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

async function openAnnotationOverlay(page: Page) {
  await page.addInitScript(({settingsKey}) => {
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
      windowBounds: {x: 0, y: 0, width: 1280, height: 720},
      canvasBounds: {x: 0, y: 0, width: 1280, height: 720},
      target: {
        type: 'screen',
        id: 'screen:primary',
        geometry: {x: 0, y: 0, width: 1280, height: 720, displayIndex: 1, nativeId: 'display-1'},
      },
      captureExcluded: false,
    }
  }, {settingsKey: browserSettingsKey})
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
      viewportWidth: 1280,
      viewportHeight: 720,
      regions: [
        expect.objectContaining({kind: 'pill'}),
        expect.objectContaining({kind: 'rect', x: 0, y: 0, width: 1280, height: 720}),
      ],
    }
    : {
      enabled: true,
      viewportWidth: 1280,
      viewportHeight: 720,
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
