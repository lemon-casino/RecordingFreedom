import {Application, Events, Screens, Window as WailsWindow} from '@wailsio/runtime'
import {RecordingFreedomService} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app'
import {
  type AnnotationCaptureRequest as BoundAnnotationCaptureRequest,
  type AnnotationCaptureResult as BoundAnnotationCaptureResult,
  type AnnotationOverlayState as BoundAnnotationOverlayState,
  type AnnotationPreviewImageRequest as BoundAnnotationPreviewImageRequest,
  type AnnotationPreviewImageResult as BoundAnnotationPreviewImageResult,
  type AnnotationRenderJobClaim as BoundAnnotationRenderJobClaim,
  type AnnotationRenderJobResult as BoundAnnotationRenderJobResult,
  type AudioState as BoundAudioState,
  type AudioStatePatchRequest as BoundAudioStatePatchRequest,
  type BootstrapState as BoundBootstrapState,
  type CapsuleWindowHitRegionsRequest as BoundCapsuleWindowHitRegionsRequest,
  type ExportRecordingPlanResult as BoundExportRecordingPlanResult,
  type ExportRecordingResult as BoundExportRecordingResult,
  type PIPPreviewImageRequest as BoundPIPPreviewImageRequest,
  type PIPPreviewImageResult as BoundPIPPreviewImageResult,
  type PIPOverlayRequest as BoundPIPOverlayRequest,
  type PIPOverlayState as BoundPIPOverlayState,
  type RegionSelectionRequest as BoundRegionSelectionRequest,
  type RegionSelectionResult as BoundRegionSelectionResult,
  type RegionSelectionSession as BoundRegionSelectionSession,
  type ScreenIndicatorRequest as BoundScreenIndicatorRequest,
  type ScreenIndicatorResult as BoundScreenIndicatorResult,
  type ScreenshotCaptureRequest as BoundScreenshotCaptureRequest,
  type ScreenshotCaptureResult as BoundScreenshotCaptureResult,
  type ScreenshotHistoryResult as BoundScreenshotHistoryResult,
  type ScreenshotImageRequest as BoundScreenshotImageRequest,
  type ScreenshotImageResult as BoundScreenshotImageResult,
  type ScreenshotItem as BoundScreenshotItem,
  type ScreenshotItemPatchRequest as BoundScreenshotItemPatchRequest,
  type ScreenshotPinState as BoundScreenshotPinState,
  type ScreenshotWhiteboardContext as BoundScreenshotWhiteboardContext,
  type SettingsPreferencesPatchRequest as BoundSettingsPreferencesPatchRequest,
  type ShortcutSettingsPatchRequest as BoundShortcutSettingsPatchRequest,
  type WhiteboardExportRequest as BoundWhiteboardExportRequest,
  type WhiteboardExportResult as BoundWhiteboardExportResult,
  type WhiteboardSceneRequest as BoundWhiteboardSceneRequest,
  type WhiteboardSceneResult as BoundWhiteboardSceneResult,
  type WhiteboardSettingsPatchRequest as BoundWhiteboardSettingsPatchRequest,
} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/models'
import {
  type Capabilities as BoundCaptureCapabilities,
  type Capability as BoundCaptureCapability,
} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/capture/models'
import {
  type Plan as BoundExportPlan,
} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/exportplan/models'
import {
  CaptureSourceType as BoundCaptureSourceType,
  type CaptureSource as BoundCaptureSource,
  type MediaDevice as BoundMediaDevice,
  type MediaInventory as BoundMediaInventory,
} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/devices/models'
import {
  type AudioOnlyRequest,
  type Session as BoundSession,
  type StartRequest,
  type StatusEvent as BoundStatusEvent,
} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/recording/models'
import {type Summary as BoundPreflightSummary} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/preflight/models'
import {type RecoverySummary as BoundRecoverySummary} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/recpackage/models'
import {type Settings as BoundSettings} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/settings/models'
import {
  createMockRecordingPackage,
  createMockAudioOnlyRecordingPackage,
  defaultSettings,
  fallbackAppData,
  fallbackCapabilities,
  fallbackStorageStatus,
  mediaInventory as fallbackMediaInventory,
  normalizeLocale,
  normalizeShortcutSettings,
  normalizeTheme,
  shortcutActions,
  sources as fallbackSources,
  type AppSettings,
  type AudioOnlyRecordingRequest,
  type AppDataInfo,
  type AppStorageStatus,
  type CaptureCapabilities,
  type CaptureCapability,
  type CaptureSource,
  type MediaDevice,
  type MediaInventory,
  type MockRecordingRequest,
  type PIPConfig,
  type RecordingPreflight,
  type ShortcutAction,
  type ShortcutSettings,
  type ScreenshotItem,
} from './mockBackend'

export type RecordingSession = {
  id: string
  packagePath: string
  manifestPath?: string
  backend?: string
  recordingMode?: string
  status?: string
}

export type RecordingStatusUpdate = {
  status: string
  message?: string
  backend?: string
  session?: RecordingSession
}

export type RecordingRecovery = {
  packagePath: string
  manifestPath?: string
  status: string
  recoverable: boolean
  reason?: string
}

export type AudioLevelUpdate = {
  deviceId: string
  level: number
  rms: number
  peak: number
  active: boolean
  error?: string
}

export type AudioControlState = {
  system: boolean
  systemDeviceId?: string
  microphone: boolean
  microphoneDeviceId?: string
  noiseSuppression: boolean
  microphoneGain: number
}

export type AudioStatePatch = {
  system?: boolean
  systemDeviceId?: string
  microphone?: boolean
  microphoneDeviceId?: string
  noiseSuppression?: boolean
  microphoneGain?: number
  clearSystemDevice?: boolean
  clearMicrophoneDevice?: boolean
}

export type SettingsPreferencesPatch = {
  theme?: AppSettings['window']['theme']
  recordingQuality?: AppSettings['recording']['quality']
  recordingFps?: number
  captureCursor?: boolean
  countdownSeconds?: number
}

export type ShortcutSettingsPatch = Partial<ShortcutSettings>

export type ShortcutTriggeredUpdate = {
  action: ShortcutAction
  accelerator: string
}

export type ScreenshotImage = {
  available: boolean
  dataUrl?: string
  path?: string
  bytes?: number
}

export type ScreenshotPinState = {
  visible: boolean
  item?: ScreenshotItem
  dataUrl?: string
  fixed: boolean
}

export type ScreenshotWhiteboardContext = {
  available: boolean
  item?: ScreenshotItem
  dataUrl?: string
}

export type WhiteboardScene = {
  available: boolean
  scenePath: string
  sceneJson?: string
  bytes: number
  updatedAt?: string
  contentType?: string
}

export type WhiteboardExport = {
  format: 'png' | 'svg' | 'excalidraw'
  outputPath: string
  bytes: number
}

export type WhiteboardSettingsPatch = Partial<AppSettings['whiteboard']>

export type AnnotationOverlayState = {
  mode?: 'annotation' | 'screenshot'
  packageDir?: string
  manifestPath?: string
  windowBounds: {x: number; y: number; width: number; height: number}
  canvasBounds: {x: number; y: number; width: number; height: number}
  target: {
    type: string
    id: string
    geometry?: {x: number; y: number; width: number; height: number; displayIndex?: number; nativeId?: string}
  }
  captureExcluded: boolean
}

export type AnnotationCapture = {
  packageDir: string
  scenePath: string
  eventsPath: string
  snapshotPath: string
  timelineSnapshotPath?: string
  bytes: number
}

export type AnnotationRenderJob = {
  id: string
  packageDir: string
  scenePath: string
  relativeScenePath: string
  outputPath: string
  relativeOutputPath: string
  sceneJson: string
  canvasWidth: number
  canvasHeight: number
  index: number
  startOffsetMs?: number
  endOffsetMs?: number
}

export type AnnotationRenderJobClaim = {
  available: boolean
  job?: AnnotationRenderJob
}

export type RecorderBootstrap = {
  appData: AppDataInfo
  storage: AppStorageStatus
  state: string
  backend: string
  sources: CaptureSource[]
  media: MediaInventory
  recoveries: RecordingRecovery[]
  settings: AppSettings
  capabilities: CaptureCapabilities
}

export type CapsuleWindowHitRegion = {
  x: number
  y: number
  width: number
  height: number
  kind?: 'rect' | 'round-rect' | 'pill'
  radius?: number
}

const browserSettingsKey = 'recordingfreedom.settings.v1'
const browserWhiteboardSceneKey = 'recordingfreedom.whiteboard.scene.v1'
const browserAnnotationSceneKey = 'recordingfreedom.annotation.scene.v1'
const browserScreenshotHistoryKey = 'recordingfreedom.screenshots.history.v1'
const browserScreenshotPinStateKey = 'recordingfreedom.screenshots.pin.v1'
const browserScreenshotWhiteboardKey = 'recordingfreedom.screenshots.whiteboard.v1'
const browserScreenshotAnnotationKey = 'recordingfreedom.screenshots.annotation.v1'
const browserWhiteboardVisibilityEvent = 'rf-whiteboard-visibility'
const browserScreenshotCapturedEvent = 'rf-screenshot-captured'
const browserScreenshotPinEvent = 'rf-screenshot-pin'
const browserCapsuleDockSideEvent = 'rf-capsule-dock-side'
const capsuleWindowWidth = 760
const capsuleWindowCompactWidth = 380
const capsuleWindowCollapsedHeight = 96
const capsuleWindowExpandedHeight = 600
const capsuleWindowSideWidth = 96
const capsuleWindowSideHeight = 560
const capsuleWindowSideCompactHeight = 360
const capsuleWindowSideExpandedWidth = 520
const capsuleSideSnapThreshold = 140
const capsuleSideCenterSnapThreshold = 96
const capsuleEdgeSnapThreshold = 48
export type CapsuleWindowExpandDirection = 'down' | 'up'
export type CapsuleWindowDockSide = 'none' | 'left' | 'right' | 'top' | 'bottom'

function isWailsDesktopRuntime(): boolean {
  if (window.navigator.userAgent.includes('Wails')) return true
  return window.location.hostname === 'wails.localhost'
}

let lastCapsuleExpandedDirection: CapsuleWindowExpandDirection = 'down'
let lastCapsuleCollapsedPosition: {x: number; y: number} | null = null
let lastCapsuleDockSide: CapsuleWindowDockSide = 'none'
let lastCapsuleHitRegionsSignature = ''
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
  try {
    await WailsWindow.Show()
    await WailsWindow.UnMinimise().catch(() => undefined)
    await WailsWindow.SetAlwaysOnTop(true).catch(() => undefined)
    if (focus) await WailsWindow.Focus().catch(() => undefined)
  } catch (error) {
    console.info('Using browser capsule window restore fallback:', error)
  }
}

export async function logClientEvent(component: string, event: string, fields: Record<string, unknown> = {}, message = ''): Promise<void> {
  try {
    await RecordingFreedomService.LogClientEvent({
      component,
      event,
      message,
      fields: Object.fromEntries(Object.entries(fields).map(([key, value]) => [key, String(value ?? '')])),
    })
  } catch (error) {
    console.info('Using browser client log fallback:', error)
  }
}

export type RegionSelectionSession = {
  id: string
  bounds: {x: number; y: number; width: number; height: number}
  captureBounds?: {x: number; y: number; width: number; height: number}
  minimumWidth: number
  minimumHeight: number
  displayCount: number
  purpose?: 'capture' | 'annotation' | 'screenshot'
}

export type RegionSelectionResult = {
  sessionId?: string
  source?: CaptureSource
  geometry?: {x: number; y: number; width: number; height: number}
  cancelled: boolean
  error?: string
}

export type ScreenIndicatorResult = {
  sourceId: string
  displayIndex: number
  label: string
  sourceBounds: {x: number; y: number; width: number; height: number}
  windowBounds: {x: number; y: number; width: number; height: number}
}

export type PIPOverlayMode = 'edit' | 'recording'

export type PIPOverlayCamera = {
  deviceId?: string
  nativeId?: string
  name?: string
}

export type PIPOverlayState = {
  config: PIPConfig
  placement: {
    visible: boolean
    rect: {x: number; y: number; width: number; height: number; visible?: boolean}
    shape: PIPConfig['shape']
    mirror: boolean
    edgeFeather: number
  }
  overlayBounds: {x: number; y: number; width: number; height: number}
  windowBounds: {x: number; y: number; width: number; height: number}
  contentBounds: {x: number; y: number; width: number; height: number}
  mode: PIPOverlayMode
  cameraName?: string
  camera?: PIPOverlayCamera
  previewImagePath?: string
  captureExcluded: boolean
  clientOperationId?: number
}

export type PIPPreviewImage = {
  available: boolean
  dataUrl?: string
  modifiedUnixNano?: number
}

export type AnnotationPreviewImage = {
  available: boolean
  dataUrl?: string
  relativePath?: string
  bytes?: number
}

export type WhiteboardVisibilityMode = 'whiteboard' | 'annotation'

export type WhiteboardVisibilityUpdate = {
  visible: boolean
  mode: WhiteboardVisibilityMode
}

