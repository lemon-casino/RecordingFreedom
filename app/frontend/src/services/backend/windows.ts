import {RecordingFreedomService, type CapsuleWindowHitRegionsRequest as BoundCapsuleWindowHitRegionsRequest, type FloatingPanelRequest as BoundFloatingPanelRequest, type FloatingPanelState as BoundFloatingPanelState, type FloatingRect as BoundFloatingRect, type FloatingSelectChosenEvent as BoundFloatingSelectChosenEvent, type FloatingSelectOption as BoundFloatingSelectOption, type FloatingSelectRequest as BoundFloatingSelectRequest, type FloatingSelectState as BoundFloatingSelectState} from '../../../bindings/github.com/lemon-casino/RecordingFreedom/app'
import {Events, Screens, Window as WailsWindow} from '@wailsio/runtime'
import {isWailsDesktopRuntime} from './shared'

export type CapsuleWindowHitRegion = {
  x: number
  y: number
  width: number
  height: number
  kind?: 'rect' | 'round-rect' | 'pill'
  radius?: number
}

export type FloatingPanelKind = 'source' | 'audio' | 'camera' | 'board' | 'language' | 'settings' | 'close' | 'ocr-result'

export type FloatingRect = {
  x: number
  y: number
  width: number
  height: number
}

export type FloatingPanelState = {
  visible: boolean
  kind?: FloatingPanelKind
  anchor: FloatingRect
  bounds: FloatingRect
  dockSide?: CapsuleWindowDockSide | string
  token: number
  screenId?: string
  direction?: string
  contextId?: string
}

export type FloatingSelectOption = {
  value: string
  label: string
  disabled?: boolean
  swatch?: string
}

export type FloatingSelectState = {
  visible: boolean
  id?: string
  anchor: FloatingRect
  bounds: FloatingRect
  value?: string
  options: FloatingSelectOption[]
  token: number
  panelToken?: number
  screenId?: string
  direction?: string
}

export type FloatingSelectChosenEvent = {
  id: string
  value: string
  token: number
  panelToken?: number
}

const browserCapsuleDockSideEvent = 'rf-capsule-dock-side'

const browserFloatingPanelEvent = 'rf-floating-panel'

const browserFloatingSelectEvent = 'rf-floating-select'

const browserFloatingSelectChosenEvent = 'rf-floating-select-chosen'

const capsuleWindowWidth = 820

const capsuleWindowCompactWidth = 440

const capsuleWindowCollapsedHeight = 96

const capsuleWindowExpandedHeight = 600

const capsuleWindowSideWidth = 96

const capsuleWindowSideHeight = 560

const capsuleWindowSideCompactHeight = 360

const capsuleWindowSideExpandedWidth = 520

const capsuleSideSnapThreshold = 32

const capsuleEdgeSnapThreshold = 32

export type CapsuleWindowExpandDirection = 'down' | 'up'

export type CapsuleWindowDockSide = 'none' | 'left' | 'right' | 'top' | 'bottom'

let lastAnnotationOverlayHitRegionsSignature = ''

export async function setCapsuleWindowHitRegions(req: {
  enabled: boolean
  force?: boolean
  viewportWidth: number
  viewportHeight: number
  devicePixelRatio: number
  regions: CapsuleWindowHitRegion[]
}): Promise<void> {
  const signature = capsuleHitRegionsSignature(req)
  if (!req.force && signature === lastCapsuleHitRegionsSignature) return
  try {
    await RecordingFreedomService.SetCapsuleWindowHitRegions(req as BoundCapsuleWindowHitRegionsRequest)
    lastCapsuleHitRegionsSignature = signature
  } catch (error) {
    console.info('Using browser capsule hit-region fallback:', error)
  }
}

export async function setAnnotationOverlayHitRegions(req: {
  enabled: boolean
  force?: boolean
  viewportWidth: number
  viewportHeight: number
  devicePixelRatio: number
  regions: CapsuleWindowHitRegion[]
}): Promise<void> {
  const signature = capsuleHitRegionsSignature(req)
  if (!req.force && signature === lastAnnotationOverlayHitRegionsSignature) return
  try {
    await RecordingFreedomService.SetAnnotationOverlayHitRegions(req as BoundCapsuleWindowHitRegionsRequest)
    lastAnnotationOverlayHitRegionsSignature = signature
  } catch (error) {
    ;(window as Window & {__RF_LAST_ANNOTATION_HIT_REGIONS__?: typeof req}).__RF_LAST_ANNOTATION_HIT_REGIONS__ = req
    console.info('Using browser annotation overlay hit-region fallback:', error)
  }
}

export async function setFloatingPanelHitRegions(req: {
  enabled: boolean
  force?: boolean
  viewportWidth: number
  viewportHeight: number
  devicePixelRatio: number
  regions: CapsuleWindowHitRegion[]
}): Promise<void> {
  try {
    await RecordingFreedomService.SetFloatingPanelHitRegions(req as BoundCapsuleWindowHitRegionsRequest)
  } catch (error) {
    console.info('Using browser floating panel hit-region fallback:', error)
  }
}

export async function setFloatingSelectHitRegions(req: {
  enabled: boolean
  force?: boolean
  viewportWidth: number
  viewportHeight: number
  devicePixelRatio: number
  regions: CapsuleWindowHitRegion[]
}): Promise<void> {
  try {
    await RecordingFreedomService.SetFloatingSelectHitRegions(req as BoundCapsuleWindowHitRegionsRequest)
  } catch (error) {
    console.info('Using browser floating select hit-region fallback:', error)
  }
}

