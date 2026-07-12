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
  type CameraStatePatchRequest as BoundCameraStatePatchRequest,
  type ExportRecordingPlanResult as BoundExportRecordingPlanResult,
  type ExportRecordingResult as BoundExportRecordingResult,
  type FloatingPanelRequest as BoundFloatingPanelRequest,
  type FloatingPanelState as BoundFloatingPanelState,
  type FloatingRect as BoundFloatingRect,
  type FloatingSelectChosenEvent as BoundFloatingSelectChosenEvent,
  type FloatingSelectOption as BoundFloatingSelectOption,
  type FloatingSelectRequest as BoundFloatingSelectRequest,
  type FloatingSelectState as BoundFloatingSelectState,
  type OcrJobEvent as BoundOcrJobEvent,
  type PIPPreviewImageRequest as BoundPIPPreviewImageRequest,
  type PIPPreviewImageResult as BoundPIPPreviewImageResult,
  type PIPOverlayRequest as BoundPIPOverlayRequest,
  type PIPOverlayState as BoundPIPOverlayState,
  type RegionAssistRequest as BoundRegionAssistRequest,
  type RegionAssistResult as BoundRegionAssistResult,
  type RegionSelectionRequest as BoundRegionSelectionRequest,
  type RegionSelectionResult as BoundRegionSelectionResult,
  type RegionSelectionSession as BoundRegionSelectionSession,
  type RegionSmartCandidate as BoundRegionSmartCandidate,
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
  type SourceControlState as BoundSourceControlState,
  type SourceGeometry as BoundSourceGeometry,
  type SourceStatePatchRequest as BoundSourceStatePatchRequest,
  type WhiteboardExportRequest as BoundWhiteboardExportRequest,
  type WhiteboardExportResult as BoundWhiteboardExportResult,
  type WhiteboardSceneRequest as BoundWhiteboardSceneRequest,
  type WhiteboardSceneResult as BoundWhiteboardSceneResult,
  type WhiteboardSettingsPatchRequest as BoundWhiteboardSettingsPatchRequest,
  type WhiteboardSnapshotRequest as BoundWhiteboardSnapshotRequest,
  type WhiteboardSnapshotResult as BoundWhiteboardSnapshotResult,
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
import {
  type Block as BoundOcrBlock,
  type JobSnapshot as BoundOcrJobSnapshot,
  type ModelDownloadEvent as BoundOcrModelDownloadEvent,
  type ModelDownloadSnapshot as BoundOcrModelDownloadSnapshot,
  type ModelInfo as BoundOcrModelInfo,
  type RecognizeRequest as BoundOcrRecognizeRequest,
  type Result as BoundOcrResult,
  SourceKind as BoundOcrSourceKind,
  type Status as BoundOcrStatus,
  type TranslateRequest as BoundOcrTranslateRequest,
  type TranslationBlock as BoundOcrTranslationBlock,
  type TranslationResult as BoundOcrTranslationResult,
  type WhiteboardRequest as BoundOcrWhiteboardRequest,
  type WorkerCapabilities as BoundOcrWorkerCapabilities,
} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/ocr/models'
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
  type RecordingMode,
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

export type CameraStatePatch = {
  enabled?: boolean
  deviceId?: string
  pipPreset?: AppSettings['camera']['pipPreset']
  pip?: PIPConfig
}

export type SettingsPreferencesPatch = {
  theme?: AppSettings['window']['theme']
  recordingQuality?: AppSettings['recording']['quality']
  recordingFps?: number
  captureCursor?: boolean
  countdownSeconds?: number
  startAtLogin?: boolean
  autoOcr?: boolean
  ocrTranslation?: Partial<AppSettings['ocr']['translation']>
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
  pins?: ScreenshotPinnedItem[]
}

