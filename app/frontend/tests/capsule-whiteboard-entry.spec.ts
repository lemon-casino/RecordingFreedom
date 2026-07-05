import {expect, test, type Page} from '@playwright/test'

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