export async function showFloatingPanel(req: {
  kind: FloatingPanelKind
  anchor: FloatingRect
  bounds: FloatingRect
  dockSide?: CapsuleWindowDockSide | string
  width: number
  height: number
  minWidth?: number
  maxHeight?: number
  token: number
  screenId?: string
  direction?: string
  contextId?: string
}): Promise<FloatingPanelState> {
  try {
    return fromBoundFloatingPanelState(await RecordingFreedomService.ShowFloatingPanel(req as BoundFloatingPanelRequest))
  } catch (error) {
    console.info('Using browser floating panel fallback:', error)
    const state: FloatingPanelState = {
      visible: true,
      kind: req.kind,
      anchor: req.anchor,
      bounds: req.bounds,
      dockSide: req.dockSide,
      token: req.token,
      screenId: req.screenId,
      direction: req.direction,
      contextId: req.contextId }
    ;(window as Window & {__RF_FLOATING_PANEL__?: FloatingPanelState}).__RF_FLOATING_PANEL__ = state
    window.dispatchEvent(new CustomEvent(browserFloatingPanelEvent, {detail: state}))
    return state
  }
}

export async function updateFloatingPanel(req: Parameters<typeof showFloatingPanel>[0]): Promise<FloatingPanelState> {
  try {
    return fromBoundFloatingPanelState(await RecordingFreedomService.UpdateFloatingPanel(req as BoundFloatingPanelRequest))
  } catch {
    return showFloatingPanel(req)
  }
}

export async function hideFloatingPanel(token = 0): Promise<void> {
  try {
    await RecordingFreedomService.HideFloatingPanel(token)
  } catch (error) {
    console.info('Using browser floating panel hide fallback:', error)
    const current = (window as Window & {__RF_FLOATING_PANEL__?: FloatingPanelState}).__RF_FLOATING_PANEL__
    const state: FloatingPanelState = {...(current ?? emptyFloatingPanelState()), visible: false, token: current?.token ?? token}
    ;(window as Window & {__RF_FLOATING_PANEL__?: FloatingPanelState}).__RF_FLOATING_PANEL__ = state
    window.dispatchEvent(new CustomEvent(browserFloatingPanelEvent, {detail: state}))
  }
}

export async function getFloatingPanelState(): Promise<FloatingPanelState> {
  try {
    return fromBoundFloatingPanelState(await RecordingFreedomService.GetFloatingPanelState())
  } catch {
    return (window as Window & {__RF_FLOATING_PANEL__?: FloatingPanelState}).__RF_FLOATING_PANEL__ ?? emptyFloatingPanelState()
  }
}

export function subscribeFloatingPanelChanged(handler: (state: FloatingPanelState) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('floating.panel.changed', (event) => {
      handler(fromBoundFloatingPanelState(event.data as BoundFloatingPanelState))
    })
  } catch (error) {
    console.info('Desktop floating panel events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler((event as CustomEvent<FloatingPanelState>).detail)
  }
  window.addEventListener(browserFloatingPanelEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserFloatingPanelEvent, onBrowserEvent)
  }
}

export async function showFloatingSelect(req: {
  id: string
  anchor: FloatingRect
  bounds: FloatingRect
  value: string
  options: FloatingSelectOption[]
  token: number
  panelToken?: number
  width?: number
  maxHeight?: number
  screenId?: string
  direction?: string
}): Promise<FloatingSelectState> {
  try {
    return fromBoundFloatingSelectState(await RecordingFreedomService.ShowFloatingSelect(req as BoundFloatingSelectRequest))
  } catch (error) {
    console.info('Using browser floating select fallback:', error)
    const state: FloatingSelectState = {
      visible: true,
      id: req.id,
      anchor: req.anchor,
      bounds: req.bounds,
      value: req.value,
      options: req.options,
      token: req.token,
      panelToken: req.panelToken,
      screenId: req.screenId,
      direction: req.direction }
    ;(window as Window & {__RF_FLOATING_SELECT__?: FloatingSelectState}).__RF_FLOATING_SELECT__ = state
    window.dispatchEvent(new CustomEvent(browserFloatingSelectEvent, {detail: state}))
    return state
  }
}

export async function hideFloatingSelect(token = 0): Promise<void> {
  try {
    await RecordingFreedomService.HideFloatingSelect(token)
  } catch (error) {
    console.info('Using browser floating select hide fallback:', error)
    const current = (window as Window & {__RF_FLOATING_SELECT__?: FloatingSelectState}).__RF_FLOATING_SELECT__
    const state: FloatingSelectState = {...(current ?? emptyFloatingSelectState()), visible: false, token: current?.token ?? token}
    ;(window as Window & {__RF_FLOATING_SELECT__?: FloatingSelectState}).__RF_FLOATING_SELECT__ = state
    window.dispatchEvent(new CustomEvent(browserFloatingSelectEvent, {detail: state}))
  }
}

export async function completeFloatingSelect(event: FloatingSelectChosenEvent): Promise<void> {
  try {
    await RecordingFreedomService.CompleteFloatingSelect(event as BoundFloatingSelectChosenEvent)
  } catch (error) {
    console.info('Using browser floating select complete fallback:', error)
    window.dispatchEvent(new CustomEvent(browserFloatingSelectChosenEvent, {detail: event}))
    await hideFloatingSelect(event.token)
  }
}

export async function getFloatingSelectState(): Promise<FloatingSelectState> {
  try {
    return fromBoundFloatingSelectState(await RecordingFreedomService.GetFloatingSelectState())
  } catch {
    return (window as Window & {__RF_FLOATING_SELECT__?: FloatingSelectState}).__RF_FLOATING_SELECT__ ?? emptyFloatingSelectState()
  }
}

export function subscribeFloatingSelectChanged(handler: (state: FloatingSelectState) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('floating.select.changed', (event) => {
      handler(fromBoundFloatingSelectState(event.data as BoundFloatingSelectState))
    })
  } catch (error) {
    console.info('Desktop floating select events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler((event as CustomEvent<FloatingSelectState>).detail)
  }
  window.addEventListener(browserFloatingSelectEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserFloatingSelectEvent, onBrowserEvent)
  }
}