export type ScreenshotPinnedItem = {
  item: ScreenshotItem
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

export type OcrSourceKind = 'region-screenshot' | 'full-screenshot' | 'window-screenshot' | 'focused-window-screenshot' | 'scrolling-screenshot' | 'pinned-screenshot' | 'whiteboard' | 'whiteboard-selection' | 'image'

export type OcrModelInfo = {
  id: string
  name: string
  channel: string
  engine: string
  language: string[]
  version?: string
  sourceUrl?: string
  license?: string
  downloadAvailable: boolean
  downloadBytes?: number
  installed: boolean
  verified: boolean
  active: boolean
  modelDir?: string
  smokeImage?: string
  smokeExpected?: string
  smokeAssetReady: boolean
  smokeError?: string
  missingFiles: string[]
  verificationError?: string
}

export type OcrStatus = {
  status: string
  activeModelId?: string
  models: OcrModelInfo[]
  workerPath?: string
  runtimeDir?: string
  workerCapabilities?: OcrWorkerCapabilities
  message?: string
}

export type OcrModelDownloadSnapshot = {
  id: string
  modelId: string
  status: 'queued' | 'running' | 'installed' | 'failed' | 'cancelled' | string
  downloadedBytes: number
  totalBytes: number
  percent: number
  error?: string
  model?: OcrModelInfo
  startedAt: string
  updatedAt: string
}

export type OcrJobSnapshot = {
  jobId: string
  status: string
  cacheKey?: string
  request: OcrRecognizeRequest
  merged: boolean
  createdAt: string
  updatedAt: string
}

export type OcrJobUpdate = {
  jobId: string
  sourceKind: OcrSourceKind
  sourceId: string
  status: string
  cacheKey?: string
  merged?: boolean
  error?: string
  result?: OcrResult
}

export type OcrWorkerCapabilities = {
  schemaVersion: number
  name: string
  version: string
  protocolVersion: string
  engine: string
  modelFormats: string[]
  supportsRecognize: boolean
  runtimeDir?: string
  runtimeLibrary?: string
  runtimeAvailable: boolean
  runtimeVersion?: string
  runtimeApiVersion?: number
  runtimeError?: string
  message?: string
}

export type OcrPoint = {x: number; y: number}

export type OcrBlock = {
  id: string
  text: string
  confidence: number
  box: OcrPoint[]
  lineIndex: number
  languageHint?: string
}

export type OcrResult = {
  id: string
  sourceKind: OcrSourceKind
  sourceId: string
  imagePath: string
  imageSha256: string
  modelId: string
  language: string
  width: number
  height: number
  blocks: OcrBlock[]
  plainText: string
  createdAt: string
  durationMs: number
}

export type OcrRecognizeRequest = {
  imagePath: string
  sourceKind: OcrSourceKind
  sourceId: string
  language?: string
  modelId?: string
  force?: boolean
  priority?: 'interactive' | 'normal' | 'background' | string
}

export type OcrWhiteboardRequest = {
  imagePath: string
  elementId?: string
  sceneId?: string
  language?: string
  force?: boolean
  priority?: 'interactive' | 'normal' | 'background' | string
}

export type OcrTranslateRequest = {
  ocrResultId: string
  blockIds?: string[]
  provider: string
  sourceLanguage: string
  targetLanguage: string
  baseUrl?: string
  apiKey?: string
  model?: string
  force?: boolean
}

export type OcrTranslationBlock = {
  blockId: string
  source: string
  translated: string
}

export type OcrTranslationResult = {
  ocrResultId: string
  provider: string
  sourceLanguage: string
  targetLanguage: string
  model?: string
  promptVersion?: string
  blocks: OcrTranslationBlock[]
  createdAt: string
}

export type AnnotationOverlayState = {
  mode?: 'annotation' | 'screenshot'
  packageDir?: string
  manifestPath?: string
  windowBounds: {x: number; y: number; width: number; height: number}
  canvasBounds: {x: number; y: number; width: number; height: number}
  toolbarBounds?: {x: number; y: number; width: number; height: number}
  toolbarPlacement?: 'top' | 'bottom'
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

export type SourceControlState = {
  recordingMode: RecordingMode
  sourceId?: string
  sourceType?: CaptureSource['type']
  sourceGeometry?: {
    x: number
    y: number
    width: number
    height: number
    displayIndex?: number
    nativeId?: string
  }
}

export type SourceStatePatch = Partial<SourceControlState> & {
  clearGeometry?: boolean
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
const browserScreenshotWhiteboardEvent = 'rf-screenshot-whiteboard'
const browserCapsuleDockSideEvent = 'rf-capsule-dock-side'
const browserFloatingPanelEvent = 'rf-floating-panel'
const browserFloatingSelectEvent = 'rf-floating-select'
const browserFloatingSelectChosenEvent = 'rf-floating-select-chosen'
const browserSourceStateEvent = 'rf-source-state'
const browserOcrJobEvent = 'rf-ocr-job'
const browserOcrModelDownloadEvent = 'rf-ocr-model-download'
const settingsSchemaVersion = 4
const legacyPipMinimumScale = 0.016
const legacyPipMaximumScale = 0.08
const pipMinimumScale = 0.08
const pipMaximumScale = 0.15
const capsuleWindowWidth = 760
const capsuleWindowCompactWidth = 380
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

export function isWailsDesktopRuntime(): boolean {
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
      contextId: req.contextId,
    }
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
      direction: req.direction,
    }
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

export async function patchSourceState(patch: SourceStatePatch): Promise<SourceControlState> {
  try {
    return fromBoundSourceControlState(await RecordingFreedomService.PatchSourceState(toBoundSourceStatePatch(patch)))
  } catch (error) {
    console.info('Using browser source state patch fallback:', error)
    const current = (window as Window & {__RF_SOURCE_STATE__?: SourceControlState}).__RF_SOURCE_STATE__ ?? {
      recordingMode: 'video' as RecordingMode,
      sourceType: 'screen' as CaptureSource['type'],
    }
    const next: SourceControlState = {
      ...current,
      ...patch,
      sourceGeometry: patch.clearGeometry ? undefined : patch.sourceGeometry ?? current.sourceGeometry,
      recordingMode: patch.recordingMode ?? current.recordingMode,
    }
    ;(window as Window & {__RF_SOURCE_STATE__?: SourceControlState}).__RF_SOURCE_STATE__ = next
    window.dispatchEvent(new CustomEvent(browserSourceStateEvent, {detail: next}))
    return next
  }
}

export async function getSourceState(): Promise<SourceControlState> {
  try {
    return fromBoundSourceControlState(await RecordingFreedomService.GetSourceState())
  } catch {
    return (window as Window & {__RF_SOURCE_STATE__?: SourceControlState}).__RF_SOURCE_STATE__ ?? {
      recordingMode: 'video',
      sourceType: 'screen',
    }
  }
}

export function subscribeSourceStateChanged(handler: (state: SourceControlState) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('source.state.changed', (event) => {
      handler(fromBoundSourceControlState(event.data as BoundSourceControlState))
    })
  } catch (error) {
    console.info('Desktop source state events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler((event as CustomEvent<SourceControlState>).detail)
  }
  window.addEventListener(browserSourceStateEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserSourceStateEvent, onBrowserEvent)
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

export type RegionSelectionPurpose = 'capture' | 'annotation' | 'screenshot' | 'scrolling-screenshot'

export type RegionSmartCandidate = {
  id: string
  kind: 'screen' | 'window' | 'element' | 'edge' | string
  label?: string
  bounds: {x: number; y: number; width: number; height: number}
  sourceId?: string
  score?: number
}

export type RegionDisplayBounds = {
  id?: string
  bounds: {x: number; y: number; width: number; height: number}
  captureBounds: {x: number; y: number; width: number; height: number}
  scaleFactor?: number
}

export type RegionSelectionSession = {
  id: string
  bounds: {x: number; y: number; width: number; height: number}
  captureBounds?: {x: number; y: number; width: number; height: number}
  displayBounds?: RegionDisplayBounds[]
  minimumWidth: number
  minimumHeight: number
  displayCount: number
  purpose?: RegionSelectionPurpose
  candidates?: RegionSmartCandidate[]
  initialPointer?: {x: number; y: number}
}

export type RegionAssistRequest = {
  sessionId?: string
  purpose?: RegionSelectionPurpose
  pointerX?: number
  pointerY?: number
  selection?: RegionSelectionSession['bounds']
  candidateLevel?: number
  candidates?: RegionSmartCandidate[]
}

export type RegionAssistResult = {
  candidates: RegionSmartCandidate[]
  best?: RegionSmartCandidate
  source?: 'element' | 'image-hover' | 'selection' | 'static'
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
    if (!expanded) {
      const collapsedPosition = capsuleReservedWindowPosition(dockSide, collapsedVisualPosition, collapsedVisualSize, collapsedVisualSize, workArea)
      await setCapsuleWindowBoundsIfChanged(position, size, collapsedPosition, collapsedVisualSize)
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
): CapsuleDockTarget {
  if (!workAreas.length) return {side: 'none', workArea: null}
  const activeWorkArea = capsuleWorkAreaForVisualRectFromAreas(position, size, workAreas)
  if (!activeWorkArea) return {side: 'none', workArea: null}
  const vertical = (['left', 'right'] as const)
    .filter((side) => isExternalWorkAreaEdge(side, activeWorkArea, workAreas))
    .map((side) => capsuleDockCandidate(side, position, size, activeWorkArea))
    .filter((candidate): candidate is CapsuleDockCandidate => candidate !== null)
  vertical.sort(compareCapsuleDockCandidates)
  if (vertical[0]) return {side: vertical[0].side, workArea: vertical[0].workArea}

  const horizontal = (['top', 'bottom'] as const)
    .filter((side) => isExternalWorkAreaEdge(side, activeWorkArea, workAreas))
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
}) {
  const target = resolveCapsuleDockTarget(input.position, input.size, input.workAreas)
  return {
    side: target.side,
    workArea: target.workArea ? {...target.workArea} : null,
  }
}

if (import.meta.env.DEV && typeof window !== 'undefined') {
  ;(window as Window & {
    __RF_TEST_RESOLVE_CAPSULE_DOCK_TARGET__?: typeof __resolveCapsuleDockTargetForTest
  }).__RF_TEST_RESOLVE_CAPSULE_DOCK_TARGET__ = __resolveCapsuleDockTargetForTest
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

function capsuleWorkAreaForVisualRectFromAreas(
  position: {x: number; y: number},
  size: {width: number; height: number},
  candidates: CapsuleWorkArea[],
): CapsuleWorkArea | null {
  if (!candidates.length) return null
  const right = position.x + size.width
  const bottom = position.y + size.height
  const intersections = candidates
    .map((area) => {
      const areaRight = area.x + area.width
      const areaBottom = area.y + area.height
      return {
        area,
        areaPixels: overlapLength(position.x, right, area.x, areaRight) * overlapLength(position.y, bottom, area.y, areaBottom),
      }
    })
    .sort((a, b) => b.areaPixels - a.areaPixels)
  if (intersections[0]?.areaPixels > 0) return intersections[0].area
  return capsuleWorkAreaForPositionFromAreas(position, size, candidates)
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
    screenDistance: distanceToWorkArea(centerX, centerY, workArea),
  }
}

function isExternalWorkAreaEdge(
  side: Exclude<CapsuleWindowDockSide, 'none'>,
  workArea: CapsuleWorkArea,
  workAreas: CapsuleWorkArea[],
) {
  const edgeTolerance = 2
  const areaRight = workArea.x + workArea.width
  const areaBottom = workArea.y + workArea.height
  return !workAreas.some((other) => {
    if (other === workArea) return false
    const otherRight = other.x + other.width
    const otherBottom = other.y + other.height
    if (side === 'left') {
      return Math.abs(otherRight - workArea.x) <= edgeTolerance &&
        overlapLength(workArea.y, areaBottom, other.y, otherBottom) > edgeTolerance
    }
    if (side === 'right') {
      return Math.abs(other.x - areaRight) <= edgeTolerance &&
        overlapLength(workArea.y, areaBottom, other.y, otherBottom) > edgeTolerance
    }
    if (side === 'top') {
      return Math.abs(otherBottom - workArea.y) <= edgeTolerance &&
        overlapLength(workArea.x, areaRight, other.x, otherRight) > edgeTolerance
    }
    return Math.abs(other.y - areaBottom) <= edgeTolerance &&
      overlapLength(workArea.x, areaRight, other.x, otherRight) > edgeTolerance
  })
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
  if (!sizeChanged && !positionChanged) return
  const bounds = {
    x: Math.round(targetPosition.x),
    y: Math.round(targetPosition.y),
    width: Math.round(targetSize.width),
    height: Math.round(targetSize.height),
  }
  try {
    await RecordingFreedomService.SetCapsuleWindowBounds(bounds)
    return
  } catch (error) {
    console.info('Using runtime capsule bounds fallback:', error)
  }
  if (positionChanged) await WailsWindow.SetPosition(bounds.x, bounds.y)
  if (sizeChanged) await WailsWindow.SetSize(bounds.width, bounds.height)
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

export function subscribeScreenshotHistoryChanged(handler: (items: ScreenshotItem[]) => void): () => void {
  try {
    return Events.On('screenshot.history.changed', (event) => {
      handler(fromBoundScreenshotHistory(event.data as BoundScreenshotHistoryResult))
    })
  } catch (error) {
    console.info('Using browser screenshot history event fallback:', error)
    return () => {}
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

export function subscribeOcrJobEvents(handler: (event: OcrJobUpdate) => void): () => void {
  const eventNames = ['ocr.job.queued', 'ocr.job.started', 'ocr.job.finished', 'ocr.job.failed', 'ocr.job.cancelled'] as const
  const disposers: Array<() => void> = []
  for (const eventName of eventNames) {
    try {
      disposers.push(Events.On(eventName, (event) => {
        handler(fromBoundOcrJobEvent(event.data as BoundOcrJobEvent))
      }))
    } catch (error) {
      console.info('Using browser OCR job event fallback:', error)
    }
  }
  const onBrowserEvent = (event: Event) => {
    if (isWailsDesktopRuntime()) return
    handler(fromBrowserOcrJobEvent((event as CustomEvent<Partial<OcrJobUpdate>>).detail))
  }
  window.addEventListener(browserOcrJobEvent, onBrowserEvent)
  return () => {
    for (const dispose of disposers) dispose()
    window.removeEventListener(browserOcrJobEvent, onBrowserEvent)
  }
}

export function subscribeOcrModelDownloadEvents(handler: (snapshot: OcrModelDownloadSnapshot) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('ocr.model.download.changed', (event) => {
      handler(fromBoundOcrModelDownloadEvent(event.data as BoundOcrModelDownloadEvent).snapshot)
    })
  } catch (error) {
    console.info('Using browser OCR model download event fallback:', error)
  }
  const onBrowserEvent = (event: Event) => {
    if (isWailsDesktopRuntime()) return
    handler(fromBrowserOcrModelDownloadEvent((event as CustomEvent<Partial<OcrModelDownloadSnapshot>>).detail).snapshot)
  }
  window.addEventListener(browserOcrModelDownloadEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserOcrModelDownloadEvent, onBrowserEvent)
  }
}

export function subscribeScreenshotWhiteboardContext(handler: (context: ScreenshotWhiteboardContext) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('screenshot.whiteboard', (event) => {
      handler(fromBoundScreenshotWhiteboardContext(event.data as BoundScreenshotWhiteboardContext))
    })
  } catch (error) {
    console.info('Desktop screenshot whiteboard events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler(fromBrowserScreenshotWhiteboardContext((event as CustomEvent<ScreenshotWhiteboardContext>).detail))
  }
  window.addEventListener(browserScreenshotWhiteboardEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserScreenshotWhiteboardEvent, onBrowserEvent)
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

export async function patchCameraState(patch: CameraStatePatch): Promise<AppSettings> {
  try {
    return fromBoundSettings(await RecordingFreedomService.PatchCameraState(patch as BoundCameraStatePatchRequest))
  } catch (error) {
    console.info('Using browser camera state patch fallback:', error)
    const current = loadBrowserSettings()
    const enabled = patch.enabled ?? current.camera.enabled
    const pipConfig = fromBoundPipConfig(patch.pip ?? current.camera.pip, patch.pipPreset ?? current.camera.pipPreset)
    const nextPip = enabled ? pipConfig : fromBoundPipConfig({...pipConfig, preset: 'off'}, 'off')
    const next: AppSettings = {
      ...current,
      camera: {
        enabled,
        deviceId: patch.deviceId ?? current.camera.deviceId,
        pipPreset: nextPip.preset,
        pip: nextPip,
      },
      updatedAt: new Date().toISOString(),
    }
    window.localStorage?.setItem(browserSettingsKey, JSON.stringify(next))
    return next
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
    return finalizeRegionSelectionSession(
      fromBoundRegionSelectionSession(await RecordingFreedomService.ShowAnnotationRegionSelector()),
      'annotation',
      `browser-annotation-region-${Date.now()}`,
    )
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation region selector fallback:', error)
    return emitBrowserRegionSession(browserRegionSelectionSession('annotation', `browser-annotation-region-${Date.now()}`))
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
    return finalizeRegionSelectionSession(
      fromBoundRegionSelectionSession(await RecordingFreedomService.ShowScreenshotRegionSelector()),
      'screenshot',
      `browser-screenshot-${Date.now()}`,
    )
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot region selector fallback:', error)
    return emitBrowserRegionSession(browserRegionSelectionSession('screenshot', `browser-screenshot-${Date.now()}`))
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

export async function completeScrollingScreenshotSelection(request: RegionSelectionSession['bounds']): Promise<ScreenshotItem> {
  try {
    const result = await RecordingFreedomService.CompleteScrollingScreenshotSelection(toBoundRegionSelectionRequest(request))
    return fromBoundScreenshotCaptureResult(result as BoundScreenshotCaptureResult).item
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser scrolling screenshot completion fallback:', error)
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
  if (!isWailsDesktopRuntime()) {
    const sceneJson = window.localStorage?.getItem(browserWhiteboardSceneKey) ?? ''
    return {
      available: sceneJson.trim() !== '',
      scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
      sceneJson,
      bytes: sceneJson.length,
      contentType: 'application/vnd.excalidraw+json',
    }
  }
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
  if (!isWailsDesktopRuntime()) {
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

export async function saveWhiteboardSnapshot(request: {sceneJson: string; snapshotDataUrl: string}): Promise<{scene: WhiteboardScene; item: ScreenshotItem}> {
  if (!isWailsDesktopRuntime()) {
    window.localStorage?.setItem(browserWhiteboardSceneKey, request.sceneJson)
    const item = createBrowserScreenshotItem('whiteboard')
    saveBrowserScreenshotHistory([item, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: item}))
    return {
      scene: {
        available: true,
        scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
        sceneJson: request.sceneJson,
        bytes: request.sceneJson.length,
        updatedAt: new Date().toISOString(),
        contentType: 'application/vnd.excalidraw+json',
      },
      item,
    }
  }
  try {
    return fromBoundWhiteboardSnapshot(await RecordingFreedomService.SaveWhiteboardSnapshot(request as BoundWhiteboardSnapshotRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard snapshot save fallback:', error)
    window.localStorage?.setItem(browserWhiteboardSceneKey, request.sceneJson)
    const item = createBrowserScreenshotItem('whiteboard')
    saveBrowserScreenshotHistory([item, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: item}))
    return {
      scene: {
        available: true,
        scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
        sceneJson: request.sceneJson,
        bytes: request.sceneJson.length,
        updatedAt: new Date().toISOString(),
        contentType: 'application/vnd.excalidraw+json',
      },
      item,
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

export async function listOcrModels(): Promise<OcrModelInfo[]> {
  try {
    return (await RecordingFreedomService.ListOcrModels() ?? []).map(fromBoundOcrModelInfo)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model fallback:', error)
    return browserOcrStatus().models
  }
}

export async function getOcrStatus(): Promise<OcrStatus> {
  try {
    return fromBoundOcrStatus(await RecordingFreedomService.GetOcrStatus())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR status fallback:', error)
    return browserOcrStatus()
  }
}

export async function installOcrModel(modelId: string): Promise<OcrModelInfo> {
  try {
    return fromBoundOcrModelInfo(await RecordingFreedomService.InstallOcrModel(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR install fallback:', error)
    throw new Error(`OCR model ${modelId} is not available in browser preview`)
  }
}

export async function installOcrModelPackage(packagePath: string): Promise<OcrModelInfo> {
  try {
    return fromBoundOcrModelInfo(await RecordingFreedomService.InstallOcrModelPackage(packagePath))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR package install fallback:', error)
    throw new Error('OCR model package import is local-only and is not available in browser preview')
  }
}

export async function removeOcrModel(modelId: string): Promise<OcrStatus> {
  try {
    return fromBoundOcrStatus(await RecordingFreedomService.RemoveOcrModel(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR remove fallback:', error)
    return browserOcrStatus()
  }
}

export async function setActiveOcrModel(modelId: string): Promise<OcrStatus> {
  try {
    return fromBoundOcrStatus(await RecordingFreedomService.SetActiveOcrModel(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR active model fallback:', error)
    const browserWindow = window as Window & {
      __RF_OCR_STATUS__?: OcrStatus
      __RF_SET_ACTIVE_OCR_MODEL_ERROR__?: string
      __RF_LAST_SET_ACTIVE_OCR_MODEL__?: {modelId: string; at: string}
    }
    browserWindow.__RF_LAST_SET_ACTIVE_OCR_MODEL__ = {modelId, at: new Date().toISOString()}
    if (browserWindow.__RF_SET_ACTIVE_OCR_MODEL_ERROR__) {
      throw new Error(browserWindow.__RF_SET_ACTIVE_OCR_MODEL_ERROR__)
    }
    const current = browserOcrStatus()
    const target = current.models.find((model) => model.id === modelId)
    if (!target?.installed || !target.verified) {
      throw new Error(`OCR model ${modelId} is not installed or failed verification`)
    }
    const next: OcrStatus = {
      ...current,
      activeModelId: modelId,
      models: current.models.map((model) => ({...model, active: model.id === modelId})),
    }
    browserWindow.__RF_OCR_STATUS__ = next
    return next
  }
}

export async function startOcrModelDownload(modelId: string): Promise<OcrModelDownloadSnapshot> {
  try {
    return fromBoundOcrModelDownloadSnapshot(await RecordingFreedomService.StartOcrModelDownload(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model download fallback:', error)
    return browserStartOcrModelDownload(modelId)
  }
}

export async function cancelOcrModelDownload(modelId: string): Promise<OcrModelDownloadSnapshot> {
  try {
    return fromBoundOcrModelDownloadSnapshot(await RecordingFreedomService.CancelOcrModelDownload(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model download cancel fallback:', error)
    return browserCancelOcrModelDownload(modelId)
  }
}

export async function getOcrModelDownloads(): Promise<OcrModelDownloadSnapshot[]> {
  try {
    return (await RecordingFreedomService.GetOcrModelDownloads() ?? []).map(fromBoundOcrModelDownloadSnapshot)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model downloads fallback:', error)
    return Object.values(browserOcrModelDownloads())
  }
}

export async function refreshOcrModelCatalog(catalogUrl = ''): Promise<OcrStatus> {
  try {
    return fromBoundOcrStatus(await RecordingFreedomService.RefreshOcrModelCatalog(catalogUrl))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model catalog refresh fallback:', error)
    const browserWindow = window as Window & {
      __RF_OCR_STATUS__?: OcrStatus
      __RF_OCR_MODEL_CATALOG_STATUS__?: OcrStatus
      __RF_LAST_OCR_MODEL_CATALOG_REFRESH__?: {catalogUrl: string; at: string}
    }
    browserWindow.__RF_LAST_OCR_MODEL_CATALOG_REFRESH__ = {catalogUrl, at: new Date().toISOString()}
    if (!browserWindow.__RF_OCR_MODEL_CATALOG_STATUS__) {
      throw new Error('OCR model catalog refresh is desktop-only in browser preview')
    }
    browserWindow.__RF_OCR_STATUS__ = browserWindow.__RF_OCR_MODEL_CATALOG_STATUS__
    return browserWindow.__RF_OCR_STATUS__
  }
}

export async function recognizeImage(request: OcrRecognizeRequest): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.RecognizeImage(toBoundOcrRecognizeRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR recognize fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function queueRecognizeImage(request: OcrRecognizeRequest): Promise<OcrJobSnapshot> {
  try {
    return fromBoundOcrJobSnapshot(await RecordingFreedomService.QueueRecognizeImage(toBoundOcrRecognizeRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR queue fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function queueRecognizeScreenshot(id: string): Promise<OcrJobSnapshot> {
  try {
    return fromBoundOcrJobSnapshot(await RecordingFreedomService.QueueRecognizeScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot OCR queue fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function recognizeScreenshot(id: string): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.RecognizeScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot OCR fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function queueRecognizePinnedScreenshot(id: string): Promise<OcrJobSnapshot> {
  try {
    return fromBoundOcrJobSnapshot(await RecordingFreedomService.QueueRecognizePinnedScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser pinned OCR queue fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function recognizePinnedScreenshot(id: string): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.RecognizePinnedScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser pinned OCR fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function queueRecognizeWhiteboard(request: OcrWhiteboardRequest): Promise<OcrJobSnapshot> {
  try {
    return fromBoundOcrJobSnapshot(await RecordingFreedomService.QueueRecognizeWhiteboard({
      ...request,
      force: request.force === true,
    } as BoundOcrWhiteboardRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard OCR queue fallback:', error)
    return browserQueuedWhiteboardOcrSnapshot(request)
  }
}

export async function recognizeWhiteboard(request: OcrWhiteboardRequest): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.RecognizeWhiteboard({
      ...request,
      force: request.force === true,
    } as BoundOcrWhiteboardRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard OCR fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function translateOcr(request: OcrTranslateRequest): Promise<OcrTranslationResult> {
  try {
    return fromBoundOcrTranslationResult(await RecordingFreedomService.TranslateOcr(request as BoundOcrTranslateRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR translation fallback:', error)
    return browserTranslateOcr(request)
  }
}

export async function openOcrResult(resultId: string): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.OpenOcrResult(resultId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR result fallback:', error)
    return browserOcrResult(resultId)
  }
}

export async function readOcrResultImage(resultId: string): Promise<ScreenshotImage> {
  try {
    return fromBoundScreenshotImage(await RecordingFreedomService.ReadOcrResultImage(resultId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR result image fallback:', error)
    return browserOcrResultImage(resultId)
  }
}

export async function cancelOcrJob(jobId: string): Promise<void> {
  try {
    await RecordingFreedomService.CancelOcrJob(jobId)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR cancel fallback:', error)
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
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id) ?? null
    ;(window as Window & {__RF_LAST_OPEN_SCREENSHOT_DIRECTORY__?: {id: string; path?: string; at: string}}).__RF_LAST_OPEN_SCREENSHOT_DIRECTORY__ = {
      id,
      path: item?.path,
      at: new Date().toISOString(),
    }
    return item
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
        pinned: false,
        fixed: patch.pinned === false ? false : patch.fixed ?? item.fixed,
      }
      : item)
    saveBrowserScreenshotHistory(next)
    const pinned = fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
    if (screenshotPinStateContains(pinned, id)) {
      const updated = next.find((item) => item.id === id)
      const state = updated ? updateBrowserScreenshotPinStateItem(pinned, updated) : removeBrowserScreenshotPinStateItem(pinned, id)
      window.localStorage?.setItem(browserScreenshotPinStateKey, JSON.stringify(state))
      window.dispatchEvent(new CustomEvent(browserScreenshotPinEvent, {detail: state}))
    }
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
    if (screenshotPinStateContains(pinned, id)) {
      const state = removeBrowserScreenshotPinStateItem(pinned, id)
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
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id)
    const current = fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
    const state = item ? appendBrowserScreenshotPinStateItem(current, {
      item,
      dataUrl: browserScreenshotDataUrl(item),
      fixed: item.fixed === true,
    }) : fromBrowserScreenshotPinState({visible: false, fixed: false})
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
  if (!isWailsDesktopRuntime()) {
    return fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
  }
  try {
    return fromBoundScreenshotPinState(await RecordingFreedomService.LoadPinnedScreenshot())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot pin state fallback:', error)
    return fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
  }
}

export async function openScreenshotInWhiteboard(id: string): Promise<ScreenshotWhiteboardContext> {
  if (!isWailsDesktopRuntime()) {
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id)
    const context: ScreenshotWhiteboardContext = {
      available: Boolean(item),
      item,
      dataUrl: browserScreenshotDataUrl(item),
    }
    window.localStorage?.setItem(browserScreenshotWhiteboardKey, JSON.stringify(context))
    window.dispatchEvent(new CustomEvent(browserScreenshotWhiteboardEvent, {detail: context}))
    const popup = window.open('/#/whiteboard', 'recordingfreedom-whiteboard', 'width=1120,height=760')
    popup?.focus()
    emitBrowserWhiteboardVisibility({visible: true, mode: 'whiteboard'})
    return context
  }
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
    window.dispatchEvent(new CustomEvent(browserScreenshotWhiteboardEvent, {detail: context}))
    const popup = window.open('/#/whiteboard', 'recordingfreedom-whiteboard', 'width=1120,height=760')
    popup?.focus()
    emitBrowserWhiteboardVisibility({visible: true, mode: 'whiteboard'})
    return context
  }
}

export async function consumeScreenshotWhiteboardContext(): Promise<ScreenshotWhiteboardContext> {
  if (!isWailsDesktopRuntime()) {
    const context = fromBrowserScreenshotWhiteboardContext(safeJSON(window.localStorage?.getItem(browserScreenshotWhiteboardKey)))
    window.localStorage?.removeItem(browserScreenshotWhiteboardKey)
    return context
  }
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

export async function startScrollingScreenshot(): Promise<RegionSelectionSession> {
  try {
    return finalizeRegionSelectionSession(
      fromBoundRegionSelectionSession(await RecordingFreedomService.StartScrollingScreenshot()),
      'scrolling-screenshot',
      `browser-scrolling-screenshot-${Date.now()}`,
    )
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser scrolling screenshot fallback:', error)
    const session = emitBrowserRegionSession(browserRegionSelectionSession('scrolling-screenshot', `browser-scrolling-screenshot-${Date.now()}`))
    const popup = window.open('/#/region-overlay', 'recordingfreedom-scrolling-screenshot', 'width=1280,height=720')
    popup?.focus()
    return session
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
    return finalizeRegionSelectionSession(
      fromBoundRegionSelectionSession(await RecordingFreedomService.ShowRegionSelector()),
      'capture',
      `browser-region-${Date.now()}`,
    )
  } catch (error) {
    console.info('Using browser region selector fallback:', error)
    return emitBrowserRegionSession(browserRegionSelectionSession('capture', `browser-region-${Date.now()}`))
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
    const result = {source, geometry: request, cancelled: false}
    ;(window as Window & {__RF_LAST_REGION_SELECTION__?: RegionSelectionResult}).__RF_LAST_REGION_SELECTION__ = result
    return result
  }
}

export async function assistRegionSelection(request: RegionAssistRequest): Promise<RegionAssistResult> {
  try {
    return fromBoundRegionAssistResult(await RecordingFreedomService.AssistRegionSelection(toBoundRegionAssistRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser region assist fallback:', error)
    return browserRegionAssist(request)
  }
}

export async function cancelRegionSelector(): Promise<RegionSelectionResult> {
  try {
    return fromBoundRegionSelectionResult(await RecordingFreedomService.CancelRegionSelection())
  } catch (error) {
    console.info('Using browser region selection cancel fallback:', error)
    ;(window as Window & {__RF_LAST_REGION_CANCEL__?: {cancelled: boolean; at: string}}).__RF_LAST_REGION_CANCEL__ = {
      cancelled: true,
      at: new Date().toISOString(),
    }
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

function browserRegionSelectionSession(purpose: NonNullable<RegionSelectionSession['purpose']>, id: string): RegionSelectionSession {
  const previous = (window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}).__RF_REGION_SESSION__
  const bounds = {x: 0, y: 0, width: window.innerWidth, height: window.innerHeight}
  return {
    id,
    bounds,
    captureBounds: bounds,
    minimumWidth: purpose === 'screenshot' ? 12 : 64,
    minimumHeight: purpose === 'screenshot' ? 12 : 64,
    displayCount: 1,
    purpose,
    candidates: previous?.candidates ? [...previous.candidates] : undefined,
  }
}

function finalizeRegionSelectionSession(
  session: RegionSelectionSession,
  purpose: NonNullable<RegionSelectionSession['purpose']>,
  browserFallbackId: string,
): RegionSelectionSession {
  if (isWailsDesktopRuntime()) return session
  const previous = (window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}).__RF_REGION_SESSION__
  const hasUsableBounds = session.bounds.width > 0 && session.bounds.height > 0
  const next = hasUsableBounds ? {...session, purpose: session.purpose ?? purpose} : browserRegionSelectionSession(purpose, browserFallbackId)
  if ((!next.candidates || next.candidates.length === 0) && previous?.candidates?.length) {
    next.candidates = [...previous.candidates]
  }
  return emitBrowserRegionSession(next)
}

function emitBrowserRegionSession(session: RegionSelectionSession): RegionSelectionSession {
  ;(window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}).__RF_REGION_SESSION__ = session
  window.dispatchEvent(new CustomEvent('rf-region-session', {detail: session}))
  return session
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
      audio: current.audio,
      camera: current.camera,
      whiteboard: current.whiteboard,
      shortcuts: current.shortcuts,
      window: {
        ...settings.window,
        theme: current.window.theme,
        startAtLogin: current.window.startAtLogin,
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
  const initialPointer = (session as BoundRegionSelectionSession & {initialPointer?: {x: number; y: number} | null}).initialPointer
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
    displayBounds: (session.displayBounds ?? []).map((display) => ({
      id: display.id,
      bounds: fromBoundRegionRect(display.bounds),
      captureBounds: fromBoundRegionRect(display.captureBounds),
      scaleFactor: display.scaleFactor,
    })),
    minimumWidth: session.minimumWidth,
    minimumHeight: session.minimumHeight,
    displayCount: session.displayCount,
    purpose: session.purpose === 'annotation' || session.purpose === 'screenshot' || session.purpose === 'scrolling-screenshot' ? session.purpose : 'capture',
    candidates: (session.candidates ?? []).map(fromBoundRegionSmartCandidate),
    initialPointer: initialPointer ? {x: initialPointer.x, y: initialPointer.y} : undefined,
  }
}

function fromBoundRegionSmartCandidate(candidate: BoundRegionSmartCandidate): RegionSmartCandidate {
  return {
    id: candidate.id,
    kind: candidate.kind,
    label: candidate.label,
    bounds: fromBoundRegionRect(candidate.bounds),
    sourceId: candidate.sourceId,
    score: candidate.score,
  }
}

function fromBoundRegionAssistResult(result: BoundRegionAssistResult): RegionAssistResult {
  return {
    candidates: (result.candidates ?? []).map(fromBoundRegionSmartCandidate),
    best: result.best ? fromBoundRegionSmartCandidate(result.best) : undefined,
    source: normalizeRegionAssistSource(result.source),
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

function fromBoundWhiteboardSnapshot(result: BoundWhiteboardSnapshotResult): {scene: WhiteboardScene; item: ScreenshotItem} {
  return {
    scene: fromBoundWhiteboardScene(result.scene),
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
    pinned: false,
    fixed: item.fixed === true,
    ocrStatus: normalizeOcrStatus(item.ocrStatus),
    ocrResultId: item.ocrResultId,
    ocrModelId: item.ocrModelId,
    ocrLanguage: item.ocrLanguage,
    ocrUpdatedAt: item.ocrUpdatedAt,
    ocrError: item.ocrError,
  }
}

function fromBoundOcrModelInfo(model: BoundOcrModelInfo): OcrModelInfo {
  return {
    id: model.id,
    name: model.name,
    channel: model.channel,
    engine: model.engine,
    language: model.language ?? [],
    version: model.version,
    sourceUrl: model.sourceUrl,
    license: model.license,
    downloadAvailable: model.downloadAvailable === true,
    downloadBytes: typeof model.downloadBytes === 'number' ? model.downloadBytes : undefined,
    installed: model.installed === true,
    verified: model.verified === true,
    active: model.active === true,
    modelDir: model.modelDir,
    smokeImage: model.smokeImage,
    smokeExpected: model.smokeExpected,
    smokeAssetReady: model.smokeAssetReady === true,
    smokeError: model.smokeError,
    missingFiles: model.missingFiles ?? [],
    verificationError: model.verificationError,
  }
}

function fromBoundOcrModelDownloadSnapshot(snapshot: BoundOcrModelDownloadSnapshot): OcrModelDownloadSnapshot {
  return {
    id: snapshot.id,
    modelId: snapshot.modelId,
    status: snapshot.status,
    downloadedBytes: Number(snapshot.downloadedBytes ?? 0),
    totalBytes: Number(snapshot.totalBytes ?? 0),
    percent: Number(snapshot.percent ?? 0),
    error: snapshot.error,
    model: snapshot.model ? fromBoundOcrModelInfo(snapshot.model) : undefined,
    startedAt: String(snapshot.startedAt ?? ''),
    updatedAt: String(snapshot.updatedAt ?? ''),
  }
}

function fromBoundOcrModelDownloadEvent(event: BoundOcrModelDownloadEvent): {snapshot: OcrModelDownloadSnapshot} {
  return {
    snapshot: fromBoundOcrModelDownloadSnapshot(event.snapshot),
  }
}

function fromBoundOcrStatus(status: BoundOcrStatus): OcrStatus {
  return {
    status: status.status,
    activeModelId: status.activeModelId,
    models: (status.models ?? []).map(fromBoundOcrModelInfo),
    workerPath: status.workerPath,
    runtimeDir: status.runtimeDir,
    workerCapabilities: status.workerCapabilities ? fromBoundOcrWorkerCapabilities(status.workerCapabilities) : undefined,
    message: status.message,
  }
}

function fromBoundOcrJobSnapshot(snapshot: BoundOcrJobSnapshot): OcrJobSnapshot {
  return {
    jobId: snapshot.jobId,
    status: snapshot.status,
    cacheKey: snapshot.cacheKey,
    request: fromBoundOcrRecognizeRequest(snapshot.request),
    merged: snapshot.merged === true,
    createdAt: String(snapshot.createdAt ?? ''),
    updatedAt: String(snapshot.updatedAt ?? ''),
  }
}

function fromBoundOcrJobEvent(event: BoundOcrJobEvent): OcrJobUpdate {
  return {
    jobId: event.jobId,
    sourceKind: fromBoundOcrSourceKind(event.sourceKind),
    sourceId: event.sourceId,
    status: event.status,
    cacheKey: event.cacheKey,
    merged: event.merged,
    error: event.error,
    result: event.result ? fromBoundOcrResult(event.result) : undefined,
  }
}

function fromBrowserOcrJobEvent(event: Partial<OcrJobUpdate>): OcrJobUpdate {
  const result = event.result ? fromBrowserOcrResult(event.result) : undefined
  return {
    jobId: String(event.jobId || result?.id || `browser-ocr-${Date.now()}`),
    sourceKind: fromOcrSourceKindString(String(event.sourceKind || result?.sourceKind || 'image')),
    sourceId: String(event.sourceId || result?.sourceId || ''),
    status: String(event.status || (result ? 'ready' : 'queued')),
    cacheKey: event.cacheKey,
    merged: event.merged,
    error: event.error,
    result,
  }
}

function fromBrowserOcrResult(result: Partial<OcrResult>): OcrResult {
  const sourceKind = fromOcrSourceKindString(String(result.sourceKind || 'image'))
  return {
    id: String(result.id || `browser-ocr-result-${Date.now()}`),
    sourceKind,
    sourceId: String(result.sourceId || ''),
    imagePath: String(result.imagePath || ''),
    imageSha256: String(result.imageSha256 || ''),
    modelId: String(result.modelId || 'browser-ocr-e2e'),
    language: String(result.language || 'zh-en'),
    width: finitePositiveInteger(result.width, 1),
    height: finitePositiveInteger(result.height, 1),
    blocks: Array.isArray(result.blocks) ? result.blocks.map(fromBrowserOcrBlock) : [],
    plainText: String(result.plainText || ''),
    createdAt: String(result.createdAt || new Date().toISOString()),
    durationMs: finiteNonNegativeNumber(result.durationMs, 0),
  }
}

function fromBrowserOcrBlock(block: Partial<OcrBlock>, index: number): OcrBlock {
  return {
    id: String(block.id || `browser-block-${index}`),
    text: String(block.text || ''),
    confidence: finiteNonNegativeNumber(block.confidence, 0),
    box: Array.isArray(block.box) ? block.box.map(fromBrowserOcrPoint).filter((point): point is OcrPoint => Boolean(point)) : [],
    lineIndex: Number.isFinite(block.lineIndex) ? Number(block.lineIndex) : index,
    languageHint: block.languageHint,
  }
}

function fromBrowserOcrPoint(point: unknown): OcrPoint | null {
  if (Array.isArray(point) && point.length >= 2) {
    const x = Number(point[0])
    const y = Number(point[1])
    if (Number.isFinite(x) && Number.isFinite(y)) return {x, y}
  }
  if (point && typeof point === 'object') {
    const candidate = point as {x?: unknown; y?: unknown}
    const x = Number(candidate.x)
    const y = Number(candidate.y)
    if (Number.isFinite(x) && Number.isFinite(y)) return {x, y}
  }
  return null
}

function fromBoundOcrWorkerCapabilities(capabilities: BoundOcrWorkerCapabilities): OcrWorkerCapabilities {
  return {
    schemaVersion: capabilities.schemaVersion,
    name: capabilities.name,
    version: capabilities.version,
    protocolVersion: capabilities.protocolVersion,
    engine: capabilities.engine,
    modelFormats: capabilities.modelFormats ?? [],
    supportsRecognize: capabilities.supportsRecognize === true,
    runtimeDir: capabilities.runtimeDir,
    runtimeLibrary: capabilities.runtimeLibrary,
    runtimeAvailable: capabilities.runtimeAvailable === true,
    runtimeVersion: capabilities.runtimeVersion,
    runtimeApiVersion: capabilities.runtimeApiVersion,
    runtimeError: capabilities.runtimeError,
    message: capabilities.message,
  }
}

function fromBoundOcrRecognizeRequest(request: BoundOcrRecognizeRequest): OcrRecognizeRequest {
  return {
    imagePath: request.imagePath,
    sourceKind: fromBoundOcrSourceKind(request.sourceKind),
    sourceId: request.sourceId,
    language: request.language,
    modelId: request.modelId,
    force: request.force,
    priority: request.priority,
  }
}

function fromBoundOcrResult(result: BoundOcrResult): OcrResult {
  return {
    id: result.id,
    sourceKind: fromBoundOcrSourceKind(result.sourceKind),
    sourceId: result.sourceId,
    imagePath: result.imagePath,
    imageSha256: result.imageSha256,
    modelId: result.modelId,
    language: result.language,
    width: result.width,
    height: result.height,
    blocks: (result.blocks ?? []).map(fromBoundOcrBlock),
    plainText: result.plainText,
    createdAt: String(result.createdAt ?? ''),
    durationMs: result.durationMs,
  }
}

function fromBoundOcrBlock(block: BoundOcrBlock): OcrBlock {
  return {
    id: block.id,
    text: block.text,
    confidence: block.confidence,
    box: (block.box ?? []).map((point) => ({x: point.x, y: point.y})),
    lineIndex: block.lineIndex,
    languageHint: block.languageHint,
  }
}

function fromBoundOcrTranslationResult(result: BoundOcrTranslationResult): OcrTranslationResult {
  return {
    ocrResultId: result.ocrResultId,
    provider: result.provider,
    sourceLanguage: result.sourceLanguage,
    targetLanguage: result.targetLanguage,
    model: result.model,
    promptVersion: result.promptVersion,
    blocks: (result.blocks ?? []).map(fromBoundOcrTranslationBlock),
    createdAt: String(result.createdAt ?? ''),
  }
}

function fromBoundOcrTranslationBlock(block: BoundOcrTranslationBlock): OcrTranslationBlock {
  return {
    blockId: block.blockId,
    source: block.source,
    translated: block.translated,
  }
}

function toBoundOcrRecognizeRequest(request: OcrRecognizeRequest): BoundOcrRecognizeRequest {
  return {
    imagePath: request.imagePath,
    sourceKind: toBoundOcrSourceKind(request.sourceKind),
    sourceId: request.sourceId,
    language: request.language ?? 'zh-en',
    modelId: request.modelId,
    force: request.force === true,
    priority: request.priority ?? 'normal',
  }
}

function fromBoundOcrSourceKind(sourceKind: BoundOcrSourceKind): OcrSourceKind {
  return fromOcrSourceKindString(String(sourceKind))
}

function fromOcrSourceKindString(sourceKind: string): OcrSourceKind {
  if (isOcrSourceKind(sourceKind)) return sourceKind
  return 'image'
}

function toBoundOcrSourceKind(sourceKind: OcrSourceKind): BoundOcrSourceKind {
  switch (sourceKind) {
    case 'region-screenshot':
      return BoundOcrSourceKind.SourceRegionScreenshot
    case 'full-screenshot':
      return BoundOcrSourceKind.SourceFullScreenshot
    case 'window-screenshot':
      return BoundOcrSourceKind.SourceWindowScreenshot
    case 'focused-window-screenshot':
      return BoundOcrSourceKind.SourceFocusedWindowScreenshot
    case 'scrolling-screenshot':
      return BoundOcrSourceKind.SourceScrollingScreenshot
    case 'pinned-screenshot':
      return BoundOcrSourceKind.SourcePinnedScreenshot
    case 'whiteboard':
      return BoundOcrSourceKind.SourceWhiteboard
    case 'whiteboard-selection':
      return BoundOcrSourceKind.SourceWhiteboardSelection
    default:
      return BoundOcrSourceKind.SourceImage
  }
}

function isOcrSourceKind(value: string): value is OcrSourceKind {
  return value === 'region-screenshot' ||
    value === 'full-screenshot' ||
    value === 'window-screenshot' ||
    value === 'focused-window-screenshot' ||
    value === 'scrolling-screenshot' ||
    value === 'pinned-screenshot' ||
    value === 'whiteboard' ||
    value === 'whiteboard-selection' ||
    value === 'image'
}

function finitePositiveInteger(value: unknown, fallback: number) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric <= 0) return fallback
  return Math.round(numeric)
}

function finiteNonNegativeNumber(value: unknown, fallback: number) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric < 0) return fallback
  return numeric
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
  return normalizeScreenshotPinState({
    visible: state?.visible === true,
    item: state?.item ? fromBoundScreenshotItem(state.item) : undefined,
    dataUrl: state?.dataUrl,
    fixed: state?.fixed === true,
    pins: Array.isArray(state?.pins) ? state.pins.map(fromBoundScreenshotPinnedItem).filter((pin): pin is ScreenshotPinnedItem => Boolean(pin)) : undefined,
  })
}

function fromBoundScreenshotPinnedItem(pin: unknown): ScreenshotPinnedItem | null {
  const record = pin && typeof pin === 'object' ? pin as {item?: BoundScreenshotItem; dataUrl?: string; fixed?: boolean} : {}
  const item = record.item ? fromBoundScreenshotItem(record.item) : null
  if (!item) return null
  return {
    item,
    dataUrl: typeof record.dataUrl === 'string' ? record.dataUrl : browserScreenshotDataUrl(item),
    fixed: record.fixed === true || item.fixed === true,
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
    toolbarBounds: state.toolbarBounds ? fromBoundRegionRect(state.toolbarBounds) : undefined,
    toolbarPlacement: state.toolbarPlacement === 'bottom' ? 'bottom' : 'top',
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
      geometry: {x: bounds?.x ?? 0, y: bounds?.y ?? 0, width: bounds?.width ?? width, height: bounds?.height ?? height},
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
    startAtLogin: patch.startAtLogin,
    autoOcr: patch.autoOcr,
    ocrTranslation: patch.ocrTranslation as BoundSettingsPreferencesPatchRequest['ocrTranslation'],
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
      startAtLogin: patch.startAtLogin ?? settings.window.startAtLogin,
    },
    ocr: {
      ...settings.ocr,
      autoRecognizeScreenshots: patch.autoOcr ?? settings.ocr.autoRecognizeScreenshots,
      translation: normalizeOcrTranslationSettings({
        ...settings.ocr.translation,
        ...(patch.ocrTranslation ?? {}),
      }),
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

function emptyFloatingPanelState(): FloatingPanelState {
  return {
    visible: false,
    anchor: {x: 0, y: 0, width: 0, height: 0},
    bounds: {x: 0, y: 0, width: 0, height: 0},
    token: 0,
  }
}

function emptyFloatingSelectState(): FloatingSelectState {
  return {
    visible: false,
    anchor: {x: 0, y: 0, width: 0, height: 0},
    bounds: {x: 0, y: 0, width: 0, height: 0},
    options: [],
    token: 0,
  }
}

function fromBoundFloatingRect(rect: BoundFloatingRect | undefined): FloatingRect {
  return {
    x: rect?.x ?? 0,
    y: rect?.y ?? 0,
    width: rect?.width ?? 0,
    height: rect?.height ?? 0,
  }
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
    contextId: (state as BoundFloatingPanelState & {contextId?: string}).contextId,
  }
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
    swatch: option.swatch,
  }
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
    direction: state.direction,
  }
}

function fromBoundFloatingSelectChosen(event: BoundFloatingSelectChosenEvent): FloatingSelectChosenEvent {
  return {
    id: event.id,
    value: event.value,
    token: event.token,
    panelToken: event.panelToken,
  }
}

function fromBoundSourceControlState(state: BoundSourceControlState): SourceControlState {
  return {
    recordingMode: state.recordingMode === 'audio' ? 'audio' : 'video',
    sourceId: state.sourceId,
    sourceType: normalizeSourceType(state.sourceType),
    sourceGeometry: state.sourceGeometry ? {
      x: state.sourceGeometry.x,
      y: state.sourceGeometry.y,
      width: state.sourceGeometry.width,
      height: state.sourceGeometry.height,
      displayIndex: state.sourceGeometry.displayIndex,
      nativeId: state.sourceGeometry.nativeId,
    } : undefined,
  }
}

function toBoundSourceStatePatch(patch: SourceStatePatch): BoundSourceStatePatchRequest {
  return {
    recordingMode: patch.recordingMode,
    sourceId: patch.sourceId,
    sourceType: patch.sourceType,
    sourceGeometry: patch.sourceGeometry ? {
      x: Math.round(patch.sourceGeometry.x),
      y: Math.round(patch.sourceGeometry.y),
      width: Math.round(patch.sourceGeometry.width),
      height: Math.round(patch.sourceGeometry.height),
      displayIndex: patch.sourceGeometry.displayIndex ?? 0,
      nativeId: patch.sourceGeometry.nativeId,
    } as BoundSourceGeometry : undefined,
    clearGeometry: patch.clearGeometry,
  }
}

function normalizeSourceType(value: unknown): CaptureSource['type'] | undefined {
  return value === 'screen' || value === 'all-screens' || value === 'region' || value === 'window' || value === 'application'
    ? value
    : undefined
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
    ocr: fromBoundOcrSettings((settings as BoundSettings & {ocr?: Partial<AppSettings['ocr']>}).ocr),
    shortcuts: normalizeShortcutSettings((settings as BoundSettings & {shortcuts?: Partial<ShortcutSettings>}).shortcuts),
    window: {
      minimizeToTray: settings.window.minimizeToTray,
      theme: normalizeTheme(settings.window.theme),
      startAtLogin: Boolean((settings.window as BoundSettings['window'] & {startAtLogin?: boolean}).startAtLogin),
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
    ocr: settings.ocr as unknown as BoundSettings['ocr'],
    shortcuts: settings.shortcuts as unknown as BoundSettings['shortcuts'],
    window: {
      minimizeToTray: settings.window.minimizeToTray,
      theme: settings.window.theme as BoundSettings['window']['theme'],
      startAtLogin: settings.window.startAtLogin,
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
    ocrStatus: 'none',
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
    pinned: false,
    fixed: record.fixed === true,
    ocrStatus: normalizeOcrStatus(record.ocrStatus),
    ocrResultId: typeof record.ocrResultId === 'string' ? record.ocrResultId : undefined,
    ocrModelId: typeof record.ocrModelId === 'string' ? record.ocrModelId : undefined,
    ocrLanguage: typeof record.ocrLanguage === 'string' ? record.ocrLanguage : undefined,
    ocrUpdatedAt: typeof record.ocrUpdatedAt === 'string' ? record.ocrUpdatedAt : undefined,
    ocrError: typeof record.ocrError === 'string' ? record.ocrError : undefined,
  }
}

function normalizeOcrStatus(status: unknown): ScreenshotItem['ocrStatus'] {
  switch (status) {
    case 'queued':
    case 'running':
    case 'ready':
    case 'failed':
      return status
    default:
      return 'none'
  }
}

type BrowserOcrModelDownloadWindow = Window & {
  __RF_OCR_STATUS__?: OcrStatus
  __RF_OCR_MODEL_DOWNLOADS__?: Record<string, OcrModelDownloadSnapshot>
  __RF_LAST_OCR_MODEL_DOWNLOAD__?: {modelId: string; at: string}
}

function browserOcrModelDownloads(): Record<string, OcrModelDownloadSnapshot> {
  const browserWindow = window as BrowserOcrModelDownloadWindow
  if (!browserWindow.__RF_OCR_MODEL_DOWNLOADS__) browserWindow.__RF_OCR_MODEL_DOWNLOADS__ = {}
  return browserWindow.__RF_OCR_MODEL_DOWNLOADS__
}

function browserStoreOcrModelDownload(snapshot: OcrModelDownloadSnapshot) {
  browserOcrModelDownloads()[snapshot.modelId] = snapshot
  window.dispatchEvent(new CustomEvent(browserOcrModelDownloadEvent, {detail: snapshot}))
}

function fromBrowserOcrModelDownloadEvent(snapshot: Partial<OcrModelDownloadSnapshot>): {snapshot: OcrModelDownloadSnapshot} {
  const now = new Date().toISOString()
  return {
    snapshot: {
      id: snapshot.id || `browser-ocr-model-download-${snapshot.modelId || 'unknown'}`,
      modelId: snapshot.modelId || '',
      status: snapshot.status || 'queued',
      downloadedBytes: Number(snapshot.downloadedBytes ?? 0),
      totalBytes: Number(snapshot.totalBytes ?? 0),
      percent: Number(snapshot.percent ?? 0),
      error: snapshot.error,
      model: snapshot.model,
      startedAt: snapshot.startedAt || now,
      updatedAt: snapshot.updatedAt || now,
    },
  }
}

function browserStartOcrModelDownload(modelId: string): OcrModelDownloadSnapshot {
  const browserWindow = window as BrowserOcrModelDownloadWindow
  const status = browserOcrStatus()
  const target = status.models.find((model) => model.id === modelId)
  if (!target?.downloadAvailable) {
    throw new Error(`OCR model ${modelId} does not have a verified RecordingFreedom package download`)
  }
  browserWindow.__RF_LAST_OCR_MODEL_DOWNLOAD__ = {modelId, at: new Date().toISOString()}
  const now = new Date().toISOString()
  const totalBytes = target.downloadBytes || 1
  const queued: OcrModelDownloadSnapshot = {
    id: `browser-ocr-model-download-${modelId}-${Date.now()}`,
    modelId,
    status: 'queued',
    downloadedBytes: 0,
    totalBytes,
    percent: 0,
    startedAt: now,
    updatedAt: now,
  }
  browserStoreOcrModelDownload(queued)
  window.setTimeout(() => {
    const running: OcrModelDownloadSnapshot = {
      ...queued,
      status: 'running',
      downloadedBytes: Math.max(1, Math.round(totalBytes * 0.55)),
      percent: 55,
      updatedAt: new Date().toISOString(),
    }
    browserStoreOcrModelDownload(running)
  }, 10)
  window.setTimeout(() => {
    const current = browserOcrStatus()
    const installedModel = current.models.find((model) => model.id === modelId) || target
    const verifiedModel: OcrModelInfo = {
      ...installedModel,
      installed: true,
      verified: true,
      active: installedModel.active === true,
      smokeAssetReady: true,
      missingFiles: [],
      verificationError: undefined,
      smokeError: undefined,
    }
    browserWindow.__RF_OCR_STATUS__ = {
      ...current,
      models: current.models.map((model) => model.id === modelId ? verifiedModel : model),
    }
    browserStoreOcrModelDownload({
      ...queued,
      status: 'installed',
      downloadedBytes: totalBytes,
      percent: 100,
      model: verifiedModel,
      updatedAt: new Date().toISOString(),
    })
  }, 25)
  return queued
}

function browserCancelOcrModelDownload(modelId: string): OcrModelDownloadSnapshot {
  const current = browserOcrModelDownloads()[modelId]
  const now = new Date().toISOString()
  const cancelled: OcrModelDownloadSnapshot = {
    ...(current || {
      id: `browser-ocr-model-download-${modelId}-${Date.now()}`,
      modelId,
      downloadedBytes: 0,
      totalBytes: 0,
      percent: 0,
      startedAt: now,
    }),
    status: 'cancelled',
    updatedAt: now,
  }
  browserStoreOcrModelDownload(cancelled)
  return cancelled
}

function browserOcrStatus(): OcrStatus {
  const injected = (window as Window & {__RF_OCR_STATUS__?: OcrStatus}).__RF_OCR_STATUS__
  if (injected) {
    return {
      ...injected,
      models: injected.models ?? [],
    }
  }
  return {
    status: 'no-model',
    activeModelId: 'ppocrv5-mobile-zh-en',
    models: [
      {
        id: 'ppocrv5-mobile-zh-en',
        name: 'PP-OCRv5 Mobile Chinese/English',
        channel: 'stable',
        engine: 'onnxruntime',
        language: ['zh', 'en'],
        downloadAvailable: false,
        downloadBytes: 0,
        installed: false,
        verified: false,
        active: true,
        smokeAssetReady: false,
        missingFiles: ['manifest.json', 'det.onnx', 'cls.onnx', 'rec.onnx', 'keys.txt'],
      },
      {
        id: 'ppocrv6-mobile-zh-en',
        name: 'PP-OCRv6 Mobile Chinese/English',
        channel: 'latest',
        engine: 'onnxruntime',
        language: ['zh', 'en'],
        downloadAvailable: false,
        downloadBytes: 0,
        installed: false,
        verified: false,
        active: false,
        smokeAssetReady: false,
        missingFiles: ['manifest.json', 'det.onnx', 'cls.onnx', 'rec.onnx', 'keys.txt'],
      },
      {
        id: 'ppocrv6-medium-zh-en',
        name: 'PP-OCRv6 Medium Chinese/English',
        channel: 'quality',
        engine: 'onnxruntime',
        language: ['zh', 'en'],
        downloadAvailable: false,
        downloadBytes: 0,
        installed: false,
        verified: false,
        active: false,
        smokeAssetReady: false,
        missingFiles: ['manifest.json', 'det.onnx', 'cls.onnx', 'rec.onnx', 'keys.txt'],
      },
    ],
    message: 'OCR is local-only and is not available in browser preview.',
  }
}

function browserOcrResult(resultId: string): OcrResult {
  const injected = browserInjectedOcrResult(resultId)
  if (injected) return injected
  const item = loadBrowserScreenshotHistory().find((entry) => entry.ocrResultId === resultId)
  if (!item || item.ocrStatus !== 'ready') {
    throw new Error(`OCR result ${resultId} is not available in browser preview`)
  }
  const width = Math.max(64, item.width || 480)
  const height = Math.max(64, item.height || 320)
  const titleTop = Math.round(height * 0.17)
  const subtitleTop = Math.round(height * 0.36)
  const titleBottom = Math.min(height - 12, titleTop + Math.max(26, Math.round(height * 0.12)))
  const subtitleBottom = Math.min(height - 10, subtitleTop + Math.max(24, Math.round(height * 0.11)))
  const left = Math.round(width * 0.08)
  const titleRight = Math.min(width - left, left + Math.round(width * 0.56))
  const subtitleRight = Math.min(width - left, left + Math.round(width * 0.42))
  return {
    id: resultId,
    sourceKind: browserScreenshotOcrSourceKind(item.mode),
    sourceId: item.id,
    imagePath: item.path,
    imageSha256: `browser-preview-${item.id}`,
    modelId: item.ocrModelId || 'ppocrv5-mobile-zh-en',
    language: item.ocrLanguage || 'zh-en',
    width,
    height,
    blocks: [
      {
        id: `${resultId}-block-title`,
        text: 'RecordingFreedom',
        confidence: 0.97,
        lineIndex: 0,
        languageHint: 'en',
        box: [
          {x: left, y: titleTop},
          {x: titleRight, y: titleTop},
          {x: titleRight, y: titleBottom},
          {x: left, y: titleBottom},
        ],
      },
      {
        id: `${resultId}-block-zh`,
        text: '文字识别',
        confidence: 0.94,
        lineIndex: 1,
        languageHint: 'zh',
        box: [
          {x: left, y: subtitleTop},
          {x: subtitleRight, y: subtitleTop},
          {x: subtitleRight, y: subtitleBottom},
          {x: left, y: subtitleBottom},
        ],
      },
    ],
    plainText: 'RecordingFreedom\n文字识别',
    createdAt: item.ocrUpdatedAt || item.createdAt,
    durationMs: 126,
  }
}

function browserOcrResultImage(resultId: string): ScreenshotImage {
  const injected = browserInjectedOcrImage(resultId)
  if (injected) return injected
  const item = loadBrowserScreenshotHistory().find((entry) => entry.ocrResultId === resultId)
  if (!item || item.ocrStatus !== 'ready') {
    return {available: false}
  }
  const dataUrl = browserScreenshotDataUrl(item)
  return {
    available: Boolean(dataUrl),
    dataUrl,
    path: item.path,
    bytes: dataUrl?.length ?? 0,
  }
}

function browserInjectedOcrResult(resultId: string): OcrResult | null {
  const injected = (window as Window & {__RF_OCR_RESULTS__?: Record<string, Partial<OcrResult>>}).__RF_OCR_RESULTS__?.[resultId]
  if (!injected) return null
  return fromBrowserOcrResult(injected)
}

function browserInjectedOcrImage(resultId: string): ScreenshotImage | null {
  const injected = (window as Window & {__RF_OCR_IMAGES__?: Record<string, Partial<ScreenshotImage>>}).__RF_OCR_IMAGES__?.[resultId]
  if (!injected) return null
  const dataUrl = typeof injected.dataUrl === 'string' && injected.dataUrl.startsWith('data:image/') ? injected.dataUrl : undefined
  return {
    available: Boolean(dataUrl),
    dataUrl,
    path: typeof injected.path === 'string' ? injected.path : undefined,
    bytes: typeof injected.bytes === 'number' && Number.isFinite(injected.bytes) ? injected.bytes : dataUrl?.length ?? 0,
  }
}

function browserTranslateOcr(request: OcrTranslateRequest): OcrTranslationResult {
  if (!request.provider || request.provider === 'disabled') throw new Error('OCR translation provider is disabled')
  if (!request.baseUrl?.trim()) throw new Error('OCR translation base URL is required')
  if (!request.apiKey?.trim()) throw new Error('OCR translation API key is required')
  if (request.provider === 'openai-compatible' && !request.model?.trim()) throw new Error('OpenAI-compatible translation model is required')
  const result = browserOcrResultForTranslation(request.ocrResultId)
  const selected = new Set((request.blockIds ?? []).filter(Boolean))
  const blocks = result.blocks
    .filter((block) => selected.size === 0 || selected.has(block.id))
    .map((block) => ({
      blockId: block.id,
      source: block.text,
      translated: browserTranslatedText(block.text, request.targetLanguage),
    }))
    .filter((block) => block.source.trim() && block.translated.trim())
  if (blocks.length === 0) throw new Error('No OCR text to translate')
  return {
    ocrResultId: request.ocrResultId,
    provider: request.provider,
    sourceLanguage: request.sourceLanguage || 'auto',
    targetLanguage: request.targetLanguage || 'zh-CN',
    model: request.model,
    promptVersion: 'browser-preview',
    blocks,
    createdAt: new Date().toISOString(),
  }
}

function browserOcrResultForTranslation(resultId: string): OcrResult {
  try {
    return browserOcrResult(resultId)
  } catch (error) {
    if (resultId !== 'whiteboard-selection-result') throw error
    return {
      id: resultId,
      sourceKind: 'whiteboard-selection',
      sourceId: 'browser-preview/data/whiteboards/board-current.excalidraw',
      imagePath: 'browser-preview/data/whiteboards/exports/whiteboard.png',
      imageSha256: 'browser-e2e-selection-image',
      modelId: 'ppocrv5-mobile-zh-en',
      language: 'zh-en',
      width: 320,
      height: 180,
      blocks: [
        {
          id: 'block-recordingfreedom',
          text: 'RecordingFreedom',
          confidence: 0.97,
          lineIndex: 0,
          languageHint: 'en',
          box: [],
        },
        {
          id: 'block-chinese',
          text: '文字识别',
          confidence: 0.94,
          lineIndex: 1,
          languageHint: 'zh',
          box: [],
        },
      ],
      plainText: 'RecordingFreedom\n文字识别',
      createdAt: '2026-07-06T10:00:00.000Z',
      durationMs: 126,
    }
  }
}

function browserTranslatedText(text: string, targetLanguage: string) {
  const trimmed = text.trim()
  if (!trimmed) return ''
  if (targetLanguage.toLowerCase().startsWith('zh')) {
    if (trimmed === 'RecordingFreedom') return '录制自由'
    if (trimmed === '文字识别') return '文字识别'
  }
  if (trimmed === 'RecordingFreedom') return 'RecordingFreedom translated'
  if (trimmed === '文字识别') return 'Text recognition translated'
  return `${trimmed} translated`
}

function browserQueuedWhiteboardOcrSnapshot(request: OcrWhiteboardRequest): OcrJobSnapshot {
  const sourceId = (request.sceneId || request.elementId || 'browser-whiteboard').trim()
  const recognizeRequest: OcrRecognizeRequest = {
    imagePath: request.imagePath,
    sourceKind: request.elementId ? 'whiteboard-selection' : 'whiteboard',
    sourceId,
    language: request.language || 'zh-en',
    force: request.force === true,
    priority: request.priority || 'interactive',
  }
  const snapshot: OcrJobSnapshot = {
    jobId: `browser-whiteboard-ocr-${Date.now()}`,
    status: 'queued',
    request: recognizeRequest,
    merged: false,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  }
  ;(window as Window & {
    __RF_LAST_WHITEBOARD_OCR_QUEUE__?: OcrJobSnapshot
    __RF_LAST_WHITEBOARD_OCR_REQUEST__?: OcrWhiteboardRequest
  }).__RF_LAST_WHITEBOARD_OCR_QUEUE__ = snapshot
  ;(window as Window & {__RF_LAST_WHITEBOARD_OCR_REQUEST__?: OcrWhiteboardRequest}).__RF_LAST_WHITEBOARD_OCR_REQUEST__ = request
  return snapshot
}

function browserScreenshotOcrSourceKind(mode: string): OcrSourceKind {
  switch (mode) {
    case 'full':
    case 'screen':
      return 'full-screenshot'
    case 'window':
      return 'window-screenshot'
    case 'focused-window':
      return 'focused-window-screenshot'
    case 'scrolling':
      return 'scrolling-screenshot'
    case 'whiteboard':
      return 'whiteboard'
    default:
      return 'region-screenshot'
  }
}

function browserScreenshotDataUrl(item: ScreenshotItem | undefined | null): string | undefined {
  if (!item) return undefined
  const width = Math.max(64, Math.min(800, item.width || 480))
  const height = Math.max(64, Math.min(600, item.height || 320))
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#151b24"/><stop offset="1" stop-color="#263445"/></linearGradient></defs><rect width="100%" height="100%" fill="url(#g)"/><rect x="18" y="18" width="${width - 36}" height="${height - 36}" rx="18" fill="none" stroke="#ef4444" stroke-width="4"/><text x="32" y="54" fill="#f8fafc" font-family="Inter, Arial" font-size="18">RecordingFreedom screenshot preview</text><text x="32" y="84" fill="#9ca3af" font-family="Inter, Arial" font-size="13">${item.id}</text></svg>`
  return `data:image/svg+xml;base64,${window.btoa(unescape(encodeURIComponent(svg)))}`
}

function browserRegionAssist(request: RegionAssistRequest): RegionAssistResult {
  const candidates = request.candidates ?? []
  let best: RegionSmartCandidate | undefined
  let bestScore = -1
  let source: RegionAssistResult['source'] = 'static'
  if (request.selection) {
    source = 'selection'
    for (const candidate of candidates) {
      const score = browserCandidateSelectionScore(candidate.bounds, request.selection) + browserRegionKindWeight(candidate.kind)
      if (score > bestScore) {
        best = {...candidate, score}
        bestScore = score
      }
    }
    if (bestScore < 0.5) best = undefined
  } else {
    const point = {x: request.pointerX ?? -1, y: request.pointerY ?? -1}
    const containing = candidates
      .filter((candidate) => browserRectContainsPoint(candidate.bounds, point))
      .sort((left, right) => {
        const kindRank = browserRegionKindWeight(right.kind) - browserRegionKindWeight(left.kind)
        if (kindRank !== 0) return kindRank
        return (left.bounds.width * left.bounds.height) - (right.bounds.width * right.bounds.height)
      })
    const level = Math.max(0, Math.min(containing.length - 1, Math.round(request.candidateLevel ?? 0)))
    const leveled = containing[level]
    if (leveled) {
      const area = Math.max(1, leveled.bounds.width * leveled.bounds.height)
      best = {...leveled, score: (leveled.score ?? 0) + browserRegionKindWeight(leveled.kind) + 1000000 / area}
      source = browserRegionAssistSourceForKind(leveled.kind)
    } else {
      for (const candidate of candidates) {
        if (!browserRectContainsPoint(candidate.bounds, point)) continue
        const area = Math.max(1, candidate.bounds.width * candidate.bounds.height)
        const score = (candidate.score ?? 0) + browserRegionKindWeight(candidate.kind) + 1000000 / area
        if (score > bestScore) {
          best = {...candidate, score}
          bestScore = score
        }
      }
      source = browserRegionAssistSourceForKind(best?.kind)
    }
  }
  return {candidates, best, source}
}

function browserRegionAssistSourceForKind(kind: RegionSmartCandidate['kind'] | undefined): RegionAssistResult['source'] {
  if (kind === 'element') return 'element'
  if (kind === 'edge') return 'image-hover'
  return 'static'
}

function normalizeRegionAssistSource(source: unknown): RegionAssistResult['source'] {
  if (source === 'element' || source === 'image-hover' || source === 'selection' || source === 'static') {
    return source
  }
  return undefined
}

function browserRegionKindWeight(kind: RegionSmartCandidate['kind']) {
  if (kind === 'element') return 0.36
  if (kind === 'edge') return 0.16
  if (kind === 'window') return 0.08
  return 0
}

function browserCandidateSelectionScore(candidate: RegionSelectionSession['bounds'], selection: RegionSelectionSession['bounds']) {
  const left = Math.max(candidate.x, selection.x)
  const top = Math.max(candidate.y, selection.y)
  const right = Math.min(candidate.x + candidate.width, selection.x + selection.width)
  const bottom = Math.min(candidate.y + candidate.height, selection.y + selection.height)
  if (right <= left || bottom <= top) return -1
  const intersection = (right - left) * (bottom - top)
  const candidateArea = Math.max(1, candidate.width * candidate.height)
  const selectionArea = Math.max(1, selection.width * selection.height)
  const overlap = intersection / Math.min(candidateArea, selectionArea)
  const edgeDistance = Math.abs(candidate.x - selection.x) +
    Math.abs(candidate.y - selection.y) +
    Math.abs(candidate.x + candidate.width - selection.x - selection.width) +
    Math.abs(candidate.y + candidate.height - selection.y - selection.height)
  const closeEdges = Math.max(0, 1 - edgeDistance / 280)
  const areaRatio = Math.min(candidateArea, selectionArea) / Math.max(candidateArea, selectionArea)
  return overlap * 0.54 + closeEdges * 0.34 + areaRatio * 0.12
}

function browserRectContainsPoint(rect: RegionSelectionSession['bounds'], point: {x: number; y: number}) {
  return rect.width > 0 &&
    rect.height > 0 &&
    point.x >= rect.x &&
    point.x < rect.x + rect.width &&
    point.y >= rect.y &&
    point.y < rect.y + rect.height
}

function fromBrowserScreenshotPinState(value: unknown): ScreenshotPinState {
  const record = value && typeof value === 'object' ? value as Partial<ScreenshotPinState> : {}
  const item = fromBrowserScreenshotItem(record.item)
  const pins = Array.isArray(record.pins)
    ? record.pins.map(fromBrowserScreenshotPinnedItem).filter((pin): pin is ScreenshotPinnedItem => Boolean(pin))
    : undefined
  return normalizeScreenshotPinState({
    visible: record.visible === true,
    item: item ?? undefined,
    dataUrl: typeof record.dataUrl === 'string' ? record.dataUrl : browserScreenshotDataUrl(item),
    fixed: record.fixed === true,
    pins,
  })
}

function fromBrowserScreenshotPinnedItem(value: unknown): ScreenshotPinnedItem | null {
  const record = value && typeof value === 'object' ? value as Partial<ScreenshotPinnedItem> : {}
  const item = fromBrowserScreenshotItem(record.item)
  if (!item) return null
  return {
    item,
    dataUrl: typeof record.dataUrl === 'string' ? record.dataUrl : browserScreenshotDataUrl(item),
    fixed: record.fixed === true || item.fixed === true,
  }
}

function normalizeScreenshotPinState(state: ScreenshotPinState): ScreenshotPinState {
  const pins = normalizeScreenshotPins(state.pins ?? (state.item ? [{
    item: state.item,
    dataUrl: state.dataUrl,
    fixed: state.fixed,
  }] : []))
  if (pins.length === 0) {
    return {visible: false, fixed: false, pins: []}
  }
  const active = pins[pins.length - 1]
  return {
    visible: state.visible === true,
    item: active.item,
    dataUrl: active.dataUrl,
    fixed: active.fixed,
    pins,
  }
}

function normalizeScreenshotPins(pins: ScreenshotPinnedItem[]) {
  const next: ScreenshotPinnedItem[] = []
  const seen = new Set<string>()
  for (const pin of pins) {
    const item = pin.item
    if (!item?.id || seen.has(item.id)) continue
    seen.add(item.id)
    next.push({
      item,
      dataUrl: pin.dataUrl || browserScreenshotDataUrl(item),
      fixed: pin.fixed === true || item.fixed === true,
    })
  }
  return next
}

function appendBrowserScreenshotPinStateItem(state: ScreenshotPinState, pin: ScreenshotPinnedItem): ScreenshotPinState {
  const pins = normalizeScreenshotPins([...(state.pins ?? []), pin])
    .filter((entry) => entry.item.id !== pin.item.id)
  pins.push(pin)
  return normalizeScreenshotPinState({visible: true, fixed: pin.fixed, pins})
}

function updateBrowserScreenshotPinStateItem(state: ScreenshotPinState, item: ScreenshotItem): ScreenshotPinState {
  const pins = normalizeScreenshotPins(state.pins ?? []).map((pin) => pin.item.id === item.id
    ? {...pin, item, fixed: item.fixed === true}
    : pin)
  return normalizeScreenshotPinState({...state, visible: pins.length > 0, pins})
}

function removeBrowserScreenshotPinStateItem(state: ScreenshotPinState, id: string): ScreenshotPinState {
  const pins = normalizeScreenshotPins(state.pins ?? []).filter((pin) => pin.item.id !== id)
  return normalizeScreenshotPinState({...state, visible: pins.length > 0, pins})
}

function screenshotPinStateContains(state: ScreenshotPinState, id: string) {
  return state.item?.id === id || (state.pins ?? []).some((pin) => pin.item.id === id)
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
    const parsed = migrateBrowserSettings(JSON.parse(raw))
    const next = {
      ...defaultSettings,
      ...parsed,
      source: {...defaultSettings.source, ...parsed.source},
      storage: {...defaultSettings.storage, ...parsed.storage},
      recording: {...defaultSettings.recording, ...parsed.recording},
      audio: {...defaultSettings.audio, ...parsed.audio},
      camera: {...defaultSettings.camera, ...parsed.camera},
      whiteboard: {...defaultSettings.whiteboard, ...parsed.whiteboard},
      ocr: fromBoundOcrSettings(parsed.ocr),
      shortcuts: normalizeShortcutSettings(parsed.shortcuts),
      window: {...defaultSettings.window, ...parsed.window},
    }
    const camera = {
      ...next.camera,
      pip: fromBoundPipConfig(next.camera.pip, next.camera.pipPreset),
    }
    camera.pipPreset = camera.pip.preset
    return {...next, camera, locale: normalizeLocale(next.locale), shortcuts: normalizeShortcutSettings(next.shortcuts), window: {...next.window, theme: normalizeTheme(next.window.theme), startAtLogin: Boolean(next.window.startAtLogin)}}
  } catch {
    return defaultSettings
  }
}

function migrateBrowserSettings(value: unknown): Partial<AppSettings> {
  const record = value && typeof value === 'object' ? value as Partial<AppSettings> : {}
  const schemaVersion = typeof record.schemaVersion === 'number' && Number.isFinite(record.schemaVersion) ? record.schemaVersion : 0
  if (schemaVersion >= settingsSchemaVersion) return record
  const next: Partial<AppSettings> = {...record, schemaVersion: settingsSchemaVersion}
  const camera = record.camera
  if (schemaVersion < 3) {
    const pip = camera?.pip
    if (!camera || !pip) return next
    const scale = pip.scale
    next.camera = {
      ...camera,
      pip: {
        ...pip,
        scale: typeof scale === 'number' && Number.isFinite(scale) ? migrateLegacyPipScale(scale) : scale,
      },
    }
  }
  return next
}

function migrateLegacyPipScale(scale: number): number {
  if (scale <= 0) return scale
  const clamped = normalizedRange(scale, legacyPipMaximumScale, legacyPipMinimumScale, legacyPipMaximumScale)
  const progress = (clamped - legacyPipMinimumScale) / (legacyPipMaximumScale - legacyPipMinimumScale)
  return pipMinimumScale + progress * (pipMaximumScale - pipMinimumScale)
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

function fromBoundOcrSettings(value: Partial<AppSettings['ocr']> | undefined): AppSettings['ocr'] {
  return {
    autoRecognizeScreenshots: value?.autoRecognizeScreenshots === true,
    translation: normalizeOcrTranslationSettings(value?.translation),
  }
}

function normalizeOcrTranslationSettings(value: Partial<AppSettings['ocr']['translation']> | undefined): AppSettings['ocr']['translation'] {
  const provider = value?.provider === 'deepl' || value?.provider === 'openai-compatible' ? value.provider : 'disabled'
  const privacyConfirmed = provider !== 'disabled' && value?.privacyConfirmed === true
  return {
    provider,
    baseUrl: trimOptionalString(value?.baseUrl),
    apiKey: trimOptionalString(value?.apiKey),
    apiKeySet: value?.apiKeySet === true || Boolean(trimOptionalString(value?.apiKey)),
    model: trimOptionalString(value?.model),
    sourceLanguage: trimOptionalString(value?.sourceLanguage) || 'auto',
    targetLanguage: trimOptionalString(value?.targetLanguage) || 'zh-CN',
    privacyConfirmed,
    privacyConfirmedAt: privacyConfirmed ? trimOptionalString(value?.privacyConfirmedAt) : undefined,
  }
}

function trimOptionalString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
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
    scale: normalizedRange(config?.scale, pipMaximumScale, pipMinimumScale, pipMaximumScale),
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

function toBoundRegionAssistRequest(request: RegionAssistRequest): BoundRegionAssistRequest {
  return {
    sessionId: request.sessionId,
    purpose: request.purpose,
    pointerX: Math.round(request.pointerX ?? 0),
    pointerY: Math.round(request.pointerY ?? 0),
    selection: request.selection ? toBoundRegionSelectionRequest(request.selection) : undefined,
    candidateLevel: Math.max(0, Math.round(request.candidateLevel ?? 0)),
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
