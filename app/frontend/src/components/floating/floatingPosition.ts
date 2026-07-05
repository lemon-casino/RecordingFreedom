import {Screens, Window as WailsWindow} from '@wailsio/runtime'
import type {CapsuleWindowDockSide} from '../../services/recorderBackend'

export type FloatingRect = {
  x: number
  y: number
  width: number
  height: number
}

type WorkArea = FloatingRect & {
  id?: string
}

export type FloatingPlacement = {
  anchor: FloatingRect
  bounds: FloatingRect
  direction: 'up' | 'down' | 'left' | 'right'
  screenId?: string
}

type PlacementOptions = {
  dockSide?: CapsuleWindowDockSide
  width: number
  height: number
  maxHeight?: number
  margin?: number
  minWidth?: number
}

export async function resolveFloatingPanelPlacement(anchorElement: Element, options: PlacementOptions): Promise<FloatingPlacement> {
  const anchor = await globalAnchorRect(anchorElement)
  const workAreas = await getWorkAreas()
  const workArea = chooseWorkArea(anchor, workAreas)
  const width = Math.max(options.minWidth ?? 260, options.width)
  const height = Math.min(options.maxHeight ?? options.height, Math.max(120, (workArea?.height ?? options.height) - 16))
  const margin = options.margin ?? 10
  const direction = resolvePanelDirection(anchor, {width, height}, workArea, options.dockSide ?? 'none', margin)
  const bounds = clampToWorkArea(positionForDirection(anchor, {width, height}, direction, margin), workArea, margin)
  return {anchor, bounds, direction, screenId: workArea?.id}
}

export async function resolveFloatingSelectPlacement(anchorElement: Element, options: {
  width?: number
  minWidth?: number
  maxWidth?: number
  maxHeight?: number
  optionCount: number
  margin?: number
}): Promise<FloatingPlacement> {
  const anchor = await globalAnchorRect(anchorElement)
  const workAreas = await getWorkAreas()
  const workArea = chooseWorkArea(anchor, workAreas)
  const margin = options.margin ?? 8
  const minWidth = options.minWidth ?? 180
  const maxWidth = Math.max(minWidth, options.maxWidth ?? 360)
  const availableWidth = Math.max(minWidth, (workArea?.width ?? maxWidth) - margin * 2)
  const width = clamp(options.width ?? anchor.width, minWidth, Math.min(maxWidth, availableWidth))
  const desiredHeight = Math.min(options.maxHeight ?? 280, Math.max(44, options.optionCount * 44 + 12))
  const height = Math.min(desiredHeight, Math.max(44, (workArea?.height ?? desiredHeight) - margin * 2))
  const browserScreen = window.screen as Screen & {availLeft?: number; availTop?: number}
  const fallbackTop = browserScreen.availTop ?? 0
  const spaceBelow = (workArea ? workArea.y + workArea.height : fallbackTop + window.screen.availHeight) - anchor.y - anchor.height
  const spaceAbove = anchor.y - (workArea?.y ?? fallbackTop)
  const direction: FloatingPlacement['direction'] = spaceBelow < height + margin && spaceAbove > spaceBelow ? 'up' : 'down'
  const bounds = clampToWorkArea(positionForDirection(anchor, {width, height}, direction, margin), workArea, margin)
  return {anchor, bounds, direction, screenId: workArea?.id}
}

async function globalAnchorRect(element: Element): Promise<FloatingRect> {
  const rect = element.getBoundingClientRect()
  try {
    const position = await WailsWindow.Position()
    return {
      x: Math.round(position.x + rect.left),
      y: Math.round(position.y + rect.top),
      width: Math.round(rect.width),
      height: Math.round(rect.height),
    }
  } catch {
    return {
      x: Math.round(rect.left),
      y: Math.round(rect.top),
      width: Math.round(rect.width),
      height: Math.round(rect.height),
    }
  }
}