export function subscribeFloatingSelectChosen(handler: (event: FloatingSelectChosenEvent) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('floating.select.chosen', (event) => {
      handler(fromBoundFloatingSelectChosen(event.data as BoundFloatingSelectChosenEvent))
    })
  } catch (error) {
    console.info('Desktop floating select chosen events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler((event as CustomEvent<FloatingSelectChosenEvent>).detail)
  }
  window.addEventListener(browserFloatingSelectChosenEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserFloatingSelectChosenEvent, onBrowserEvent)
  }
}

export type CapsuleWindowMoveReason = 'window-did-move' | 'window-end-move'

export function subscribeCapsuleWindowMoveEnded(handler: (reason: CapsuleWindowMoveReason) => void): () => void {
  const events: Array<{name: string; reason: CapsuleWindowMoveReason}> = [
    {name: 'common:WindowDidMove', reason: 'window-did-move'},
    {name: 'mac:WindowDidMove', reason: 'window-did-move'},
    {name: 'linux:WindowDidMove', reason: 'window-did-move'},
    {name: 'windows:WindowDidMove', reason: 'window-did-move'},
    {name: 'windows:WindowEndMove', reason: 'window-end-move'},
    {name: 'windows:WindowEndResize', reason: 'window-end-move'},
  ]
  const disposers = events.map(({name, reason}) => Events.On(name, (event) => {
    if (event.sender && event.sender !== 'capsule-recorder') return
    handler(reason)
  }))
  return () => disposers.forEach((dispose) => dispose())
}

export function subscribeCapsuleDockSide(handler: (side: CapsuleWindowDockSide) => void): () => void {
  const onDockSide = (event: Event) => {
    const side = (event as CustomEvent<CapsuleWindowDockSide>).detail
    if (side === 'left' || side === 'right' || side === 'top' || side === 'bottom' || side === 'none') {
      handler(side)
    }
  }
  window.addEventListener(browserCapsuleDockSideEvent, onDockSide)
  return () => window.removeEventListener(browserCapsuleDockSideEvent, onDockSide)
}

export async function restoreCapsuleWindow(focus = true): Promise<void> {
  if (!isWailsDesktopRuntime()) {
    const browserWindow = window as Window & {__RF_TEST_CAPSULE_RESTORE_COUNT__?: number}
    browserWindow.__RF_TEST_CAPSULE_RESTORE_COUNT__ = (browserWindow.__RF_TEST_CAPSULE_RESTORE_COUNT__ ?? 0) + 1
    return
  }
  try {
    await WailsWindow.Show()
    await WailsWindow.UnMinimise().catch(() => undefined)
    await WailsWindow.SetAlwaysOnTop(true).catch(() => undefined)
    if (focus) await WailsWindow.Focus().catch(() => undefined)
  } catch (error) {
    console.info('Using browser capsule window restore fallback:', error)
  }
}

export async function hideCapsuleWindow(): Promise<void> {
  if (!isWailsDesktopRuntime()) {
    const browserWindow = window as Window & {__RF_CAPSULE_HIDDEN__?: boolean}
    browserWindow.__RF_CAPSULE_HIDDEN__ = true
    window.dispatchEvent(new Event('rf-capsule-hidden'))
    return
  }
  try {
    await WailsWindow.Hide()
  } catch (error) {
    console.info('Using browser capsule window hide fallback:', error)
  }
}

export async function setCapsuleWindowExpanded(
  expanded: boolean,
  expandedHeight = capsuleWindowExpandedHeight,
  preferredDirection: CapsuleWindowExpandDirection | 'auto' = 'auto',
  compactCollapsed = false,
): Promise<CapsuleWindowExpandDirection> {
  try {
    // Layout runs as soon as the webview mounts, including while a
    // start-at-login window is intentionally hidden. Never change visibility
    // from this geometry-only path.
    const position = await WailsWindow.Position()
    const size = await WailsWindow.Size().catch(() => ({
      width: expanded ? capsuleWindowWidth : compactCollapsed ? capsuleWindowCompactWidth : capsuleWindowWidth,
      height: expanded ? expandedHeight : capsuleWindowCollapsedHeight }))
    const dockSide = lastCapsuleDockSide
    const initialVisualSize = capsuleCollapsedVisualSize(compactCollapsed, dockSide, null)
    const initialVisualPosition = capsuleVisualPositionFromWindow(dockSide, position, size, initialVisualSize)
    const workArea = await capsuleWorkAreaForWindow(position, size, undefined, initialVisualPosition, initialVisualSize).catch(() => null)
    const collapsedVisualSize = capsuleCollapsedVisualSize(compactCollapsed, dockSide, workArea)
    const collapsedVisualPosition = capsuleVisibleCollapsedPosition(dockSide, position, size, collapsedVisualSize, workArea)
    if (!expanded) {
      const collapsedWindowSize = capsuleCollapsedWindowSize(compactCollapsed, dockSide, workArea)
      const collapsedPosition = capsuleReservedWindowPosition(dockSide, collapsedVisualPosition, collapsedVisualSize, collapsedWindowSize, workArea)
      await setCapsuleWindowBoundsIfChanged(position, size, collapsedPosition, collapsedWindowSize)
      lastCapsuleCollapsedPosition = null
      return lastCapsuleExpandedDirection
    }

    const reservedWindowSize = capsuleReservedWindowSize(compactCollapsed, dockSide, workArea)
    if (isSideDock(dockSide)) {
      const targetExpandedSize = capsuleReservedWindowSize(compactCollapsed, dockSide, workArea)
      const expandedPosition = capsuleReservedWindowPosition(dockSide, collapsedVisualPosition, collapsedVisualSize, targetExpandedSize, workArea)
      lastCapsuleExpandedDirection = 'down'
      lastCapsuleCollapsedPosition = collapsedVisualPosition
      await setCapsuleWindowBoundsIfChanged(position, size, expandedPosition, targetExpandedSize)
      return 'down'
    }

    const direction = dockSide === 'bottom'
      ? 'up'
      : dockSide === 'top'
        ? 'down'
        : resolveCapsuleExpandDirection(collapsedVisualPosition.y, expandedHeight, preferredDirection, workArea)
    const expandedPosition = capsuleReservedWindowPosition(dockSide, collapsedVisualPosition, collapsedVisualSize, reservedWindowSize, workArea, direction)
    lastCapsuleExpandedDirection = direction
    lastCapsuleCollapsedPosition = collapsedVisualPosition
    await setCapsuleWindowBoundsIfChanged(position, size, expandedPosition, reservedWindowSize)
    return direction
  } catch (error) {
    console.info('Using browser capsule window size fallback:', error)
    return preferredDirection === 'up' ? 'up' : 'down'
  }
}

