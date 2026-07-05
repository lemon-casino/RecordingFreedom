import {expect, test, type Page} from '@playwright/test'

test('capsule docking only snaps near external monitor edges', async ({page}) => {
  await page.goto('/')

  const workAreas = [
    {x: 0, y: 0, width: 1920, height: 1040},
    {x: 1920, y: 0, width: 1600, height: 900},
  ]
  const size = {width: 760, height: 96}

  await expect(dockSide(page, {position: {x: 4, y: 240}, size, workAreas})).resolves.toBe('left')
  await expect(dockSide(page, {position: {x: 80, y: 240}, size, workAreas})).resolves.toBe('none')
  await expect(dockSide(page, {position: {x: 1920 - size.width + 4, y: 240}, size, workAreas})).resolves.toBe('none')
  await expect(dockSide(page, {position: {x: 1920 - 20, y: 240}, size, workAreas})).resolves.toBe('none')
  await expect(dockSide(page, {position: {x: 1920 + 4, y: 240}, size, workAreas})).resolves.toBe('none')
  await expect(dockSide(page, {position: {x: 1920 + 1600 - size.width - 4, y: 240}, size, workAreas})).resolves.toBe('right')
})

test('capsule docking ignores vertical monitor seams and keeps top or bottom external edges available', async ({page}) => {
  await page.goto('/')

  const workAreas = [
    {x: 0, y: 0, width: 1920, height: 1040},
    {x: 0, y: 1040, width: 1920, height: 1080},
  ]
  const size = {width: 760, height: 96}

  await expect(dockSide(page, {position: {x: 500, y: 6}, size, workAreas})).resolves.toBe('top')
  await expect(dockSide(page, {position: {x: 500, y: 1040 - size.height + 8}, size, workAreas})).resolves.toBe('none')
  await expect(dockSide(page, {position: {x: 500, y: 1040 + 8}, size, workAreas})).resolves.toBe('none')
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

async function dockSide(page: Page, input: {
  position: {x: number; y: number}
  size: {width: number; height: number}
  workAreas: Array<{x: number; y: number; width: number; height: number}>
}) {
  return page.evaluate((value) => {
    const fn = (window as Window & {
      __RF_TEST_RESOLVE_CAPSULE_DOCK_TARGET__?: (request: typeof value) => {side: string}
    }).__RF_TEST_RESOLVE_CAPSULE_DOCK_TARGET__
    if (!fn) return 'missing-test-hook'
    return fn(value).side
  }, input)
}

async function draggableValue(locator: ReturnType<Page['locator']>) {
  return locator.evaluate((element) => getComputedStyle(element).getPropertyValue('--wails-draggable').trim())
}