async function getWorkAreas(): Promise<WorkArea[]> {
  try {
    const screens = await Screens.GetAll()
    return screens
      .map((screen, index): WorkArea | null => {
        const rect = screen.WorkArea ?? screen.Bounds
        if (!rect || rect.Width <= 0 || rect.Height <= 0) return null
        return {
          id: String((screen as {ID?: unknown}).ID ?? index + 1),
          x: rect.X,
          y: rect.Y,
          width: rect.Width,
          height: rect.Height,
        }
      })
      .filter((area): area is WorkArea => area !== null)
  } catch {
    const browserScreen = window.screen as Screen & {availLeft?: number; availTop?: number}
    const x = Number.isFinite(browserScreen.availLeft) ? browserScreen.availLeft ?? 0 : 0
    const y = Number.isFinite(browserScreen.availTop) ? browserScreen.availTop ?? 0 : 0
    const width = window.screen.availWidth || window.innerWidth || 1280
    const height = window.screen.availHeight || window.innerHeight || 720
    return [{id: 'browser', x, y, width, height}]
  }
}

function chooseWorkArea(anchor: FloatingRect, workAreas: WorkArea[]): WorkArea | undefined {
  if (workAreas.length === 0) return undefined
  const centerX = anchor.x + anchor.width / 2
  const centerY = anchor.y + anchor.height / 2
  return workAreas.find((area) => pointInside(centerX, centerY, area)) ?? nearestWorkArea(centerX, centerY, workAreas)
}

function resolvePanelDirection(
  anchor: FloatingRect,
  size: {width: number; height: number},
  workArea: WorkArea | undefined,
  dockSide: CapsuleWindowDockSide,
  margin: number,
): FloatingPlacement['direction'] {
  if (dockSide === 'left') return 'right'
  if (dockSide === 'right') return 'left'
  if (dockSide === 'top') return 'down'
  if (dockSide === 'bottom') return 'up'
  const top = workArea?.y ?? 0
  const bottom = workArea ? workArea.y + workArea.height : window.screen.availHeight || window.innerHeight
  const spaceBelow = bottom - (anchor.y + anchor.height)
  const spaceAbove = anchor.y - top
  if (spaceBelow < size.height + margin && spaceAbove > spaceBelow) return 'up'
  return 'down'
}

function positionForDirection(
  anchor: FloatingRect,
  size: {width: number; height: number},
  direction: FloatingPlacement['direction'],
  margin: number,
): FloatingRect {
  if (direction === 'left') {
    return {
      x: anchor.x - size.width - margin,
      y: anchor.y + anchor.height / 2 - size.height / 2,
      width: size.width,
      height: size.height,
    }
  }
  if (direction === 'right') {
    return {
      x: anchor.x + anchor.width + margin,
      y: anchor.y + anchor.height / 2 - size.height / 2,
      width: size.width,
      height: size.height,
    }
  }
  return {
    x: anchor.x + anchor.width / 2 - size.width / 2,
    y: direction === 'up' ? anchor.y - size.height - margin : anchor.y + anchor.height + margin,
    width: size.width,
    height: size.height,
  }
}

function clampToWorkArea(rect: FloatingRect, workArea: WorkArea | undefined, margin: number): FloatingRect {
  if (!workArea) return roundRect(rect)
  const minX = workArea.x + margin
  const minY = workArea.y + margin
  const maxX = Math.max(minX, workArea.x + workArea.width - rect.width - margin)
  const maxY = Math.max(minY, workArea.y + workArea.height - rect.height - margin)
  return {
    x: Math.round(clamp(rect.x, minX, maxX)),
    y: Math.round(clamp(rect.y, minY, maxY)),
    width: Math.round(rect.width),
    height: Math.round(rect.height),
  }
}

function pointInside(x: number, y: number, area: WorkArea) {
  return x >= area.x && y >= area.y && x < area.x + area.width && y < area.y + area.height
}

function nearestWorkArea(x: number, y: number, areas: WorkArea[]) {
  return areas.reduce((nearest, area) => distanceToArea(x, y, area) < distanceToArea(x, y, nearest) ? area : nearest, areas[0])
}

function distanceToArea(x: number, y: number, area: WorkArea) {
  const nearestX = clamp(x, area.x, area.x + area.width)
  const nearestY = clamp(y, area.y, area.y + area.height)
  return Math.hypot(x - nearestX, y - nearestY)
}

function clamp(value: number, min: number, max: number) {
  if (max < min) return min
  return Math.min(max, Math.max(min, value))
}

function roundRect(rect: FloatingRect): FloatingRect {
  return {
    x: Math.round(rect.x),
    y: Math.round(rect.y),
    width: Math.round(rect.width),
    height: Math.round(rect.height),
  }
}