export async function snapCapsuleWindowToEdge(compactCollapsed = false): Promise<CapsuleWindowDockSide> {
  try {
    // Snapping may be triggered by startup/layout events for a hidden window.
    // Showing the capsule belongs to explicit tray/user actions only.
    const position = await WailsWindow.Position()
    const size = await WailsWindow.Size().catch(() => capsuleCollapsedWindowSize(compactCollapsed, lastCapsuleDockSide, null))
    let workAreas = await capsuleWorkAreas().catch(() => [])
    const dockSideBeforeSnap = lastCapsuleDockSide
    const visualSizeBeforeSnap = capsuleCollapsedVisualSize(compactCollapsed, dockSideBeforeSnap, null)
    lastCapsuleCollapsedPosition = null
    const visualPositionBeforeSnap = capsuleVisualPositionFromWindow(dockSideBeforeSnap, position, size, visualSizeBeforeSnap)
    const workArea = await capsuleWorkAreaForWindow(position, size, workAreas, visualPositionBeforeSnap, visualSizeBeforeSnap)
    if (workArea && !workAreas.some((area) => area.id === workArea.id && area.x === workArea.x && area.y === workArea.y)) {
      workAreas = [...workAreas, workArea]
    }
    const currentVisualSize = capsuleCollapsedVisualSize(compactCollapsed, lastCapsuleDockSide, workArea)
    const currentVisualPosition = capsuleVisibleCollapsedPosition(lastCapsuleDockSide, position, size, currentVisualSize, workArea)
    const dockTarget = resolveCapsuleDockTarget(currentVisualPosition, currentVisualSize, workAreas, workArea)
    const dockSide = dockTarget.side
    const targetWorkArea = dockTarget.workArea ?? workArea
    lastCapsuleExpandedDirection = dockSide === 'bottom' ? 'up' : 'down'
    const targetSize = capsuleCollapsedWindowSize(compactCollapsed, dockSide, targetWorkArea)
    const targetPosition = capsuleReservedWindowPosition(dockSide, currentVisualPosition, currentVisualSize, targetSize, targetWorkArea)
    await setCapsuleWindowBoundsIfChanged(position, size, targetPosition, targetSize)
    publishCapsuleDockSide(dockSide)
    return dockSide
  } catch (error) {
    console.info('Using browser capsule edge snap fallback:', error)
    return lastCapsuleDockSide
  }
}

function publishCapsuleDockSide(side: CapsuleWindowDockSide) {
  if (lastCapsuleDockSide === side) return
  lastCapsuleDockSide = side
  window.dispatchEvent(new CustomEvent(browserCapsuleDockSideEvent, {detail: side}))
}

function resolveCapsuleExpandDirection(
  windowY: number,
  expandedHeight: number,
  preferredDirection: CapsuleWindowExpandDirection | 'auto',
  workArea: CapsuleWorkArea | null = null,
): CapsuleWindowExpandDirection {
  if (preferredDirection === 'up' || preferredDirection === 'down') return preferredDirection
  const top = workArea?.y ?? capsuleScreenTop()
  const bottom = workArea ? workArea.y + workArea.height : capsuleScreenBottom()
  const wouldOverflowBottom = windowY + expandedHeight > bottom
  const canFitAbove = windowY + capsuleWindowCollapsedHeight - expandedHeight >= top
  return wouldOverflowBottom && canFitAbove ? 'up' : 'down'
}

function resolveCapsuleDockTarget(
  position: {x: number; y: number},
  size: {width: number; height: number},
  workAreas: CapsuleWorkArea[],
  preferredWorkArea: CapsuleWorkArea | null = null,
): CapsuleDockTarget {
  if (!workAreas.length) return {side: 'none', workArea: null}
  const activeWorkArea = chooseCapsuleWorkArea(position, size, workAreas, preferredWorkArea)
  if (!activeWorkArea) return {side: 'none', workArea: null}
  const vertical = (['left', 'right'] as const)
    .map((side) => capsuleDockCandidate(side, position, size, activeWorkArea))
    .filter((candidate): candidate is CapsuleDockCandidate => candidate !== null)
  vertical.sort(compareCapsuleDockCandidates)
  if (vertical[0]) return {side: vertical[0].side, workArea: vertical[0].workArea}

  const horizontal = (['top', 'bottom'] as const)
    .map((side) => capsuleDockCandidate(side, position, size, activeWorkArea))
    .filter((candidate): candidate is CapsuleDockCandidate => candidate !== null)
  horizontal.sort(compareCapsuleDockCandidates)
  if (horizontal[0]) return {side: horizontal[0].side, workArea: horizontal[0].workArea}
  return {side: 'none', workArea: activeWorkArea}
}