export type RecordingExportPlan = {
  packageDir: string
  outputPath: string
  screenInputPath: string
  webcamInputPath?: string
  pipVisible: boolean
  annotationsVisible: boolean
  annotationInputPath?: string
  annotationEventsPath?: string
  annotationStartMs?: number
  annotationTimeline?: string
  annotationRenderMode?: string
  annotationSnapshots?: Array<{
    inputPath: string
    relativePath?: string
    startOffsetMs: number
    endOffsetMs?: number
    durationMs?: number
    bytes?: number
  }>
  annotationElementScenes?: Array<{
    inputPath: string
    relativePath?: string
    renderInputPath?: string
    renderRelativePath?: string
    startOffsetMs?: number
    endOffsetMs?: number
    durationMs?: number
    canvasWidth?: number
    canvasHeight?: number
    elementCount?: number
    sourceEventSequence?: number
    bytes?: number
  }>
  annotationSummary?: {
    mode?: string
    eventCount?: number
    snapshotCount?: number
    exportedSnapshotCount?: number
    skippedSnapshotCount?: number
    elementEventCount?: number
    elementTimelineMode?: string
    elementKeyframeCount?: number
    finalElementCount?: number
    deletedElementCount?: number
    missingElementPayloads?: number
    startOffsetMs?: number
    endOffsetMs?: number
    eventFileBytes?: number
    snapshotBytes?: number
    elementTypeCounts?: Record<string, number>
    elementPreviewFrames?: Array<{
      sequence?: number
      startOffsetMs?: number
      eventType?: string
      elementId?: string
      elementType?: string
      activeElementCount?: number
      hasElementPayload: boolean
    }>
  }
  warnings: string[]
}

export type RecordingExportResult = RecordingExportPlan & {
  bytes: number
  ffmpegPath?: string
  outputVerified: boolean
}

export type RecordingExportOptions = {
  includeAnnotations?: boolean
}

export async function setCapsuleWindowExpanded(
  expanded: boolean,
  expandedHeight = capsuleWindowExpandedHeight,
  preferredDirection: CapsuleWindowExpandDirection | 'auto' = 'auto',
  compactCollapsed = false,
): Promise<CapsuleWindowExpandDirection> {
  try {
    await restoreCapsuleWindow(false)
    const position = await WailsWindow.Position()
    const size = await WailsWindow.Size().catch(() => ({
      width: expanded ? capsuleWindowWidth : compactCollapsed ? capsuleWindowCompactWidth : capsuleWindowWidth,
      height: expanded ? expandedHeight : capsuleWindowCollapsedHeight,
    }))
    const workArea = await capsuleWorkAreaForPosition(position, size).catch(() => null)
    const dockSide = lastCapsuleDockSide
    const collapsedVisualSize = capsuleCollapsedWindowSize(compactCollapsed, dockSide, workArea)
    const collapsedVisualPosition = capsuleVisibleCollapsedPosition(dockSide, position, size, collapsedVisualSize, workArea)
    const reservedWindowSize = capsuleReservedWindowSize(compactCollapsed, dockSide, workArea)
    if (!expanded) {
      const reservedPosition = capsuleReservedWindowPosition(dockSide, collapsedVisualPosition, collapsedVisualSize, reservedWindowSize, workArea)
      await setCapsuleWindowBoundsIfChanged(position, size, reservedPosition, reservedWindowSize)
      await restoreCapsuleWindow(false)
      lastCapsuleCollapsedPosition = null
      return lastCapsuleExpandedDirection
    }

    if (isSideDock(dockSide)) {
      const targetExpandedSize = capsuleReservedWindowSize(compactCollapsed, dockSide, workArea)
      const expandedPosition = capsuleReservedWindowPosition(dockSide, collapsedVisualPosition, collapsedVisualSize, targetExpandedSize, workArea)
      lastCapsuleExpandedDirection = 'down'
      lastCapsuleCollapsedPosition = collapsedVisualPosition
      await setCapsuleWindowBoundsIfChanged(position, size, expandedPosition, targetExpandedSize)
      await restoreCapsuleWindow(false)
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
    await restoreCapsuleWindow(false)
    return direction
  } catch (error) {
    console.info('Using browser capsule window size fallback:', error)
    return preferredDirection === 'up' ? 'up' : 'down'
  }
}

export async function snapCapsuleWindowToEdge(compactCollapsed = false): Promise<CapsuleWindowDockSide> {
  try {
    await restoreCapsuleWindow(false)
    const position = await WailsWindow.Position()
    const size = await WailsWindow.Size().catch(() => capsuleCollapsedWindowSize(compactCollapsed, lastCapsuleDockSide, null))
    const workAreas = await capsuleWorkAreas().catch(() => [])
    const workArea = capsuleWorkAreaForPositionFromAreas(position, size, workAreas)
    const currentVisualSize = capsuleCollapsedWindowSize(compactCollapsed, lastCapsuleDockSide, workArea)
    lastCapsuleCollapsedPosition = null
    const currentVisualPosition = capsuleVisibleCollapsedPosition(lastCapsuleDockSide, position, size, currentVisualSize, workArea)
    const dockTarget = resolveCapsuleDockTarget(currentVisualPosition, currentVisualSize, workAreas)
    const dockSide = dockTarget.side
    const targetWorkArea = dockTarget.workArea ?? workArea
    publishCapsuleDockSide(dockSide)
    lastCapsuleExpandedDirection = dockSide === 'bottom' ? 'up' : 'down'
    await waitForAnimationFrame()
    await waitForAnimationFrame()
    const targetSize = capsuleReservedWindowSize(compactCollapsed, dockSide, targetWorkArea)
    const targetPosition = capsuleReservedWindowPosition(dockSide, currentVisualPosition, currentVisualSize, targetSize, targetWorkArea)
    await setCapsuleWindowBoundsIfChanged(position, size, targetPosition, targetSize)
    await restoreCapsuleWindow(false)
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

function waitForAnimationFrame() {
  return new Promise<void>((resolve) => window.requestAnimationFrame(() => resolve()))
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
): CapsuleDockTarget {
  if (!workAreas.length) return {side: 'none', workArea: null}
  const vertical = workAreas
    .flatMap((area) => [
      capsuleDockCandidate('left', position, size, area),
      capsuleDockCandidate('right', position, size, area),
    ])
    .filter((candidate): candidate is CapsuleDockCandidate => candidate !== null)
  vertical.sort(compareCapsuleDockCandidates)
  if (vertical[0]) return {side: vertical[0].side, workArea: vertical[0].workArea}

  const horizontal = workAreas
    .flatMap((area) => [
      capsuleDockCandidate('top', position, size, area),
      capsuleDockCandidate('bottom', position, size, area),
    ])
    .filter((candidate): candidate is CapsuleDockCandidate => candidate !== null)
  horizontal.sort(compareCapsuleDockCandidates)
  if (horizontal[0]) return {side: horizontal[0].side, workArea: horizontal[0].workArea}
  return {side: 'none', workArea: capsuleWorkAreaForPositionFromAreas(position, size, workAreas)}
}

type CapsuleWorkArea = {
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

async function capsuleWorkAreaForPosition(
  position: {x: number; y: number},
  size: {width: number; height: number},
): Promise<CapsuleWorkArea | null> {
  return capsuleWorkAreaForPositionFromAreas(position, size, await capsuleWorkAreas())
}

async function capsuleWorkAreas(): Promise<CapsuleWorkArea[]> {
  const screens = await Screens.GetAll()
  return screens
    .map((screen) => normalizeCapsuleWorkArea(screen.WorkArea) ?? normalizeCapsuleWorkArea(screen.Bounds))
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

function normalizeCapsuleWorkArea(rect: {X: number; Y: number; Width: number; Height: number} | undefined): CapsuleWorkArea | null {
  if (!rect || rect.Width <= 0 || rect.Height <= 0) return null
  return {x: rect.X, y: rect.Y, width: rect.Width, height: rect.Height}
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
  const centerDistance = sideIsVertical
    ? Math.abs(centerX - edgePosition)
    : Math.abs(centerY - edgePosition)
  const active = sideIsVertical
    ? edgeDistance <= capsuleSideSnapThreshold || centerDistance <= capsuleSideCenterSnapThreshold
    : edgeDistance <= capsuleEdgeSnapThreshold
  if (!active) return null

  return {
    side,
    workArea,
    distance: sideIsVertical ? Math.min(edgeDistance, centerDistance + 24) : edgeDistance,
    overlapRatio,
    originInside: pointInsideWorkAreaOpenEnd(position.x, centerY, workArea),
    centerInside: pointInsideWorkAreaOpenEnd(centerX, centerY, workArea),
    screenDistance: distanceToWorkArea(centerX, centerY, workArea),
  }
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
    return {
      width: Math.min(capsuleWindowSideWidth, Math.max(64, workArea?.width ?? capsuleWindowSideWidth)),
      height: Math.min(requestedHeight, Math.max(capsuleWindowCollapsedHeight, workArea?.height ?? requestedHeight)),
    }
  }
  return {
    width: compactCollapsed ? capsuleWindowCompactWidth : capsuleWindowWidth,
    height: capsuleWindowCollapsedHeight,
  }
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
      height: Math.min(height, Math.max(capsuleWindowCollapsedHeight, workArea?.height ?? height)),
    }
  }
  return {
    width: capsuleWindowWidth,
    height: Math.min(capsuleWindowExpandedHeight, Math.max(capsuleWindowCollapsedHeight, workArea?.height ?? capsuleWindowExpandedHeight)),
  }
}

function capsuleVisibleCollapsedPosition(
  dockSide: CapsuleWindowDockSide,
  position: {x: number; y: number},
  size: {width: number; height: number},
  visualSize: {width: number; height: number},
  workArea: CapsuleWorkArea | null,
) {
  if (lastCapsuleCollapsedPosition) return lastCapsuleCollapsedPosition
  if (dockSide === 'left') {
    return clampCapsuleWindowPosition(position.x, position.y + (size.height - visualSize.height) / 2, visualSize.width, visualSize.height, workArea)
  }
  if (dockSide === 'right') {
    return clampCapsuleWindowPosition(position.x + size.width - visualSize.width, position.y + (size.height - visualSize.height) / 2, visualSize.width, visualSize.height, workArea)
  }
  if (dockSide === 'bottom' || lastCapsuleExpandedDirection === 'up') {
    return clampCapsuleWindowPosition(position.x + (size.width - visualSize.width) / 2, position.y + size.height - visualSize.height, visualSize.width, visualSize.height, workArea)
  }
  return clampCapsuleWindowPosition(position.x + (size.width - visualSize.width) / 2, position.y, visualSize.width, visualSize.height, workArea)
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
      y: Math.round(position.y + (size.height - targetSize.height) / 2),
    }
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
  if (sizeChanged) await WailsWindow.SetSize(Math.round(targetSize.width), Math.round(targetSize.height))
  if (positionChanged) await WailsWindow.SetPosition(Math.round(targetPosition.x), Math.round(targetPosition.y))
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
    y: Math.round(clampNumber(y, workArea.y, maxY)),
  }
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

export async function quitApplication(): Promise<void> {
  try {
    await Application.Quit()
  } catch (error) {
    console.info('Using browser quit fallback:', error)
    window.close()
  }
}

export async function loadBootstrap(): Promise<RecorderBootstrap> {
  try {
    return fromBoundBootstrap(await RecordingFreedomService.Bootstrap())
  } catch (error) {
    console.info('Using browser mock bootstrap:', error)
    return {
      appData: fallbackAppData,
      storage: fallbackStorageStatus,
      state: 'idle',
      backend: 'browser-mock',
      sources: fallbackSources,
      media: fallbackMediaInventory,
      recoveries: [],
      settings: loadBrowserSettings(),
      capabilities: fallbackCapabilities,
    }
  }
}

export function subscribeRecordingStatus(handler: (event: RecordingStatusUpdate) => void): () => void {
  try {
    return Events.On('recording.status', (event) => {
      handler(fromBoundStatusEvent(event.data as BoundStatusEvent))
    })
  } catch (error) {
    console.info('Using browser recording event fallback:', error)
    return () => {}
  }
}

export function subscribeRegionSelection(handler: (event: RegionSelectionResult) => void): () => void {
  try {
    return Events.On('capture.region.selected', (event) => {
      handler(fromBoundRegionSelectionResult(event.data as BoundRegionSelectionResult))
    })
  } catch (error) {
    console.info('Using browser region selection event fallback:', error)
    return () => {}
  }
}

export function subscribeSettingsChanged(handler: (settings: AppSettings) => void): () => void {
  try {
    return Events.On('settings.changed', (event) => {
      handler(fromBoundSettings(event.data as BoundSettings))
    })
  } catch (error) {
    console.info('Using browser settings event fallback:', error)
    return () => {}
  }
}

export function subscribeWhiteboardVisibility(handler: (event: WhiteboardVisibilityUpdate) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('whiteboard.visibility', (event) => {
      handler(fromWhiteboardVisibilityEvent(event.data))
    })
  } catch (error) {
    console.info('Using browser whiteboard visibility event fallback:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler(fromWhiteboardVisibilityEvent((event as CustomEvent<WhiteboardVisibilityUpdate>).detail))
  }
  window.addEventListener(browserWhiteboardVisibilityEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserWhiteboardVisibilityEvent, onBrowserEvent)
  }
}

