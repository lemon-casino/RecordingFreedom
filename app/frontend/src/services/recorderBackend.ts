import {Application, Events, Window as WailsWindow} from '@wailsio/runtime'
import {RecordingFreedomService} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app'
import {
  type BootstrapState as BoundBootstrapState,
  type CapsuleWindowHitRegionsRequest as BoundCapsuleWindowHitRegionsRequest,
  type ExportRecordingResult as BoundExportRecordingResult,
  type PIPOverlayRequest as BoundPIPOverlayRequest,
  type PIPOverlayState as BoundPIPOverlayState,
  type RegionSelectionRequest as BoundRegionSelectionRequest,
  type RegionSelectionResult as BoundRegionSelectionResult,
  type RegionSelectionSession as BoundRegionSelectionSession,
  type ScreenIndicatorRequest as BoundScreenIndicatorRequest,
  type ScreenIndicatorResult as BoundScreenIndicatorResult,
} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/models'
import {
  type Capabilities as BoundCaptureCapabilities,
  type Capability as BoundCaptureCapability,
} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/capture/models'
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
const capsuleWindowWidth = 760
const capsuleWindowCompactWidth = 380
const capsuleWindowCollapsedHeight = 96
const capsuleWindowExpandedHeight = 600
export type CapsuleWindowExpandDirection = 'down' | 'up'

let lastCapsuleExpandedDirection: CapsuleWindowExpandDirection = 'down'
let lastCapsuleExpandedHeight = capsuleWindowExpandedHeight
let lastCapsuleCollapsedPosition: {x: number; y: number} | null = null