export function __resolveCapsuleDockTargetForTest(input: {
  position: {x: number; y: number}
  size: {width: number; height: number}
  workAreas: CapsuleWorkArea[]
  activeScreenId?: string
}) {
  const preferredWorkArea = input.activeScreenId
    ? input.workAreas.find((area) => area.id === input.activeScreenId) ?? null
    : null
  const target = resolveCapsuleDockTarget(input.position, input.size, input.workAreas, preferredWorkArea)
  return {
    side: target.side,
    workArea: target.workArea ? {...target.workArea} : null }
}

export function __capsuleCollapsedWindowGeometryForTest(input: {
  compactCollapsed: boolean
  dockSide: CapsuleWindowDockSide
  workArea?: CapsuleWorkArea | null
}) {
  return {
    windowSize: capsuleCollapsedWindowSize(input.compactCollapsed, input.dockSide, input.workArea ?? null),
    visualSize: capsuleCollapsedVisualSize(input.compactCollapsed, input.dockSide, input.workArea ?? null) }
}

if (import.meta.env.DEV && typeof window !== 'undefined') {
  ;(window as Window & {
    __RF_TEST_RESOLVE_CAPSULE_DOCK_TARGET__?: typeof __resolveCapsuleDockTargetForTest
    __RF_TEST_CAPSULE_COLLAPSED_WINDOW_GEOMETRY__?: typeof __capsuleCollapsedWindowGeometryForTest
  }).__RF_TEST_RESOLVE_CAPSULE_DOCK_TARGET__ = __resolveCapsuleDockTargetForTest
  ;(window as Window & {
    __RF_TEST_CAPSULE_COLLAPSED_WINDOW_GEOMETRY__?: typeof __capsuleCollapsedWindowGeometryForTest
  }).__RF_TEST_CAPSULE_COLLAPSED_WINDOW_GEOMETRY__ = __capsuleCollapsedWindowGeometryForTest
}

type CapsuleWorkArea = {
  id?: string
  x: number
  y: number
  width: number
  height: number
}

type CapsuleDockTarget = {
  side: CapsuleWindowDockSide
  workArea: CapsuleWorkArea | null
}

type CapsuleDockCandidate = {
  side: Exclude<CapsuleWindowDockSide, 'none'>
  workArea: CapsuleWorkArea
  distance: number
  overlapRatio: number
  originInside: boolean
  centerInside: boolean
  screenDistance: number
}

async function capsuleWorkAreaForWindow(
  position: {x: number; y: number},
  size: {width: number; height: number},
  candidates?: CapsuleWorkArea[],
  visualPosition = position,
  visualSize = size,
): Promise<CapsuleWorkArea | null> {
  const workAreas = candidates ?? await capsuleWorkAreas().catch(() => [])
  const owningScreen = await capsuleOwningScreenWorkArea()
  if (!workAreas.length) return owningScreen
  return chooseCapsuleWorkArea(visualPosition, visualSize, workAreas, owningScreen)
}

async function capsuleOwningScreenWorkArea(): Promise<CapsuleWorkArea | null> {
  try {
    const screen = await WailsWindow.GetScreen()
    const screenId = typeof screen?.ID === 'string' ? screen.ID.trim() : ''
    const id = screenId || undefined
    return normalizeCapsuleWorkArea(screen.WorkArea, id) ?? normalizeCapsuleWorkArea(screen.Bounds, id)
  } catch {
    // Browser previews and older runtimes do not expose the owning screen.
    return null
  }
}

function chooseCapsuleWorkArea(
  position: {x: number; y: number},
  size: {width: number; height: number},
  workAreas: CapsuleWorkArea[],
  owningScreen: CapsuleWorkArea | null = null,
): CapsuleWorkArea | null {
  const geometricWorkArea = capsuleWorkAreaForVisualRectFromAreas(position, size, workAreas)
  if (!geometricWorkArea || !owningScreen || owningScreen === geometricWorkArea) return geometricWorkArea

  const geometricOverlap = capsuleWorkAreaOverlapPixels(position, size, geometricWorkArea)
  const owningScreenOverlap = capsuleWorkAreaOverlapPixels(position, size, owningScreen)
  const overlapDifference = Math.abs(geometricOverlap - owningScreenOverlap)
  const tieTolerance = Math.max(4, size.width * size.height * 0.005)
  return overlapDifference <= tieTolerance ? owningScreen : geometricWorkArea
}

async function capsuleWorkAreas(): Promise<CapsuleWorkArea[]> {
  const screens = await Screens.GetAll()
  return screens
    .map((screen, index) => {
      const id = typeof screen.ID === 'string' && screen.ID.trim() ? screen.ID.trim() : String(index + 1)
      return normalizeCapsuleWorkArea(screen.WorkArea, id) ?? normalizeCapsuleWorkArea(screen.Bounds, id)
    })
    .filter((area): area is CapsuleWorkArea => area !== null)
}

function capsuleWorkAreaForPositionFromAreas(
  position: {x: number; y: number},
  size: {width: number; height: number},
  candidates: CapsuleWorkArea[],
): CapsuleWorkArea | null {
  if (!candidates.length) return null
  const centerX = position.x + size.width / 2
  const centerY = position.y + size.height / 2
  return candidates.find((area) => pointInsideWorkArea(centerX, centerY, area)) ?? nearestWorkArea(centerX, centerY, candidates)
}

function capsuleWorkAreaForVisualRectFromAreas(
  position: {x: number; y: number},
  size: {width: number; height: number},
  candidates: CapsuleWorkArea[],
): CapsuleWorkArea | null {
  if (!candidates.length) return null
  const intersections = candidates
    .map((area) => {
      return {
        area,
        areaPixels: capsuleWorkAreaOverlapPixels(position, size, area) }
    })
    .sort((a, b) => b.areaPixels - a.areaPixels)
  if (intersections[0]?.areaPixels > 0) return intersections[0].area
  return capsuleWorkAreaForPositionFromAreas(position, size, candidates)
}