export function subscribeAudioLevel(handler: (event: AudioLevelUpdate) => void): () => void {
  try {
    return Events.On('audio.level', (event) => {
      handler(fromBoundAudioLevel(event.data))
    })
  } catch (error) {
    console.info('Desktop microphone level events unavailable:', error)
    return () => {}
  }
}

export function subscribeAudioState(handler: (event: AudioControlState) => void): () => void {
  try {
    return Events.On('audio.state', (event) => {
      handler(fromBoundAudioState(event.data as BoundAudioState))
    })
  } catch (error) {
    console.info('Desktop audio state events unavailable:', error)
    return () => {}
  }
}

export function subscribeShortcutTriggered(handler: (event: ShortcutTriggeredUpdate) => void): () => void {
  try {
    return Events.On('shortcut.triggered', (event) => {
      handler(fromShortcutTriggeredEvent(event.data))
    })
  } catch (error) {
    console.info('Desktop shortcut events unavailable:', error)
    return () => {}
  }
}

export function subscribeScreenshotCaptured(handler: (item: ScreenshotItem) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('screenshot.captured', (event) => {
      const record = event.data && typeof event.data === 'object' ? event.data as {item?: BoundScreenshotItem} : {}
      if (record.item) handler(fromBoundScreenshotItem(record.item))
    })
  } catch (error) {
    console.info('Desktop screenshot captured events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    const item = (event as CustomEvent<ScreenshotItem>).detail
    if (item) handler(item)
  }
  window.addEventListener(browserScreenshotCapturedEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserScreenshotCapturedEvent, onBrowserEvent)
  }
}

export function subscribeScreenshotPin(handler: (state: ScreenshotPinState) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('screenshot.pin', (event) => {
      handler(fromBoundScreenshotPinState(event.data as BoundScreenshotPinState))
    })
  } catch (error) {
    console.info('Desktop screenshot pin events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler(fromBrowserScreenshotPinState((event as CustomEvent<ScreenshotPinState>).detail))
  }
  window.addEventListener(browserScreenshotPinEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserScreenshotPinEvent, onBrowserEvent)
  }
}

export async function patchAudioState(patch: AudioStatePatch): Promise<AudioControlState> {
  try {
    return fromBoundAudioState(await RecordingFreedomService.PatchAudioState(toBoundAudioStatePatch(patch)))
  } catch (error) {
    console.info('Using browser audio state patch fallback:', error)
    const current = loadBrowserSettings()
    const nextAudio = applyBrowserAudioPatch(current.audio, patch)
    const next = {...current, audio: nextAudio, updatedAt: new Date().toISOString()}
    window.localStorage?.setItem(browserSettingsKey, JSON.stringify(next))
    return {
      system: nextAudio.system,
      systemDeviceId: nextAudio.systemDeviceId,
      microphone: nextAudio.microphone,
      microphoneDeviceId: nextAudio.microphoneDeviceId,
      noiseSuppression: nextAudio.microphone && nextAudio.noiseSuppression,
      microphoneGain: nextAudio.microphoneGain,
    }
  }
}

export async function patchSettingsPreferences(patch: SettingsPreferencesPatch): Promise<AppSettings> {
  try {
    return fromBoundSettings(await RecordingFreedomService.PatchSettingsPreferences(toBoundSettingsPreferencesPatch(patch)))
  } catch (error) {
    console.info('Using browser settings preference patch fallback:', error)
    const current = loadBrowserSettings()
    const next = applyBrowserSettingsPreferencesPatch(current, patch)
    window.localStorage?.setItem(browserSettingsKey, JSON.stringify(next))
    return next
  }
}