export async function setCapsuleWindowHitRegions(req: {
  enabled: boolean
  viewportWidth: number
  viewportHeight: number
  devicePixelRatio: number
  regions: CapsuleWindowHitRegion[]
}): Promise<void> {
  try {
    await RecordingFreedomService.SetCapsuleWindowHitRegions(req as BoundCapsuleWindowHitRegionsRequest)
  } catch (error) {
    console.info('Using browser capsule hit-region fallback:', error)
  }
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

export type RegionSelectionSession = {
  id: string
  bounds: {x: number; y: number; width: number; height: number}
  minimumWidth: number
  minimumHeight: number
  displayCount: number
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
  captureExcluded: boolean
}

export type RecordingExportResult = {
  outputPath: string
  bytes: number
  screenInputPath: string
  webcamInputPath?: string
  pipVisible: boolean
  ffmpegPath?: string
  outputVerified: boolean
  warnings: string[]
}

export async function setCapsuleWindowExpanded(
  expanded: boolean,
  expandedHeight = capsuleWindowExpandedHeight,
  preferredDirection: CapsuleWindowExpandDirection | 'auto' = 'auto',
  compactCollapsed = false,
): Promise<CapsuleWindowExpandDirection> {
  try {
    const position = await WailsWindow.Position()
    const size = await WailsWindow.Size().catch(() => ({
      width: expanded ? capsuleWindowWidth : compactCollapsed ? capsuleWindowCompactWidth : capsuleWindowWidth,
      height: expanded ? expandedHeight : capsuleWindowCollapsedHeight,
    }))
    const targetCollapsedWidth = compactCollapsed ? capsuleWindowCompactWidth : capsuleWindowWidth
    if (!expanded) {
      const collapsedY = lastCapsuleExpandedDirection === 'up'
        ? lastCapsuleCollapsedPosition?.y ?? position.y + Math.max(0, lastCapsuleExpandedHeight - capsuleWindowCollapsedHeight)
        : position.y
      const collapsedX = Math.round(position.x + (size.width - targetCollapsedWidth) / 2)
      await WailsWindow.SetSize(targetCollapsedWidth, capsuleWindowCollapsedHeight)
      await WailsWindow.SetPosition(collapsedX, collapsedY)
      lastCapsuleCollapsedPosition = null
      return lastCapsuleExpandedDirection
    }

    const direction = resolveCapsuleExpandDirection(position.y, expandedHeight, preferredDirection)
    const nextY = direction === 'up'
      ? Math.max(capsuleScreenTop(), position.y + capsuleWindowCollapsedHeight - expandedHeight)
      : position.y
    const expandedX = Math.round(position.x + (size.width - capsuleWindowWidth) / 2)
    lastCapsuleExpandedDirection = direction
    lastCapsuleExpandedHeight = expandedHeight
    lastCapsuleCollapsedPosition = {x: position.x, y: position.y}
    await WailsWindow.SetSize(capsuleWindowWidth, expandedHeight)
    await WailsWindow.SetPosition(expandedX, nextY)
    return direction
  } catch (error) {
    console.info('Using browser capsule window size fallback:', error)
    return preferredDirection === 'up' ? 'up' : 'down'
  }
}

function resolveCapsuleExpandDirection(
  windowY: number,
  expandedHeight: number,
  preferredDirection: CapsuleWindowExpandDirection | 'auto',
): CapsuleWindowExpandDirection {
  if (preferredDirection === 'up' || preferredDirection === 'down') return preferredDirection
  const top = capsuleScreenTop()
  const bottom = capsuleScreenBottom()
  const wouldOverflowBottom = windowY + expandedHeight > bottom
  const canFitAbove = windowY + capsuleWindowCollapsedHeight - expandedHeight >= top
  return wouldOverflowBottom && canFitAbove ? 'up' : 'down'
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

export async function showPipOverlay(config: PIPConfig, mode: PIPOverlayMode = 'edit', camera: string | PIPOverlayCamera = ''): Promise<PIPOverlayState> {
  try {
    return fromBoundPipOverlayState(await RecordingFreedomService.ShowPIPOverlay(toBoundPipOverlayRequest(config, mode, camera)))
  } catch (error) {
    console.info('Using browser PIP overlay fallback:', error)
    return browserPipOverlayState(config, mode, camera)
  }
}

export async function updatePipOverlay(config: PIPConfig, mode: PIPOverlayMode = 'edit', camera: string | PIPOverlayCamera = ''): Promise<PIPOverlayState> {
  try {
    return fromBoundPipOverlayState(await RecordingFreedomService.UpdatePIPOverlay(toBoundPipOverlayRequest(config, mode, camera)))
  } catch (error) {
    console.info('Using browser PIP overlay update fallback:', error)
    return browserPipOverlayState(config, mode, camera)
  }
}

export async function hidePipOverlay(): Promise<void> {
  try {
    await RecordingFreedomService.HidePIPOverlay()
  } catch (error) {
    console.info('Using browser PIP overlay hide fallback:', error)
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
    const next = {...settings, updatedAt: new Date().toISOString()}
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

export async function exportRecordingPackage(packagePath: string): Promise<RecordingExportResult> {
  try {
    return fromBoundExport(await RecordingFreedomService.ExportRecordingPackage({packageDir: packagePath}))
  } catch (error) {
    console.info('Desktop recording export unavailable:', error)
    throw error
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
    minimumWidth: session.minimumWidth,
    minimumHeight: session.minimumHeight,
    displayCount: session.displayCount,
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
    captureExcluded: state.captureExcluded,
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
    window: {
      minimizeToTray: settings.window.minimizeToTray,
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
    window: {
      minimizeToTray: settings.window.minimizeToTray,
    },
    updatedAt: settings.updatedAt ?? new Date(0).toISOString(),
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
      window: {...defaultSettings.window, ...parsed.window},
    }
    const camera = {
      ...next.camera,
      pip: fromBoundPipConfig(next.camera.pip, next.camera.pipPreset),
    }
    camera.pipPreset = camera.pip.preset
    return {...next, camera, locale: normalizeLocale(next.locale)}
  } catch {
    return defaultSettings
  }
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
    scale: normalizedRange(config?.scale, 0.08, 0.08, 0.32),
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

function toBoundPipOverlayRequest(config: PIPConfig, mode: PIPOverlayMode, camera: string | PIPOverlayCamera): BoundPIPOverlayRequest {
  const target = normalizePipOverlayCamera(camera)
  return {
    config: config as unknown as BoundPIPOverlayRequest['config'],
    mode,
    cameraName: target.name,
    camera: target as BoundPIPOverlayRequest['camera'],
  }
}

function browserPipOverlayState(config: PIPConfig, mode: PIPOverlayMode, camera: string | PIPOverlayCamera): PIPOverlayState {
  const target = normalizePipOverlayCamera(camera)
  const overlayBounds = {x: 0, y: 0, width: Math.max(320, window.innerWidth || 1280), height: Math.max(240, window.innerHeight || 720)}
  const normalized = fromBoundPipConfig(config, config.preset)
  const size = Math.round(Math.max(72, Math.min(overlayBounds.width, overlayBounds.height) * normalized.scale))
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
    captureExcluded: false,
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

function fromBoundExport(result: BoundExportRecordingResult): RecordingExportResult {
  return {
    outputPath: result.export.outputPath,
    bytes: result.export.bytes,
    screenInputPath: result.export.screenInputPath,
    webcamInputPath: result.export.webcamInputPath,
    pipVisible: result.export.pipVisible,
    ffmpegPath: result.export.ffmpegPath,
    outputVerified: result.export.outputVerified === true,
    warnings: result.plan.warnings ?? [],
  }
}