function capsuleWorkAreaOverlapPixels(
  position: {x: number; y: number},
  size: {width: number; height: number},
  area: CapsuleWorkArea,
) {
  const right = position.x + size.width
  const bottom = position.y + size.height
  const areaRight = area.x + area.width
  const areaBottom = area.y + area.height
  return overlapLength(position.x, right, area.x, areaRight) * overlapLength(position.y, bottom, area.y, areaBottom)
}

function pointInsideWorkArea(x: number, y: number, area: CapsuleWorkArea) {
  return x >= area.x && y >= area.y && x <= area.x + area.width && y <= area.y + area.height
}

function pointInsideWorkAreaOpenEnd(x: number, y: number, area: CapsuleWorkArea) {
  return x >= area.x && y >= area.y && x < area.x + area.width && y < area.y + area.height
}

function nearestWorkArea(x: number, y: number, areas: CapsuleWorkArea[]) {
  return areas.reduce((nearest, area) => {
    const nearestDistance = distanceToWorkArea(x, y, nearest)
    const areaDistance = distanceToWorkArea(x, y, area)
    return areaDistance < nearestDistance ? area : nearest
  }, areas[0])
}

function distanceToWorkArea(x: number, y: number, area: CapsuleWorkArea) {
  const nearestX = clampNumber(x, area.x, area.x + area.width)
  const nearestY = clampNumber(y, area.y, area.y + area.height)
  return Math.hypot(x - nearestX, y - nearestY)
}

function capsuleDockCandidate(
  side: Exclude<CapsuleWindowDockSide, 'none'>,
  position: {x: number; y: number},
  size: {width: number; height: number},
  workArea: CapsuleWorkArea,
): CapsuleDockCandidate | null {
  const right = position.x + size.width
  const bottom = position.y + size.height
  const centerX = position.x + size.width / 2
  const centerY = position.y + size.height / 2
  const areaRight = workArea.x + workArea.width
  const areaBottom = workArea.y + workArea.height
  const verticalOverlap = overlapLength(position.y, bottom, workArea.y, areaBottom)
  const horizontalOverlap = overlapLength(position.x, right, workArea.x, areaRight)
  const sideIsVertical = side === 'left' || side === 'right'
  const overlapRatio = sideIsVertical
    ? verticalOverlap / Math.max(1, Math.min(size.height, workArea.height))
    : horizontalOverlap / Math.max(1, Math.min(size.width, workArea.width))
  if (overlapRatio < 0.18) return null

  const edgePosition = side === 'left'
    ? workArea.x
    : side === 'right'
      ? areaRight
      : side === 'top'
        ? workArea.y
        : areaBottom
  const edgeDistance = side === 'left'
    ? Math.abs(position.x - edgePosition)
    : side === 'right'
      ? Math.abs(right - edgePosition)
      : side === 'top'
        ? Math.abs(position.y - edgePosition)
        : Math.abs(bottom - edgePosition)
  const active = sideIsVertical
    ? edgeDistance <= capsuleSideSnapThreshold
    : edgeDistance <= capsuleEdgeSnapThreshold
  if (!active) return null

  return {
    side,
    workArea,
    distance: edgeDistance,
    overlapRatio,
    originInside: pointInsideWorkAreaOpenEnd(position.x, centerY, workArea),
    centerInside: pointInsideWorkAreaOpenEnd(centerX, centerY, workArea),
    screenDistance: distanceToWorkArea(centerX, centerY, workArea) }
}

function compareCapsuleDockCandidates(a: CapsuleDockCandidate, b: CapsuleDockCandidate) {
  if (Math.abs(a.distance - b.distance) > 1) return a.distance - b.distance
  if (Math.abs(a.overlapRatio - b.overlapRatio) > 0.01) return b.overlapRatio - a.overlapRatio
  if (a.originInside !== b.originInside) return a.originInside ? -1 : 1
  if (a.centerInside !== b.centerInside) return a.centerInside ? -1 : 1
  return a.screenDistance - b.screenDistance
}

function overlapLength(aStart: number, aEnd: number, bStart: number, bEnd: number) {
  return Math.max(0, Math.min(aEnd, bEnd) - Math.max(aStart, bStart))
}

function isSideDock(side: CapsuleWindowDockSide) {
  return side === 'left' || side === 'right'
}

function capsuleCollapsedWindowSize(
  compactCollapsed: boolean,
  dockSide: CapsuleWindowDockSide,
  workArea: CapsuleWorkArea | null,
) {
  if (isSideDock(dockSide)) {
    const requestedHeight = compactCollapsed ? capsuleWindowSideCompactHeight : capsuleWindowSideHeight
    const requestedWidth = compactCollapsed ? capsuleWindowSideWidth : capsuleWindowWidth
    return {
      width: Math.min(requestedWidth, Math.max(64, workArea?.width ?? requestedWidth)),
      height: Math.min(requestedHeight, Math.max(capsuleWindowCollapsedHeight, workArea?.height ?? requestedHeight)) }
  }
  return {
    width: compactCollapsed ? capsuleWindowCompactWidth : capsuleWindowWidth,
    height: capsuleWindowCollapsedHeight }
}

function capsuleCollapsedVisualSize(
  compactCollapsed: boolean,
  dockSide: CapsuleWindowDockSide,
  workArea: CapsuleWorkArea | null,
) {
  if (isSideDock(dockSide)) {
    const requestedHeight = compactCollapsed ? capsuleWindowSideCompactHeight : capsuleWindowSideHeight
    return {
      width: Math.min(capsuleWindowSideWidth, Math.max(64, workArea?.width ?? capsuleWindowSideWidth)),
      height: Math.min(requestedHeight, Math.max(capsuleWindowCollapsedHeight, workArea?.height ?? requestedHeight)) }
  }
  return {
    width: compactCollapsed ? capsuleWindowCompactWidth : capsuleWindowWidth,
    height: capsuleWindowCollapsedHeight }
}