export async function patchShortcutSettings(patch: ShortcutSettingsPatch): Promise<AppSettings> {
  try {
    return fromBoundSettings(await RecordingFreedomService.PatchShortcutSettings(patch as BoundShortcutSettingsPatchRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser shortcut settings fallback:', error)
    const next = applyBrowserShortcutPatch(loadBrowserSettings(), patch)
    window.localStorage?.setItem(browserSettingsKey, JSON.stringify(next))
    return next
  }
}

export async function startMicrophoneLevelMonitor(deviceId?: string): Promise<void> {
  try {
    await RecordingFreedomService.StartMicrophoneLevelMonitor(deviceId || 'microphone:default')
  } catch (error) {
    console.info('Desktop microphone level monitor unavailable:', error)
    throw error
  }
}

export async function stopMicrophoneLevelMonitor(): Promise<void> {
  try {
    await RecordingFreedomService.StopMicrophoneLevelMonitor()
  } catch (error) {
    console.info('Desktop microphone level monitor stop fallback:', error)
  }
}

export async function showSettingsWindow(): Promise<void> {
  try {
    await RecordingFreedomService.ShowSettingsWindow()
  } catch (error) {
    console.info('Using browser settings window fallback:', error)
    const popup = window.open('/#/settings', 'recordingfreedom-settings', 'width=920,height=720')
    popup?.focus()
  }
}

export async function showWhiteboardWindow(): Promise<void> {
  try {
    await RecordingFreedomService.ShowWhiteboardWindow()
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard window fallback:', error)
    ;(window as Window & {__RF_LAST_WHITEBOARD_LAUNCH__?: {mode: string; url: string; at: string}}).__RF_LAST_WHITEBOARD_LAUNCH__ = {
      mode: 'whiteboard',
      url: '/#/whiteboard',
      at: new Date().toISOString(),
    }
    emitBrowserWhiteboardVisibility({visible: true, mode: 'whiteboard'})
    const popup = window.open('/#/whiteboard', 'recordingfreedom-whiteboard', 'width=1120,height=760')
    popup?.focus()
  }
}

export async function showAnnotationOverlay(): Promise<AnnotationOverlayState> {
  try {
    return fromBoundAnnotationOverlayState(await RecordingFreedomService.ShowAnnotationOverlay())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation overlay fallback:', error)
    ;(window as Window & {__RF_LAST_WHITEBOARD_LAUNCH__?: {mode: string; url: string; at: string}}).__RF_LAST_WHITEBOARD_LAUNCH__ = {
      mode: 'annotation',
      url: '/#/annotation-overlay',
      at: new Date().toISOString(),
    }
    emitBrowserWhiteboardVisibility({visible: true, mode: 'annotation'})
    const popup = window.open('/#/annotation-overlay', 'recordingfreedom-annotation-overlay', 'width=1280,height=720')
    popup?.focus()
    return browserAnnotationOverlayState()
  }
}

export async function showAnnotationRegionSelector(): Promise<RegionSelectionSession> {
  try {
    return fromBoundRegionSelectionSession(await RecordingFreedomService.ShowAnnotationRegionSelector())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation region selector fallback:', error)
    return {
      id: `browser-annotation-region-${Date.now()}`,
      bounds: {x: 0, y: 0, width: window.innerWidth, height: window.innerHeight},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'annotation',
    }
  }
}

export async function completeAnnotationRegionSelection(request: RegionSelectionSession['bounds']): Promise<AnnotationOverlayState> {
  try {
    return fromBoundAnnotationOverlayState(await RecordingFreedomService.CompleteAnnotationRegionSelection(toBoundRegionSelectionRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation region completion fallback:', error)
    return browserAnnotationOverlayState()
  }
}

export async function showScreenshotRegionSelector(): Promise<RegionSelectionSession> {
  try {
    return fromBoundRegionSelectionSession(await RecordingFreedomService.ShowScreenshotRegionSelector())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot region selector fallback:', error)
    return {
      id: `browser-screenshot-${Date.now()}`,
      bounds: {x: 0, y: 0, width: window.innerWidth, height: window.innerHeight},
      captureBounds: {x: 0, y: 0, width: window.innerWidth, height: window.innerHeight},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
      purpose: 'screenshot',
    }
  }
}

export async function completeScreenshotRegionSelection(request: RegionSelectionSession['bounds']): Promise<ScreenshotItem> {
  try {
    const result = await RecordingFreedomService.CompleteScreenshotRegionSelection(toBoundRegionSelectionRequest(request))
    return fromBoundScreenshotCaptureResult(result as BoundScreenshotCaptureResult).item
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot completion fallback:', error)
    const item = createBrowserScreenshotItem('region', request)
    saveBrowserScreenshotHistory([item, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: item}))
    return item
  }
}

export async function beginScreenshotRegionEdit(request: RegionSelectionSession['bounds']): Promise<void> {
  try {
    await RecordingFreedomService.BeginScreenshotRegionEdit(toBoundRegionSelectionRequest(request))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot region edit fallback:', error)
    emitBrowserRegionFrame(request, 'screenshot')
  }
}

export async function beginScreenshotAnnotationOverlay(request: RegionSelectionSession['bounds']): Promise<AnnotationOverlayState> {
  try {
    return fromBoundAnnotationOverlayState(await RecordingFreedomService.BeginScreenshotAnnotationOverlay(toBoundRegionSelectionRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot annotation overlay fallback:', error)
    const item = createBrowserScreenshotItem('region', request)
    const context: ScreenshotWhiteboardContext = {
      available: true,
      item,
      dataUrl: browserScreenshotDataUrl(item),
    }
    window.localStorage?.setItem(browserScreenshotAnnotationKey, JSON.stringify(context))
    const state = browserAnnotationOverlayState('screenshot', request)
    ;(window as Window & {__RF_ANNOTATION_OVERLAY__?: AnnotationOverlayState}).__RF_ANNOTATION_OVERLAY__ = state
    ;(window as Window & {__RF_LAST_WHITEBOARD_LAUNCH__?: {mode: string; url: string; at: string}}).__RF_LAST_WHITEBOARD_LAUNCH__ = {
      mode: 'screenshot',
      url: '/#/annotation-overlay',
      at: new Date().toISOString(),
    }
    emitBrowserWhiteboardVisibility({visible: true, mode: 'annotation'})
    const popup = window.open('/#/annotation-overlay', 'recordingfreedom-screenshot-annotation', 'width=1280,height=720')
    popup?.focus()
    return state
  }
}

export async function updateScreenshotRegionSelection(request: RegionSelectionSession['bounds']): Promise<void> {
  try {
    await RecordingFreedomService.UpdateScreenshotRegionSelection(toBoundRegionSelectionRequest(request))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot region update fallback:', error)
    emitBrowserRegionFrame(request, 'screenshot')
  }
}

export async function captureScreenshot(request: {mode?: string; region?: RegionSelectionSession['bounds']} = {}): Promise<ScreenshotItem> {
  try {
    const result = await RecordingFreedomService.CaptureScreenshot(request as unknown as BoundScreenshotCaptureRequest)
    return fromBoundScreenshotCaptureResult(result as BoundScreenshotCaptureResult).item
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot capture fallback:', error)
    const item = createBrowserScreenshotItem(request.mode ?? 'full', request.region)
    saveBrowserScreenshotHistory([item, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: item}))
    return item
  }
}

export async function reselectAnnotationRegion(): Promise<RegionSelectionSession> {
  try {
    return fromBoundRegionSelectionSession(await RecordingFreedomService.ReselectAnnotationRegion())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation region reselect fallback:', error)
    window.localStorage?.removeItem(browserAnnotationSceneKey)
    const session = await showAnnotationRegionSelector()
    ;(window as Window & {__RF_LAST_ANNOTATION_REGION_RESELECT__?: RegionSelectionSession}).__RF_LAST_ANNOTATION_REGION_RESELECT__ = session
    return session
  }
}

export async function reselectScreenshotAnnotationRegion(): Promise<RegionSelectionSession> {
  try {
    return fromBoundRegionSelectionSession(await RecordingFreedomService.ReselectScreenshotAnnotationRegion())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot annotation region reselect fallback:', error)
    window.localStorage?.removeItem(browserScreenshotAnnotationKey)
    const session = await showScreenshotRegionSelector()
    ;(window as Window & {__RF_LAST_SCREENSHOT_REGION_RESELECT__?: RegionSelectionSession}).__RF_LAST_SCREENSHOT_REGION_RESELECT__ = session
    return session
  }
}

export async function hideAnnotationOverlay(): Promise<void> {
  try {
    await RecordingFreedomService.HideAnnotationOverlay()
  } catch (error) {
    console.info('Using browser annotation overlay hide fallback:', error)
    emitBrowserWhiteboardVisibility({visible: false, mode: 'annotation'})
    window.close()
  }
}

export async function hideScreenshotAnnotationOverlay(): Promise<void> {
  try {
    await RecordingFreedomService.HideScreenshotAnnotationOverlay()
  } catch (error) {
    console.info('Using browser screenshot annotation hide fallback:', error)
    window.localStorage?.removeItem(browserScreenshotAnnotationKey)
    emitBrowserWhiteboardVisibility({visible: false, mode: 'annotation'})
    window.close()
  }
}

export async function loadAnnotationCapture(): Promise<WhiteboardScene> {
  try {
    return fromBoundWhiteboardScene(await RecordingFreedomService.LoadAnnotationCapture())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation capture load fallback:', error)
    const sceneJson = window.localStorage?.getItem(browserAnnotationSceneKey) ?? ''
    return {
      available: sceneJson.trim() !== '',
      scenePath: 'browser-preview/data/video/recording-preview.rfrec/annotations/scene.excalidraw',
      sceneJson,
      bytes: sceneJson.length,
      contentType: 'application/vnd.excalidraw+json',
    }
  }
}

export async function loadScreenshotAnnotationCapture(): Promise<ScreenshotWhiteboardContext> {
  try {
    return fromBoundScreenshotWhiteboardContext(await RecordingFreedomService.LoadScreenshotAnnotationCapture())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot annotation load fallback:', error)
    const raw = window.localStorage?.getItem(browserScreenshotAnnotationKey)
    const record = raw ? safeJSON(raw) : null
    return fromBoundScreenshotWhiteboardContext(record as Partial<BoundScreenshotWhiteboardContext> | undefined)
  }
}

export async function saveAnnotationCapture(request: {sceneJson: string; snapshotDataUrl: string; eventsJsonl?: string}): Promise<AnnotationCapture> {
  try {
    return fromBoundAnnotationCapture(await RecordingFreedomService.SaveAnnotationCapture(request as BoundAnnotationCaptureRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation capture fallback:', error)
    window.localStorage?.setItem(browserAnnotationSceneKey, request.sceneJson)
    return {
      packageDir: 'browser-preview/data/video/recording-preview.rfrec',
      scenePath: 'browser-preview/data/video/recording-preview.rfrec/annotations/scene.excalidraw',
      eventsPath: 'browser-preview/data/video/recording-preview.rfrec/annotations/events.jsonl',
      snapshotPath: 'browser-preview/data/video/recording-preview.rfrec/annotations/exports/annotation.png',
      timelineSnapshotPath: 'browser-preview/data/video/recording-preview.rfrec/annotations/snapshots/annotation-000001.png',
      bytes: request.sceneJson.length + request.snapshotDataUrl.length,
    }
  }
}

export async function saveScreenshotAnnotationCapture(request: {sceneJson: string; snapshotDataUrl: string; eventsJsonl?: string}): Promise<ScreenshotItem> {
  try {
    const result = await RecordingFreedomService.SaveScreenshotAnnotationCapture(request as BoundAnnotationCaptureRequest)
    return fromBoundScreenshotCaptureResult(result as BoundScreenshotCaptureResult).item
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot annotation save fallback:', error)
    const raw = window.localStorage?.getItem(browserScreenshotAnnotationKey)
    const context = fromBoundScreenshotWhiteboardContext((raw ? safeJSON(raw) : null) as Partial<BoundScreenshotWhiteboardContext> | undefined)
    const item = context.item ?? createBrowserScreenshotItem('region')
    const saved = {
      ...item,
      id: `browser-screenshot-${Date.now()}`,
      createdAt: new Date().toISOString(),
    }
    saveBrowserScreenshotHistory([saved, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: saved}))
    return saved
  }
}

export async function claimAnnotationRenderJob(): Promise<AnnotationRenderJobClaim> {
  try {
    return fromBoundAnnotationRenderJobClaim(await RecordingFreedomService.ClaimAnnotationRenderJob())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation renderer idle fallback:', error)
    return {available: false}
  }
}

export async function completeAnnotationRenderJob(result: {id: string; dataUrl?: string; error?: string}): Promise<void> {
  try {
    await RecordingFreedomService.CompleteAnnotationRenderJob(result as BoundAnnotationRenderJobResult)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation renderer completion fallback:', error)
  }
}

export async function hideWhiteboardWindow(): Promise<void> {
  try {
    await RecordingFreedomService.HideWhiteboardWindow()
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard hide fallback:', error)
    emitBrowserWhiteboardVisibility({visible: false, mode: 'whiteboard'})
    window.close()
  }
}

export async function loadWhiteboardScene(): Promise<WhiteboardScene> {
  try {
    return fromBoundWhiteboardScene(await RecordingFreedomService.LoadWhiteboardScene())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard scene fallback:', error)
    const sceneJson = window.localStorage?.getItem(browserWhiteboardSceneKey) ?? ''
    return {
      available: sceneJson.trim() !== '',
      scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
      sceneJson,
      bytes: sceneJson.length,
      contentType: 'application/vnd.excalidraw+json',
    }
  }
}

export async function saveWhiteboardScene(sceneJson: string): Promise<WhiteboardScene> {
  try {
    return fromBoundWhiteboardScene(await RecordingFreedomService.SaveWhiteboardScene({sceneJson} as BoundWhiteboardSceneRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard scene save fallback:', error)
    window.localStorage?.setItem(browserWhiteboardSceneKey, sceneJson)
    return {
      available: true,
      scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
      sceneJson,
      bytes: sceneJson.length,
      updatedAt: new Date().toISOString(),
      contentType: 'application/vnd.excalidraw+json',
    }
  }
}

export async function saveWhiteboardExport(request: {format: 'png' | 'svg' | 'excalidraw'; dataUrl?: string; payload?: string}): Promise<WhiteboardExport> {
  try {
    return fromBoundWhiteboardExport(await RecordingFreedomService.SaveWhiteboardExport(request as BoundWhiteboardExportRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard export fallback:', error)
    return {
      format: request.format,
      outputPath: `browser-preview/data/whiteboards/exports/whiteboard.${request.format}`,
      bytes: (request.payload ?? request.dataUrl ?? '').length,
    }
  }
}

export async function listScreenshots(): Promise<ScreenshotItem[]> {
  try {
    const result = await RecordingFreedomService.ListScreenshots()
    return fromBoundScreenshotHistory(result as BoundScreenshotHistoryResult)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot history fallback:', error)
    return loadBrowserScreenshotHistory()
  }
}

export async function readScreenshotImage(id: string, thumbnail = false): Promise<ScreenshotImage> {
  try {
    return fromBoundScreenshotImage(await RecordingFreedomService.ReadScreenshotImage({id, thumbnail} as BoundScreenshotImageRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot image fallback:', error)
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id)
    return {
      available: Boolean(item),
      dataUrl: browserScreenshotDataUrl(item),
      path: item?.path,
      bytes: browserScreenshotDataUrl(item)?.length ?? 0,
    }
  }
}

export async function openScreenshot(id: string): Promise<ScreenshotItem | null> {
  try {
    return fromBoundScreenshotItem(await RecordingFreedomService.OpenScreenshot({id} as BoundScreenshotImageRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot open fallback:', error)
    return loadBrowserScreenshotHistory().find((item) => item.id === id) ?? null
  }
}

export async function openScreenshotDirectory(id: string): Promise<ScreenshotItem | null> {
  try {
    return fromBoundScreenshotItem(await RecordingFreedomService.OpenScreenshotDirectory({id} as BoundScreenshotImageRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot directory fallback:', error)
    return loadBrowserScreenshotHistory().find((item) => item.id === id) ?? null
  }
}

export async function patchScreenshotItem(id: string, patch: {pinned?: boolean; fixed?: boolean}): Promise<ScreenshotItem[]> {
  try {
    const result = await RecordingFreedomService.PatchScreenshotItem({id, ...patch} as BoundScreenshotItemPatchRequest)
    return fromBoundScreenshotHistory(result as BoundScreenshotHistoryResult)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot patch fallback:', error)
    const next = loadBrowserScreenshotHistory().map((item) => item.id === id
      ? {
        ...item,
        pinned: patch.fixed === true ? true : patch.pinned ?? item.pinned,
        fixed: patch.pinned === false ? false : patch.fixed ?? item.fixed,
      }
      : item)
    saveBrowserScreenshotHistory(next)
    return next
  }
}

export async function deleteScreenshotItem(id: string): Promise<ScreenshotItem[]> {
  try {
    const result = await RecordingFreedomService.DeleteScreenshotItem(id)
    return fromBoundScreenshotHistory(result as BoundScreenshotHistoryResult)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot delete fallback:', error)
    const next = loadBrowserScreenshotHistory().filter((item) => item.id !== id)
    saveBrowserScreenshotHistory(next)
    const pinned = fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
    if (pinned.item?.id === id) {
      const state = {visible: false, fixed: false}
      window.localStorage?.setItem(browserScreenshotPinStateKey, JSON.stringify(state))
      window.dispatchEvent(new CustomEvent(browserScreenshotPinEvent, {detail: state}))
    }
    return next
  }
}

export async function showPinnedScreenshot(id: string): Promise<ScreenshotPinState> {
  try {
    return fromBoundScreenshotPinState(await RecordingFreedomService.ShowPinnedScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot pin fallback:', error)
    let items = await patchScreenshotItem(id, {pinned: true})
    const item = items.find((entry) => entry.id === id)
    const state = fromBrowserScreenshotPinState({
      visible: Boolean(item),
      item,
      dataUrl: browserScreenshotDataUrl(item),
      fixed: item?.fixed === true,
    })
    window.localStorage?.setItem(browserScreenshotPinStateKey, JSON.stringify(state))
    window.dispatchEvent(new CustomEvent(browserScreenshotPinEvent, {detail: state}))
    return state
  }
}

export async function hidePinnedScreenshot(): Promise<void> {
  try {
    await RecordingFreedomService.HidePinnedScreenshot()
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot pin hide fallback:', error)
    const state = {visible: false, fixed: false}
    window.localStorage?.setItem(browserScreenshotPinStateKey, JSON.stringify(state))
    window.dispatchEvent(new CustomEvent(browserScreenshotPinEvent, {detail: state}))
  }
}

export async function loadPinnedScreenshot(): Promise<ScreenshotPinState> {
  try {
    return fromBoundScreenshotPinState(await RecordingFreedomService.LoadPinnedScreenshot())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot pin state fallback:', error)
    return fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
  }
}

export async function openScreenshotInWhiteboard(id: string): Promise<ScreenshotWhiteboardContext> {
  try {
    return fromBoundScreenshotWhiteboardContext(await RecordingFreedomService.OpenScreenshotInWhiteboard(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot whiteboard fallback:', error)
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id)
    const context: ScreenshotWhiteboardContext = {
      available: Boolean(item),
      item,
      dataUrl: browserScreenshotDataUrl(item),
    }
    window.localStorage?.setItem(browserScreenshotWhiteboardKey, JSON.stringify(context))
    const popup = window.open('/#/whiteboard', 'recordingfreedom-whiteboard', 'width=1120,height=760')
    popup?.focus()
    emitBrowserWhiteboardVisibility({visible: true, mode: 'whiteboard'})
    return context
  }
}

export async function consumeScreenshotWhiteboardContext(): Promise<ScreenshotWhiteboardContext> {
  try {
    return fromBoundScreenshotWhiteboardContext(await RecordingFreedomService.ConsumeScreenshotWhiteboardContext())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot whiteboard context fallback:', error)
    const context = fromBrowserScreenshotWhiteboardContext(safeJSON(window.localStorage?.getItem(browserScreenshotWhiteboardKey)))
    window.localStorage?.removeItem(browserScreenshotWhiteboardKey)
    return context
  }
}

export async function startScrollingScreenshot(): Promise<void> {
  try {
    await RecordingFreedomService.StartScrollingScreenshot()
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser scrolling screenshot fallback:', error)
    throw error
  }
}

export async function patchWhiteboardSettings(patch: WhiteboardSettingsPatch): Promise<AppSettings> {
  try {
    return fromBoundSettings(await RecordingFreedomService.PatchWhiteboardSettings(patch as BoundWhiteboardSettingsPatchRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard settings fallback:', error)
    const next = applyBrowserWhiteboardPatch(loadBrowserSettings(), patch)
    window.localStorage?.setItem(browserSettingsKey, JSON.stringify(next))
    return next
  }
}

export async function showRegionSelector(): Promise<RegionSelectionSession> {
  try {
    return fromBoundRegionSelectionSession(await RecordingFreedomService.ShowRegionSelector())
  } catch (error) {
    console.info('Using browser region selector fallback:', error)
    return {
      id: `browser-region-${Date.now()}`,
      bounds: {x: 0, y: 0, width: window.innerWidth, height: window.innerHeight},
      minimumWidth: 64,
      minimumHeight: 64,
      displayCount: 1,
    }
  }
}

export async function completeRegionSelection(request: RegionSelectionSession['bounds']): Promise<RegionSelectionResult> {
  try {
    return fromBoundRegionSelectionResult(await RecordingFreedomService.CompleteRegionSelection(toBoundRegionSelectionRequest(request)))
  } catch (error) {
    console.info('Using browser region selection completion fallback:', error)
    const source: CaptureSource = {
      id: 'region:custom',
      type: 'region',
      label: 'Region',
      name: 'Custom Region',
      meta: `${request.width} x ${request.height} selected region`,
      x: request.x,
      y: request.y,
      width: request.width,
      height: request.height,
      nativeId: 'region:browser-preview',
      available: false,
      capability: 'native-backend-queued',
      unavailableReason: 'Desktop region overlay is only available in the Wails runtime.',
    }
    return {source, geometry: request, cancelled: false}
  }
}

export async function cancelRegionSelector(): Promise<RegionSelectionResult> {
  try {
    return fromBoundRegionSelectionResult(await RecordingFreedomService.CancelRegionSelection())
  } catch (error) {
    console.info('Using browser region selection cancel fallback:', error)
    return {cancelled: true}
  }
}

export async function updateSelectedRegion(request: RegionSelectionSession['bounds']): Promise<RegionSelectionResult> {
  try {
    return fromBoundRegionSelectionResult(await RecordingFreedomService.UpdateSelectedRegion(toBoundRegionSelectionRequest(request)))
  } catch (error) {
    console.info('Using browser selected region update fallback:', error)
    return {geometry: request, cancelled: false}
  }
}

function emitBrowserRegionFrame(bounds: RegionSelectionSession['bounds'], purpose: NonNullable<RegionSelectionSession['purpose']>) {
  const frame = {
    bounds,
    overlayBounds: {x: 0, y: 0, width: window.innerWidth, height: window.innerHeight},
    mode: 'edit',
    purpose,
  }
  ;(window as Window & {__RF_REGION_FRAME__?: typeof frame}).__RF_REGION_FRAME__ = frame
  window.dispatchEvent(new CustomEvent('rf-region-frame', {detail: frame}))
}

export async function cancelSelectedRegion(): Promise<RegionSelectionResult> {
  try {
    return fromBoundRegionSelectionResult(await RecordingFreedomService.CancelSelectedRegion())
  } catch (error) {
    console.info('Using browser selected region cancel fallback:', error)
    return {cancelled: true}
  }
}

export async function hideRegionFrame(): Promise<void> {
  try {
    await RecordingFreedomService.HideRegionFrame()
  } catch (error) {
    console.info('Using browser region frame hide fallback:', error)
  }
}

export async function showPipOverlay(config: PIPConfig, mode: PIPOverlayMode = 'edit', camera: string | PIPOverlayCamera = '', previewImagePath = '', clientOperationId = 0): Promise<PIPOverlayState> {
  try {
    return fromBoundPipOverlayState(await RecordingFreedomService.ShowPIPOverlay(toBoundPipOverlayRequest(config, mode, camera, previewImagePath, clientOperationId)))
  } catch (error) {
    console.info('Using browser PIP overlay fallback:', error)
    return browserPipOverlayState(config, mode, camera, previewImagePath, clientOperationId)
  }
}

export async function updatePipOverlay(config: PIPConfig, mode: PIPOverlayMode = 'edit', camera: string | PIPOverlayCamera = '', previewImagePath = '', clientOperationId = 0): Promise<PIPOverlayState> {
  try {
    return fromBoundPipOverlayState(await RecordingFreedomService.UpdatePIPOverlay(toBoundPipOverlayRequest(config, mode, camera, previewImagePath, clientOperationId)))
  } catch (error) {
    console.info('Using browser PIP overlay update fallback:', error)
    return browserPipOverlayState(config, mode, camera, previewImagePath, clientOperationId)
  }
}

export async function hidePipOverlay(): Promise<void> {
  try {
    await RecordingFreedomService.HidePIPOverlay()
  } catch (error) {
    console.info('Using browser PIP overlay hide fallback:', error)
  }
}

export async function readPipPreviewImage(path: string, knownModifiedUnixNano = 0): Promise<PIPPreviewImage> {
  try {
    const result = await RecordingFreedomService.ReadPIPPreviewImage({
      path,
      knownModifiedUnixNano,
    } as BoundPIPPreviewImageRequest)
    return fromBoundPipPreviewImage(result)
  } catch (error) {
    console.info('Using browser PIP preview image fallback:', error)
    return {available: false}
  }
}

export async function showScreenIndicator(sourceId: string): Promise<ScreenIndicatorResult | null> {
  try {
    return fromBoundScreenIndicatorResult(await RecordingFreedomService.ShowScreenIndicator(toBoundScreenIndicatorRequest(sourceId)))
  } catch (error) {
    console.info('Using browser screen indicator fallback:', error)
    return null
  }
}

export async function hideScreenIndicator(): Promise<void> {
  try {
    await RecordingFreedomService.HideScreenIndicator()
  } catch (error) {
    console.info('Using browser screen indicator hide fallback:', error)
  }
}

export async function hideSettingsWindow(): Promise<void> {
  try {
    await RecordingFreedomService.HideSettingsWindow()
  } catch (error) {
    console.info('Using browser settings close fallback:', error)
    if (window.history.length > 1) {
      window.history.back()
      return
    }
    window.close()
  }
}

export async function loadSources(): Promise<CaptureSource[]> {
  try {
    const bootstrap = await RecordingFreedomService.Bootstrap()
    const boundSources = bootstrap.sources ?? []
    return boundSources.length > 0 ? boundSources.map(fromBoundSource) : fallbackSources
  } catch (error) {
    console.info('Using browser mock sources:', error)
    return fallbackSources
  }
}

export async function loadMediaDevices(): Promise<MediaInventory> {
  try {
    return fromBoundMediaInventory(await RecordingFreedomService.ListMediaDevices())
  } catch (error) {
    console.info('Using browser mock media devices:', error)
    return fallbackMediaInventory
  }
}

export async function loadCaptureCapabilities(): Promise<CaptureCapabilities> {
  try {
    return fromBoundCapabilities(await RecordingFreedomService.GetCaptureCapabilities())
  } catch (error) {
    console.info('Using browser mock capture capabilities:', error)
    return fallbackCapabilities
  }
}

export async function loadSettings(): Promise<AppSettings> {
  try {
    return fromBoundSettings(await RecordingFreedomService.GetSettings())
  } catch (error) {
    console.info('Using browser mock settings:', error)
    return loadBrowserSettings()
  }
}

export async function saveSettings(settings: AppSettings): Promise<AppSettings> {
  try {
    return fromBoundSettings(await RecordingFreedomService.SaveSettings(toBoundSettings(settings)))
  } catch (error) {
    console.info('Using browser mock settings save:', error)
    const current = loadBrowserSettings()
    const next = {
      ...settings,
      recording: current.recording,
      shortcuts: current.shortcuts,
      window: {
        ...settings.window,
        theme: current.window.theme,
      },
      updatedAt: new Date().toISOString(),
    }
    window.localStorage?.setItem(browserSettingsKey, JSON.stringify(next))
    return next
  }
}

export async function setDataRoot(rootDir: string): Promise<AppDataInfo> {
  try {
    const info = await RecordingFreedomService.SetDataRoot(rootDir)
    return {
      rootDir: info.rootDir,
      videoDir: info.videoDir,
    }
  } catch (error) {
    console.info('Using browser mock data root apply:', error)
    const cleanRoot = rootDir.trim() || fallbackAppData.rootDir
    const separator = cleanRoot.includes('\\') ? '\\' : '/'
    return {
      rootDir: cleanRoot,
      videoDir: `${cleanRoot.replace(/[\\/]+$/, '')}${separator}data${separator}video`,
    }
  }
}

export async function openVideoDirectory(): Promise<AppDataInfo> {
  try {
    const info = await RecordingFreedomService.OpenVideoDirectory()
    return {
      rootDir: info.rootDir,
      videoDir: info.videoDir,
    }
  } catch (error) {
    console.info('Desktop video directory open unavailable:', error)
    throw error
  }
}

export async function openRecordingPackage(packagePath: string): Promise<RecordingRecovery> {
  try {
    return fromBoundRecovery(await RecordingFreedomService.OpenRecordingPackage(packagePath))
  } catch (error) {
    console.info('Desktop recording package open unavailable:', error)
    throw error
  }
}

export async function exportRecordingPackage(packagePath: string, options: RecordingExportOptions = {}): Promise<RecordingExportResult> {
  try {
    return fromBoundExport(await RecordingFreedomService.ExportRecordingPackage({
      packageDir: packagePath,
      includeAnnotations: options.includeAnnotations,
    }))
  } catch (error) {
    console.info('Desktop recording export unavailable:', error)
    throw error
  }
}

export async function previewExportRecordingPackage(packagePath: string, options: RecordingExportOptions = {}): Promise<RecordingExportPlan> {
  try {
    const result: BoundExportRecordingPlanResult = await RecordingFreedomService.PreviewExportRecordingPackage({
      packageDir: packagePath,
      includeAnnotations: options.includeAnnotations,
    })
    return fromBoundExportPlan(result.plan)
  } catch (error) {
    console.info('Desktop recording export preview unavailable:', error)
    throw error
  }
}

export async function readAnnotationPreviewImage(packagePath: string, snapshotPath: string): Promise<AnnotationPreviewImage> {
  try {
    const result: BoundAnnotationPreviewImageResult = await RecordingFreedomService.ReadAnnotationPreviewImage({
      packageDir: packagePath,
      snapshotPath,
    } as BoundAnnotationPreviewImageRequest)
    return {
      available: result.available === true,
      dataUrl: result.dataUrl,
      relativePath: result.relativePath,
      bytes: result.bytes,
    }
  } catch (error) {
    console.info('Desktop annotation preview image unavailable:', error)
    return {available: false}
  }
}

export async function scanRecordingPackages(): Promise<RecordingRecovery[]> {
  try {
    const recoveries = await RecordingFreedomService.ScanRecordingPackages()
    return (recoveries ?? []).map(fromBoundRecovery)
  } catch (error) {
    console.info('Using browser mock recovery scan:', error)
    return []
  }
}

export async function recoverRecordingPackage(packagePath: string): Promise<RecordingRecovery | null> {
  try {
    return fromBoundRecovery(await RecordingFreedomService.RecoverRecordingPackage(packagePath))
  } catch (error) {
    console.info('Using browser mock package recovery:', error)
    return null
  }
}

export async function preflightRecording(request: MockRecordingRequest): Promise<RecordingPreflight> {
  try {
    return fromBoundPreflight(await RecordingFreedomService.PreflightRecording(toStartRequest(request)))
  } catch (error) {
    console.info('Using browser mock preflight:', error)
    return {
      status: 'ready',
      backend: 'browser-mock',
      message: 'Browser UI shell is ready to create a mock recording package.',
      checks: [
        {
          id: 'browser-mock',
          label: 'Browser Preview',
          status: 'ready',
          reason: 'Preview mode validates UI flow only; desktop runtime performs native preflight.',
        },
      ],
    }
  }
}

export async function preflightAudioOnlyRecording(request: AudioOnlyRecordingRequest): Promise<RecordingPreflight> {
  try {
    return fromBoundPreflight(await RecordingFreedomService.PreflightAudioOnlyRecording(toAudioOnlyRequest(request)))
  } catch (error) {
    console.info('Using browser mock audio-only preflight:', error)
    return {
      status: request.systemAudio || request.microphone ? 'ready' : 'blocked',
      backend: 'browser-mock',
      message: 'Browser UI shell is ready to create a mock audio-only package.',
      checks: [
        {
          id: 'browser-mock',
          label: 'Browser Preview',
          status: request.systemAudio || request.microphone ? 'ready' : 'blocked',
          reason: 'Preview mode validates UI flow only; desktop runtime performs native preflight.',
        },
      ],
    }
  }
}

export async function startRecording(request: MockRecordingRequest): Promise<RecordingSession> {
  try {
    const session = await RecordingFreedomService.StartRecording(toStartRequest(request))
    return fromBoundSession(session)
  } catch (error) {
    console.info('Using browser mock recording package:', error)
    const session = createMockRecordingPackage(request)
    return {id: session.id, packagePath: session.packagePath, backend: 'browser-mock'}
  }
}

export async function startAudioOnlyRecording(request: AudioOnlyRecordingRequest): Promise<RecordingSession> {
  try {
    const session = await RecordingFreedomService.StartAudioOnlyRecording(toAudioOnlyRequest(request))
    return fromBoundSession(session)
  } catch (error) {
    console.info('Using browser mock audio-only recording package:', error)
    const session = createMockAudioOnlyRecordingPackage(request)
    return {id: session.id, packagePath: session.packagePath, backend: 'browser-mock', recordingMode: 'audio-only'}
  }
}

function fromBoundPreflight(summary: BoundPreflightSummary): RecordingPreflight {
  return {
    status: summary.status as RecordingPreflight['status'],
    backend: summary.backend,
    message: summary.message,
    checks: (summary.checks ?? []).map((check) => ({
      id: check.id,
      label: check.label,
      status: check.status as RecordingPreflight['status'],
      reason: check.reason,
    })),
  }
}

export async function pauseRecording(): Promise<RecordingSession | null> {
  try {
    return fromBoundSession(await RecordingFreedomService.PauseRecording())
  } catch (error) {
    console.info('Using browser mock pause:', error)
    return null
  }
}

export async function resumeRecording(): Promise<RecordingSession | null> {
  try {
    return fromBoundSession(await RecordingFreedomService.ResumeRecording())
  } catch (error) {
    console.info('Using browser mock resume:', error)
    return null
  }
}

export async function stopRecording(): Promise<RecordingSession | null> {
  try {
    return fromBoundSession(await RecordingFreedomService.StopRecording())
  } catch (error) {
    console.info('Using browser mock stop:', error)
    return null
  }
}

function fromBoundBootstrap(bootstrap: BoundBootstrapState): RecorderBootstrap {
  const boundSources = bootstrap.sources ?? []
  const recoveries = bootstrap.recoveries ?? []
  return {
    appData: {
      rootDir: bootstrap.appData.rootDir,
      videoDir: bootstrap.appData.videoDir,
    },
    storage: fromBoundStorageStatus(bootstrap.storage),
    state: bootstrap.state,
    backend: bootstrap.backend,
    sources: boundSources.length > 0 ? boundSources.map(fromBoundSource) : fallbackSources,
    media: fromBoundMediaInventory(bootstrap.media),
    recoveries: recoveries.map(fromBoundRecovery),
    settings: fromBoundSettings(bootstrap.settings),
    capabilities: fromBoundCapabilities(bootstrap.capabilities),
  }
}

function fromBoundStorageStatus(storage: BoundBootstrapState['storage']): AppStorageStatus {
  if (!storage) return fallbackStorageStatus
  return {
    rootDir: storage.rootDir,
    videoDir: storage.videoDir,
    writable: storage.writable,
    freeSpaceKnown: storage.freeSpaceKnown,
    availableBytes: storage.availableBytes,
    minimumRecommendedBytes: storage.minimumRecommendedBytes,
    status: storage.status as AppStorageStatus['status'],
    reason: storage.reason,
  }
}

function fromBoundSource(source: BoundCaptureSource): CaptureSource {
  const type = source.type as CaptureSource['type']
  return {
    id: source.id,
    type,
    label: sourceLabel(type),
    name: source.name,
    meta: sourceMeta(source),
    x: source.x,
    y: source.y,
    width: source.width,
    height: source.height,
    displayIndex: source.displayIndex,
    nativeId: source.nativeId,
    processId: source.processId,
    available: source.available,
    capability: source.capability,
    unavailableReason: source.unavailableReason,
  }
}

function fromBoundRegionSelectionSession(session: BoundRegionSelectionSession): RegionSelectionSession {
  return {
    id: session.id,
    bounds: {
      x: session.bounds.x,
      y: session.bounds.y,
      width: session.bounds.width,
      height: session.bounds.height,
    },
    captureBounds: session.captureBounds ? {
      x: session.captureBounds.x,
      y: session.captureBounds.y,
      width: session.captureBounds.width,
      height: session.captureBounds.height,
    } : undefined,
    minimumWidth: session.minimumWidth,
    minimumHeight: session.minimumHeight,
    displayCount: session.displayCount,
    purpose: session.purpose === 'annotation' || session.purpose === 'screenshot' ? session.purpose : 'capture',
  }
}

function fromBoundRegionSelectionResult(result: BoundRegionSelectionResult): RegionSelectionResult {
  return {
    sessionId: result.sessionId,
    source: result.source?.id ? fromBoundSource(result.source as BoundCaptureSource) : undefined,
    geometry: result.geometry ? {
      x: result.geometry.x,
      y: result.geometry.y,
      width: result.geometry.width,
      height: result.geometry.height,
    } : undefined,
    cancelled: result.cancelled,
    error: result.error,
  }
}

function fromBoundScreenIndicatorResult(result: BoundScreenIndicatorResult): ScreenIndicatorResult {
  return {
    sourceId: result.sourceId,
    displayIndex: result.displayIndex,
    label: result.label,
    sourceBounds: fromBoundRegionRect(result.sourceBounds),
    windowBounds: fromBoundRegionRect(result.windowBounds),
  }
}

function fromBoundPipOverlayState(state: BoundPIPOverlayState): PIPOverlayState {
  const config = fromBoundPipConfig(state.config as Partial<PIPConfig>, (state.config?.preset as PIPConfig['preset'] | undefined) ?? 'bottom-right')
  return {
    config,
    placement: {
      visible: state.placement.visible,
      rect: {
        x: state.placement.rect.x,
        y: state.placement.rect.y,
        width: state.placement.rect.width,
        height: state.placement.rect.height,
        visible: state.placement.rect.visible,
      },
      shape: state.placement.shape as PIPConfig['shape'],
      mirror: state.placement.mirror,
      edgeFeather: state.placement.edgeFeather,
    },
    overlayBounds: fromBoundRegionRect(state.overlayBounds),
    windowBounds: fromBoundRegionRect(state.windowBounds),
    contentBounds: fromBoundRegionRect(state.contentBounds),
    mode: state.mode === 'recording' ? 'recording' : 'edit',
    cameraName: state.cameraName,
    camera: fromBoundPipCamera(state.camera),
    previewImagePath: state.previewImagePath,
    captureExcluded: state.captureExcluded,
    clientOperationId: state.clientOperationId,
  }
}

function fromBoundPipPreviewImage(result: BoundPIPPreviewImageResult): PIPPreviewImage {
  return {
    available: result.available,
    dataUrl: result.dataUrl,
    modifiedUnixNano: result.modifiedUnixNano,
  }
}

function fromBoundWhiteboardScene(result: BoundWhiteboardSceneResult): WhiteboardScene {
  return {
    available: result.available,
    scenePath: result.scenePath,
    sceneJson: result.sceneJson,
    bytes: result.bytes,
    updatedAt: result.updatedAt,
    contentType: result.contentType,
  }
}

function fromBoundWhiteboardExport(result: BoundWhiteboardExportResult): WhiteboardExport {
  return {
    format: result.format === 'svg' || result.format === 'excalidraw' ? result.format : 'png',
    outputPath: result.outputPath,
    bytes: result.bytes,
  }
}

function fromBoundScreenshotCaptureResult(result: BoundScreenshotCaptureResult): {item: ScreenshotItem} {
  return {
    item: fromBoundScreenshotItem(result.item),
  }
}

function fromBoundScreenshotHistory(result: BoundScreenshotHistoryResult): ScreenshotItem[] {
  return (result.items ?? []).map(fromBoundScreenshotItem)
}

function fromBoundScreenshotItem(item: BoundScreenshotItem): ScreenshotItem {
  return {
    id: item.id,
    path: item.path,
    thumbnailPath: item.thumbnailPath,
    createdAt: item.createdAt,
    width: item.width,
    height: item.height,
    mode: item.mode || 'region',
    region: item.region ? fromBoundRegionRect(item.region) : undefined,
    pinned: item.pinned === true,
    fixed: item.fixed === true,
  }
}

function fromBoundScreenshotImage(result: BoundScreenshotImageResult): ScreenshotImage {
  return {
    available: result.available === true,
    dataUrl: result.dataUrl,
    path: result.path,
    bytes: result.bytes,
  }
}

function fromBoundScreenshotPinState(state: Partial<BoundScreenshotPinState> | undefined): ScreenshotPinState {
  return {
    visible: state?.visible === true,
    item: state?.item ? fromBoundScreenshotItem(state.item) : undefined,
    dataUrl: state?.dataUrl,
    fixed: state?.fixed === true,
  }
}

function fromBoundScreenshotWhiteboardContext(context: Partial<BoundScreenshotWhiteboardContext> | undefined): ScreenshotWhiteboardContext {
  return {
    available: context?.available === true,
    item: context?.item ? fromBoundScreenshotItem(context.item) : undefined,
    dataUrl: context?.dataUrl,
  }
}

function fromBoundAnnotationOverlayState(state: BoundAnnotationOverlayState): AnnotationOverlayState {
  return {
    mode: state.mode === 'screenshot' ? 'screenshot' : 'annotation',
    packageDir: state.packageDir,
    manifestPath: state.manifestPath,
    windowBounds: fromBoundRegionRect(state.windowBounds),
    canvasBounds: fromBoundRegionRect(state.canvasBounds),
    target: {
      type: state.target.type,
      id: state.target.id,
      geometry: state.target.geometry ? {
        x: state.target.geometry.x,
        y: state.target.geometry.y,
        width: state.target.geometry.width,
        height: state.target.geometry.height,
        displayIndex: state.target.geometry.displayIndex,
        nativeId: state.target.geometry.nativeId,
      } : undefined,
    },
    captureExcluded: state.captureExcluded === true,
  }
}

function fromBoundAnnotationCapture(result: BoundAnnotationCaptureResult): AnnotationCapture {
  return {
    packageDir: result.packageDir,
    scenePath: result.scenePath,
    eventsPath: result.eventsPath,
    snapshotPath: result.snapshotPath,
    timelineSnapshotPath: result.timelineSnapshotPath,
    bytes: result.bytes,
  }
}

function fromBoundAnnotationRenderJobClaim(result: BoundAnnotationRenderJobClaim): AnnotationRenderJobClaim {
  const job = result.job
  return {
    available: result.available === true && Boolean(job),
    job: job
      ? {
        id: job.id,
        packageDir: job.packageDir,
        scenePath: job.scenePath,
        relativeScenePath: job.relativeScenePath,
        outputPath: job.outputPath,
        relativeOutputPath: job.relativeOutputPath,
        sceneJson: job.sceneJson,
        canvasWidth: job.canvasWidth,
        canvasHeight: job.canvasHeight,
        index: job.index,
        startOffsetMs: job.startOffsetMs,
        endOffsetMs: job.endOffsetMs,
      }
      : undefined,
  }
}

function browserAnnotationOverlayState(
  mode: AnnotationOverlayState['mode'] = 'annotation',
  bounds?: RegionSelectionSession['bounds'],
): AnnotationOverlayState {
  const width = Math.max(320, bounds?.width ?? window.innerWidth ?? 1280)
  const height = Math.max(240, bounds?.height ?? window.innerHeight ?? 720)
  return {
    mode,
    packageDir: mode === 'screenshot' ? undefined : 'browser-preview/data/video/recording-preview.rfrec',
    manifestPath: mode === 'screenshot' ? undefined : 'browser-preview/data/video/recording-preview.rfrec/manifest.json',
    windowBounds: {x: 0, y: 0, width, height},
    canvasBounds: {x: 0, y: 0, width, height},
    target: {
      type: mode === 'screenshot' ? 'screenshot-region' : 'screen',
      id: mode === 'screenshot' ? 'browser-screenshot-region' : 'browser-preview',
      geometry: {x: bounds?.x ?? 0, y: bounds?.y ?? 0, width, height},
    },
    captureExcluded: false,
  }
}

function fromBoundPipCamera(camera: BoundPIPOverlayState['camera']): PIPOverlayCamera | undefined {
  if (!camera) return undefined
  return normalizePipOverlayCamera({
    deviceId: camera.deviceId,
    nativeId: camera.nativeId,
    name: camera.name,
  })
}

function fromBoundRegionRect(rect: {x: number; y: number; width: number; height: number}) {
  return {
    x: rect.x,
    y: rect.y,
    width: rect.width,
    height: rect.height,
  }
}

function sourceMeta(source: BoundCaptureSource) {
  const base = source.subtitle || sourceDimensions(source)
  if (source.available === false && source.unavailableReason) {
    return `${base} · ${source.unavailableReason}`
  }
  if (source.available === false) {
    return `${base} · ${source.capability}`
  }
  return base
}

function sourceDimensions(source: BoundCaptureSource) {
  if (source.width && source.height) {
    return `${source.width} x ${source.height}`
  }
  return 'Ready'
}

function sourceLabel(type: CaptureSource['type']) {
  if (type === 'screen') return 'Screen'
  if (type === 'all-screens') return 'All Screens'
  if (type === 'region') return 'Region'
  if (type === 'window') return 'Window'
  return 'Program'
}

function fromBoundMediaInventory(inventory: BoundMediaInventory): MediaInventory {
  return {
    systemAudio: (inventory.systemAudio ?? []).map(fromBoundMediaDevice),
    microphones: (inventory.microphones ?? []).map(fromBoundMediaDevice),
    cameras: (inventory.cameras ?? []).map(fromBoundMediaDevice),
    enhancement: {
      engine: inventory.enhancement.engine,
      appliesTo: inventory.enhancement.appliesTo,
      available: inventory.enhancement.available,
      capability: inventory.enhancement.capability,
      unavailableReason: inventory.enhancement.unavailableReason,
    },
  }
}

function fromBoundMediaDevice(device: BoundMediaDevice): MediaDevice {
  return {
    id: device.id,
    type: device.type as MediaDevice['type'],
    name: device.name,
    meta: mediaDeviceMeta(device),
    nativeId: device.nativeId,
    isDefault: device.isDefault,
    available: device.available,
    capability: device.capability,
    unavailableReason: device.unavailableReason,
    rnnoiseEligible: device.rnnoiseEligible,
    sidecarEligible: device.sidecarEligible,
  }
}

function fromBoundAudioState(state: BoundAudioState): AudioControlState {
  return {
    system: state.system,
    systemDeviceId: state.systemDeviceId,
    microphone: state.microphone,
    microphoneDeviceId: state.microphoneDeviceId,
    noiseSuppression: state.microphone && state.noiseSuppression,
    microphoneGain: state.microphoneGain || 1,
  }
}

function toBoundAudioStatePatch(patch: AudioStatePatch): BoundAudioStatePatchRequest {
  return {
    system: patch.system,
    systemDeviceId: patch.systemDeviceId,
    microphone: patch.microphone,
    microphoneDeviceId: patch.microphoneDeviceId,
    noiseSuppression: patch.noiseSuppression,
    microphoneGain: patch.microphoneGain,
    clearSystemDevice: patch.clearSystemDevice,
    clearMicrophoneDevice: patch.clearMicrophoneDevice,
  }
}

function toBoundSettingsPreferencesPatch(patch: SettingsPreferencesPatch): BoundSettingsPreferencesPatchRequest {
  return {
    theme: patch.theme as BoundSettingsPreferencesPatchRequest['theme'],
    recordingQuality: patch.recordingQuality,
    recordingFps: patch.recordingFps,
    captureCursor: patch.captureCursor,
    countdownSeconds: patch.countdownSeconds,
  }
}

function applyBrowserSettingsPreferencesPatch(settings: AppSettings, patch: SettingsPreferencesPatch): AppSettings {
  return {
    ...settings,
    recording: {
      ...settings.recording,
      quality: patch.recordingQuality ?? settings.recording.quality,
      fps: patch.recordingFps ?? settings.recording.fps,
      captureCursor: patch.captureCursor ?? settings.recording.captureCursor,
      countdownSeconds: patch.countdownSeconds ?? settings.recording.countdownSeconds,
    },
    window: {
      ...settings.window,
      theme: patch.theme ?? settings.window.theme,
    },
    updatedAt: new Date().toISOString(),
  }
}

function applyBrowserShortcutPatch(settings: AppSettings, patch: ShortcutSettingsPatch): AppSettings {
  const nextShortcuts = normalizeShortcutSettings({
    ...settings.shortcuts,
    ...patch,
  })
  return {
    ...settings,
    shortcuts: nextShortcuts,
    updatedAt: new Date().toISOString(),
  }
}

function applyBrowserWhiteboardPatch(settings: AppSettings, patch: WhiteboardSettingsPatch): AppSettings {
  return {
    ...settings,
    whiteboard: fromBoundWhiteboardSettings({
      ...settings.whiteboard,
      ...patch,
    }),
    updatedAt: new Date().toISOString(),
  }
}

function applyBrowserAudioPatch(audio: AppSettings['audio'], patch: AudioStatePatch): AppSettings['audio'] {
  const next = {...audio}
  if (patch.system !== undefined) next.system = patch.system
  if (patch.clearSystemDevice) next.systemDeviceId = undefined
  if (patch.systemDeviceId !== undefined) next.systemDeviceId = patch.systemDeviceId
  if (patch.microphone !== undefined) next.microphone = patch.microphone
  if (patch.clearMicrophoneDevice) next.microphoneDeviceId = undefined
  if (patch.microphoneDeviceId !== undefined) next.microphoneDeviceId = patch.microphoneDeviceId
  if (patch.noiseSuppression !== undefined) next.noiseSuppression = patch.noiseSuppression
  if (patch.microphoneGain !== undefined) next.microphoneGain = patch.microphoneGain
  if (!next.microphone) next.noiseSuppression = false
  return next
}

function fromBoundCapabilities(capabilities: BoundCaptureCapabilities): CaptureCapabilities {
  return {
    platform: capabilities.platform,
    sourceEnumeration: fromBoundCapability(capabilities.sourceEnumeration),
    screenRecording: fromBoundCapability(capabilities.screenRecording),
    windowRecording: fromBoundCapability(capabilities.windowRecording),
    applicationRecording: fromBoundCapability(capabilities.applicationRecording),
    systemAudio: fromBoundCapability(capabilities.systemAudio),
    microphone: fromBoundCapability(capabilities.microphone),
    microphoneEnhancement: fromBoundCapability(capabilities.microphoneEnhancement),
    cameraSidecar: fromBoundCapability(capabilities.cameraSidecar),
    pipExport: fromBoundCapability(capabilities.pipExport),
    packageRecovery: fromBoundCapability(capabilities.packageRecovery),
  }
}

function fromBoundCapability(capability: BoundCaptureCapability): CaptureCapability {
  return {
    id: capability.id,
    label: capability.label,
    status: capability.status as CaptureCapability['status'],
    backend: capability.backend,
    permission: capability.permission as CaptureCapability['permission'],
    reason: capability.reason,
  }
}

function mediaDeviceMeta(device: BoundMediaDevice) {
  const base = device.subtitle || 'Ready'
  if (device.available === false && device.unavailableReason) {
    return `${base} · ${device.unavailableReason}`
  }
  if (device.available === false) {
    return `${base} · ${device.capability}`
  }
  return base
}

function fromBoundSettings(settings: BoundSettings): AppSettings {
  return {
    schemaVersion: settings.schemaVersion,
    locale: normalizeLocale(settings.locale),
    source: {
      lastSourceId: settings.source.lastSourceId,
      lastSourceType: settings.source.lastSourceType as CaptureSource['type'],
    },
    storage: {
      dataRootDir: settings.storage?.dataRootDir,
    },
    recording: {
      quality: normalizeRecordingQuality(settings.recording.quality),
      fps: normalizeRecordingFPS(settings.recording.fps),
      captureCursor: settings.recording.captureCursor,
      countdownSeconds: normalizeCountdown(settings.recording.countdownSeconds),
    },
    audio: {
      system: settings.audio.system,
      systemDeviceId: settings.audio.systemDeviceId,
      microphone: settings.audio.microphone,
      microphoneDeviceId: settings.audio.microphoneDeviceId,
      noiseSuppression: settings.audio.noiseSuppression,
      microphoneGain: settings.audio.microphoneGain,
    },
    camera: {
      enabled: settings.camera.enabled,
      deviceId: settings.camera.deviceId,
      pipPreset: settings.camera.pipPreset as AppSettings['camera']['pipPreset'],
      pip: fromBoundPipConfig((settings.camera as BoundSettings['camera'] & {pip?: PIPConfig}).pip, settings.camera.pipPreset as AppSettings['camera']['pipPreset']),
    },
    whiteboard: fromBoundWhiteboardSettings((settings as BoundSettings & {whiteboard?: Partial<AppSettings['whiteboard']>}).whiteboard),
    shortcuts: normalizeShortcutSettings((settings as BoundSettings & {shortcuts?: Partial<ShortcutSettings>}).shortcuts),
    window: {
      minimizeToTray: settings.window.minimizeToTray,
      theme: normalizeTheme(settings.window.theme),
    },
    updatedAt: typeof settings.updatedAt === 'string' ? settings.updatedAt : undefined,
  }
}

function toBoundSettings(settings: AppSettings): BoundSettings {
  return {
    schemaVersion: settings.schemaVersion,
    locale: settings.locale as BoundSettings['locale'],
    source: {
      lastSourceId: settings.source.lastSourceId,
      lastSourceType: settings.source.lastSourceType,
    },
    storage: {
      dataRootDir: settings.storage.dataRootDir,
    },
    recording: {
      quality: settings.recording.quality,
      fps: settings.recording.fps,
      captureCursor: settings.recording.captureCursor,
      countdownSeconds: settings.recording.countdownSeconds,
    },
    audio: {
      system: settings.audio.system,
      systemDeviceId: settings.audio.systemDeviceId,
      microphone: settings.audio.microphone,
      microphoneDeviceId: settings.audio.microphoneDeviceId,
      noiseSuppression: settings.audio.noiseSuppression,
      microphoneGain: settings.audio.microphoneGain,
    },
    camera: {
      enabled: settings.camera.enabled,
      deviceId: settings.camera.deviceId,
      pipPreset: settings.camera.pipPreset,
      pip: settings.camera.pip as unknown as BoundSettings['camera']['pip'],
    },
    whiteboard: settings.whiteboard as unknown as BoundSettings['whiteboard'],
    shortcuts: settings.shortcuts as unknown as BoundSettings['shortcuts'],
    window: {
      minimizeToTray: settings.window.minimizeToTray,
      theme: settings.window.theme as BoundSettings['window']['theme'],
    },
    updatedAt: settings.updatedAt ?? new Date(0).toISOString(),
  }
}

function loadBrowserScreenshotHistory(): ScreenshotItem[] {
  const parsed = safeJSON(window.localStorage?.getItem(browserScreenshotHistoryKey))
  if (!Array.isArray(parsed)) return []
  return parsed.map(fromBrowserScreenshotItem).filter((item): item is ScreenshotItem => item !== null)
}

function saveBrowserScreenshotHistory(items: ScreenshotItem[]) {
  const unique = new Map<string, ScreenshotItem>()
  for (const item of items) {
    if (!item.id || unique.has(item.id)) continue
    unique.set(item.id, item)
  }
  window.localStorage?.setItem(browserScreenshotHistoryKey, JSON.stringify(Array.from(unique.values()).slice(0, 200)))
}

function createBrowserScreenshotItem(mode: string, region?: RegionSelectionSession['bounds']): ScreenshotItem {
  const createdAt = new Date().toISOString()
  const width = Math.max(64, Math.round(region?.width ?? window.innerWidth ?? 1280))
  const height = Math.max(64, Math.round(region?.height ?? window.innerHeight ?? 720))
  const id = `browser-screenshot-${Date.now()}`
  return {
    id,
    path: `browser-preview/data/screenshots/${id}.png`,
    thumbnailPath: `browser-preview/data/screenshots/thumbnails/${id}.png`,
    createdAt,
    width,
    height,
    mode,
    region: region ? {
      x: Math.round(region.x),
      y: Math.round(region.y),
      width,
      height,
    } : undefined,
    pinned: false,
    fixed: false,
  }
}

function fromBrowserScreenshotItem(value: unknown): ScreenshotItem | null {
  const record = value && typeof value === 'object' ? value as Partial<ScreenshotItem> : {}
  if (typeof record.id !== 'string' || typeof record.path !== 'string') return null
  return {
    id: record.id,
    path: record.path,
    thumbnailPath: typeof record.thumbnailPath === 'string' ? record.thumbnailPath : undefined,
    createdAt: typeof record.createdAt === 'string' ? record.createdAt : new Date().toISOString(),
    width: typeof record.width === 'number' ? record.width : 1280,
    height: typeof record.height === 'number' ? record.height : 720,
    mode: typeof record.mode === 'string' ? record.mode : 'region',
    region: record.region,
    pinned: record.pinned === true,
    fixed: record.fixed === true,
  }
}

function browserScreenshotDataUrl(item: ScreenshotItem | undefined | null): string | undefined {
  if (!item) return undefined
  const width = Math.max(64, Math.min(800, item.width || 480))
  const height = Math.max(64, Math.min(600, item.height || 320))
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#151b24"/><stop offset="1" stop-color="#263445"/></linearGradient></defs><rect width="100%" height="100%" fill="url(#g)"/><rect x="18" y="18" width="${width - 36}" height="${height - 36}" rx="18" fill="none" stroke="#ef4444" stroke-width="4"/><text x="32" y="54" fill="#f8fafc" font-family="Inter, Arial" font-size="18">RecordingFreedom screenshot preview</text><text x="32" y="84" fill="#9ca3af" font-family="Inter, Arial" font-size="13">${item.id}</text></svg>`
  return `data:image/svg+xml;base64,${window.btoa(unescape(encodeURIComponent(svg)))}`
}

function fromBrowserScreenshotPinState(value: unknown): ScreenshotPinState {
  const record = value && typeof value === 'object' ? value as Partial<ScreenshotPinState> : {}
  const item = fromBrowserScreenshotItem(record.item)
  return {
    visible: record.visible === true,
    item: item ?? undefined,
    dataUrl: typeof record.dataUrl === 'string' ? record.dataUrl : browserScreenshotDataUrl(item),
    fixed: record.fixed === true,
  }
}

function fromBrowserScreenshotWhiteboardContext(value: unknown): ScreenshotWhiteboardContext {
  const record = value && typeof value === 'object' ? value as Partial<ScreenshotWhiteboardContext> : {}
  const item = fromBrowserScreenshotItem(record.item)
  return {
    available: record.available === true && Boolean(item),
    item: item ?? undefined,
    dataUrl: typeof record.dataUrl === 'string' ? record.dataUrl : browserScreenshotDataUrl(item),
  }
}

function safeJSON(value: string | null | undefined): unknown {
  if (!value) return null
  try {
    return JSON.parse(value)
  } catch {
    return null
  }
}

function loadBrowserSettings(): AppSettings {
  const raw = window.localStorage?.getItem(browserSettingsKey)
  if (!raw) return defaultSettings
  try {
    const parsed = JSON.parse(raw)
    const next = {
      ...defaultSettings,
      ...parsed,
      source: {...defaultSettings.source, ...parsed.source},
      storage: {...defaultSettings.storage, ...parsed.storage},
      recording: {...defaultSettings.recording, ...parsed.recording},
      audio: {...defaultSettings.audio, ...parsed.audio},
      camera: {...defaultSettings.camera, ...parsed.camera},
      whiteboard: {...defaultSettings.whiteboard, ...parsed.whiteboard},
      shortcuts: normalizeShortcutSettings(parsed.shortcuts),
      window: {...defaultSettings.window, ...parsed.window},
    }
    const camera = {
      ...next.camera,
      pip: fromBoundPipConfig(next.camera.pip, next.camera.pipPreset),
    }
    camera.pipPreset = camera.pip.preset
    return {...next, camera, locale: normalizeLocale(next.locale), shortcuts: normalizeShortcutSettings(next.shortcuts), window: {...next.window, theme: normalizeTheme(next.window.theme)}}
  } catch {
    return defaultSettings
  }
}

function fromBoundWhiteboardSettings(value: Partial<AppSettings['whiteboard']> | undefined): AppSettings['whiteboard'] {
  const next = {...defaultSettings.whiteboard, ...(value ?? {})}
  return {
    enabled: next.enabled !== false,
    lastMode: next.lastMode === 'annotation' ? 'annotation' : 'board',
    lastTool: normalizeWhiteboardTool(next.lastTool),
    lastStrokeColor: typeof next.lastStrokeColor === 'string' && next.lastStrokeColor.trim() ? next.lastStrokeColor : defaultSettings.whiteboard.lastStrokeColor,
    lastStrokeWidth: next.lastStrokeWidth === 'thin' || next.lastStrokeWidth === 'bold' ? next.lastStrokeWidth : 'medium',
    lastOpacity: normalizedRange(next.lastOpacity, defaultSettings.whiteboard.lastOpacity, 5, 100),
    capturePolicy: next.capturePolicy === 'preview-only' ? 'preview-only' : 'export-compose',
  }
}

function normalizeWhiteboardTool(value: unknown): AppSettings['whiteboard']['lastTool'] {
  return value === 'selection' ||
    value === 'hand' ||
    value === 'freedraw' ||
    value === 'laser' ||
    value === 'arrow' ||
    value === 'line' ||
    value === 'rectangle' ||
    value === 'ellipse' ||
    value === 'text' ||
    value === 'eraser'
    ? value
    : 'freedraw'
}

function toStartRequest(request: MockRecordingRequest): StartRequest {
  return {
    sourceId: request.source.id,
    sourceType: toBoundSourceType(request.source.type),
    sourceName: request.source.name,
    sourceGeometry: toSourceGeometry(request.source),
    recording: request.recording,
    audio: {
      system: request.systemAudio,
      systemDeviceId: request.systemAudioDeviceId,
      microphone: request.microphone,
      microphoneDeviceId: request.microphoneDeviceId,
      noiseSuppression: request.noiseSuppression,
      microphoneGain: 1,
    },
    camera: {
      enabled: request.camera,
      deviceId: request.cameraDeviceId,
      deviceNativeId: request.cameraDeviceNativeId,
      pipPreset: request.camera ? request.pipPreset : 'off',
      pip: (request.camera ? request.pip : {...request.pip, preset: 'off'}) as unknown as StartRequest['camera']['pip'],
    },
  }
}

function fromBoundPipConfig(config: Partial<PIPConfig> | undefined, fallbackPreset: AppSettings['camera']['pipPreset']): PIPConfig {
  const preset = normalizePipPreset(config?.preset ?? fallbackPreset)
  return {
    preset,
    shape: config?.shape === 'square' ? 'square' : 'circle',
    mirror: config?.mirror !== false,
    position: {
      x: normalizedUnit(config?.position?.x ?? (preset === 'bottom-left' ? 0 : 1)),
      y: normalizedUnit(config?.position?.y ?? 1),
    },
    scale: normalizedRange(config?.scale, 0.08, 0.016, 0.08),
    edgeFeather: normalizedRange(config?.edgeFeather, 0.16, 0.02, 0.42),
  }
}

function normalizePipPreset(value: unknown): AppSettings['camera']['pipPreset'] {
  return value === 'off' || value === 'bottom-right' || value === 'bottom-left' || value === 'free' ? value : 'bottom-right'
}

function toSourceGeometry(source: CaptureSource): StartRequest['sourceGeometry'] {
  if (!source.width || !source.height) return undefined
  return {
    x: source.x ?? 0,
    y: source.y ?? 0,
    width: source.width,
    height: source.height,
    displayIndex: source.displayIndex ?? 0,
    nativeId: source.nativeId,
  }
}

function toAudioOnlyRequest(request: AudioOnlyRecordingRequest): AudioOnlyRequest {
  return {
    recording: request.recording,
    audio: {
      system: request.systemAudio,
      systemDeviceId: request.systemAudioDeviceId,
      microphone: request.microphone,
      microphoneDeviceId: request.microphoneDeviceId,
      noiseSuppression: request.noiseSuppression,
      microphoneGain: 1,
    },
  }
}

function toBoundRegionSelectionRequest(request: RegionSelectionSession['bounds']): BoundRegionSelectionRequest {
  return {
    x: Math.round(request.x),
    y: Math.round(request.y),
    width: Math.round(request.width),
    height: Math.round(request.height),
  }
}

function toBoundScreenIndicatorRequest(sourceId: string): BoundScreenIndicatorRequest {
  return {
    sourceId,
  }
}

function toBoundPipOverlayRequest(config: PIPConfig, mode: PIPOverlayMode, camera: string | PIPOverlayCamera, previewImagePath = '', clientOperationId = 0): BoundPIPOverlayRequest {
  const target = normalizePipOverlayCamera(camera)
  return {
    config: config as unknown as BoundPIPOverlayRequest['config'],
    mode,
    cameraName: target.name,
    camera: target as BoundPIPOverlayRequest['camera'],
    previewImagePath,
    clientOperationId,
  }
}

function browserPipOverlayState(config: PIPConfig, mode: PIPOverlayMode, camera: string | PIPOverlayCamera, previewImagePath = '', clientOperationId = 0): PIPOverlayState {
  const target = normalizePipOverlayCamera(camera)
  const overlayBounds = {x: 0, y: 0, width: Math.max(320, window.innerWidth || 1280), height: Math.max(240, window.innerHeight || 720)}
  const normalized = fromBoundPipConfig(config, config.preset)
  const size = Math.round(Math.max(24, Math.min(overlayBounds.width, overlayBounds.height) * normalized.scale))
  const contentBounds = {x: 24, y: 24, width: size, height: size}
  return {
    config: normalized,
    placement: {
      visible: normalized.preset !== 'off',
      rect: {...contentBounds, visible: normalized.preset !== 'off'},
      shape: normalized.shape,
      mirror: normalized.mirror,
      edgeFeather: normalized.edgeFeather,
    },
    overlayBounds,
    windowBounds: {x: 0, y: 0, width: size + 48, height: size + 48},
    contentBounds,
    mode,
    cameraName: target.name,
    camera: target,
    previewImagePath,
    captureExcluded: false,
    clientOperationId,
  }
}

function normalizePipOverlayCamera(camera: string | PIPOverlayCamera | undefined): PIPOverlayCamera {
  const target = typeof camera === 'string' ? {name: camera} : {...(camera ?? {})}
  const next = {
    deviceId: cleanOptionalString(target.deviceId),
    nativeId: cleanOptionalString(target.nativeId),
    name: cleanOptionalString(target.name),
  }
  if (!next.name) next.name = next.nativeId || next.deviceId
  return next
}

function cleanOptionalString(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined
  const trimmed = value.trim()
  return trimmed || undefined
}

function normalizeRecordingQuality(value: string): AppSettings['recording']['quality'] {
  return value === 'standard' || value === 'balanced' || value === 'high' ? value : 'balanced'
}

function normalizeRecordingFPS(value: number): number {
  return value === 24 || value === 30 || value === 60 ? value : 30
}

function normalizeCountdown(value: number): number {
  if (value < 0) return 0
  if (value > 10) return 10
  return value
}

function toBoundSourceType(type: CaptureSource['type']) {
  if (type === 'screen') return BoundCaptureSourceType.SourceScreen
  if (type === 'all-screens') return BoundCaptureSourceType.SourceAllScreens
  if (type === 'region') return BoundCaptureSourceType.SourceRegion
  if (type === 'window') return BoundCaptureSourceType.SourceWindow
  return BoundCaptureSourceType.SourceApplication
}

function fromBoundSession(session: BoundSession): RecordingSession {
  return {
    id: session.id,
    packagePath: session.packageDir,
    manifestPath: session.manifest,
    backend: session.backend,
    recordingMode: session.recordingMode,
    status: session.status,
  }
}

function fromBoundStatusEvent(event: BoundStatusEvent): RecordingStatusUpdate {
  const session = event.sessionId && event.packageDir ? {
    id: event.sessionId,
    packagePath: event.packageDir,
    manifestPath: event.manifest,
    backend: event.backend,
    recordingMode: undefined,
    status: event.status,
  } : undefined
  return {
    status: event.status,
    message: event.message,
    backend: event.backend,
    session,
  }
}

function fromWhiteboardVisibilityEvent(value: unknown): WhiteboardVisibilityUpdate {
  const record = value && typeof value === 'object' ? value as Record<string, unknown> : {}
  return {
    visible: record.visible === true,
    mode: record.mode === 'annotation' ? 'annotation' : 'whiteboard',
  }
}

function fromShortcutTriggeredEvent(value: unknown): ShortcutTriggeredUpdate {
  const record = value && typeof value === 'object' ? value as Record<string, unknown> : {}
  const action = shortcutActions.includes(record.action as ShortcutAction) ? record.action as ShortcutAction : 'toggleRecording'
  return {
    action,
    accelerator: typeof record.accelerator === 'string' ? record.accelerator : '',
  }
}

function emitBrowserWhiteboardVisibility(event: WhiteboardVisibilityUpdate) {
  const normalized = fromWhiteboardVisibilityEvent(event)
  ;(window as Window & {__RF_LAST_WHITEBOARD_VISIBILITY__?: WhiteboardVisibilityUpdate}).__RF_LAST_WHITEBOARD_VISIBILITY__ = normalized
  window.dispatchEvent(new CustomEvent(browserWhiteboardVisibilityEvent, {detail: normalized}))
}

function fromBoundAudioLevel(event: unknown): AudioLevelUpdate {
  const data = (event ?? {}) as Partial<AudioLevelUpdate>
  return {
    deviceId: typeof data.deviceId === 'string' ? data.deviceId : '',
    level: normalizedUnit(data.level),
    rms: normalizedUnit(data.rms),
    peak: normalizedUnit(data.peak),
    active: data.active === true,
    error: typeof data.error === 'string' ? data.error : undefined,
  }
}

function normalizedUnit(value: unknown): number {
  const numeric = typeof value === 'number' && Number.isFinite(value) ? value : 0
  if (numeric < 0) return 0
  if (numeric > 1) return 1
  return numeric
}

function normalizedRange(value: unknown, fallback: number, minimum: number, maximum: number): number {
  const numeric = typeof value === 'number' && Number.isFinite(value) ? value : fallback
  if (numeric < minimum) return minimum
  if (numeric > maximum) return maximum
  return numeric
}

function fromBoundRecovery(recovery: BoundRecoverySummary): RecordingRecovery {
  return {
    packagePath: recovery.packageDir,
    manifestPath: recovery.manifestPath,
    status: recovery.status,
    recoverable: recovery.recoverable,
    reason: recovery.reason,
  }
}

function fromBoundExportPlan(plan: BoundExportPlan): RecordingExportPlan {
  return {
    packageDir: plan.packageDir,
    outputPath: plan.outputPath,
    screenInputPath: plan.screenInputPath,
    webcamInputPath: plan.webcamInputPath,
    pipVisible: plan.pipLayout?.visible === true,
    annotationsVisible: plan.annotationsVisible === true,
    annotationInputPath: plan.annotationInputPath,
    annotationEventsPath: plan.annotationEventsPath,
    annotationStartMs: plan.annotationStartMs,
    annotationTimeline: plan.annotationTimeline,
    annotationRenderMode: plan.annotationRenderMode,
    annotationSnapshots: plan.annotationSnapshots?.map((snapshot) => ({
      inputPath: snapshot.inputPath,
      relativePath: snapshot.relativePath,
      startOffsetMs: snapshot.startOffsetMs,
      endOffsetMs: snapshot.endOffsetMs,
      durationMs: snapshot.durationMs,
      bytes: snapshot.bytes,
    })),
    annotationElementScenes: plan.annotationElementScenes?.map((scene) => ({
      inputPath: scene.inputPath,
      relativePath: scene.relativePath,
      renderInputPath: scene.renderInputPath,
      renderRelativePath: scene.renderRelativePath,
      startOffsetMs: scene.startOffsetMs,
      endOffsetMs: scene.endOffsetMs,
      durationMs: scene.durationMs,
      canvasWidth: scene.canvasWidth,
      canvasHeight: scene.canvasHeight,
      elementCount: scene.elementCount,
      sourceEventSequence: scene.sourceEventSequence,
      bytes: scene.bytes,
    })),
    annotationSummary: plan.annotationSummary
      ? {
        ...plan.annotationSummary,
        elementTypeCounts: normalizedNumberRecord(plan.annotationSummary.elementTypeCounts),
        elementPreviewFrames: plan.annotationSummary.elementPreviewFrames ?? undefined,
      }
      : undefined,
    warnings: plan.warnings ?? [],
  }
}

function normalizedNumberRecord(value: Record<string, number | undefined> | null | undefined): Record<string, number> | undefined {
  if (!value) return undefined
  const entries = Object.entries(value).filter((entry): entry is [string, number] => typeof entry[1] === 'number' && Number.isFinite(entry[1]))
  if (entries.length === 0) return undefined
  return Object.fromEntries(entries)
}

function fromBoundExport(result: BoundExportRecordingResult): RecordingExportResult {
  return {
    ...fromBoundExportPlan(result.plan),
    outputPath: result.export.outputPath,
    bytes: result.export.bytes,
    screenInputPath: result.export.screenInputPath,
    webcamInputPath: result.export.webcamInputPath,
    pipVisible: result.export.pipVisible,
    ffmpegPath: result.export.ffmpegPath,
    outputVerified: result.export.outputVerified === true,
  }
}
