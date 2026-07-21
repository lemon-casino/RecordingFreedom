import {expect, test, type Page} from '@playwright/test'

test('capsule docking only snaps near external monitor edges', async ({page}) => {
  await page.goto('/')

  const workAreas = [
    {x: 0, y: 0, width: 1920, height: 1040},
    {x: 1920, y: 0, width: 1600, height: 900},
  ]
  const size = {width: 820, height: 96}

  await expect(dockSide(page, {position: {x: 4, y: 240}, size, workAreas})).resolves.toBe('left')
  await expect(dockSide(page, {position: {x: 80, y: 240}, size, workAreas})).resolves.toBe('none')
  await expect(dockSide(page, {position: {x: 1920 - size.width + 4, y: 240}, size, workAreas})).resolves.toBe('right')
  await expect(dockSide(page, {position: {x: 1920 - 20, y: 240}, size, workAreas})).resolves.toBe('left')
  await expect(dockSide(page, {position: {x: 1920 + 4, y: 240}, size, workAreas})).resolves.toBe('left')
  await expect(dockSide(page, {position: {x: 1920 + 1600 - size.width - 4, y: 240}, size, workAreas})).resolves.toBe('right')
})

test('capsule docking follows the drop rectangle when returning from a secondary screen', async ({page}) => {
  await page.goto('/')

  const workAreas = [
    {id: 'primary', x: 0, y: 0, width: 5000, height: 1040},
    {id: 'secondary', x: 5000, y: 0, width: 1000, height: 900},
  ]
  const target = await dockTarget(page, {
    position: {x: 5000 - 820 + 4, y: 240},
    size: {width: 820, height: 96},
    workAreas,
    activeScreenId: 'secondary',
  })

  expect(target.side).toBe('right')
  expect(target.workArea).toMatchObject({id: 'primary', x: 0, width: 5000})
})

test('capsule docking keeps a single-screen right edge inside the work area', async ({page}) => {
  await page.goto('/')

  const target = await dockTarget(page, {
    position: {x: 1920 - 820 + 4, y: 240},
    size: {width: 820, height: 96},
    workAreas: [{id: 'primary', x: 0, y: 0, width: 1920, height: 1040}],
    activeScreenId: 'primary',
  })

  expect(target.side).toBe('right')
  expect(target.workArea).toMatchObject({id: 'primary', x: 0, width: 1920})
})

test('idle side docking keeps a wide native host while recording stays compact', async ({page}) => {
  await page.goto('/')

  const geometry = await page.evaluate(() => {
    const fn = (window as Window & {
      __RF_TEST_CAPSULE_COLLAPSED_WINDOW_GEOMETRY__?: (input: {
        compactCollapsed: boolean
        dockSide: 'left' | 'right'
        workArea: {x: number; y: number; width: number; height: number}
      }) => {windowSize: {width: number; height: number}; visualSize: {width: number; height: number}}
    }).__RF_TEST_CAPSULE_COLLAPSED_WINDOW_GEOMETRY__
    if (!fn) return null
    const workArea = {x: 0, y: 0, width: 1920, height: 1040}
    return {
      idle: fn({compactCollapsed: false, dockSide: 'right', workArea}),
      recording: fn({compactCollapsed: true, dockSide: 'right', workArea}),
    }
  })

  expect(geometry).not.toBeNull()
  expect(geometry?.idle.windowSize).toEqual({width: 820, height: 560})
  expect(geometry?.idle.visualSize).toEqual({width: 96, height: 560})
  expect(geometry?.recording.windowSize).toEqual({width: 96, height: 360})
  expect(geometry?.recording.visualSize).toEqual({width: 96, height: 360})
})

test('capsule docking keeps the selected monitor side at vertical monitor seams', async ({page}) => {
  await page.goto('/')

  const workAreas = [
    {x: 0, y: 0, width: 1920, height: 1040},
    {x: 0, y: 1040, width: 1920, height: 1080},
  ]
  const size = {width: 820, height: 96}

  await expect(dockSide(page, {position: {x: 500, y: 6}, size, workAreas})).resolves.toBe('top')
  await expect(dockSide(page, {position: {x: 500, y: 1040 - size.height + 8}, size, workAreas})).resolves.toBe('bottom')
  await expect(dockSide(page, {position: {x: 500, y: 1040 + 8}, size, workAreas})).resolves.toBe('top')
  await expect(dockSide(page, {position: {x: 500, y: 1040 + 1080 - size.height - 6}, size, workAreas})).resolves.toBe('bottom')
})

test('capsule controls are excluded from native drag regions while the grabber remains draggable', async ({page}) => {
  await page.goto('/')

  const toolsButton = page.getByRole('button', {name: /Screenshot \/ board|截图 \/ 画板/})
  await expect(toolsButton).toBeVisible()
  await expect(draggableValue(page.locator('.rf-shell'))).resolves.toBe('no-drag')
  await expect(draggableValue(page.locator('.recorder-stage'))).resolves.toBe('no-drag')
  await expect(draggableValue(page.locator('.capsule'))).resolves.toBe('drag')
  await expect(draggableValue(toolsButton)).resolves.toBe('no-drag')
  await expect(draggableValue(page.getByRole('button', {name: /Move recorder|移动录制器/}))).resolves.toBe('drag')

  await toolsButton.click()
  const boardMenu = page.getByRole('dialog', {name: /board (menu|菜单)|画板菜单/})
  await expect(boardMenu).toBeVisible()
  await expect(draggableValue(boardMenu)).resolves.toBe('no-drag')
  await expect(draggableValue(page.getByRole('button', {name: /Region screenshot|区域截图/}))).resolves.toBe('no-drag')
})

test('capsule exposes a minimize action and a distinct language icon', async ({page}) => {
  await page.goto('/')

  const languageButton = page.getByRole('button', {name: /Select language|选择语言/})
  await expect(languageButton.locator('svg')).toHaveClass(/lucide-earth/)
  await expect(page.getByRole('button', {name: /Minimize window|最小化窗口/})).toBeVisible()

  await page.getByRole('button', {name: /Minimize window|最小化窗口/}).click()
  await expect.poll(async () => page.evaluate(() => (window as Window & {__RF_WINDOW_MINIMIZED__?: boolean}).__RF_WINDOW_MINIMIZED__)).toBe(true)
})

async function dockSide(page: Page, input: {
  position: {x: number; y: number}
  size: {width: number; height: number}
  workAreas: Array<{id?: string; x: number; y: number; width: number; height: number}>
  activeScreenId?: string
}) {
  return (await dockTarget(page, input)).side
}

async function dockTarget(page: Page, input: {
  position: {x: number; y: number}
  size: {width: number; height: number}
  workAreas: Array<{id?: string; x: number; y: number; width: number; height: number}>
  activeScreenId?: string
}) {
  return page.evaluate((value) => {
    const fn = (window as Window & {
      __RF_TEST_RESOLVE_CAPSULE_DOCK_TARGET__?: (request: typeof value) => {side: string; workArea: unknown}
    }).__RF_TEST_RESOLVE_CAPSULE_DOCK_TARGET__
    if (!fn) return {side: 'missing-test-hook', workArea: null}
    return fn(value)
  }, input)
}

async function draggableValue(locator: ReturnType<Page['locator']>) {
  return locator.evaluate((element) => getComputedStyle(element).getPropertyValue('--wails-draggable').trim())
}