function capsuleReservedWindowSize(
  compactCollapsed: boolean,
  dockSide: CapsuleWindowDockSide,
  workArea: CapsuleWorkArea | null,
) {
  if (isSideDock(dockSide)) {
    const height = Math.max(
      compactCollapsed ? capsuleWindowSideCompactHeight : capsuleWindowSideHeight,
      capsuleWindowExpandedHeight,
    )
    return {
      width: Math.min(capsuleWindowSideExpandedWidth, Math.max(capsuleWindowSideWidth, workArea?.width ?? capsuleWindowSideExpandedWidth)),
      height: Math.min(height, Math.max(capsuleWindowCollapsedHeight, workArea?.height ?? height)) }
  }
  return {
    width: capsuleWindowWidth,
    height: Math.min(capsuleWindowExpandedHeight, Math.max(capsuleWindowCollapsedHeight, workArea?.height ?? capsuleWindowExpandedHeight)) }
}

function capsuleVisibleCollapsedPosition(
  dockSide: CapsuleWindowDockSide,
  position: {x: number; y: number},
  size: {width: number; height: number},
  visualSize: {width: number; height: number},
  workArea: CapsuleWorkArea | null,
) {
  if (lastCapsuleCollapsedPosition) return lastCapsuleCollapsedPosition
  const visualPosition = capsuleVisualPositionFromWindow(dockSide, position, size, visualSize)
  return clampCapsuleWindowPosition(visualPosition.x, visualPosition.y, visualSize.width, visualSize.height, workArea)
}

function capsuleVisualPositionFromWindow(
  dockSide: CapsuleWindowDockSide,
  position: {x: number; y: number},
  size: {width: number; height: number},
  visualSize: {width: number; height: number},
) {
  if (dockSide === 'left') {
    return {x: position.x, y: position.y + (size.height - visualSize.height) / 2}
  }
  if (dockSide === 'right') {
    return {x: position.x + size.width - visualSize.width, y: position.y + (size.height - visualSize.height) / 2}
  }
  if (dockSide === 'bottom' || lastCapsuleExpandedDirection === 'up') {
    return {x: position.x + (size.width - visualSize.width) / 2, y: position.y + size.height - visualSize.height}
  }
  return {x: position.x + (size.width - visualSize.width) / 2, y: position.y}
}

function capsuleReservedWindowPosition(
  dockSide: CapsuleWindowDockSide,
  visualPosition: {x: number; y: number},
  visualSize: {width: number; height: number},
  reserveSize: {width: number; height: number},
  workArea: CapsuleWorkArea | null,
  direction: CapsuleWindowExpandDirection = lastCapsuleExpandedDirection,
) {
  if (dockSide === 'left' || dockSide === 'top') {
    return capsuleDockedWindowPosition(dockSide, visualPosition, visualSize, reserveSize, workArea)
  }
  if (dockSide === 'right' || dockSide === 'bottom') {
    return capsuleDockedWindowPosition(dockSide, visualPosition, visualSize, reserveSize, workArea)
  }
  const x = visualPosition.x + (visualSize.width - reserveSize.width) / 2
  const y = direction === 'up'
    ? visualPosition.y + visualSize.height - reserveSize.height
    : visualPosition.y
  return clampCapsuleWindowPosition(x, y, reserveSize.width, reserveSize.height, workArea)
}

function capsuleDockedWindowPosition(
  dockSide: CapsuleWindowDockSide,
  position: {x: number; y: number},
  size: {width: number; height: number},
  targetSize: {width: number; height: number},
  workArea: CapsuleWorkArea | null,
) {
  if (!workArea) {
    return {
      x: Math.round(position.x + (size.width - targetSize.width) / 2),
      y: Math.round(position.y + (size.height - targetSize.height) / 2) }
  }
  const centerX = position.x + size.width / 2
  const centerY = position.y + size.height / 2
  if (dockSide === 'left') {
    return clampCapsuleWindowPosition(workArea.x, centerY - targetSize.height / 2, targetSize.width, targetSize.height, workArea)
  }
  if (dockSide === 'right') {
    return clampCapsuleWindowPosition(workArea.x + workArea.width - targetSize.width, centerY - targetSize.height / 2, targetSize.width, targetSize.height, workArea)
  }
  if (dockSide === 'top') {
    return clampCapsuleWindowPosition(centerX - targetSize.width / 2, workArea.y, targetSize.width, targetSize.height, workArea)
  }
  if (dockSide === 'bottom') {
    return clampCapsuleWindowPosition(centerX - targetSize.width / 2, workArea.y + workArea.height - targetSize.height, targetSize.width, targetSize.height, workArea)
  }
  if (dockSide === 'none' && lastCapsuleCollapsedPosition) {
    return clampCapsuleWindowPosition(
      lastCapsuleCollapsedPosition.x,
      lastCapsuleCollapsedPosition.y,
      targetSize.width,
      targetSize.height,
      workArea,
    )
  }
  return clampCapsuleWindowPosition(
    centerX - targetSize.width / 2,
    centerY - targetSize.height / 2,
    targetSize.width,
    targetSize.height,
    workArea,
  )
}

