// @vitest-environment jsdom
import {beforeEach, describe, expect, it, vi} from 'vitest'

vi.mock('@wailsio/runtime', () => ({
  Screens: {GetAll: vi.fn()},
  Window: {Position: vi.fn()},
}))

import {Screens, Window as WailsWindow} from '@wailsio/runtime'
import {resolveFloatingPanelPlacement, type FloatingRect} from './floatingPosition'

const workArea = {id: '1', x: 0, y: 0, width: 1280, height: 720}

function anchorRect(rect: {left: number; top: number; width: number; height: number}): Element {
  return {
    getBoundingClientRect: () => rect,
  } as unknown as Element
}

function expectRectClose(actual: FloatingRect, expected: {x: number; y: number; width: number; height: number}) {
  expect(actual.x).toBeCloseTo(expected.x)
  expect(actual.y).toBeCloseTo(expected.y)
  expect(actual.width).toBe(expected.width)
  expect(actual.height).toBe(expected.height)
}

beforeEach(() => {
  vi.mocked(Screens.GetAll).mockResolvedValue([
    {
      ID: '1',
      Bounds: {X: 0, Y: 0, Width: 1280, Height: 720},
      WorkArea: {X: 0, Y: 0, Width: 1280, Height: 720},
    },
  ] as never)
  vi.mocked(WailsWindow.Position).mockResolvedValue({x: 0, y: 0} as never)
})

describe('resolveFloatingPanelPlacement', () => {
  it('opens downward when there is room below the anchor', async () => {
    const placement = await resolveFloatingPanelPlacement(anchorRect({left: 600, top: 40, width: 120, height: 40}), {
      width: 260,
      height: 300,
    })
    expect(placement.direction).toBe('down')
    expectRectClose(placement.bounds, {x: 530, y: 90, width: 260, height: 300})
  })

  it('flips upward when the panel would overflow the work area bottom', async () => {
    const placement = await resolveFloatingPanelPlacement(anchorRect({left: 600, top: 600, width: 120, height: 40}), {
      width: 260,
      height: 300,
    })
    expect(placement.direction).toBe('up')
    expectRectClose(placement.bounds, {x: 530, y: 290, width: 260, height: 300})
  })

  it('docks sideways when the capsule sits on a screen edge', async () => {
    const placement = await resolveFloatingPanelPlacement(anchorRect({left: 600, top: 300, width: 120, height: 40}), {
      width: 260,
      height: 300,
      dockSide: 'left',
    })
    expect(placement.direction).toBe('right')
    expectRectClose(placement.bounds, {x: 730, y: 170, width: 260, height: 300})
  })

  it('clamps sideways panels back into the work area', async () => {
    const placement = await resolveFloatingPanelPlacement(anchorRect({left: 1230, top: 300, width: 40, height: 40}), {
      width: 260,
      height: 300,
      dockSide: 'left',
    })
    expect(placement.direction).toBe('right')
    expect(placement.bounds.x + placement.bounds.width).toBeLessThanOrEqual(workArea.width)
    expect(placement.bounds.x).toBeGreaterThanOrEqual(0)
  })

  it('enforces the minimum width', async () => {
    const placement = await resolveFloatingPanelPlacement(anchorRect({left: 600, top: 40, width: 120, height: 40}), {
      width: 120,
      minWidth: 260,
      height: 300,
    })
    expect(placement.bounds.width).toBe(260)
  })
})