async function setCapsuleWindowBoundsIfChanged(
  currentPosition: {x: number; y: number},
  currentSize: {width: number; height: number},
  targetPosition: {x: number; y: number},
  targetSize: {width: number; height: number},
) {
  const sizeChanged = Math.abs(currentSize.width - targetSize.width) > 1 || Math.abs(currentSize.height - targetSize.height) > 1
  const positionChanged = Math.abs(currentPosition.x - targetPosition.x) > 1 || Math.abs(currentPosition.y - targetPosition.y) > 1
  if (!sizeChanged && !positionChanged) return
  const bounds = {
    x: Math.round(targetPosition.x),
    y: Math.round(targetPosition.y),
    width: Math.round(targetSize.width),
    height: Math.round(targetSize.height) }
  try {
    await RecordingFreedomService.SetCapsuleWindowBounds(bounds)
    return
  } catch (error) {
    console.info('Using runtime capsule bounds fallback:', error)
  }
  if (positionChanged) await WailsWindow.SetPosition(bounds.x, bounds.y)
  if (sizeChanged) await WailsWindow.SetSize(bounds.width, bounds.height)
}

function clampNumber(value: number, min: number, max: number) {
  if (max < min) return min
  return Math.min(max, Math.max(min, value))
}

function capsuleHitRegionsSignature(req: {
  enabled: boolean
  viewportWidth: number
  viewportHeight: number
  devicePixelRatio: number
  regions: CapsuleWindowHitRegion[]
}) {
  const viewport = [
    req.enabled ? 1 : 0,
    Math.round(req.viewportWidth),
    Math.round(req.viewportHeight),
    Math.round((req.devicePixelRatio || 1) * 100),
  ].join(':')
  const regions = req.regions
    .map((region) => [
      Math.round(region.x),
      Math.round(region.y),
      Math.round(region.width),
      Math.round(region.height),
      region.kind ?? '',
      Math.round(region.radius ?? 0),
    ].join(','))
    .join('|')
  return `${viewport}|${regions}`
}

function capsuleScreenTop() {
  const screen = window.screen as Screen & {availTop?: number}
  return Number.isFinite(screen.availTop) ? screen.availTop ?? 0 : 0
}

function capsuleScreenBottom() {
  const top = capsuleScreenTop()
  const height = Number.isFinite(window.screen.availHeight) && window.screen.availHeight > 0
    ? window.screen.availHeight
    : window.screen.height || window.innerHeight || capsuleWindowExpandedHeight
  return top + height
}

function emptyFloatingPanelState(): FloatingPanelState {
  return {
    visible: false,
    anchor: {x: 0, y: 0, width: 0, height: 0},
    bounds: {x: 0, y: 0, width: 0, height: 0},
    token: 0 }
}

function emptyFloatingSelectState(): FloatingSelectState {
  return {
    visible: false,
    anchor: {x: 0, y: 0, width: 0, height: 0},
    bounds: {x: 0, y: 0, width: 0, height: 0},
    options: [],
    token: 0 }
}

function fromBoundFloatingRect(rect: BoundFloatingRect | undefined): FloatingRect {
  return {
    x: rect?.x ?? 0,
    y: rect?.y ?? 0,
    width: rect?.width ?? 0,
    height: rect?.height ?? 0 }
}

function fromBoundFloatingPanelState(state: BoundFloatingPanelState): FloatingPanelState {
  return {
    visible: Boolean(state.visible),
    kind: normalizeFloatingPanelKind(state.kind),
    anchor: fromBoundFloatingRect(state.anchor),
    bounds: fromBoundFloatingRect(state.bounds),
    dockSide: state.dockSide,
    token: state.token ?? 0,
    screenId: state.screenId,
    direction: state.direction,
    contextId: (state as BoundFloatingPanelState & {contextId?: string}).contextId }
}

function normalizeFloatingPanelKind(value: unknown): FloatingPanelKind | undefined {
  return value === 'source' || value === 'audio' || value === 'camera' || value === 'board' || value === 'language' || value === 'settings' || value === 'close' || value === 'ocr-result'
    ? value
    : undefined
}

function fromBoundFloatingSelectOption(option: BoundFloatingSelectOption): FloatingSelectOption {
  return {
    value: option.value,
    label: option.label,
    disabled: option.disabled,
    swatch: option.swatch }
}

function fromBoundFloatingSelectState(state: BoundFloatingSelectState): FloatingSelectState {
  return {
    visible: Boolean(state.visible),
    id: state.id,
    anchor: fromBoundFloatingRect(state.anchor),
    bounds: fromBoundFloatingRect(state.bounds),
    value: state.value,
    options: (state.options ?? []).map(fromBoundFloatingSelectOption),
    token: state.token ?? 0,
    panelToken: state.panelToken,
    screenId: state.screenId,
    direction: state.direction }
}

function fromBoundFloatingSelectChosen(event: BoundFloatingSelectChosenEvent): FloatingSelectChosenEvent {
  return {
    id: event.id,
    value: event.value,
    token: event.token,
    panelToken: event.panelToken }
}

let lastCapsuleExpandedDirection: CapsuleWindowExpandDirection = 'down'

let lastCapsuleCollapsedPosition: {x: number; y: number} | null = null

let lastCapsuleDockSide: CapsuleWindowDockSide = 'none'

let lastCapsuleHitRegionsSignature = ''

function normalizeCapsuleWorkArea(rect: {X: number; Y: number; Width: number; Height: number} | undefined, id?: string): CapsuleWorkArea | null {
  if (!rect || rect.Width <= 0 || rect.Height <= 0) return null
  return {id, x: rect.X, y: rect.Y, width: rect.Width, height: rect.Height}
}

function clampCapsuleWindowPosition(
  x: number,
  y: number,
  width: number,
  height: number,
  workArea: CapsuleWorkArea | null,
) {
  if (!workArea) return {x, y}
  const maxX = Math.max(workArea.x, workArea.x + workArea.width - width)
  const maxY = Math.max(workArea.y, workArea.y + workArea.height - height)
  return {
    x: Math.round(clampNumber(x, workArea.x, maxX)),
    y: Math.round(clampNumber(y, workArea.y, maxY)) }
}
