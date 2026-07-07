import {
  AppWindow,
  Camera,
  Check,
  ChevronDown,
  ChevronLeft,
  CircleDot,
  Copy as CopyIcon,
  Crosshair,
  Eye,
  FileText,
  FlipHorizontal,
  FolderOpen,
  Gauge,
  History,
  Image as ImageIcon,
  Languages,
  Lock,
  Maximize2,
  Minimize2,
  Move,
  MousePointer2,
  Monitor,
  Pause,
  PenLine,
  Play,
  Radio,
  Settings,
  ScrollText,
  Square,
  Trash2,
  Video,
  Volume2,
  Wand2,
  X,
  Pin,
  Unlock,
} from 'lucide-react'
import {Suspense, lazy, useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type CSSProperties, type PointerEvent as ReactPointerEvent, type ReactNode, type RefObject} from 'react'
import {copyByLocale, type RecorderCopy, type RecoveryMessageKey, type SourceSelectionMessageKey, type StatusMessageKey, type StorageMessageKey} from './i18n'
import {
  cameraDevices,
  defaultSettings,
  fallbackAppData,
  localeOptions,
  normalizeLocale,
  normalizeTheme,
  shortcutActions,
  sources,
  systemAudioDevices,
  themeOptions,
  themeSwatches,
  type AppDataInfo,
  type AppSettings,
  type AppStorageStatus,
  type AudioOnlyRecordingRequest,
  type CaptureCapabilities,
  type CaptureCapability,
  type CaptureSource,
  type LocaleCode,
  type PIPConfig,
  type MediaDevice,
  type MediaInventory,
  type MockRecordingRequest,
  type PIPShape,
  type PIPPreset,
  type RecordingMode,
  type RecordingPreflight,
  type RecordingQuality,
  type RecordingState,
  type ShortcutAction,
  type ShortcutSettings,
  type ScreenshotItem,
  type ThemeCode,
  fallbackCapabilities,
  fallbackStorageStatus,
} from './services/mockBackend'
import {assistRegionSelection, beginScreenshotAnnotationOverlay, cancelOcrModelDownload, cancelRegionSelector, cancelSelectedRegion, captureScreenshot, completeAnnotationRegionSelection, completeRegionSelection, completeScreenshotRegionSelection, completeScrollingScreenshotSelection, completeFloatingSelect, deleteScreenshotItem, exportRecordingPackage, getFloatingPanelState, getFloatingSelectState, getOcrModelDownloads, getOcrStatus, getSourceState, hideFloatingPanel, hideFloatingSelect, hidePinnedScreenshot, hidePipOverlay, hideRegionFrame, hideScreenIndicator, hideSettingsWindow, installOcrModelPackage, isWailsDesktopRuntime, listScreenshots, loadBootstrap, loadPinnedScreenshot, loadSettings, logClientEvent, openOcrResult, openRecordingPackage, openScreenshotDirectory, openScreenshotInWhiteboard, openVideoDirectory, patchAudioState, patchCameraState, patchScreenshotItem, patchSettingsPreferences, patchShortcutSettings, patchSourceState, patchWhiteboardSettings, pauseRecording, preflightAudioOnlyRecording, preflightRecording, previewExportRecordingPackage, queueRecognizePinnedScreenshot, queueRecognizeScreenshot, quitApplication, readAnnotationPreviewImage, readPipPreviewImage, readOcrResultImage, readScreenshotImage, recoverRecordingPackage, refreshOcrModelCatalog, removeOcrModel, restoreCapsuleWindow, resumeRecording, saveSettings, setActiveOcrModel, setCapsuleWindowExpanded, setCapsuleWindowHitRegions, setDataRoot, setFloatingPanelHitRegions, setFloatingSelectHitRegions, showAnnotationOverlay, showAnnotationRegionSelector, showFloatingPanel, showFloatingSelect, showPinnedScreenshot, showPipOverlay, showRegionSelector, showScreenIndicator, showScreenshotRegionSelector, showWhiteboardWindow, snapCapsuleWindowToEdge, startAudioOnlyRecording, startMicrophoneLevelMonitor, startOcrModelDownload, startRecording, startScrollingScreenshot, stopMicrophoneLevelMonitor, stopRecording, subscribeAudioLevel, subscribeAudioState, subscribeCapsuleDockSide, subscribeCapsuleWindowMoveEnded, subscribeFloatingPanelChanged, subscribeFloatingSelectChanged, subscribeFloatingSelectChosen, subscribeOcrJobEvents, subscribeOcrModelDownloadEvents, subscribeRecordingStatus, subscribeRegionSelection, subscribeScreenshotCaptured, subscribeScreenshotHistoryChanged, subscribeScreenshotPin, subscribeSettingsChanged, subscribeShortcutTriggered, subscribeSourceStateChanged, subscribeWhiteboardVisibility, translateOcr, updatePipOverlay, updateScreenshotRegionSelection, updateSelectedRegion, type AudioControlState, type AudioLevelUpdate, type AudioStatePatch, type CapsuleWindowDockSide, type CapsuleWindowExpandDirection, type CapsuleWindowHitRegion, type FloatingPanelKind, type FloatingPanelState, type FloatingSelectOption, type FloatingSelectState, type OcrBlock, type OcrModelDownloadSnapshot, type OcrModelInfo, type OcrResult, type OcrStatus, type OcrTranslationResult, type PIPOverlayCamera, type PIPOverlayState, type RecordingExportPlan, type RecordingRecovery, type RecordingStatusUpdate, type RegionSelectionSession, type RegionSmartCandidate, type ScreenshotPinnedItem, type ScreenshotPinState, type SettingsPreferencesPatch, type ShortcutSettingsPatch, type SourceControlState, type WhiteboardSettingsPatch, type WhiteboardVisibilityUpdate} from './services/recorderBackend'
import {resolveFloatingPanelPlacement, resolveFloatingSelectPlacement} from './components/floating/floatingPosition'
import {ocrPanelContext, ocrResultPanelExpandedSize, ocrResultPanelSize, parseOcrPanelContext} from './components/floating/ocrResultPanel'
import {OcrPositionTextLayer, ocrBlockPolygonPoints, ocrBlockStableId} from './components/ocr/OcrPositionTextLayer'
import {writeClipboardText} from './utils/clipboard'

const AnnotationOverlayWindow = lazy(() => import('./AnnotationOverlayWindow'))
const AnnotationRenderWindow = lazy(() => import('./AnnotationRenderWindow'))
const WhiteboardWindow = lazy(() => import('./WhiteboardWindow'))

const sourceIcon = {
  screen: Monitor,
  'all-screens': Maximize2,
  region: Crosshair,
  window: AppWindow,
  application: Radio,
}

const pipPresetOptions: PIPPreset[] = ['bottom-right', 'bottom-left', 'free']
const allPipPresetOptions: PIPPreset[] = [...pipPresetOptions, 'off']
const pipShapeOptions: PIPShape[] = ['circle', 'square']
const pipMinimumScale = 0.08
const pipMaximumScale = 0.15
const pipDefaultScale = pipMaximumScale
const pipMinimumDisplayPercent = 20
const pipMaximumDisplayPercent = 100

const recordingQualityOptions: RecordingQuality[] = ['standard', 'balanced', 'high']
const fpsOptions = [24, 30, 60]
const countdownOptions = [0, 3, 5, 10]
const ocrTranslationProviders: AppSettings['ocr']['translation']['provider'][] = ['disabled', 'deepl', 'openai-compatible']
const ocrTranslationLanguageOptions = ['auto', 'zh-CN', 'en', 'ja', 'ko', 'fr', 'de', 'es']
const previewPackagePath = 'data/video/recording-preview.rfrec'
type ActivePanel = 'source' | 'audio' | 'camera' | 'language' | 'board'

const floatingPanelStandardSize = {width: 320, height: 340, maxHeight: 340, minWidth: 300}
const floatingPanelCompactSize = {width: 260, height: 120, maxHeight: 120, minWidth: 240}
const floatingPanelSettingsSize = {width: 340, height: 340, maxHeight: 340, minWidth: 320}
const floatingPanelOcrResultSize = ocrResultPanelSize
const floatingPanelOcrResultExpandedSize = ocrResultPanelExpandedSize
const floatingSelectMinWidth = 180
const floatingSelectMaxWidth = 280
const floatingSelectMaxHeight = 220

const floatingPanelSizes: Record<FloatingPanelKind, {width: number; height: number; maxHeight: number; minWidth?: number}> = {
  source: floatingPanelStandardSize,
  audio: floatingPanelStandardSize,
  camera: floatingPanelStandardSize,
  board: floatingPanelStandardSize,
  language: floatingPanelCompactSize,
  settings: floatingPanelSettingsSize,
  close: {width: 340, height: 170, maxHeight: 170, minWidth: 320},
  'ocr-result': floatingPanelOcrResultSize,
}

function normalizePipPreset(value: PIPPreset): PIPPreset {
  return allPipPresetOptions.includes(value) ? value : 'bottom-right'
}

function normalizePipShape(value: PIPShape): PIPShape {
  return pipShapeOptions.includes(value) ? value : 'circle'
}

function defaultPipPosition(preset: PIPPreset) {
  return {x: preset === 'bottom-left' ? 0 : 1, y: 1}
}

function clampNumber(value: number, min: number, max: number) {
  if (!Number.isFinite(value)) return min
  return Math.min(max, Math.max(min, value))
}

function formatPipScalePercent(scale: number) {
  const normalized = (clampNumber(scale, pipMinimumScale, pipMaximumScale) - pipMinimumScale) / (pipMaximumScale - pipMinimumScale)
  return `${Math.round(pipMinimumDisplayPercent + normalized * (pipMaximumDisplayPercent - pipMinimumDisplayPercent))}%`
}

function normalizePipConfig(value: Partial<PIPConfig> | undefined, fallbackPreset: PIPPreset): PIPConfig {
  const preset = normalizePipPreset((value?.preset as PIPPreset | undefined) ?? fallbackPreset)
  const fallbackPosition = defaultPipPosition(preset)
  return {
    preset,
    shape: normalizePipShape((value?.shape as PIPShape | undefined) ?? 'circle'),
    mirror: value?.mirror !== false,
    position: {
      x: clampNumber(value?.position?.x ?? fallbackPosition.x, 0, 1),
      y: clampNumber(value?.position?.y ?? fallbackPosition.y, 0, 1),
    },
    scale: clampNumber(value?.scale ?? pipDefaultScale, pipMinimumScale, pipMaximumScale),
    edgeFeather: clampNumber(value?.edgeFeather ?? 0.16, 0.02, 0.42),
  }
}

function ensureVisiblePipConfig(value: PIPConfig): PIPConfig {
  if (value.preset !== 'off') return value
  return normalizePipConfig({
    ...value,
    preset: 'bottom-right',
    position: defaultPipPosition('bottom-right'),
  }, 'bottom-right')
}

function normalizeRecordingQuality(value: string): RecordingQuality {
  return recordingQualityOptions.includes(value as RecordingQuality) ? value as RecordingQuality : 'balanced'
}

function normalizeOcrTranslationSettings(value: Partial<AppSettings['ocr']['translation']> | undefined): AppSettings['ocr']['translation'] {
  const provider = value?.provider === 'deepl' || value?.provider === 'openai-compatible' ? value.provider : 'disabled'
  const privacyConfirmed = provider !== 'disabled' && value?.privacyConfirmed === true
  return {
    provider,
    baseUrl: cleanOptionalString(value?.baseUrl),
    apiKey: cleanOptionalString(value?.apiKey),
    apiKeySet: value?.apiKeySet === true || Boolean(cleanOptionalString(value?.apiKey)),
    model: cleanOptionalString(value?.model),
    sourceLanguage: cleanOptionalString(value?.sourceLanguage) || 'auto',
    targetLanguage: cleanOptionalString(value?.targetLanguage) || 'zh-CN',
    privacyConfirmed,
    privacyConfirmedAt: privacyConfirmed ? cleanOptionalString(value?.privacyConfirmedAt) : undefined,
  }
}

function ocrTranslationUnavailableMessage(translation: AppSettings['ocr']['translation'], copy: RecorderCopy): string {
  if (translation.provider === 'disabled') return copy.screenshot.translationUnavailable
  if (!translation.privacyConfirmed) return copy.screenshot.translationPrivacyRequired
  if (!translation.baseUrl) return copy.screenshot.translationMissingBaseUrl
  if (!translation.apiKey && !translation.apiKeySet) return copy.screenshot.translationMissingApiKey
  if (translation.provider === 'openai-compatible' && !translation.model) return copy.screenshot.translationMissingModel
  return ''
}

function cleanOptionalString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

function annotationCapturePolicy(includeAnnotations: boolean): AppSettings['whiteboard']['capturePolicy'] {
  return includeAnnotations ? 'export-compose' : 'preview-only'
}

function whiteboardLaunchMode(state: RecordingState, recordingMode: RecordingMode): 'whiteboard' | 'annotation' {
  return (state === 'recording' || state === 'paused') && recordingMode === 'video' ? 'annotation' : 'whiteboard'
}

function formatTime(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return hours > 0
    ? `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
    : `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
}

function joinDisplayPath(root: string, leaf: string) {
  if (!root || root === 'browser-preview') return leaf
  const separator = root.includes('\\') ? '\\' : '/'
  return `${root.replace(/[\\/]+$/, '')}${separator}${leaf}`
}

function eventPathContains(event: Event, element: Element | null) {
  if (!element) return false
  if (typeof event.composedPath === 'function' && event.composedPath().includes(element)) return true
  return event.target instanceof Node && element.contains(event.target)
}

function packageDisplayName(packagePath: string) {
  const parts = packagePath.split(/[\\/]/).filter(Boolean)
  return parts[parts.length - 1] || packagePath
}

function isRecordingPackagePath(packagePath: string) {
  return /(^|[\\/])[^\\/]+\.rfrec$/.test(packagePath)
}

function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes < 0) return ''
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  return `${value >= 10 || unitIndex === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unitIndex]}`
}

function readableError(error: unknown) {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  try {
    return JSON.stringify(error)
  } catch {
    return String(error)
  }
}

function normalizedClientRect(startX: number, startY: number, currentX: number, currentY: number) {
  const x = Math.round(Math.min(startX, currentX))
  const y = Math.round(Math.min(startY, currentY))
  const width = Math.round(Math.abs(currentX - startX))
  const height = Math.round(Math.abs(currentY - startY))
  return {x, y, width, height}
}

function clampedClientPoint(clientX: number, clientY: number) {
  const maxX = Math.max(0, (window.innerWidth || 1) - 1)
  const maxY = Math.max(0, (window.innerHeight || 1) - 1)
  return {
    x: clampNumber(clientX, 0, maxX),
    y: clampNumber(clientY, 0, maxY),
  }
}

function elementHitRegion(
  element: Element | null,
  viewportWidth: number,
  viewportHeight: number,
  kind: CapsuleWindowHitRegion['kind'] = 'round-rect',
  radius = 18,
): CapsuleWindowHitRegion | null {
  if (!element) return null
  const rect = element.getBoundingClientRect()
  if (rect.width <= 0 || rect.height <= 0) return null
  const x = Math.max(0, Math.floor(rect.left))
  const y = Math.max(0, Math.floor(rect.top))
  const right = Math.min(viewportWidth, Math.ceil(rect.right))
  const bottom = Math.min(viewportHeight, Math.ceil(rect.bottom))
  if (right <= x || bottom <= y) return null
  return {x, y, width: right - x, height: bottom - y, kind, radius}
}

function capsuleHitRegionRequestSignature(req: {
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

function currentWindowRoute() {
  const hashRoute = window.location.hash.replace(/^#/, '')
  if (hashRoute.startsWith('/')) return hashRoute
  return window.location.pathname
}

type StatusMessageState = {
  key: StatusMessageKey
  fallback?: string
}

type RecoveryMessageState = {
  key: RecoveryMessageKey
  count?: number
}

type ExportMessageState = {
  key: 'ready' | 'failed'
  path?: string
  fallback?: string
}

type StorageMessageState = {
  key: StorageMessageKey
  path?: string
}

type SourceSelectionMessageState = {
  key: SourceSelectionMessageKey
  width?: number
  height?: number
  fallback?: string
}

type ApplySettingsOptions = {
  preserveRecordingSettings?: boolean
  preserveTheme?: boolean
  preserveOcr?: boolean
  preserveAudioEnabled?: boolean
  preserveAudioSelection?: boolean
  preserveCameraEnabled?: boolean
  preserveCameraSelection?: boolean
  preservePipConfig?: boolean
  preserveWhiteboard?: boolean
}

function App() {
  const route = currentWindowRoute()
  const isSettingsWindow = route === '/settings'
  const isFloatingPanelWindow = route === '/floating-panel'
  const isFloatingSelectWindow = route === '/floating-select'
  const isRegionOverlayWindow = route === '/region-overlay'
  const isScreenIndicatorWindow = route === '/screen-indicator'
  const isPipOverlayWindow = route === '/pip-overlay'
  const isScreenshotPinWindow = route === '/screenshot-pin'
  const isWhiteboardWindow = route === '/whiteboard'
  const isAnnotationOverlayWindow = route === '/annotation-overlay'
  const isAnnotationRendererWindow = route === '/annotation-renderer'
  if (isFloatingSelectWindow) {
    return <FloatingSelectWindow />
  }
  if (isScreenIndicatorWindow) {
    return <ScreenIndicatorWindow />
  }
  if (isAnnotationOverlayWindow) {
    return (
      <Suspense fallback={<main className="whiteboard-loading"><span>Loading annotation</span></main>}>
        <AnnotationOverlayWindow />
      </Suspense>
    )
  }
  if (isAnnotationRendererWindow) {
    return (
      <Suspense fallback={<main className="whiteboard-loading"><span>Loading annotation renderer</span></main>}>
        <AnnotationRenderWindow />
      </Suspense>
    )
  }
  if (isRegionOverlayWindow) {
    return <RegionOverlayWindow />
  }
  if (isPipOverlayWindow) {
    return <PIPOverlayWindow />
  }
  if (isScreenshotPinWindow) {
    return <ScreenshotPinWindow />
  }
  if (isWhiteboardWindow) {
    return (
      <Suspense fallback={<main className="whiteboard-loading"><span>Loading whiteboard</span></main>}>
        <WhiteboardWindow />
      </Suspense>
    )
  }
  if (isFloatingPanelWindow) {
    return <FloatingPanelWindow />
  }

  const [selectedSource, setSelectedSource] = useState<CaptureSource>(sources[0])
  const [availableSources, setAvailableSources] = useState<CaptureSource[]>(sources)
  const [activePanel, setActivePanel] = useState<ActivePanel | null>(null)
  const [sourcePickerView, setSourcePickerView] = useState<'overview' | 'windows'>('overview')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [whiteboardVisibility, setWhiteboardVisibility] = useState<WhiteboardVisibilityUpdate | null>(null)
  const [closePromptOpen, setClosePromptOpen] = useState(false)
  const [closeBusy, setCloseBusy] = useState(false)
  const [recordingMode, setRecordingMode] = useState<RecordingMode>('video')
  const [state, setState] = useState<RecordingState>('idle')
  const [elapsed, setElapsed] = useState(0)
  const [countdownRemaining, setCountdownRemaining] = useState(0)
  const [recordingQuality, setRecordingQuality] = useState<RecordingQuality>('balanced')
  const [recordingFPS, setRecordingFPS] = useState(30)
  const [captureCursor, setCaptureCursor] = useState(true)
  const [countdownSeconds, setCountdownSeconds] = useState(0)
  const [systemAudio, setSystemAudio] = useState(false)
  const [availableSystemAudio, setAvailableSystemAudio] = useState<MediaDevice[]>(systemAudioDevices)
  const [selectedSystemAudio, setSelectedSystemAudio] = useState(systemAudioDevices[0].id)
  const [microphone, setMicrophone] = useState(false)
  const [noiseSuppression, setNoiseSuppression] = useState(false)
  const [availableMicrophones, setAvailableMicrophones] = useState<MediaDevice[]>([])
  const [selectedMic, setSelectedMic] = useState('')
  const [micLevel, setMicLevel] = useState(0)
  const [micPeak, setMicPeak] = useState(0)
  const [micMonitorActive, setMicMonitorActive] = useState(false)
  const [micMonitorError, setMicMonitorError] = useState<string | null>(null)
  const [camera, setCamera] = useState(false)
  const [availableCameras, setAvailableCameras] = useState<MediaDevice[]>(cameraDevices)
  const [selectedCamera, setSelectedCamera] = useState(cameraDevices[0].id)
  const [pipPreset, setPipPreset] = useState<PIPPreset>('bottom-right')
  const [pipShape, setPipShape] = useState<PIPShape>('circle')
  const [pipMirror, setPipMirror] = useState(true)
  const [pipPosition, setPipPosition] = useState(defaultPipPosition('bottom-right'))
  const [pipScale, setPipScale] = useState(pipDefaultScale)
  const [pipEdgeFeather, setPipEdgeFeather] = useState(0.16)
  const [locale, setLocale] = useState<LocaleCode>('zh-CN')
  const [theme, setTheme] = useState<ThemeCode>('night-teal')
  const [startAtLogin, setStartAtLogin] = useState(false)
  const [autoRecognizeScreenshots, setAutoRecognizeScreenshots] = useState(false)
  const [ocrTranslation, setOcrTranslation] = useState<AppSettings['ocr']['translation']>(defaultSettings.ocr.translation)
  const [shortcuts, setShortcuts] = useState<ShortcutSettings>(defaultSettings.shortcuts)
  const [shortcutCapture, setShortcutCapture] = useState<ShortcutAction | null>(null)
  const [shortcutError, setShortcutError] = useState('')
  const [screenshots, setScreenshots] = useState<ScreenshotItem[]>([])
  const [screenshotMessage, setScreenshotMessage] = useState('')
  const [includeAnnotationsInExport, setIncludeAnnotationsInExport] = useState(true)
  const [lastPackage, setLastPackage] = useState<string>(previewPackagePath)
  const [lastBackend, setLastBackend] = useState<string>('ui-preview')
  const [lastStatusMessage, setLastStatusMessage] = useState<StatusMessageState>({key: 'waiting'})
  const [lastPreflight, setLastPreflight] = useState<RecordingPreflight | null>(null)
  const [preflightBusy, setPreflightBusy] = useState(false)
  const [recoveries, setRecoveries] = useState<RecordingRecovery[]>([])
  const [recoveryBusy, setRecoveryBusy] = useState(false)
  const [recoveryMessage, setRecoveryMessage] = useState<RecoveryMessageState | null>(null)
  const [exportBusy, setExportBusy] = useState(false)
  const [exportMessage, setExportMessage] = useState<ExportMessageState | null>(null)
  const [exportPlanPreview, setExportPlanPreview] = useState<RecordingExportPlan | null>(null)
  const [exportPlanBusy, setExportPlanBusy] = useState(false)
  const [exportPlanError, setExportPlanError] = useState('')
  const [settingsLoaded, setSettingsLoaded] = useState(false)
  const [capsuleExpandDirection, setCapsuleExpandDirection] = useState<CapsuleWindowExpandDirection>('down')
  const [capsuleDockSide, setCapsuleDockSide] = useState<CapsuleWindowDockSide>('none')
  const [capabilities, setCapabilities] = useState<CaptureCapabilities>(fallbackCapabilities)
  const [appData, setAppData] = useState<AppDataInfo>(fallbackAppData)
  const [storageStatus, setStorageStatus] = useState<AppStorageStatus>(fallbackStorageStatus)
  const [storageRootDraft, setStorageRootDraft] = useState(fallbackAppData.rootDir)
  const [storageBusy, setStorageBusy] = useState(false)
  const [storageMessage, setStorageMessage] = useState<StorageMessageState | null>(null)
  const [sourceSelectionMessage, setSourceSelectionMessage] = useState<SourceSelectionMessageState | null>(null)
  const shellRef = useRef<HTMLElement | null>(null)
  const capsuleRef = useRef<HTMLDivElement | null>(null)
  const popoverRef = useRef<HTMLDivElement | null>(null)
  const settingsPanelRef = useRef<HTMLElement | null>(null)
  const closePromptRef = useRef<HTMLElement | null>(null)
  const capsuleHitRegionSignatureRef = useRef('')
  const forceCapsuleHitRegionPublishRef = useRef<(() => void) | null>(null)
  const capsuleWindowLayoutChangingRef = useRef(false)
  const capsuleWindowLayoutTokenRef = useRef(0)
  const capsuleDragCandidateRef = useRef(false)
  const capsuleDragObservedMoveRef = useRef(false)
  const capsuleDragStartPointRef = useRef<{x: number; y: number} | null>(null)
  const capsuleDragPendingUntilRef = useRef(0)
  const capsuleLastDragStabilizedAtRef = useRef(0)
  const capsuleProgrammaticMoveRef = useRef(false)
  const capsuleIgnoreMoveEventsUntilRef = useRef(0)
  const capsuleDockSideRef = useRef<CapsuleWindowDockSide>('none')
  const floatingPointerInsideAtRef = useRef(0)
  const floatingPointerInsideRef = useRef(false)
  const countdownTimerRef = useRef<number | null>(null)
  const countdownTokenRef = useRef(0)
  const cameraPreviewGenerationRef = useRef(0)
  const audioPatchTokenRef = useRef(0)
  const preferencePatchTokenRef = useRef(0)
  const whiteboardPatchTokenRef = useRef(0)
  const shortcutPatchTokenRef = useRef(0)
  const exportPlanTokenRef = useRef(0)
  const floatingPanelTokenRef = useRef(0)
  const localAudioIntentUntilRef = useRef(0)
  const localPreferenceIntentUntilRef = useRef(0)
  const localWhiteboardIntentUntilRef = useRef(0)
  const localCameraIntentUntilRef = useRef(0)
  const localPipIntentUntilRef = useRef(0)
  const selectedSystemAudioRef = useRef(selectedSystemAudio)
  const selectedMicRef = useRef(selectedMic)
  const selectedCameraRef = useRef(selectedCamera)
  const systemAudioRef = useRef(systemAudio)
  const microphoneRef = useRef(microphone)
  const noiseSuppressionRef = useRef(noiseSuppression)
  const cameraRef = useRef(camera)
  const shortcutCaptureRef = useRef<ShortcutAction | null>(null)
  const shortcutActionsRef = useRef<Record<ShortcutAction, () => void>>({
    toggleRecording: () => undefined,
    togglePause: () => undefined,
    toggleCamera: () => undefined,
    openWhiteboard: () => undefined,
    openScreenshot: () => undefined,
  })
  const currentSettingsRef = useRef<AppSettings | null>(null)
  const persistedSettingsRef = useRef<AppSettings | null>(null)
  const rnnoiseActive = microphone && noiseSuppression

  const copy = copyByLocale[locale]
  const lastStatusText = lastStatusMessage.fallback ?? copy.statusMessages[lastStatusMessage.key]
  const recoveryText = recoveryMessage ? formatRecoveryMessage(recoveryMessage, copy) : ''
  const exportText = exportMessage ? formatExportMessage(exportMessage, copy) : ''
  const storageText = storageMessage ? formatStorageMessage(storageMessage, copy) : ''
  const sourceSelectionText = sourceSelectionMessage ? formatSourceSelectionMessage(sourceSelectionMessage, copy) : ''
  const isRecording = state === 'recording' || state === 'paused' || state === 'preparing' || state === 'stopping'
  const recordingConfigLocked = isRecording
  const whiteboardButtonActive = whiteboardVisibility?.visible === true
  const capsuleExpanded = closePromptOpen
  const capsuleWindowCompact = recordingConfigLocked && !capsuleExpanded
  const capsuleExpandedHeight = settingsOpen
    ? 560
    : closePromptOpen
      ? 300
      : activePanel === 'audio'
        ? 500
      : activePanel === 'camera'
          ? 570
          : activePanel === 'language'
            ? 280
            : 500
  const SourceIcon = recordingMode === 'audio' ? Volume2 : sourceIcon[selectedSource.type]
  const sourceTitle = recordingMode === 'audio' ? copy.recordingModes.audio : sourceTypeLabel(selectedSource, copy)
  const sourceSubtitle = recordingMode === 'audio' ? audioOnlySourceMeta(systemAudio, microphone, copy) : sourceName(selectedSource, copy)
  const titleWithShortcut = (label: string, action: ShortcutAction) => `${label} · ${formatShortcutForDisplay(shortcuts[action])}`
  const currentSettings = useMemo<AppSettings>(() => {
    const rawPipConfig = normalizePipConfig({
      preset: pipPreset,
      shape: pipShape,
      mirror: pipMirror,
      position: pipPosition,
      scale: pipScale,
      edgeFeather: pipEdgeFeather,
    }, pipPreset)
    const savedPipConfig = camera ? ensureVisiblePipConfig(rawPipConfig) : rawPipConfig
    const nextSettings = {
      schemaVersion: 4,
      locale,
      source: {
        lastSourceId: selectedSource.id,
        lastSourceType: selectedSource.type,
      },
      storage: {
        dataRootDir: appData.rootDir,
      },
      recording: {
        quality: recordingQuality,
        fps: recordingFPS,
        captureCursor,
        countdownSeconds,
      },
      audio: persistedSettingsRef.current?.audio ?? {
        system: systemAudio,
        systemDeviceId: selectedSystemAudio || undefined,
        microphone,
        microphoneDeviceId: selectedMic || undefined,
        noiseSuppression,
        microphoneGain: 1,
      },
      camera: {
        enabled: camera,
        deviceId: selectedCamera,
        pipPreset: savedPipConfig.preset,
        pip: savedPipConfig,
      },
      whiteboard: {
        ...(persistedSettingsRef.current?.whiteboard ?? defaultSettings.whiteboard),
        capturePolicy: annotationCapturePolicy(includeAnnotationsInExport),
      },
      ocr: {
        ...(persistedSettingsRef.current?.ocr ?? defaultSettings.ocr),
        autoRecognizeScreenshots,
        translation: ocrTranslation,
      },
      shortcuts,
      window: {
        minimizeToTray: true,
        theme,
        startAtLogin,
      },
    }
    if (!isSettingsWindow || !persistedSettingsRef.current) return nextSettings
    return {
      ...persistedSettingsRef.current,
      schemaVersion: 4,
      locale,
      storage: {
        dataRootDir: appData.rootDir,
      },
      recording: {
        quality: recordingQuality,
        fps: recordingFPS,
        captureCursor,
        countdownSeconds,
      },
      whiteboard: {
        ...persistedSettingsRef.current.whiteboard,
        capturePolicy: annotationCapturePolicy(includeAnnotationsInExport),
      },
      ocr: {
        ...persistedSettingsRef.current.ocr,
        autoRecognizeScreenshots,
        translation: ocrTranslation,
      },
      shortcuts,
      window: {
        minimizeToTray: true,
        theme,
        startAtLogin,
      },
    }
  }, [appData.rootDir, autoRecognizeScreenshots, camera, captureCursor, countdownSeconds, includeAnnotationsInExport, isSettingsWindow, locale, microphone, noiseSuppression, ocrTranslation, pipEdgeFeather, pipMirror, pipPosition, pipPreset, pipScale, pipShape, recordingFPS, recordingQuality, selectedCamera, selectedMic, selectedSource.id, selectedSource.type, selectedSystemAudio, shortcuts, startAtLogin, systemAudio, theme])
  const settingsAutosaveKey = useMemo(() => JSON.stringify({
    locale,
    sourceId: selectedSource.id,
    sourceType: selectedSource.type,
    dataRootDir: appData.rootDir,
  }), [appData.rootDir, locale, selectedSource.id, selectedSource.type])
  const capabilityRows = useMemo(() => [
    capabilities.sourceEnumeration,
    capabilities.screenRecording,
    capabilities.windowRecording,
    capabilities.systemAudio,
    capabilities.microphone,
    capabilities.microphoneEnhancement,
    capabilities.cameraSidecar,
    capabilities.pipExport,
    capabilities.packageRecovery,
  ], [capabilities])
  const recoverableRecoveries = useMemo(() => recoveries.filter((recovery) => recovery.recoverable), [recoveries])
  const recoverablePackages = recoverableRecoveries.length
  const allScreensSource = useMemo(() => availableSources.find((source) => source.type === 'all-screens'), [availableSources])
  const screenSources = useMemo(() => availableSources.filter((source) => source.type === 'screen'), [availableSources])
  const regionSource = useMemo(() => availableSources.find((source) => source.type === 'region'), [availableSources])
  const windowSources = useMemo(() => availableSources.filter((source) => source.type === 'window'), [availableSources])
  const selectedWindowSource = selectedSource.type === 'window' ? selectedSource : windowSources.find((source) => source.id === selectedSource.id)
  const selectedCameraDevice = useMemo(
    () => availableCameras.find((device) => device.id === selectedCamera),
    [availableCameras, selectedCamera],
  )
  const currentPipConfig = useMemo<PIPConfig>(() => normalizePipConfig({
    preset: pipPreset,
    shape: pipShape,
    mirror: pipMirror,
    position: pipPosition,
    scale: pipScale,
    edgeFeather: pipEdgeFeather,
  }, pipPreset), [pipEdgeFeather, pipMirror, pipPosition, pipPreset, pipScale, pipShape])
  const pipCameraLabel = selectedCameraDevice?.name || selectedCamera
  const pipCameraTarget = useMemo<PIPOverlayCamera>(() => ({
    deviceId: selectedCameraDevice?.id || selectedCamera,
    nativeId: selectedCameraDevice?.nativeId,
    name: pipCameraLabel,
  }), [pipCameraLabel, selectedCamera, selectedCameraDevice?.id, selectedCameraDevice?.nativeId])

  useEffect(() => {
    currentSettingsRef.current = currentSettings
  }, [currentSettings])

  const selectedMicrophoneDevice = useMemo(
    () => availableMicrophones.find((device) => device.id === selectedMic),
    [availableMicrophones, selectedMic],
  )
  const hasAvailableMicrophone = useMemo(
    () => availableMicrophones.some((device) => device.available !== false),
    [availableMicrophones],
  )
  const hasUsableCamera = useMemo(
    () => availableCameras.some(isUsableCameraDevice),
    [availableCameras],
  )
  const selectedCameraUsable = selectedCameraDevice ? isUsableCameraDevice(selectedCameraDevice) : hasUsableCamera
  const fallbackUsableCameraDevice = useMemo(
    () => availableCameras.find(isUsableCameraDevice),
    [availableCameras],
  )
  const cameraUnavailableText = selectedCameraDevice?.unavailableReason || selectedCameraDevice?.meta || copy.pipOverlay.cameraUnavailable
  const cameraStatusText = !hasUsableCamera || !selectedCameraUsable
    ? cameraUnavailableText
    : camera
      ? copy.panels.cameraEnabled
      : copy.panels.cameraOff
  const micMonitorStatusText = micMonitorError
    ? copy.panels.microphoneLevelError
    : !microphone
      ? copy.panels.microphoneLevelOff
      : selectedMicrophoneDevice?.available === false || !hasAvailableMicrophone
        ? copy.panels.microphoneLevelUnavailable
        : micMonitorActive
          ? copy.panels.microphoneLevelLive
          : copy.panels.microphoneLevelWaiting
  const micMeterLevel = microphone && micMonitorActive ? micLevel : 0
  const micMeterBars = useMemo(() => Array.from({length: 18}, (_, index) => {
    const threshold = (index + 1) / 18
    const active = micMeterLevel >= threshold
    const height = active ? Math.max(14, Math.min(100, micMeterLevel * 100 + index * 0.9)) : 8
    return {active, height: `${height}%`}
  }), [micMeterLevel])
  const canOpenLastPackage = isRecordingPackagePath(lastPackage) && lastPackage !== previewPackagePath
  const lastPackageName = canOpenLastPackage ? packageDisplayName(lastPackage) : copy.settings.noRecordingPackage
  const exportPlanValue = exportPlanBusy
    ? copy.settings.exportPlanLoading
    : exportPlanPreview
      ? formatExportPlanValue(exportPlanPreview, copy)
      : copy.settings.exportPlanUnavailable
  const exportPlanDetail = exportPlanError || (exportPlanPreview ? formatExportPlanDetail(exportPlanPreview, copy) : copy.settings.exportPlanPendingDetail)

  useEffect(() => {
    const token = exportPlanTokenRef.current + 1
    exportPlanTokenRef.current = token
    if (!canOpenLastPackage || isRecording) {
      setExportPlanPreview(null)
      setExportPlanError('')
      setExportPlanBusy(false)
      return
    }
    setExportPlanBusy(true)
    setExportPlanError('')
    void previewExportRecordingPackage(lastPackage, {includeAnnotations: includeAnnotationsInExport})
      .then((plan) => {
        if (token !== exportPlanTokenRef.current) return
        setExportPlanPreview(plan)
      })
      .catch((error) => {
        if (token !== exportPlanTokenRef.current) return
        setExportPlanPreview(null)
        setExportPlanError(error instanceof Error ? error.message : String(error))
      })
      .finally(() => {
        if (token === exportPlanTokenRef.current) setExportPlanBusy(false)
      })
  }, [canOpenLastPackage, includeAnnotationsInExport, isRecording, lastPackage])

  const openPipEditor = async () => {
    if (!cameraRef.current || !hasUsableCamera || recordingMode !== 'video') return
    const nextConfig = currentPipConfig.preset === 'off'
      ? normalizePipConfig({...currentPipConfig, preset: 'bottom-right', position: defaultPipPosition('bottom-right')}, 'bottom-right')
      : currentPipConfig
    if (nextConfig.preset !== pipPreset) {
      setPipPreset(nextConfig.preset)
      setPipPosition(nextConfig.position)
    }
    try {
      const generation = cameraPreviewGenerationRef.current
      await showPipOverlay(nextConfig, 'edit', pipCameraTarget)
      if (generation !== cameraPreviewGenerationRef.current || !cameraRef.current) {
        await hidePipOverlay()
      }
    } catch (error) {
      console.info('PIP editor unavailable:', error)
    }
  }
  const stopCameraPreview = (reason = 'unspecified') => {
    void logClientEvent('camera', 'preview-stop-request', {reason})
    cameraPreviewGenerationRef.current += 1
    void hidePipOverlay()
  }
  const applyLocalPipConfigState = (config: PIPConfig) => {
    const nextConfig = normalizePipConfig(config, config.preset)
    setPipPreset(nextConfig.preset)
    setPipShape(nextConfig.shape)
    setPipMirror(nextConfig.mirror)
    setPipPosition(nextConfig.position)
    setPipScale(nextConfig.scale)
    setPipEdgeFeather(nextConfig.edgeFeather)
  }
  const persistCameraSettings = (enabled: boolean, pipConfig: PIPConfig, deviceId = selectedCameraRef.current || selectedCamera) => {
    const normalizedPip = enabled
      ? ensureVisiblePipConfig(pipConfig)
      : normalizePipConfig({...pipConfig, preset: 'off'}, 'off')
    const baseSettings = currentSettingsRef.current ?? currentSettings
    const nextSettings: AppSettings = {
      ...baseSettings,
      camera: {
        ...baseSettings.camera,
        enabled,
        deviceId,
        pipPreset: normalizedPip.preset,
        pip: normalizedPip,
      },
    }
    currentSettingsRef.current = nextSettings
    persistedSettingsRef.current = nextSettings
    void patchCameraState({
      enabled,
      deviceId,
      pipPreset: normalizedPip.preset,
      pip: normalizedPip,
    })
      .then((saved) => {
        currentSettingsRef.current = saved
        persistedSettingsRef.current = saved
        applySettingsState(saved, undefined, undefined, {
          preserveCameraEnabled: hasLocalCameraIntent(),
          preserveCameraSelection: hasLocalCameraIntent(),
          preservePipConfig: hasLocalPipIntent(),
        })
      })
      .catch((error) => console.error('Failed to persist camera settings:', error))
  }
  const commitPipConfigFromPanel = (patch: Partial<PIPConfig>) => {
    if (!cameraRef.current || !hasUsableCamera || recordingMode !== 'video') return
    const nextConfig = ensureVisiblePipConfig(normalizePipConfig({
      ...currentPipConfig,
      ...patch,
      position: patch.position ?? currentPipConfig.position,
    }, (patch.preset as PIPPreset | undefined) ?? currentPipConfig.preset))
    markLocalPipIntent()
    applyLocalPipConfigState(nextConfig)
    void updatePipOverlay(nextConfig, 'edit', pipCameraTarget)
      .then((state) => {
        if (!currentSettingsRef.current) return
        const nextSettings: AppSettings = {
          ...currentSettingsRef.current,
          camera: {
            ...currentSettingsRef.current.camera,
            enabled: true,
            pipPreset: state.config.preset,
            pip: state.config,
          },
        }
        currentSettingsRef.current = nextSettings
        persistedSettingsRef.current = nextSettings
      })
      .catch((error) => {
        console.info('PIP panel update failed:', error)
        persistCameraSettings(true, nextConfig)
      })
  }
  const setCameraEnabled = (enabled: boolean, deviceId = selectedCamera) => {
    const nextEnabled = enabled && hasUsableCamera
    markLocalCameraIntent()
    markLocalPipIntent()
    const nextPipConfig = nextEnabled
      ? ensureVisiblePipConfig(currentPipConfig.preset === 'off'
          ? normalizePipConfig({...currentPipConfig, preset: 'bottom-right', position: defaultPipPosition('bottom-right')}, 'bottom-right')
          : currentPipConfig)
      : normalizePipConfig({...currentPipConfig, preset: 'off'}, 'off')
    void logClientEvent('camera', 'toggle', {
      requested: enabled,
      enabled: nextEnabled,
      hasUsableCamera,
      selectedCamera: deviceId,
      selectedCameraName: selectedCameraDevice?.name ?? '',
    })
    cameraPreviewGenerationRef.current += 1
    cameraRef.current = nextEnabled
    setCamera(nextEnabled)
    applyLocalPipConfigState(nextPipConfig)
    persistCameraSettings(nextEnabled, nextPipConfig, deviceId)
    if (!nextEnabled) {
      void hidePipOverlay()
    }
  }
  const chooseCameraDevice = (deviceId: string) => {
    if (recordingConfigLocked) return
    selectedCameraRef.current = deviceId
    setSelectedCamera(deviceId)
    markLocalCameraIntent()
    const nextPipConfig = cameraRef.current
      ? ensureVisiblePipConfig(currentPipConfig)
      : normalizePipConfig(currentPipConfig, currentPipConfig.preset)
    persistCameraSettings(cameraRef.current, nextPipConfig, deviceId)
  }
  const applyRecordingStatus = (update: RecordingStatusUpdate) => {
    const nextStatus = update.status as RecordingState
    setState(nextStatus)
    if (update.status !== 'preparing') setCountdownRemaining(0)
    if (nextStatus === 'idle' || nextStatus === 'ready' || nextStatus === 'failed') {
      setElapsed(0)
    }
    if (!isSettingsWindow && !isFloatingPanelWindow && (nextStatus === 'ready' || nextStatus === 'failed')) {
      void restoreCapsuleWindow(false)
    }
    if (update.session?.packagePath) setLastPackage(update.session.packagePath)
    const backend = update.session?.backend || update.backend
    if (backend) setLastBackend(backend)
    if (update.message) setLastStatusMessage(statusMessageFromBackend(update.message))
  }
  const markLocalCameraIntent = () => {
    localCameraIntentUntilRef.current = Date.now() + 5000
  }
  const markLocalAudioIntent = () => {
    localAudioIntentUntilRef.current = Date.now() + 5000
  }
  const markLocalPreferenceIntent = () => {
    localPreferenceIntentUntilRef.current = Date.now() + 5000
  }
  const markLocalWhiteboardIntent = () => {
    localWhiteboardIntentUntilRef.current = Date.now() + 5000
  }
  const markLocalPipIntent = () => {
    localPipIntentUntilRef.current = Date.now() + 5000
  }
  const hasLocalAudioIntent = () => Date.now() < localAudioIntentUntilRef.current
  const hasLocalPreferenceIntent = () => Date.now() < localPreferenceIntentUntilRef.current
  const hasLocalWhiteboardIntent = () => Date.now() < localWhiteboardIntentUntilRef.current
  const hasLocalCameraIntent = () => Date.now() < localCameraIntentUntilRef.current
  const hasLocalPipIntent = () => Date.now() < localPipIntentUntilRef.current
  const settingsWithPreferencePatch = (settings: AppSettings | null, patch: SettingsPreferencesPatch): AppSettings | null => {
    if (!settings) return settings
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
    }
  }
  const applyLocalPreferencePatch = (patch: SettingsPreferencesPatch) => {
    if (patch.theme !== undefined) setTheme(normalizeTheme(patch.theme))
    if (patch.startAtLogin !== undefined) setStartAtLogin(patch.startAtLogin)
    if (patch.autoOcr !== undefined) setAutoRecognizeScreenshots(patch.autoOcr)
    if (patch.ocrTranslation !== undefined) setOcrTranslation((current) => normalizeOcrTranslationSettings({...current, ...patch.ocrTranslation}))
    if (patch.recordingQuality !== undefined) setRecordingQuality(normalizeRecordingQuality(patch.recordingQuality))
    if (patch.recordingFps !== undefined) setRecordingFPS(fpsOptions.includes(patch.recordingFps) ? patch.recordingFps : 30)
    if (patch.captureCursor !== undefined) setCaptureCursor(patch.captureCursor)
    if (patch.countdownSeconds !== undefined) setCountdownSeconds(countdownOptions.includes(patch.countdownSeconds) ? patch.countdownSeconds : 0)
    currentSettingsRef.current = settingsWithPreferencePatch(currentSettingsRef.current, patch)
    persistedSettingsRef.current = settingsWithPreferencePatch(persistedSettingsRef.current, patch)
  }
  const completeSettingsPreferencePatch = (patch: SettingsPreferencesPatch): SettingsPreferencesPatch => {
    if (patch.ocrTranslation === undefined) return patch
    return {
      ...patch,
      ocrTranslation: normalizeOcrTranslationSettings({
        ...(currentSettingsRef.current?.ocr.translation ?? ocrTranslation),
        ...patch.ocrTranslation,
      }),
    }
  }
  const commitSettingsPreferencePatch = (rawPatch: SettingsPreferencesPatch) => {
    const patch = completeSettingsPreferencePatch(rawPatch)
    markLocalPreferenceIntent()
    const token = preferencePatchTokenRef.current + 1
    preferencePatchTokenRef.current = token
    applyLocalPreferencePatch(patch)
    void logClientEvent('settings-preferences', 'patch-request', {
      theme: patch.theme ?? '',
      startAtLogin: patch.startAtLogin ?? '',
      recordingQuality: patch.recordingQuality ?? '',
      recordingFps: patch.recordingFps ?? '',
      captureCursor: patch.captureCursor ?? '',
      countdownSeconds: patch.countdownSeconds ?? '',
      autoOcr: patch.autoOcr ?? '',
      ocrTranslationProvider: patch.ocrTranslation?.provider ?? '',
      ocrTranslationApiKeySet: patch.ocrTranslation?.apiKey !== undefined ? String(Boolean(patch.ocrTranslation.apiKey)) : '',
    })
    void patchSettingsPreferences(patch)
      .then((settings) => {
        if (token !== preferencePatchTokenRef.current) return
        localPreferenceIntentUntilRef.current = Date.now() + 3000
        void logClientEvent('settings-preferences', 'patch-success', {
          theme: settings.window.theme,
          startAtLogin: settings.window.startAtLogin,
          recordingQuality: settings.recording.quality,
          recordingFps: settings.recording.fps,
          captureCursor: settings.recording.captureCursor,
          countdownSeconds: settings.recording.countdownSeconds,
          autoOcr: settings.ocr.autoRecognizeScreenshots,
          ocrTranslationProvider: settings.ocr.translation.provider,
          ocrTranslationApiKeySet: Boolean(settings.ocr.translation.apiKey || settings.ocr.translation.apiKeySet),
        })
        applySettingsState(settings, undefined, undefined, {
          preserveRecordingSettings: hasLocalPreferenceIntent(),
          preserveTheme: hasLocalPreferenceIntent(),
          preserveOcr: hasLocalPreferenceIntent(),
        })
      })
      .catch((error) => {
        console.error('Failed to patch settings preferences:', error)
        void logClientEvent('settings-preferences', 'patch-error', {}, readableError(error))
        void loadSettings()
          .then((settings) => applySettingsState(settings))
          .catch((loadError) => console.error('Failed to reload settings preferences:', loadError))
      })
  }
  const settingsWithWhiteboardPatch = (settings: AppSettings | null, patch: WhiteboardSettingsPatch): AppSettings | null => {
    if (!settings) return settings
    return {
      ...settings,
      whiteboard: {
        ...settings.whiteboard,
        ...patch,
      },
    }
  }
  const applyLocalWhiteboardPatch = (patch: WhiteboardSettingsPatch) => {
    if (patch.capturePolicy !== undefined) {
      setIncludeAnnotationsInExport(patch.capturePolicy !== 'preview-only')
    }
    currentSettingsRef.current = settingsWithWhiteboardPatch(currentSettingsRef.current, patch)
    persistedSettingsRef.current = settingsWithWhiteboardPatch(persistedSettingsRef.current, patch)
  }
  const commitWhiteboardSettingsPatch = (patch: WhiteboardSettingsPatch) => {
    markLocalWhiteboardIntent()
    const token = whiteboardPatchTokenRef.current + 1
    whiteboardPatchTokenRef.current = token
    applyLocalWhiteboardPatch(patch)
    void logClientEvent('whiteboard-settings', 'patch-request', {
      capturePolicy: patch.capturePolicy ?? '',
      lastMode: patch.lastMode ?? '',
      lastTool: patch.lastTool ?? '',
    })
    void patchWhiteboardSettings(patch)
      .then((settings) => {
        if (token !== whiteboardPatchTokenRef.current) return
        localWhiteboardIntentUntilRef.current = Date.now() + 3000
        persistedSettingsRef.current = settings
        currentSettingsRef.current = settings
        setIncludeAnnotationsInExport(settings.whiteboard.capturePolicy !== 'preview-only')
      })
      .catch((error) => {
        console.error('Failed to patch whiteboard settings:', error)
        void logClientEvent('whiteboard-settings', 'patch-error', {}, readableError(error))
        void loadSettings()
          .then((settings) => applySettingsState(settings))
          .catch((loadError) => console.error('Failed to reload whiteboard settings:', loadError))
      })
  }
  const settingsWithShortcutPatch = (settings: AppSettings | null, patch: ShortcutSettingsPatch): AppSettings | null => {
    if (!settings) return settings
    return {
      ...settings,
      shortcuts: {
        ...settings.shortcuts,
        ...patch,
      },
    }
  }
  const applyLocalShortcutPatch = (patch: ShortcutSettingsPatch) => {
    setShortcuts((current) => ({...current, ...patch}))
    currentSettingsRef.current = settingsWithShortcutPatch(currentSettingsRef.current, patch)
    persistedSettingsRef.current = settingsWithShortcutPatch(persistedSettingsRef.current, patch)
  }
  const commitShortcutSettingsPatch = (action: ShortcutAction, accelerator: string) => {
    const conflict = shortcutActions.find((candidate) => (
      candidate !== action &&
      shortcutIdentity(shortcuts[candidate]) === shortcutIdentity(accelerator)
    ))
    if (conflict) {
      setShortcutError(copy.settings.shortcutConflict(copy.settings.shortcutActionLabels[conflict]))
      return
    }
    const patch = {[action]: accelerator} as ShortcutSettingsPatch
    const token = shortcutPatchTokenRef.current + 1
    shortcutPatchTokenRef.current = token
    setShortcutCapture(null)
    setShortcutError('')
    applyLocalShortcutPatch(patch)
    void logClientEvent('shortcuts', 'patch-request', {
      action,
      accelerator,
    })
    void patchShortcutSettings(patch)
      .then((settings) => {
        if (token !== shortcutPatchTokenRef.current) return
        persistedSettingsRef.current = settings
        currentSettingsRef.current = settings
        setShortcuts(settings.shortcuts)
      })
      .catch((error) => {
        const message = readableError(error)
        setShortcutError(message || copy.settings.shortcutInvalid)
        console.error('Failed to patch shortcut settings:', error)
        void logClientEvent('shortcuts', 'patch-error', {action, accelerator}, message)
        void loadSettings()
          .then((settings) => applySettingsState(settings))
          .catch((loadError) => console.error('Failed to reload shortcut settings:', loadError))
      })
  }
  const mergeAudioIntoSettingsCache = (audio: AudioControlState) => {
    const apply = (settings: AppSettings | null): AppSettings | null => {
      if (!settings) return settings
      return {
        ...settings,
        audio: {
          system: audio.system,
          systemDeviceId: audio.systemDeviceId,
          microphone: audio.microphone,
          microphoneDeviceId: audio.microphoneDeviceId,
          noiseSuppression: audio.microphone && audio.noiseSuppression,
          microphoneGain: audio.microphoneGain || 1,
        },
      }
    }
    currentSettingsRef.current = apply(currentSettingsRef.current)
    persistedSettingsRef.current = apply(persistedSettingsRef.current)
  }
  const applyAudioControlState = (audio: AudioControlState) => {
    const nextNoiseSuppression = audio.microphone && audio.noiseSuppression
    systemAudioRef.current = audio.system
    microphoneRef.current = audio.microphone
    noiseSuppressionRef.current = nextNoiseSuppression
    setSystemAudio(audio.system)
    setMicrophone(audio.microphone)
    setNoiseSuppression(nextNoiseSuppression)
    if (audio.systemDeviceId) {
      selectedSystemAudioRef.current = audio.systemDeviceId
      setSelectedSystemAudio(audio.systemDeviceId)
    }
    if (audio.microphoneDeviceId) {
      selectedMicRef.current = audio.microphoneDeviceId
      setSelectedMic(audio.microphoneDeviceId)
    }
    if (!audio.microphone) {
      setMicMonitorError(null)
      setMicMonitorActive(false)
      setMicLevel(0)
      setMicPeak(0)
    }
    mergeAudioIntoSettingsCache(audio)
  }
  const optimisticAudioState = (patch: AudioStatePatch): AudioControlState => {
    const nextMicrophone = patch.microphone ?? microphoneRef.current
    const nextNoiseSuppression = nextMicrophone ? (patch.noiseSuppression ?? noiseSuppressionRef.current) : false
    return {
      system: patch.system ?? systemAudioRef.current,
      systemDeviceId: patch.clearSystemDevice ? undefined : (patch.systemDeviceId ?? selectedSystemAudioRef.current),
      microphone: nextMicrophone,
      microphoneDeviceId: patch.clearMicrophoneDevice ? undefined : (patch.microphoneDeviceId ?? selectedMicRef.current),
      noiseSuppression: nextNoiseSuppression,
      microphoneGain: patch.microphoneGain ?? currentSettingsRef.current?.audio.microphoneGain ?? 1,
    }
  }
  const commitAudioStatePatch = (patch: AudioStatePatch) => {
    markLocalAudioIntent()
    const token = audioPatchTokenRef.current + 1
    audioPatchTokenRef.current = token
    const optimistic = optimisticAudioState(patch)
    applyAudioControlState(optimistic)
    void patchAudioState(patch)
      .then((state) => {
        if (token === audioPatchTokenRef.current) {
          applyAudioControlState(state)
        }
      })
      .catch((error) => {
        console.error('Failed to patch audio state:', error)
        void loadSettings()
          .then((settings) => applyAudioControlState({
            system: settings.audio.system,
            systemDeviceId: settings.audio.systemDeviceId,
            microphone: settings.audio.microphone,
            microphoneDeviceId: settings.audio.microphoneDeviceId,
            noiseSuppression: settings.audio.noiseSuppression,
            microphoneGain: settings.audio.microphoneGain,
          }))
          .catch((loadError) => console.error('Failed to reload audio settings:', loadError))
      })
  }
  const applySettingsState = (nextSettings: AppSettings, nextMedia?: MediaInventory, nextSources?: CaptureSource[], options: ApplySettingsOptions = {}) => {
    let effectiveSettings = nextSettings
    if ((options.preserveRecordingSettings || options.preserveTheme || options.preserveOcr || options.preservePipConfig || options.preserveWhiteboard) && currentSettingsRef.current) {
      effectiveSettings = {
        ...effectiveSettings,
        recording: options.preserveRecordingSettings
          ? currentSettingsRef.current.recording
          : effectiveSettings.recording,
        window: options.preserveTheme
          ? {
              ...effectiveSettings.window,
              theme: currentSettingsRef.current.window.theme,
              startAtLogin: currentSettingsRef.current.window.startAtLogin,
            }
          : effectiveSettings.window,
        ocr: options.preserveOcr
          ? currentSettingsRef.current.ocr
          : effectiveSettings.ocr,
        camera: options.preservePipConfig
          ? {
              ...effectiveSettings.camera,
              pipPreset: currentSettingsRef.current.camera.pipPreset,
              pip: currentSettingsRef.current.camera.pip,
            }
          : effectiveSettings.camera,
        whiteboard: options.preserveWhiteboard
          ? currentSettingsRef.current.whiteboard
          : effectiveSettings.whiteboard,
      }
    }
    persistedSettingsRef.current = effectiveSettings
    setIncludeAnnotationsInExport(effectiveSettings.whiteboard.capturePolicy !== 'preview-only')
    setShortcuts(effectiveSettings.shortcuts ?? defaultSettings.shortcuts)
    const systemAudioList = nextMedia?.systemAudio
    const microphoneList = nextMedia?.microphones
    const cameraList = nextMedia?.cameras
    setLocale(normalizeLocale(effectiveSettings.locale))
    if (!options.preserveTheme) {
      setTheme(normalizeTheme(effectiveSettings.window.theme))
    }
    setStartAtLogin(Boolean(effectiveSettings.window.startAtLogin))
    if (!options.preserveOcr) {
      setAutoRecognizeScreenshots(effectiveSettings.ocr.autoRecognizeScreenshots)
      setOcrTranslation(normalizeOcrTranslationSettings(effectiveSettings.ocr.translation))
    }
    if (!options.preserveRecordingSettings) {
      setRecordingQuality(normalizeRecordingQuality(effectiveSettings.recording.quality))
      setRecordingFPS(fpsOptions.includes(effectiveSettings.recording.fps) ? effectiveSettings.recording.fps : 30)
      setCaptureCursor(effectiveSettings.recording.captureCursor)
      setCountdownSeconds(countdownOptions.includes(effectiveSettings.recording.countdownSeconds) ? effectiveSettings.recording.countdownSeconds : 0)
    }
    const nextHasAvailableMicrophone = !microphoneList || microphoneList.some((device) => device.available !== false)
    const nextSystemAudioEnabled = options.preserveAudioEnabled
      ? systemAudioRef.current
      : effectiveSettings.audio.system
    const nextMicrophoneEnabled = options.preserveAudioEnabled
      ? microphoneRef.current && nextHasAvailableMicrophone
      : effectiveSettings.audio.microphone && nextHasAvailableMicrophone
    const nextNoiseSuppressionEnabled = options.preserveAudioEnabled
      ? noiseSuppressionRef.current && nextMicrophoneEnabled
      : effectiveSettings.audio.microphone && nextHasAvailableMicrophone && effectiveSettings.audio.noiseSuppression
    const nextCameraDevice = selectPreferredCameraDevice(cameraList, effectiveSettings.camera.deviceId)
    const nextHasUsableCamera = !cameraList || Boolean(nextCameraDevice)
    const nextCameraEnabled = options.preserveCameraEnabled
      ? cameraRef.current
      : effectiveSettings.camera.enabled && nextHasUsableCamera
    systemAudioRef.current = nextSystemAudioEnabled
    microphoneRef.current = nextMicrophoneEnabled
    noiseSuppressionRef.current = nextNoiseSuppressionEnabled
    setSystemAudio(nextSystemAudioEnabled)
    setMicrophone(nextMicrophoneEnabled)
    setNoiseSuppression(nextNoiseSuppressionEnabled)
    cameraRef.current = nextCameraEnabled
    setCamera(nextCameraEnabled)
    const nextPip = nextCameraEnabled
      ? ensureVisiblePipConfig(normalizePipConfig(effectiveSettings.camera.pip, normalizePipPreset(effectiveSettings.camera.pipPreset)))
      : normalizePipConfig(effectiveSettings.camera.pip, normalizePipPreset(effectiveSettings.camera.pipPreset))
    setPipPreset(nextPip.preset)
    setPipShape(nextPip.shape)
    setPipMirror(nextPip.mirror)
    setPipPosition(nextPip.position)
    setPipScale(nextPip.scale)
    setPipEdgeFeather(nextPip.edgeFeather)

    if (systemAudioList) setAvailableSystemAudio(systemAudioList)
    if (microphoneList) setAvailableMicrophones(microphoneList)
    if (cameraList) setAvailableCameras(cameraList)
    if (!options.preserveAudioSelection) {
      if (effectiveSettings.audio.systemDeviceId && (!systemAudioList || systemAudioList.some((device) => device.id === effectiveSettings.audio.systemDeviceId))) {
        selectedSystemAudioRef.current = effectiveSettings.audio.systemDeviceId
        setSelectedSystemAudio(effectiveSettings.audio.systemDeviceId)
      } else if (systemAudioList?.[0]) {
        selectedSystemAudioRef.current = systemAudioList[0].id
        setSelectedSystemAudio(systemAudioList[0].id)
      }
      if (effectiveSettings.audio.microphoneDeviceId && (!microphoneList || microphoneList.some((device) => device.id === effectiveSettings.audio.microphoneDeviceId))) {
        selectedMicRef.current = effectiveSettings.audio.microphoneDeviceId
        setSelectedMic(effectiveSettings.audio.microphoneDeviceId)
      } else if (microphoneList?.[0]) {
        selectedMicRef.current = microphoneList[0].id
        setSelectedMic(microphoneList[0].id)
      } else if (microphoneList) {
        selectedMicRef.current = ''
        setSelectedMic('')
      }
    }
    if (!options.preserveCameraSelection) {
      if (!cameraList) {
        if (effectiveSettings.camera.deviceId) setSelectedCamera(effectiveSettings.camera.deviceId)
      } else if (nextCameraDevice) {
        setSelectedCamera(nextCameraDevice.id)
      } else {
        setSelectedCamera(cameraList[0]?.id ?? '')
      }
    }
    if (nextSources) {
      setSelectedSource(selectVisibleInitialSource(nextSources, effectiveSettings.source.lastSourceId, effectiveSettings.source.lastSourceType))
    }
  }

  useEffect(() => {
    document.body.classList.toggle('rf-settings-window', isSettingsWindow)
    document.body.classList.toggle('rf-floating-panel-window', isFloatingPanelWindow)
    document.body.classList.toggle('rf-recorder-window', !isSettingsWindow && !isFloatingPanelWindow)
    return () => {
      document.body.classList.remove('rf-settings-window', 'rf-recorder-window', 'rf-floating-panel-window')
    }
  }, [isFloatingPanelWindow, isSettingsWindow])

  useEffect(() => {
    document.documentElement.lang = locale
  }, [locale])

  useEffect(() => {
    document.documentElement.dataset.theme = theme
  }, [theme])

  useEffect(() => {
    selectedSystemAudioRef.current = selectedSystemAudio
  }, [selectedSystemAudio])

  useEffect(() => {
    selectedMicRef.current = selectedMic
  }, [selectedMic])

  useEffect(() => {
    selectedCameraRef.current = selectedCamera
  }, [selectedCamera])

  useEffect(() => {
    systemAudioRef.current = systemAudio
  }, [systemAudio])

  useEffect(() => {
    microphoneRef.current = microphone
  }, [microphone])

  useEffect(() => {
    noiseSuppressionRef.current = noiseSuppression
  }, [noiseSuppression])

  useEffect(() => {
    cameraRef.current = camera
  }, [camera])

  useEffect(() => {
    shortcutCaptureRef.current = shortcutCapture
  }, [shortcutCapture])

  useEffect(() => {
    capsuleDockSideRef.current = capsuleDockSide
  }, [capsuleDockSide])

  useEffect(() => subscribeCapsuleDockSide((side) => {
    capsuleDockSideRef.current = side
    setCapsuleDockSide(side)
  }), [])

  useEffect(() => {
    let cancelled = false
    void getFloatingPanelState()
      .then((state) => {
        if (cancelled) return
        applyFloatingPanelState(state)
      })
      .catch((error) => console.info('Floating panel state unavailable:', error))
    const unsubscribe = subscribeFloatingPanelChanged(applyFloatingPanelState)
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  const applyFloatingPanelState = (state: FloatingPanelState) => {
    if (!state.visible) {
      setActivePanel(null)
      setSettingsOpen(false)
      return
    }
    if (state.kind === 'settings') {
      setActivePanel(null)
      setSettingsOpen(true)
      return
    }
    if (state.kind === 'close') {
      setActivePanel(null)
      setSettingsOpen(false)
      return
    }
    if (state.kind === 'source' || state.kind === 'audio' || state.kind === 'camera' || state.kind === 'language' || state.kind === 'board') {
      setSettingsOpen(false)
      setActivePanel(state.kind)
    }
  }

  useEffect(() => {
    let cancelled = false
    void getSourceState()
      .then((sourceState) => {
        if (!cancelled) applySourceControlState(sourceState)
      })
      .catch((error) => console.info('Source state unavailable:', error))
    const unsubscribe = subscribeSourceStateChanged(applySourceControlState)
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [availableSources])

  const applySourceControlState = (sourceState: SourceControlState) => {
    if (sourceState.recordingMode) setRecordingMode(sourceState.recordingMode)
    if (!sourceState.sourceId && !sourceState.sourceType) return
    setSelectedSource((current) => {
      const direct = sourceState.sourceId
        ? availableSources.find((source) => source.id === sourceState.sourceId)
        : undefined
      const byType = sourceState.sourceType
        ? availableSources.find((source) => source.type === sourceState.sourceType)
        : undefined
      const picked = direct ?? byType
      if (!picked) return current
      if (picked.type !== 'region' || !sourceState.sourceGeometry) return picked
      return {
        ...picked,
        x: sourceState.sourceGeometry.x,
        y: sourceState.sourceGeometry.y,
        width: sourceState.sourceGeometry.width,
        height: sourceState.sourceGeometry.height,
        displayIndex: sourceState.sourceGeometry.displayIndex,
        nativeId: sourceState.sourceGeometry.nativeId,
      }
    })
  }

  useEffect(() => subscribeAudioLevel((update: AudioLevelUpdate) => {
    const currentMic = selectedMicRef.current
    if (update.deviceId && currentMic && update.deviceId !== currentMic) return
    if (update.error) {
      setMicMonitorError(update.error)
      setMicMonitorActive(false)
      setMicLevel(0)
      setMicPeak(0)
      return
    }
    setMicMonitorError(null)
    setMicMonitorActive(update.active)
    setMicLevel(update.active ? update.level : 0)
    setMicPeak(update.active ? update.peak : 0)
  }), [])

  useLayoutEffect(() => {
    if (isSettingsWindow || isFloatingPanelWindow) return
    const token = capsuleWindowLayoutTokenRef.current + 1
    capsuleWindowLayoutTokenRef.current = token
    capsuleWindowLayoutChangingRef.current = true
    capsuleProgrammaticMoveRef.current = true
    capsuleIgnoreMoveEventsUntilRef.current = Date.now() + 700
    let disposed = false
    let forceTimer = 0
    let secondForceTimer = 0
    let programmaticMoveTimer = 0
    const releaseProgrammaticMove = () => {
      if (programmaticMoveTimer) window.clearTimeout(programmaticMoveTimer)
      programmaticMoveTimer = window.setTimeout(() => {
        if (!disposed && token === capsuleWindowLayoutTokenRef.current) {
          capsuleProgrammaticMoveRef.current = false
        }
      }, 180)
    }
    const forceStableRegion = () => {
      if (disposed || token !== capsuleWindowLayoutTokenRef.current) return
      capsuleWindowLayoutChangingRef.current = false
      forceCapsuleHitRegionPublishRef.current?.()
      forceTimer = window.setTimeout(() => {
        if (disposed || token !== capsuleWindowLayoutTokenRef.current) return
        forceCapsuleHitRegionPublishRef.current?.()
      }, 80)
      secondForceTimer = window.setTimeout(() => {
        if (disposed || token !== capsuleWindowLayoutTokenRef.current) return
        forceCapsuleHitRegionPublishRef.current?.()
      }, 220)
    }
    void setCapsuleWindowExpanded(capsuleExpanded, capsuleExpandedHeight, 'auto', capsuleWindowCompact)
      .then((direction) => {
        if (!disposed && token === capsuleWindowLayoutTokenRef.current) setCapsuleExpandDirection(direction)
      })
      .finally(() => {
        window.requestAnimationFrame(() => {
          window.requestAnimationFrame(forceStableRegion)
        })
        releaseProgrammaticMove()
      })
    return () => {
      disposed = true
      if (forceTimer) window.clearTimeout(forceTimer)
      if (secondForceTimer) window.clearTimeout(secondForceTimer)
      if (programmaticMoveTimer) window.clearTimeout(programmaticMoveTimer)
    }
  }, [capsuleExpanded, capsuleExpandedHeight, capsuleWindowCompact, isFloatingPanelWindow, isSettingsWindow])

  useLayoutEffect(() => {
    if (isSettingsWindow || isFloatingPanelWindow) return
    let disposed = false
    let frame = 0
    let pendingForce = false

    const publish = (force = false) => {
      if (disposed) return
      if (!force && capsuleWindowLayoutChangingRef.current) return
      const viewportWidth = window.innerWidth || document.documentElement.clientWidth || 0
      const viewportHeight = window.innerHeight || document.documentElement.clientHeight || 0
      const regions = [
        elementHitRegion(capsuleRef.current, viewportWidth, viewportHeight, 'pill', 999),
        closePromptOpen ? elementHitRegion(closePromptRef.current, viewportWidth, viewportHeight, 'round-rect', 22) : null,
      ].filter((region): region is CapsuleWindowHitRegion => region !== null)
      const request = {
        enabled: regions.length > 0,
        force,
        viewportWidth,
        viewportHeight,
        devicePixelRatio: window.devicePixelRatio || 1,
        regions,
      }
      const signature = capsuleHitRegionRequestSignature(request)
      if (!force && signature === capsuleHitRegionSignatureRef.current) return
      capsuleHitRegionSignatureRef.current = signature
      void setCapsuleWindowHitRegions(request)
    }

    const schedule = (force = false) => {
      pendingForce = pendingForce || force
      if (frame) window.cancelAnimationFrame(frame)
      frame = window.requestAnimationFrame(() => {
        const forceCurrent = pendingForce
        pendingForce = false
        publish(forceCurrent)
      })
    }

    schedule()
    forceCapsuleHitRegionPublishRef.current = () => schedule(true)
    const scheduleNormal = () => schedule()
    const resizeObserver = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(scheduleNormal)
    ;[
      shellRef.current,
      capsuleRef.current,
      closePromptRef.current,
    ].forEach((element) => {
      if (element) resizeObserver?.observe(element)
    })
    const shellElement = shellRef.current
    const mutationObserver = typeof MutationObserver === 'undefined' || !shellElement
      ? null
      : new MutationObserver(scheduleNormal)
    if (shellElement) mutationObserver?.observe(shellElement, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['class', 'style'],
    })
    window.addEventListener('resize', scheduleNormal)

    return () => {
      disposed = true
      forceCapsuleHitRegionPublishRef.current = null
      if (frame) window.cancelAnimationFrame(frame)
      resizeObserver?.disconnect()
      mutationObserver?.disconnect()
      window.removeEventListener('resize', scheduleNormal)
    }
  }, [capsuleDockSide, capsuleExpanded, capsuleExpandedHeight, closePromptOpen, isFloatingPanelWindow, isSettingsWindow])

  useEffect(() => {
    if (isSettingsWindow || isFloatingPanelWindow) return
    let disposed = false
    let settleTimer = 0
    let forceTimer = 0

    const clearTimers = () => {
      if (settleTimer) window.clearTimeout(settleTimer)
      if (forceTimer) window.clearTimeout(forceTimer)
      settleTimer = 0
      forceTimer = 0
    }
    const stabilize = (reason: string) => {
      const now = Date.now()
      if (reason !== 'pointer-end' && now < capsuleIgnoreMoveEventsUntilRef.current) return
      if (capsuleProgrammaticMoveRef.current && reason !== 'pointer-end') return
      if (now - capsuleLastDragStabilizedAtRef.current < 220) return
      capsuleLastDragStabilizedAtRef.current = now
      clearTimers()
      const settleDelay = 90
      settleTimer = window.setTimeout(() => {
        if (disposed) return
        capsuleProgrammaticMoveRef.current = true
        capsuleIgnoreMoveEventsUntilRef.current = Date.now() + 700
        void logClientEvent('capsule-window', 'stabilize', {reason, expanded: capsuleExpanded, compact: capsuleWindowCompact})
        const snapBeforeLayout = !capsuleExpanded
        const snapTask = snapBeforeLayout
          ? snapCapsuleWindowToEdge(capsuleWindowCompact)
          : Promise.resolve(capsuleDockSideRef.current)
        void snapTask
          .then((dockSide) => {
            if (!disposed) setCapsuleDockSide(dockSide)
          })
          .then(() => setCapsuleWindowExpanded(capsuleExpanded, capsuleExpandedHeight, 'auto', capsuleWindowCompact))
          .then((direction) => {
            if (!disposed) setCapsuleExpandDirection(direction)
          })
          .finally(() => {
            if (disposed) return
            forceCapsuleHitRegionPublishRef.current?.()
            forceTimer = window.setTimeout(() => {
              if (!disposed) {
                capsuleProgrammaticMoveRef.current = false
                forceCapsuleHitRegionPublishRef.current?.()
              }
            }, 140)
          })
      }, settleDelay)
    }
    const isCapsuleDragTarget = (event: PointerEvent) => {
      const target = event.target
      if (!(target instanceof Element)) return false
      if (target.closest('.grabber')) return true
      return Boolean(target.closest('.capsule')) && !Boolean(target.closest('button, select, input, textarea, label, [role="button"], [role="option"], .select-menu-list'))
    }
    const onPointerDown = (event: PointerEvent) => {
      const dragCandidate = isCapsuleDragTarget(event)
      capsuleDragCandidateRef.current = dragCandidate
      capsuleDragObservedMoveRef.current = false
      capsuleDragStartPointRef.current = dragCandidate ? {x: event.clientX, y: event.clientY} : null
      capsuleDragPendingUntilRef.current = 0
    }
    const onPointerMove = (event: PointerEvent) => {
      if (!capsuleDragCandidateRef.current || capsuleDragObservedMoveRef.current) return
      const startPoint = capsuleDragStartPointRef.current
      if (!startPoint) return
      if (Math.hypot(event.clientX - startPoint.x, event.clientY - startPoint.y) >= 4) {
        capsuleDragObservedMoveRef.current = true
      }
    }
    const onPointerEnd = () => {
      if (!capsuleDragCandidateRef.current) return
      const shouldStabilize = capsuleDragObservedMoveRef.current
      capsuleDragCandidateRef.current = false
      capsuleDragObservedMoveRef.current = false
      capsuleDragStartPointRef.current = null
      if (shouldStabilize) {
        capsuleDragPendingUntilRef.current = 0
        stabilize('pointer-end')
        return
      }
      capsuleDragPendingUntilRef.current = Date.now() + 420
    }

    const unsubscribeMoveEnded = subscribeCapsuleWindowMoveEnded((reason) => {
      const dragPending = Date.now() <= capsuleDragPendingUntilRef.current
      if (reason === 'window-did-move') {
        if (capsuleDragCandidateRef.current || dragPending) capsuleDragObservedMoveRef.current = true
        return
      }
      if (!capsuleDragCandidateRef.current && !capsuleDragObservedMoveRef.current) return
      capsuleDragCandidateRef.current = false
      capsuleDragObservedMoveRef.current = false
      capsuleDragStartPointRef.current = null
      capsuleDragPendingUntilRef.current = 0
      stabilize(reason)
    })
    document.addEventListener('pointerdown', onPointerDown, true)
    document.addEventListener('pointermove', onPointerMove, true)
    document.addEventListener('pointerup', onPointerEnd, true)
    document.addEventListener('pointercancel', onPointerEnd, true)

    return () => {
      disposed = true
      clearTimers()
      unsubscribeMoveEnded()
      document.removeEventListener('pointerdown', onPointerDown, true)
      document.removeEventListener('pointermove', onPointerMove, true)
      document.removeEventListener('pointerup', onPointerEnd, true)
      document.removeEventListener('pointercancel', onPointerEnd, true)
      capsuleDragCandidateRef.current = false
      capsuleDragObservedMoveRef.current = false
      capsuleDragStartPointRef.current = null
      capsuleDragPendingUntilRef.current = 0
    }
  }, [capsuleExpanded, capsuleExpandedHeight, capsuleWindowCompact, isFloatingPanelWindow, isSettingsWindow])

  useEffect(() => {
    if (recordingMode === 'audio' && activePanel === 'camera') {
      setActivePanel(null)
    }
  }, [activePanel, recordingMode])

  useEffect(() => {
    if (isSettingsWindow) return
    if (!camera || recordingMode === 'audio') {
      stopCameraPreview(!camera ? 'camera-disabled' : 'audio-mode')
    }
  }, [camera, isSettingsWindow, recordingMode])

  useEffect(() => {
    if (isRecording) return
    if (isSettingsWindow || !camera || recordingMode !== 'video' || !hasUsableCamera) return

    const generation = cameraPreviewGenerationRef.current
    void logClientEvent('camera', 'preview-show-request', {
      generation,
      preset: ensureVisiblePipConfig(currentPipConfig).preset,
      selectedCamera,
      selectedCameraName: selectedCameraDevice?.name ?? '',
    })
    void showPipOverlay(ensureVisiblePipConfig(currentPipConfig), 'edit', pipCameraTarget)
      .then(() => {
        if (generation !== cameraPreviewGenerationRef.current || !cameraRef.current || recordingMode !== 'video') {
          void logClientEvent('camera', 'preview-show-cancelled-after-show', {
            generation,
            currentGeneration: cameraPreviewGenerationRef.current,
            camera: cameraRef.current,
            recordingMode,
          })
          void hidePipOverlay()
        }
      })
      .catch((error) => {
        void logClientEvent('camera', 'preview-show-failed', {error: readableError(error)})
        console.info('PIP camera preview unavailable:', error)
      })
  }, [camera, currentPipConfig, hasUsableCamera, isRecording, isSettingsWindow, pipCameraTarget, recordingMode])

  useEffect(() => {
    if (!recordingConfigLocked) return
    if (activePanel === 'source' || activePanel === 'audio' || activePanel === 'camera' || activePanel === 'board') {
      setActivePanel(null)
    }
  }, [activePanel, recordingConfigLocked])

  useEffect(() => {
    if (isSettingsWindow) return
    const shouldMonitor = activePanel === 'audio' &&
      microphone &&
      !isRecording &&
      selectedMic !== '' &&
      selectedMicrophoneDevice?.available !== false &&
      hasAvailableMicrophone
    if (!shouldMonitor) {
      setMicMonitorActive(false)
      setMicLevel(0)
      setMicPeak(0)
      void stopMicrophoneLevelMonitor()
      return
    }

    let cancelled = false
    setMicMonitorError(null)
    void startMicrophoneLevelMonitor(selectedMic)
      .then(() => {
        if (!cancelled) setMicMonitorActive(true)
      })
      .catch((error) => {
        if (cancelled) return
        setMicMonitorError(readableError(error))
        setMicMonitorActive(false)
        setMicLevel(0)
        setMicPeak(0)
      })
    return () => {
      cancelled = true
      void stopMicrophoneLevelMonitor()
    }
  }, [activePanel, hasAvailableMicrophone, isRecording, isSettingsWindow, microphone, selectedMic, selectedMicrophoneDevice?.available])

  useEffect(() => {
    let cancelled = false
    loadBootstrap()
      .then((bootstrap) => {
        if (cancelled) return
        const nextSources = bootstrap.sources.length > 0 ? bootstrap.sources : sources
        const nextSettings = bootstrap.settings
        setAppData(bootstrap.appData)
        setStorageStatus(bootstrap.storage)
        setStorageRootDraft(bootstrap.appData.rootDir)
        setCapabilities(bootstrap.capabilities)
        setAvailableSources(nextSources)
        setRecoveries(bootstrap.recoveries)
        setState(bootstrap.state as RecordingState)
        setLastBackend(bootstrap.backend || 'ui-preview')
        applySettingsState(nextSettings, bootstrap.media, nextSources)
        setSettingsLoaded(true)
      })
      .catch((error) => {
        console.error('Failed to bootstrap recorder:', error)
        setSettingsLoaded(true)
      })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!settingsLoaded) return
    const saveTimer = window.setTimeout(() => {
      void saveSettings(currentSettings)
        .then((saved) => {
          persistedSettingsRef.current = saved
        })
        .catch((error) => console.error('Failed to save settings:', error))
    }, 300)
    return () => window.clearTimeout(saveTimer)
  }, [settingsAutosaveKey, settingsLoaded])

  useEffect(() => {
    if (state !== 'recording') {
      return
    }
    const timer = window.setInterval(() => setElapsed((value) => value + 1), 1000)
    return () => window.clearInterval(timer)
  }, [state])

  useEffect(() => () => {
    countdownTokenRef.current += 1
    if (countdownTimerRef.current !== null) {
      window.clearTimeout(countdownTimerRef.current)
      countdownTimerRef.current = null
    }
  }, [])

  useEffect(() => subscribeRecordingStatus(applyRecordingStatus), [])

  useEffect(() => subscribeWhiteboardVisibility((event) => {
    setWhiteboardVisibility(event.visible ? event : null)
  }), [])

  useEffect(() => {
    let cancelled = false
    void listScreenshots()
      .then((items) => {
        if (!cancelled) setScreenshots(items)
      })
      .catch((error) => {
        console.error('Failed to load screenshot history:', error)
      })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => subscribeScreenshotCaptured((item) => {
    setScreenshots((current) => [item, ...current.filter((entry) => entry.id !== item.id)].slice(0, 200))
    setScreenshotMessage(copy.screenshot.captured(item.width, item.height))
  }), [copy])

  useEffect(() => subscribeScreenshotHistoryChanged((items) => {
    setScreenshots(items)
  }), [])

  useEffect(() => subscribeOcrJobEvents((event) => {
    if (!event.sourceId) return
    if (event.status === 'queued') {
      setScreenshotMessage(copy.screenshot.ocrQueued)
      return
    }
    if (event.status === 'running') {
      setScreenshotMessage(copy.screenshot.ocrStatusRunning)
      return
    }
    if (event.status === 'ready') {
      setScreenshotMessage(copy.screenshot.ocrStatusReady)
      return
    }
    if (event.status === 'failed') {
      setScreenshotMessage(event.error || copy.screenshot.ocrStatusFailed)
    }
  }), [copy])

  useEffect(() => subscribeAudioState((audio) => {
    void logClientEvent('audio', 'state', {
      system: audio.system,
      systemDeviceId: audio.systemDeviceId ?? '',
      microphone: audio.microphone,
      microphoneDeviceId: audio.microphoneDeviceId ?? '',
      noiseSuppression: audio.noiseSuppression,
    })
    applyAudioControlState(audio)
  }), [])

  useEffect(() => subscribeSettingsChanged((settings) => {
    const incomingCameraOff = !settings.camera.enabled
    const preservePreferences = hasLocalPreferenceIntent()
    const preserveWhiteboard = hasLocalWhiteboardIntent()
    const preserveAudioEnabled = !isSettingsWindow && hasLocalAudioIntent()
    const preserveCameraEnabled = !isSettingsWindow && hasLocalCameraIntent()
    const preserveAudioSelection = preserveAudioEnabled
    const preservePipConfig = !isSettingsWindow && hasLocalPipIntent()
    void logClientEvent('settings', 'changed', {
      window: isSettingsWindow ? 'settings' : 'recorder',
      systemAudio: settings.audio.system,
      microphone: settings.audio.microphone,
      noiseSuppression: settings.audio.noiseSuppression,
      recordingQuality: settings.recording.quality,
      recordingFps: settings.recording.fps,
      theme: settings.window.theme,
      autoOcr: settings.ocr.autoRecognizeScreenshots,
      whiteboardCapturePolicy: settings.whiteboard.capturePolicy,
      preservePreferences,
      preserveWhiteboard,
      preserveAudioEnabled,
      preserveAudioSelection,
      cameraEnabled: settings.camera.enabled,
      currentCamera: cameraRef.current,
      pipPreset: settings.camera.pipPreset,
      preserveCameraEnabled,
      preservePipConfig,
    })
    if (!isSettingsWindow && incomingCameraOff && cameraRef.current && !preserveCameraEnabled) {
      stopCameraPreview('settings-camera-off')
    }
    applySettingsState(settings, undefined, undefined, {
      preserveRecordingSettings: preservePreferences,
      preserveTheme: preservePreferences,
      preserveOcr: preservePreferences,
      preserveAudioEnabled,
      preserveAudioSelection,
      preserveCameraEnabled,
      preserveCameraSelection: preserveCameraEnabled,
      preservePipConfig,
      preserveWhiteboard,
    })
  }), [isSettingsWindow])

  useEffect(() => subscribeRegionSelection((result) => {
    if (result.cancelled) {
      void hideRegionFrame()
      setAvailableSources((current) => {
        const next = current.filter((source) => source.id !== 'region:custom')
        setSelectedSource((selected) => selected.type === 'region' ? fallbackVisibleSource(next) : selected)
        return next
      })
      setSourceSelectionMessage(result.error ? {key: 'regionTooSmall', fallback: result.error} : {key: 'regionCancelled'})
      return
    }
    const pickedSource = result.source
    if (!pickedSource) return
    setAvailableSources((current) => {
      const next = current.filter((source) => !(source.id === pickedSource.id && source.type === pickedSource.type))
      return [pickedSource, ...next]
    })
    setSelectedSource(pickedSource)
    void patchSourceState({
      recordingMode: 'video',
      sourceId: pickedSource.id,
      sourceType: pickedSource.type,
      sourceGeometry: {
        x: result.geometry?.x ?? pickedSource.x ?? 0,
        y: result.geometry?.y ?? pickedSource.y ?? 0,
        width: result.geometry?.width ?? pickedSource.width ?? 0,
        height: result.geometry?.height ?? pickedSource.height ?? 0,
        displayIndex: pickedSource.displayIndex,
        nativeId: pickedSource.nativeId,
      },
    })
    setSourceSelectionMessage({
      key: 'regionSelected',
      width: result.geometry?.width ?? pickedSource.width,
      height: result.geometry?.height ?? pickedSource.height,
    })
  }), [])

  useEffect(() => {
    if (activePanel !== 'source') {
      setSourcePickerView('overview')
      void hideScreenIndicator()
    }
  }, [activePanel])

  useEffect(() => {
    const inlinePanelOpen = !isWailsDesktopRuntime() && (activePanel || settingsOpen)
    if (isSettingsWindow || isFloatingPanelWindow || (!inlinePanelOpen && !closePromptOpen)) return
    const pointerInsideFloatingPanels = (event: Event) => (
      eventPathContains(event, capsuleRef.current) ||
      eventPathContains(event, popoverRef.current) ||
      eventPathContains(event, settingsPanelRef.current) ||
      eventPathContains(event, closePromptRef.current)
    )
    const markFloatingPointer = (inside: boolean) => {
      floatingPointerInsideRef.current = inside
      if (inside) floatingPointerInsideAtRef.current = Date.now()
    }
    const closeFloatingPanels = () => {
      setActivePanel(null)
      setSettingsOpen(false)
      setClosePromptOpen(false)
      void hideScreenIndicator()
    }
    const onPointerDown = (event: PointerEvent) => {
      if (pointerInsideFloatingPanels(event)) {
        markFloatingPointer(true)
        return
      }
      markFloatingPointer(false)
      closeFloatingPanels()
    }
    const onPointerMove = (event: PointerEvent) => {
      markFloatingPointer(pointerInsideFloatingPanels(event))
    }
    const onWindowBlur = () => {
      window.setTimeout(() => {
        if (floatingPointerInsideRef.current || Date.now() - floatingPointerInsideAtRef.current < 650) return
        closeFloatingPanels()
      }, 120)
    }
    const onVisibilityChange = () => {
      if (document.visibilityState !== 'visible' && !floatingPointerInsideRef.current) closeFloatingPanels()
    }
    document.addEventListener('pointerdown', onPointerDown, true)
    document.addEventListener('pointermove', onPointerMove, true)
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('blur', onWindowBlur)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown, true)
      document.removeEventListener('pointermove', onPointerMove, true)
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.removeEventListener('blur', onWindowBlur)
      floatingPointerInsideRef.current = false
    }
  }, [activePanel, closePromptOpen, isFloatingPanelWindow, isSettingsWindow, settingsOpen])

  const statusLabel = useMemo(() => {
    return copy.statusChips[state] ?? copy.statusChips.idle
  }, [copy, state])
  const timeChipLabel = countdownRemaining > 0 && state === 'preparing' ? copy.settings.countdown : statusLabel
  const timeChipValue = countdownRemaining > 0 && state === 'preparing' ? formatTime(countdownRemaining) : formatTime(elapsed)

  const currentRecordingProfile = () => ({
    quality: recordingQuality,
    fps: recordingFPS,
    captureCursor,
    countdownSeconds,
  })

  const buildAudioOnlyRequest = (): AudioOnlyRecordingRequest => ({
    recording: currentRecordingProfile(),
    systemAudio,
    systemAudioDeviceId: selectedSystemAudio || undefined,
    microphone,
    microphoneDeviceId: selectedMic || undefined,
    noiseSuppression,
  })

  const buildVideoRequest = (): MockRecordingRequest => {
    const requestedCameraId = selectedCameraRef.current || selectedCamera
    const requestCameraDevice = availableCameras.find((device) => device.id === requestedCameraId && isUsableCameraDevice(device)) ??
      fallbackUsableCameraDevice ??
      selectedCameraDevice
    const requestCameraEnabled = cameraRef.current && recordingMode === 'video' && (requestCameraDevice ? isUsableCameraDevice(requestCameraDevice) : false)
    const requestPip = requestCameraEnabled
      ? ensureVisiblePipConfig(currentPipConfig)
      : normalizePipConfig({...currentPipConfig, preset: 'off'}, 'off')
    return {
      source: selectedSource,
      recording: currentRecordingProfile(),
      systemAudio,
      systemAudioDeviceId: selectedSystemAudio || undefined,
      microphone,
      microphoneDeviceId: selectedMic || undefined,
      noiseSuppression,
      camera: requestCameraEnabled,
      cameraDeviceId: requestCameraDevice?.id ?? requestedCameraId,
      cameraDeviceNativeId: requestCameraDevice?.nativeId,
      pipPreset: requestCameraEnabled ? requestPip.preset : 'off',
      pip: requestPip,
    }
  }

  const runCurrentPreflight = async () => {
    if (isRecording || preflightBusy) return null
    setPreflightBusy(true)
    try {
      const preflight = recordingMode === 'audio'
        ? await preflightAudioOnlyRecording(buildAudioOnlyRequest())
        : await preflightRecording(buildVideoRequest())
      setLastPreflight(preflight)
      setLastBackend(preflight.backend || lastBackend)
      if (preflight.status === 'blocked') {
        setLastStatusMessage({key: 'preflightBlocked'})
      }
      return preflight
    } catch (error) {
      console.error('Failed to run preflight:', error)
      setLastStatusMessage({key: 'failedToStart'})
      return null
    } finally {
      setPreflightBusy(false)
    }
  }

  const clearCountdownTimer = () => {
    if (countdownTimerRef.current !== null) {
      window.clearTimeout(countdownTimerRef.current)
      countdownTimerRef.current = null
    }
  }

  const cancelCountdown = () => {
    countdownTokenRef.current += 1
    clearCountdownTimer()
    setCountdownRemaining(0)
  }

  const waitForCountdown = async (seconds: number) => {
    const total = Math.max(0, Math.trunc(seconds))
    cancelCountdown()
    if (total <= 0) return
    const token = countdownTokenRef.current + 1
    countdownTokenRef.current = token
    for (let remaining = total; remaining > 0; remaining -= 1) {
      if (countdownTokenRef.current !== token) throw new Error('countdown cancelled')
      setCountdownRemaining(remaining)
      await new Promise<void>((resolve) => {
        countdownTimerRef.current = window.setTimeout(resolve, 1000)
      })
      countdownTimerRef.current = null
    }
    if (countdownTokenRef.current === token) {
      setCountdownRemaining(0)
    }
  }

  const beginRecording = async () => {
    setActivePanel(null)
    setSettingsOpen(false)
    void hideFloatingPanel()
    setElapsed(0)
    setState('preparing')
    try {
      if (recordingMode === 'audio') {
        const request = buildAudioOnlyRequest()
        const preflight = await preflightAudioOnlyRecording(request)
        setLastPreflight(preflight)
        setLastBackend(preflight.backend || lastBackend)
        if (preflight.status === 'blocked') {
          setState('failed')
          setCountdownRemaining(0)
          setLastStatusMessage({key: 'preflightBlocked'})
          return
        }
        await waitForCountdown(request.recording.countdownSeconds)
        const session = await startAudioOnlyRecording(request)
        applyRecordingStatus({
          status: session.status ?? 'recording',
          message: 'Audio-only recording started',
          backend: session.backend,
          session,
        })
        return
      }

      const request = buildVideoRequest()
      void logClientEvent('recording', 'start-request-built', {
        sourceType: request.source.type,
        cameraEnabled: request.camera,
        cameraDeviceId: request.cameraDeviceId ?? '',
        cameraNativeId: request.cameraDeviceNativeId ?? '',
        cameraPipPreset: request.pipPreset,
        microphoneEnabled: request.microphone,
        systemAudio: request.systemAudio,
      })
      const preflight = await preflightRecording(request)
      setLastPreflight(preflight)
      setLastBackend(preflight.backend || lastBackend)
      if (preflight.status === 'blocked') {
        setState('failed')
        setCountdownRemaining(0)
        setLastStatusMessage({key: 'preflightBlocked'})
        return
      }
      await waitForCountdown(request.recording.countdownSeconds)
      const session = await startRecording(request)
      applyRecordingStatus({
        status: session.status ?? 'recording',
        message: 'Recording started',
        backend: session.backend,
        session,
      })
    } catch (error) {
      console.error('Failed to start recording:', error)
      setLastStatusMessage({key: 'failedToStart'})
      setCountdownRemaining(0)
      setState('failed')
    }
  }

  const finishRecording = async () => {
    cancelCountdown()
    setState('stopping')
    setLastStatusMessage({key: 'finalizing'})
    try {
      const session = await stopRecording()
      if (session) {
        applyRecordingStatus({
          status: session.status ?? 'ready',
          message: 'Recording package ready',
          backend: session.backend,
          session,
        })
        if (shouldAutoExportPipRecording(session)) {
          await autoExportPipRecording(session.packagePath)
        }
      } else {
        setState('ready')
        setLastStatusMessage({key: 'ready'})
      }
      await hideRegionFrame()
      await restoreCapsuleWindow()
      return true
    } catch (error) {
      console.error('Failed to stop recording:', error)
      setLastStatusMessage({key: 'failedToStop'})
      setState('failed')
      await hideRegionFrame().catch((hideError) => console.info('Region frame cleanup failed:', hideError))
      await restoreCapsuleWindow()
      return false
    }
  }

  const shouldAutoExportPipRecording = (session: {packagePath?: string; recordingMode?: string}) => {
    return Boolean(
      session.packagePath &&
      recordingMode === 'video' &&
      session.recordingMode !== 'audio-only' &&
      camera &&
      currentPipConfig.preset !== 'off',
    )
  }

  const autoExportPipRecording = async (packagePath: string) => {
    setExportBusy(true)
    setExportMessage(null)
    setLastStatusMessage({key: 'exportingPip'})
    try {
      const result = await exportRecordingPackage(packagePath, {includeAnnotations: includeAnnotationsInExport})
      setExportMessage({key: 'ready', path: result.outputPath})
      setLastStatusMessage({key: result.pipVisible ? 'pipReady' : 'ready'})
    } catch (error) {
      console.error('Failed to auto export PIP recording:', error)
      setExportMessage({key: 'failed', fallback: error instanceof Error ? error.message : undefined})
      setLastStatusMessage({key: 'ready'})
    } finally {
      setExportBusy(false)
    }
  }

  const toggleRecord = () => {
    if (state === 'recording' || state === 'paused') {
      finishRecording()
      return
    }
    if (state === 'preparing' || state === 'stopping') {
      return
    }
    void beginRecording()
  }

  const togglePause = () => {
    if (state === 'recording') {
      setLastStatusMessage({key: 'pausing'})
      void pauseRecording().then((session) => {
        if (session) {
          applyRecordingStatus({status: session.status ?? 'paused', message: 'Recording paused', backend: session.backend, session})
          return
        }
        setState('paused')
        setLastStatusMessage({key: 'paused'})
      })
      return
    }
    if (state === 'paused') {
      setLastStatusMessage({key: 'resuming'})
      void resumeRecording().then((session) => {
        if (session) {
          applyRecordingStatus({status: session.status ?? 'recording', message: 'Recording resumed', backend: session.backend, session})
          return
        }
        setState('recording')
        setLastStatusMessage({key: 'resumed'})
      })
    }
  }

  const recoverPackages = async () => {
    if (recoverableRecoveries.length === 0 || recoveryBusy) return
    setRecoveryBusy(true)
    setRecoveryMessage({key: 'recovering'})
    try {
      let recovered = 0
      for (const recovery of recoverableRecoveries) {
        const result = await recoverRecordingPackage(recovery.packagePath)
        if (result?.status === 'ready') recovered += 1
      }
      const bootstrap = await loadBootstrap()
      setRecoveries(bootstrap.recoveries)
      setStorageStatus(bootstrap.storage)
      setRecoveryMessage(recovered > 0 ? {key: 'recovered', count: recovered} : {key: 'refreshed'})
    } catch (error) {
      console.error('Failed to recover packages:', error)
      setRecoveryMessage({key: 'failed'})
    } finally {
      setRecoveryBusy(false)
    }
  }

  const applyDataRoot = async () => {
    const nextRoot = storageRootDraft.trim()
    if (!nextRoot || storageBusy) return
    setStorageBusy(true)
    setStorageMessage({key: 'applying'})
    try {
      const info = await setDataRoot(nextRoot)
      setAppData(info)
      setStorageRootDraft(info.rootDir)
      const bootstrap = await loadBootstrap()
      setAppData(bootstrap.appData)
      setStorageStatus(bootstrap.storage)
      setStorageRootDraft(bootstrap.appData.rootDir)
      setRecoveries(bootstrap.recoveries)
      setCapabilities(bootstrap.capabilities)
      setLastBackend(bootstrap.backend || lastBackend)
      setStorageMessage({key: 'changed', path: bootstrap.appData.videoDir})
    } catch (error) {
      console.error('Failed to change data root:', error)
      setStorageMessage({key: 'failed'})
    } finally {
      setStorageBusy(false)
    }
  }

  const openRecordingsDirectory = async () => {
    setStorageMessage(null)
    try {
      const info = await openVideoDirectory()
      setAppData(info)
      setStorageRootDraft(info.rootDir)
    } catch (error) {
      console.error('Failed to open recordings directory:', error)
      setStorageMessage({key: 'failed'})
    }
  }

  const openLastRecordingPackage = async () => {
    if (!canOpenLastPackage) return
    try {
      const summary = await openRecordingPackage(lastPackage)
      setLastPackage(summary.packagePath)
    } catch (error) {
      console.error('Failed to open recording package:', error)
    }
  }

  const exportLastRecordingPackage = async () => {
    if (!canOpenLastPackage || exportBusy || isRecording) return
    setExportBusy(true)
    setExportMessage(null)
    try {
      const result = await exportRecordingPackage(lastPackage, {includeAnnotations: includeAnnotationsInExport})
      setExportMessage({key: 'ready', path: result.outputPath})
    } catch (error) {
      console.error('Failed to export recording package:', error)
      setExportMessage({key: 'failed', fallback: error instanceof Error ? error.message : undefined})
    } finally {
      setExportBusy(false)
    }
  }

  const openFloatingPanelFromAnchor = async (panel: FloatingPanelKind, anchorElement: Element, contextId = '') => {
    const size = floatingPanelSizes[panel]
    const token = floatingPanelTokenRef.current + 1
    floatingPanelTokenRef.current = token
    const placement = await resolveFloatingPanelPlacement(anchorElement, {
      dockSide: capsuleDockSideRef.current,
      width: size.width,
      height: size.height,
      maxHeight: size.maxHeight,
      minWidth: size.minWidth,
    })
    await showFloatingPanel({
      kind: panel,
      anchor: placement.anchor,
      bounds: placement.bounds,
      dockSide: capsuleDockSideRef.current,
      width: placement.bounds.width,
      height: placement.bounds.height,
      minWidth: size.minWidth,
      maxHeight: size.maxHeight,
      token,
      screenId: placement.screenId,
      direction: placement.direction,
      contextId,
    })
  }

  const togglePanel = (panel: ActivePanel, anchorElement?: Element) => {
    setSettingsOpen(false)
    setClosePromptOpen(false)
    if (isWailsDesktopRuntime() && anchorElement) {
      if (activePanel === panel) {
        void hideFloatingPanel()
        return
      }
      void openFloatingPanelFromAnchor(panel, anchorElement)
      return
    }
    setActivePanel(activePanel === panel ? null : panel)
  }

  const openSettings = (anchorElement?: Element) => {
    setActivePanel(null)
    setClosePromptOpen(false)
    if (isWailsDesktopRuntime() && anchorElement) {
      if (settingsOpen) {
        void hideFloatingPanel()
        return
      }
      void openFloatingPanelFromAnchor('settings', anchorElement)
      return
    }
    setSettingsOpen((open) => !open)
  }

  const requestCloseApplication = (anchorElement?: Element) => {
    setActivePanel(null)
    setSettingsOpen(false)
    if (isWailsDesktopRuntime() && anchorElement) {
      setClosePromptOpen(false)
      void openFloatingPanelFromAnchor('close', anchorElement)
      return
    }
    void hideFloatingPanel()
    setClosePromptOpen(true)
  }

  const openRecordingWhiteboard = async () => {
    try {
      await showAnnotationOverlay()
      return
    } catch (error) {
      const message = readableError(error)
      if (!message.includes('selected annotation region')) {
        throw error
      }
      void logClientEvent('whiteboard', 'annotation-region-required', {
        state,
        recordingMode,
      }, message)
    }
    await showAnnotationRegionSelector()
  }

  const openWhiteboard = () => {
    setActivePanel(null)
    setSettingsOpen(false)
    setClosePromptOpen(false)
    void hideFloatingPanel()
    const launchMode = whiteboardLaunchMode(state, recordingMode)
    void (async () => {
      if (launchMode === 'whiteboard') {
        await showWhiteboardWindow()
        return
      }
      try {
        await openRecordingWhiteboard()
      } catch (error) {
        console.error('Failed to open recording annotation overlay:', error)
        void logClientEvent('whiteboard', 'annotation-open-fallback', {
          state,
          recordingMode,
          message: error instanceof Error ? error.message : String(error),
        })
        await showWhiteboardWindow()
      }
    })()
  }

  const openBoardTools = (anchorElement?: Element) => {
    if (isRecording) {
      openWhiteboard()
      return
    }
    togglePanel('board', anchorElement)
  }

  const beginScreenshotCapture = () => {
    if (isRecording) return
    setActivePanel(null)
    setSettingsOpen(false)
    setClosePromptOpen(false)
    void hideFloatingPanel()
    setScreenshotMessage(copy.screenshot.selecting)
    void showScreenshotRegionSelector()
      .catch((error) => {
        const message = readableError(error)
        console.error('Failed to show screenshot selector:', error)
        setScreenshotMessage(message || copy.screenshot.captureFailed)
      })
  }

  const beginScrollingScreenshot = () => {
    if (isRecording) return
    void hideFloatingPanel()
    setScreenshotMessage(copy.screenshot.scrollingPreparing)
    void startScrollingScreenshot()
      .then(() => setScreenshotMessage(copy.screenshot.scrollingStarted))
      .catch((error) => {
        const message = readableError(error)
        console.error('Failed to start scrolling screenshot:', error)
        setScreenshotMessage(message || copy.screenshot.scrollingUnavailable)
      })
  }

  const captureScreenshotMode = (mode: 'full' | 'screen' | 'window' | 'focused-window') => {
    if (isRecording) return
    setActivePanel(null)
    setSettingsOpen(false)
    setClosePromptOpen(false)
    void hideFloatingPanel()
    void captureScreenshot({mode})
      .then((item) => {
        setScreenshots((current) => [item, ...current.filter((entry) => entry.id !== item.id)].slice(0, 200))
        setScreenshotMessage(copy.screenshot.captured(item.width, item.height))
      })
      .catch((error) => {
        const message = readableError(error)
        console.error('Failed to capture screenshot:', error)
        setScreenshotMessage(message || copy.screenshot.captureFailed)
      })
  }

  const openScreenshotPreview = (item: ScreenshotItem) => {
    void showPinnedScreenshot(item.id)
      .catch((error) => {
        console.error('Failed to preview screenshot:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const openScreenshotFolder = (item: ScreenshotItem) => {
    void openScreenshotDirectory(item.id)
      .then((opened) => {
        if (opened) setScreenshotMessage(copy.screenshot.openFolder)
      })
      .catch((error) => {
        console.error('Failed to open screenshot directory:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const annotateScreenshot = (item: ScreenshotItem) => {
    setActivePanel(null)
    void hideFloatingPanel(0)
    void openScreenshotInWhiteboard(item.id)
      .catch((error) => {
        console.error('Failed to open screenshot in whiteboard:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const pinScreenshot = (item: ScreenshotItem) => {
    void showPinnedScreenshot(item.id)
      .then(() => listScreenshots())
      .then(setScreenshots)
      .catch((error) => {
        console.error('Failed to pin screenshot:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const deleteScreenshot = (item: ScreenshotItem) => {
    void deleteScreenshotItem(item.id)
      .then((items) => {
        setScreenshots(items)
        setScreenshotMessage(copy.screenshot.deleted)
      })
      .catch((error) => {
        console.error('Failed to delete screenshot:', error)
        setScreenshotMessage(readableError(error) || copy.screenshot.deleteFailed)
      })
  }

  const toggleScreenshotFixed = (item: ScreenshotItem) => {
    void patchScreenshotItem(item.id, {fixed: !item.fixed})
      .then(setScreenshots)
      .then(() => {
        if (!item.fixed) return showPinnedScreenshot(item.id).then(() => undefined)
        return undefined
      })
      .catch((error) => {
        console.error('Failed to toggle screenshot fixed state:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const queueScreenshotOcr = (item: ScreenshotItem) => {
    if (item.ocrStatus === 'queued' || item.ocrStatus === 'running') return
    setScreenshotMessage(copy.screenshot.ocrQueued)
    void queueRecognizeScreenshot(item.id)
      .catch((error) => {
        const message = readableError(error)
        console.error('Failed to queue screenshot OCR:', error)
        setScreenshotMessage(message || copy.screenshot.ocrStatusFailed)
      })
  }

  const copyScreenshotOcrText = (item: ScreenshotItem) => {
    if (!item.ocrResultId || item.ocrStatus !== 'ready') {
      queueScreenshotOcr(item)
      return
    }
    void openOcrResult(item.ocrResultId)
      .then((result) => copyOcrResultText(result, copy))
      .then((message) => setScreenshotMessage(message))
      .catch((error) => {
        console.error('Failed to copy screenshot OCR text:', error)
        setScreenshotMessage(readableError(error) || copy.screenshot.copyTextEmpty)
      })
  }

  const translateScreenshotOcrText = (item: ScreenshotItem, anchorElement?: Element) => {
    if (!item.ocrResultId || item.ocrStatus !== 'ready') {
      queueScreenshotOcr(item)
      return
    }
    if (shouldUseFloatingPanelWindows() && anchorElement) {
      void openFloatingPanelFromAnchor('ocr-result', anchorElement, ocrPanelContext(item.ocrResultId, true))
      return
    }
    setScreenshotMessage(copy.screenshot.translationWorking)
    void translateAndCopyOcrResultText(item.ocrResultId, ocrTranslation, copy)
      .then(setScreenshotMessage)
      .catch((error) => {
        console.error('Failed to translate screenshot OCR text:', error)
        setScreenshotMessage(`${copy.screenshot.translationFailed}: ${readableError(error)}`)
      })
  }

  const toggleCameraFromShortcut = () => {
    if (recordingConfigLocked || recordingMode !== 'video') {
      void logClientEvent('shortcuts', 'camera-ignored', {
        recordingConfigLocked,
        recordingMode,
      })
      return
    }
    setActivePanel(null)
    setSettingsOpen(false)
    setClosePromptOpen(false)
    setCameraEnabled(!cameraRef.current)
  }

  shortcutActionsRef.current = {
    toggleRecording: toggleRecord,
    togglePause,
    toggleCamera: toggleCameraFromShortcut,
    openWhiteboard,
    openScreenshot: beginScreenshotCapture,
  }

  useEffect(() => subscribeShortcutTriggered((event) => {
    if (isSettingsWindow || isFloatingPanelWindow || shortcutCaptureRef.current) return
    shortcutActionsRef.current[event.action]?.()
  }), [isFloatingPanelWindow, isSettingsWindow])

  const confirmCloseApplication = async () => {
    if (closeBusy) return
    setCloseBusy(true)
    try {
      if (state === 'recording' || state === 'paused') {
        const stopped = await finishRecording()
        if (!stopped) return
      }
      await hideRegionFrame()
      await hideScreenIndicator()
      await stopMicrophoneLevelMonitor()
      await quitApplication()
    } finally {
      setCloseBusy(false)
    }
  }

  const chooseSource = (source: CaptureSource) => {
    if (recordingConfigLocked) return
    void hideScreenIndicator()
    if (source.type !== 'region') {
      void hideRegionFrame()
    }
    setSelectedSource(source)
    setSourceSelectionMessage(source.available === false ? {key: 'sourceQueued'} : null)
    setActivePanel(null)
    void patchSourceState({
      recordingMode,
      sourceId: source.id,
      sourceType: source.type,
      sourceGeometry: source.type === 'region' && source.width && source.height
        ? {
            x: source.x ?? 0,
            y: source.y ?? 0,
            width: source.width,
            height: source.height,
            displayIndex: source.displayIndex,
            nativeId: source.nativeId,
          }
        : undefined,
      clearGeometry: source.type !== 'region',
    })
    void hideFloatingPanel()
    setSourcePickerView('overview')
  }

  const chooseRegion = async (source: CaptureSource) => {
    if (recordingConfigLocked) return
    await hideScreenIndicator()
    setActivePanel(null)
    void hideFloatingPanel()
    setSourcePickerView('overview')
    setSourceSelectionMessage({key: 'regionSelecting'})
    try {
      await showRegionSelector()
    } catch (error) {
      console.error('Failed to show region selector:', error)
      setSourceSelectionMessage({key: 'sourceQueued'})
    }
  }

  const showScreenMarker = (source: CaptureSource) => {
    if (source.type !== 'screen') return
    void showScreenIndicator(source.id)
  }

  const hideScreenMarker = () => {
    void hideScreenIndicator()
  }

  const closeSettings = () => {
    if (isSettingsWindow) {
      void hideSettingsWindow()
      return
    }
    setSettingsOpen(false)
  }

  const settingsPanel = (
    <section ref={settingsPanelRef} className={`settings-panel ${isSettingsWindow ? 'settings-window-panel' : 'settings-sheet'}`} role="dialog" aria-modal={!isSettingsWindow} aria-label={copy.aria.settingsDialog}>
      <div className="sheet-header">
        <div>
          <strong>RecordingFreedom</strong>
          <span>{copy.settings.title}</span>
        </div>
        <button type="button" className="sheet-close" onClick={closeSettings}>{copy.common.close}</button>
      </div>
      <div className="settings-list">
        <SettingLine
          title={copy.settings.storage}
          value={appData.videoDir}
          detail={copy.settings.storageDetail}
          actionLabel={copy.settings.openRecordings}
          onAction={() => void openRecordingsDirectory()}
        />
        <SettingLine
          title={copy.settings.storageHealth}
          value={formatStorageStatusValue(storageStatus, copy)}
          status={storageStatusForBadge(storageStatus.status)}
          statusLabel={copy.capabilityStatusLabels[storageStatusForBadge(storageStatus.status)]}
          detail={storageStatusDetail(storageStatus, copy)}
        />
        <SettingTextAction
          title={copy.settings.dataRoot}
          value={storageRootDraft}
          detail={storageText || copy.settings.dataRootDetail}
          actionLabel={storageBusy ? copy.common.applying : copy.common.apply}
          actionDisabled={storageBusy || isRecording || storageRootDraft.trim() === '' || storageRootDraft.trim() === appData.rootDir}
          onChange={setStorageRootDraft}
          onAction={() => void applyDataRoot()}
        />
        <SettingLine title={copy.settings.appData} value={appData.rootDir} />
        <SettingLine title={copy.settings.settingsFile} value={joinDisplayPath(appData.rootDir, 'settings.json')} />
        <SettingSelect
          title={copy.settings.language}
          value={locale}
          options={localeOptions.map((code) => ({value: code, label: copy.localeNames[code]}))}
          onChange={(value) => setLocale(normalizeLocale(value))}
        />
        <SettingSelect
          title={copy.settings.theme}
          value={theme}
          options={themeOptions.map((code) => ({value: code, label: copy.themeNames[code], swatch: themeSwatches[code]}))}
          detail={copy.settings.themeDetail}
          onChange={(value) => commitSettingsPreferencePatch({theme: normalizeTheme(value)})}
        />
        <SettingToggle
          title={copy.settings.startAtLogin}
          checked={startAtLogin}
          detail={copy.settings.startAtLoginDetail}
          onChange={(value) => commitSettingsPreferencePatch({startAtLogin: value})}
        />
        <SettingToggle
          title={copy.settings.autoOcr}
          checked={autoRecognizeScreenshots}
          detail={copy.settings.autoOcrDetail}
          onChange={(value) => commitSettingsPreferencePatch({autoOcr: value})}
        />
        <OcrTranslationSettingsPanel
          copy={copy}
          translation={currentSettings.ocr.translation}
          onChange={(ocrTranslation) => commitSettingsPreferencePatch({ocrTranslation})}
        />
        <OcrModelSettings copy={copy} />
        <SettingLine
          title={copy.settings.shortcuts}
          value={copy.settings.shortcutSummary}
          detail={shortcutError || copy.settings.shortcutDetail}
        />
        {shortcutActions.map((action) => (
          <SettingShortcut
            key={action}
            title={copy.settings.shortcutActionLabels[action]}
            value={formatShortcutForDisplay(shortcuts[action])}
            detail={shortcutCapture === action ? copy.settings.shortcutHint : undefined}
            actionLabel={shortcutCapture === action ? copy.settings.shortcutRecording : copy.settings.shortcutRecord}
            capturing={shortcutCapture === action}
            onStart={() => {
              setShortcutError('')
              setShortcutCapture(action)
            }}
            onCancel={() => {
              setShortcutCapture(null)
              setShortcutError('')
            }}
            onCapture={(accelerator) => commitShortcutSettingsPatch(action, accelerator)}
          />
        ))}
        <SettingLine
          title={copy.settings.recordingBackend}
          value={lastBackend}
          detail={copy.settings.recordingBackendDetail}
        />
        <SettingLine
          title={copy.settings.preflight}
          value={lastPreflight ? `${copy.preflightLabels[lastPreflight.status]} · ${lastPreflight.backend}` : copy.common.notRun}
          status={lastPreflight ? preflightStatusForBadge(lastPreflight.status) : undefined}
          statusLabel={lastPreflight ? copy.capabilityStatusLabels[preflightStatusForBadge(lastPreflight.status)] : undefined}
          detail={lastPreflight ? preflightDetail(lastPreflight, copy) : copy.settings.preflightPendingDetail}
          actionLabel={preflightBusy ? copy.settings.preflightRunning : (lastPreflight ? copy.settings.preflightRerun : copy.settings.preflightAction)}
          actionDisabled={preflightBusy || isRecording}
          onAction={() => void runCurrentPreflight()}
        />
        <SettingLine
          title={copy.settings.recordingPackage}
          value={lastPackageName}
          detail={copy.settings.packageContentDetail}
          actionLabel={copy.settings.openPackage}
          actionDisabled={!canOpenLastPackage}
          onAction={() => void openLastRecordingPackage()}
        />
        <SettingLine
          title={copy.settings.exportPackage}
          value={exportBusy ? copy.settings.exporting : copy.settings.exportPackageValue}
          detail={exportText || copy.settings.exportPackageDetail}
          actionLabel={exportBusy ? copy.settings.exporting : copy.settings.exportPackage}
          actionDisabled={!canOpenLastPackage || exportBusy || isRecording}
          onAction={() => void exportLastRecordingPackage()}
        />
        <SettingLine
          title={copy.settings.exportPlan}
          value={exportPlanValue}
          detail={exportPlanDetail}
        />
        <ExportPlanTimelinePreview plan={exportPlanPreview} copy={copy} />
        <SettingToggle
          title={copy.settings.includeAnnotations}
          checked={includeAnnotationsInExport}
          detail={copy.settings.includeAnnotationsDetail}
          onChange={(value) => commitWhiteboardSettingsPatch({capturePolicy: annotationCapturePolicy(value)})}
        />
        <SettingSelect
          title={copy.settings.quality}
          value={recordingQuality}
          options={recordingQualityOptions.map((quality) => ({value: quality, label: copy.recordingQualityLabels[quality]}))}
          detail={copy.settings.qualityDetail}
          onChange={(value) => commitSettingsPreferencePatch({recordingQuality: normalizeRecordingQuality(value)})}
        />
        <SettingSelect
          title={copy.settings.fps}
          value={String(recordingFPS)}
          options={fpsOptions.map((fps) => ({value: String(fps), label: `${fps} ${copy.settings.fps}`}))}
          onChange={(value) => commitSettingsPreferencePatch({recordingFps: Number(value)})}
        />
        <SettingToggle title={copy.settings.captureCursor} checked={captureCursor} onChange={(value) => commitSettingsPreferencePatch({captureCursor: value})} />
        <SettingSelect
          title={copy.settings.countdown}
          value={String(countdownSeconds)}
          options={countdownOptions.map((seconds) => ({value: String(seconds), label: seconds === 0 ? copy.common.off : `${seconds}s`}))}
          onChange={(value) => commitSettingsPreferencePatch({countdownSeconds: Number(value)})}
        />
        <SettingLine
          title={copy.settings.recovery}
          value={recoverablePackages > 0 ? copy.common.packageCount(recoverablePackages) : copy.common.clean}
          detail={recoveryText || (recoverablePackages > 0 ? copy.settings.recoveryFoundDetail(recoverablePackages) : copy.settings.recoveryCleanDetail)}
          actionLabel={recoverablePackages > 0 ? (recoveryBusy ? copy.settings.recovering : copy.settings.recover) : undefined}
          actionDisabled={recoveryBusy}
          onAction={recoverPackages}
        />
        <SettingLine title={copy.settings.platform} value={capabilities.platform} />
        {capabilityRows.map((capability) => (
          <SettingLine
            key={capability.id}
            title={capabilityTitle(capability, copy)}
            value={formatCapabilityValue(capability, copy)}
            status={capability.status}
            statusLabel={copy.capabilityStatusLabels[capability.status]}
            detail={capabilityDetail(capability, copy)}
          />
        ))}
        <SettingLine title={copy.settings.release} value="GitHub Actions Windows portable + setup" />
      </div>
    </section>
  )

  if (isSettingsWindow) {
    return (
      <main className="rf-settings-shell" aria-label={copy.aria.settingsShell}>
        {settingsPanel}
      </main>
    )
  }

  return (
    <main ref={shellRef} className={`rf-shell ${capsuleExpanded ? 'is-expanded' : 'is-collapsed'} ${recordingConfigLocked ? 'is-recording-compact' : ''} drop-${capsuleExpandDirection} dock-${capsuleDockSide}`} aria-label={copy.aria.recorderShell}>
      <section className="recorder-stage" aria-label={copy.aria.recorderControls}>
        <div ref={capsuleRef} className={`capsule ${isRecording ? 'capsule-active capsule-compact' : ''}`}>
          <button
            className="grabber"
            type="button"
            aria-label={copy.aria.moveRecorder}
            title={copy.aria.moveRecorder}
          >
            <span />
            <span />
          </button>

          <div className="capsule-config-segment" aria-hidden={recordingConfigLocked}>
            <button
              className="source-pill"
              type="button"
              aria-expanded={activePanel === 'source'}
              disabled={recordingConfigLocked}
              onClick={(event) => togglePanel('source', event.currentTarget)}
            >
              <SourceIcon size={18} />
              <span className="source-text">
                <strong>{sourceTitle}</strong>
                <small>{sourceSubtitle}</small>
              </span>
              <ChevronDown size={14} />
            </button>

            <div className="control-group" aria-label={copy.aria.audioCameraControls}>
              <button
                className={`icon-button ${systemAudio || microphone ? 'is-on' : ''} ${rnnoiseActive ? 'strong' : ''}`}
                type="button"
                aria-label={copy.aria.openAudioSettings}
                title={copy.panels.audio}
                aria-expanded={activePanel === 'audio'}
                disabled={recordingConfigLocked}
                onClick={(event) => togglePanel('audio', event.currentTarget)}
              >
                <Volume2 size={18} />
              </button>
              <button
                className={`icon-button ${recordingMode === 'video' && camera ? 'is-on' : ''}`}
                type="button"
                aria-label={copy.aria.openCameraSettings}
                title={titleWithShortcut(copy.panels.cameraSidecar, 'toggleCamera')}
                disabled={recordingConfigLocked || recordingMode === 'audio'}
                onClick={(event) => togglePanel('camera', event.currentTarget)}
              >
                <Camera size={18} />
              </button>
            </div>
          </div>

          <button
            className={`record-button ${state}`}
            type="button"
            aria-label={state === 'recording' || state === 'paused' ? copy.aria.stopRecording : copy.aria.startRecording}
            title={titleWithShortcut(state === 'recording' || state === 'paused' ? copy.aria.stopRecording : copy.aria.startRecording, 'toggleRecording')}
            onClick={toggleRecord}
          >
            {state === 'recording' || state === 'paused' ? <Square size={20} /> : <CircleDot size={22} />}
          </button>

          <button
            className={`icon-button soft whiteboard-open-button ${whiteboardButtonActive ? 'selected' : ''}`}
            type="button"
            aria-label={isRecording ? copy.aria.openWhiteboard : copy.screenshot.tools}
            aria-pressed={whiteboardButtonActive}
            title={isRecording ? titleWithShortcut(copy.aria.openWhiteboard, 'openWhiteboard') : `${copy.screenshot.tools} · ${formatShortcutForDisplay(shortcuts.openScreenshot)} / ${formatShortcutForDisplay(shortcuts.openWhiteboard)}`}
            aria-expanded={!isRecording && activePanel === 'board'}
            onClick={(event) => openBoardTools(event.currentTarget)}
          >
            <PenLine size={18} />
          </button>

          {canOpenLastPackage && !isRecording && (
            <button
              className="icon-button soft latest-recording-button"
              type="button"
              aria-label={copy.aria.openLastRecording}
              title={`${copy.aria.openLastRecording}: ${lastPackageName}`}
              onClick={() => void openLastRecordingPackage()}
            >
              <FolderOpen size={18} />
            </button>
          )}

          <div className="time-chip" aria-live="polite">
            <span className={`status-dot ${state}`} />
            <strong>{timeChipLabel}</strong>
            <span>{timeChipValue}</span>
          </div>

          <button
            className="icon-button soft pause-button"
            type="button"
            aria-label={state === 'paused' ? copy.aria.resumeRecording : copy.aria.pauseRecording}
            title={titleWithShortcut(state === 'paused' ? copy.aria.resumeRecording : copy.aria.pauseRecording, 'togglePause')}
            disabled={state !== 'recording' && state !== 'paused'}
            onClick={togglePause}
          >
            {state === 'paused' ? <Play size={18} /> : <Pause size={18} />}
          </button>

          <div className="capsule-utility-segment" aria-hidden={recordingConfigLocked}>
            <button
              className="icon-button soft"
              type="button"
              aria-label={copy.aria.selectLanguage}
              title={copy.localeNames[locale]}
              aria-expanded={activePanel === 'language'}
              disabled={recordingConfigLocked}
              onClick={(event) => togglePanel('language', event.currentTarget)}
            >
              <Languages size={18} />
            </button>
            <button
              className="icon-button soft"
              type="button"
              aria-label={copy.aria.openSettings}
              title={copy.settings.title}
              disabled={recordingConfigLocked}
              onClick={(event) => openSettings(event.currentTarget)}
            >
              <Settings size={18} />
            </button>
          </div>

          <button
            className="icon-button close-app-button"
            type="button"
            aria-label={copy.aria.closeApplication}
            title={copy.aria.closeApplication}
            onClick={(event) => requestCloseApplication(event.currentTarget)}
          >
            <X size={18} />
          </button>
        </div>

        {!isWailsDesktopRuntime() && activePanel && (
          <div ref={popoverRef} className={`popover panel-${activePanel} drop-${capsuleExpandDirection}`} role="dialog" aria-label={copy.aria.menu(activePanel)}>
            {activePanel === 'source' && (
              <div className="menu-grid source-menu">
                <div className="mode-toggle" role="group" aria-label={copy.aria.recordingMode}>
                  {(['video', 'audio'] as RecordingMode[]).map((mode) => {
                    const ModeIcon = mode === 'video' ? Video : Volume2
                    const selected = recordingMode === mode
                    return (
                      <button
                        key={mode}
                        type="button"
                        className={selected ? 'selected' : ''}
                        aria-pressed={selected}
                        disabled={isRecording}
                        onClick={() => {
                          setRecordingMode(mode)
                          if (mode === 'audio') void hideRegionFrame()
                        }}
                      >
                        <ModeIcon size={16} />
                        <span>{copy.recordingModes[mode]}</span>
                      </button>
                    )
                  })}
                </div>

                {recordingMode === 'video' ? (
                  sourcePickerView === 'windows' ? (
                    <div className="source-window-picker">
                      <div className="source-panel-header">
                        <button type="button" className="source-back-button" onClick={() => setSourcePickerView('overview')} aria-label={copy.sourceActions.backToSources}>
                          <ChevronLeft size={16} />
                        </button>
                        <span>
                          <strong>{copy.sourceGroups.window}</strong>
                          <small>{copy.sourceActions.lockedWindowHint}</small>
                        </span>
                      </div>
                      <div className="source-list-scroll">
                        {windowSources.length > 0 ? windowSources.map((source) => (
                          <SourceMenuRow
                            key={source.id}
                            source={source}
                            copy={copy}
                            selected={selectedSource.id === source.id}
                            disabled={recordingConfigLocked}
                            onSelect={() => chooseSource(source)}
                          />
                        )) : (
                          <div className="source-empty">
                            <AppWindow size={18} />
                            <span>{copy.sourceActions.noWindows}</span>
                          </div>
                        )}
                      </div>
                    </div>
                  ) : (
                    <div className="source-list-scroll">
                      <SourceGroup title={copy.sourceGroups.screen}>
                        {allScreensSource && allScreensSource.available !== false && (
                          <SourceMenuRow
                            source={allScreensSource}
                            copy={copy}
                            selected={selectedSource.id === allScreensSource.id}
                            disabled={recordingConfigLocked}
                            onSelect={() => chooseSource(allScreensSource)}
                          />
                        )}
                        {screenSources.map((source) => (
                          <SourceMenuRow
                            key={source.id}
                            source={source}
                            copy={copy}
                            selected={selectedSource.id === source.id}
                            disabled={recordingConfigLocked}
                            onSelect={() => chooseSource(source)}
                            onPreviewStart={() => showScreenMarker(source)}
                            onPreviewEnd={hideScreenMarker}
                          />
                        ))}
                      </SourceGroup>

                      <SourceGroup title={copy.sourceGroups.region}>
                        {regionSource ? (
                          <SourceMenuRow
                            source={regionSource}
                            copy={copy}
                            selected={selectedSource.id === regionSource.id}
                            actionLabel={copy.sourceActions.chooseRegion}
                            disabled={recordingConfigLocked}
                            onSelect={() => void chooseRegion(regionSource)}
                          />
                        ) : (
                          <div className="source-empty">
                            <Crosshair size={18} />
                            <span>{copy.sourceActions.regionUnavailable}</span>
                          </div>
                        )}
                      </SourceGroup>

                      <SourceGroup title={copy.sourceGroups.window}>
                        <button className={`menu-row ${selectedWindowSource ? 'selected' : ''}`} type="button" disabled={recordingConfigLocked} onClick={() => setSourcePickerView('windows')}>
                          <AppWindow size={18} />
                          <span>
                            <strong>{copy.sourceActions.chooseLockedWindow}</strong>
                            <small>{selectedWindowSource ? sourceName(selectedWindowSource, copy) : copy.sourceActions.lockedWindowHint}</small>
                          </span>
                          <ChevronDown size={16} />
                        </button>
                      </SourceGroup>

                      {sourceSelectionText && <div className="source-selection-note">{sourceSelectionText}</div>}
                    </div>
                  )
                ) : (
                  <div className="menu-row selected audio-mode-summary" aria-live="polite">
                    <Volume2 size={18} />
                    <span>
                      <strong>{copy.sourceAudioOnly.name}</strong>
                      <small>{audioOnlySourceMeta(systemAudio, microphone, copy)}</small>
                    </span>
                    <Check size={16} />
                  </div>
                )}
              </div>
            )}

            {activePanel === 'audio' && (
              <div className="menu-stack">
                <SwitchRow label={copy.panels.systemAudio} checked={systemAudio} disabled={recordingConfigLocked} onChange={(value) => commitAudioStatePatch({system: value})} />
                <label className="field-label" htmlFor="system-audio-device">{copy.panels.systemAudioDevice}</label>
                <SelectMenu
                  id="system-audio-device"
                  value={selectedSystemAudio}
                  disabled={recordingConfigLocked}
                  options={availableSystemAudio.map((device) => ({value: device.id, label: mediaDeviceName(device, copy), disabled: device.available === false}))}
                  onChange={(value) => commitAudioStatePatch({systemDeviceId: value})}
                />
                <SwitchRow
                  label={copy.panels.microphone}
                  checked={microphone && hasAvailableMicrophone}
                  disabled={recordingConfigLocked || !hasAvailableMicrophone}
                  onChange={(value) => {
                    commitAudioStatePatch(value
                      ? {microphone: true, microphoneDeviceId: selectedMic || availableMicrophones.find((device) => device.available !== false)?.id}
                      : {microphone: false, noiseSuppression: false})
                  }}
                />
                <SwitchRow
                  label={copy.panels.rnnoise}
                  checked={rnnoiseActive}
                  disabled={recordingConfigLocked || !microphone || selectedMicrophoneDevice?.rnnoiseEligible === false}
                  onChange={(value) => commitAudioStatePatch({noiseSuppression: value && microphoneRef.current})}
                />
                <label className="field-label" htmlFor="mic-device">{copy.panels.microphoneDevice}</label>
                <SelectMenu
                  id="mic-device"
                  value={selectedMic}
                  disabled={recordingConfigLocked || !microphone || !hasAvailableMicrophone}
                  options={availableMicrophones.length === 0
                    ? [{value: '', label: copy.panels.noMicrophones, disabled: true}]
                    : availableMicrophones.map((device) => ({value: device.id, label: mediaDeviceName(device, copy), disabled: device.available === false}))}
                  onChange={(value) => commitAudioStatePatch({microphoneDeviceId: value})}
                />
                <div
                  className={`meter ${micMonitorActive ? 'live' : ''} ${micMonitorError ? 'error' : ''}`}
                  aria-label={copy.aria.microphoneLevel}
                  title={micMonitorError ?? micMonitorStatusText}
                >
                  {micMeterBars.map((bar, index) => (
                    <span key={index} className={bar.active ? 'active' : ''} style={{height: bar.height}} />
                  ))}
                </div>
                <div className="meter-status">
                  <span>{micMonitorStatusText}</span>
                  <b>{Math.round(micPeak * 100)}%</b>
                </div>
              </div>
            )}

            {activePanel === 'camera' && (
              <div className="menu-stack">
                <SwitchRow
                  label={copy.panels.cameraSidecar}
                  checked={camera}
                  disabled={recordingConfigLocked || !hasUsableCamera}
                  onChange={(value) => {
                    const enabled = value && hasUsableCamera
                    let nextCameraId = selectedCamera
                    if (enabled && !selectedCameraUsable && fallbackUsableCameraDevice) {
                      nextCameraId = fallbackUsableCameraDevice.id
                      setSelectedCamera(nextCameraId)
                    }
                    setCameraEnabled(enabled, nextCameraId)
                  }}
                />
                <label className="field-label" htmlFor="camera-device">{copy.panels.cameraDevice}</label>
                <SelectMenu
                  id="camera-device"
                  value={selectedCamera}
                  disabled={recordingConfigLocked || availableCameras.length === 0}
                  options={availableCameras.map((device) => ({value: device.id, label: mediaDeviceName(device, copy), disabled: !isUsableCameraDevice(device)}))}
                  onChange={chooseCameraDevice}
                />
                <div className={`meter-status ${!hasUsableCamera || !selectedCameraUsable ? 'error' : ''}`}>
                  <span>{cameraStatusText}</span>
                </div>
                <label className="field-label" htmlFor="pip-preset">{copy.panels.pipPreset}</label>
                <SelectMenu
                  id="pip-preset"
                  value={pipPreset}
                  disabled={recordingConfigLocked || !camera || !hasUsableCamera}
                  options={pipPresetOptions.map((preset) => ({value: preset, label: copy.pipPresetLabels[preset]}))}
                  onChange={(value) => {
                    const nextPreset = value as PIPPreset
                    commitPipConfigFromPanel({
                      preset: nextPreset,
                      position: nextPreset !== 'free' ? defaultPipPosition(nextPreset) : currentPipConfig.position,
                    })
                  }}
                />
                <span className="field-label">{copy.panels.pipShape}</span>
                <div className="mode-toggle">
                  {pipShapeOptions.map((shape) => {
                    const ShapeIcon = shape === 'circle' ? CircleDot : Square
                    return (
                      <button
                        key={shape}
                        type="button"
                        className={pipShape === shape ? 'selected' : ''}
                        disabled={recordingConfigLocked || !camera || !hasUsableCamera}
                        onClick={() => commitPipConfigFromPanel({shape})}
                      >
                        <ShapeIcon size={15} />
                        <span>{copy.pipShapeLabels[shape]}</span>
                      </button>
                    )
                  })}
                </div>
                <SwitchRow label={copy.panels.pipMirror} checked={pipMirror} disabled={recordingConfigLocked || !camera || !hasUsableCamera} onChange={(value) => commitPipConfigFromPanel({mirror: value})} />
                <label className="field-label" htmlFor="pip-size">{copy.panels.pipSize}</label>
                <div className="pip-slider-row">
                  <input
                    id="pip-size"
                    type="range"
                    min={pipMinimumScale}
                    max={pipMaximumScale}
                    step="0.001"
                    value={pipScale}
                    disabled={recordingConfigLocked || !camera || !hasUsableCamera}
                    onChange={(event) => commitPipConfigFromPanel({scale: Number(event.currentTarget.value)})}
                  />
                  <b>{formatPipScalePercent(pipScale)}</b>
                </div>
                <label className="field-label" htmlFor="pip-edge">{copy.panels.pipEdge}</label>
                <div className="pip-slider-row">
                  <input
                    id="pip-edge"
                    type="range"
                    min="0.02"
                    max="0.42"
                    step="0.01"
                    value={pipEdgeFeather}
                    disabled={recordingConfigLocked || !camera || !hasUsableCamera}
                    onChange={(event) => commitPipConfigFromPanel({edgeFeather: Number(event.currentTarget.value)})}
                  />
                  <b>{Math.round(pipEdgeFeather * 100)}%</b>
                </div>
                <button
                  type="button"
                  className="pip-edit-button"
                  disabled={recordingConfigLocked || !camera || !hasUsableCamera}
                  onClick={() => void openPipEditor()}
                >
                  <Maximize2 size={16} />
                  <span>{copy.panels.pipEdit}</span>
                </button>
              </div>
            )}

            {activePanel === 'board' && (
              <div className="menu-stack board-tools-menu">
                <div className="board-tool-actions">
                  <button type="button" className="menu-row selected" onClick={beginScreenshotCapture}>
                    <ImageIcon size={18} />
                    <span>
                      <strong>{copy.screenshot.region}</strong>
                      <small>{formatShortcutForDisplay(shortcuts.openScreenshot)}</small>
                    </span>
                  </button>
                  <button type="button" className="menu-row" onClick={() => captureScreenshotMode('full')}>
                    <Maximize2 size={18} />
                    <span>
                      <strong>{copy.screenshot.full}</strong>
                      <small>{copy.screenshot.fullDetail}</small>
                    </span>
                  </button>
                  <button type="button" className="menu-row" onClick={beginScrollingScreenshot}>
                    <ScrollText size={18} />
                    <span>
                      <strong>{copy.screenshot.scrolling}</strong>
                      <small>{copy.screenshot.scrollingDetail}</small>
                    </span>
                  </button>
                  <button type="button" className="menu-row" onClick={openWhiteboard}>
                    <PenLine size={18} />
                    <span>
                      <strong>{copy.whiteboard.open}</strong>
                      <small>{formatShortcutForDisplay(shortcuts.openWhiteboard)}</small>
                    </span>
                  </button>
                </div>
                <div className="screenshot-history-header">
                  <span>
                    <History size={15} />
                    <strong>{copy.screenshot.history}</strong>
                  </span>
                  <small>{screenshotMessage || copy.screenshot.historyDetail}</small>
                </div>
                <div className="screenshot-history-list">
                  {screenshots.length > 0 ? screenshots.slice(0, 6).map((item) => (
                    <ScreenshotHistoryRow
                      key={item.id}
                      item={item}
                      copy={copy}
                      onOpen={openScreenshotPreview}
                      onCopyOcr={copyScreenshotOcrText}
                      onTranslateOcr={translateScreenshotOcrText}
                      onOpenFolder={openScreenshotFolder}
                      onAnnotate={annotateScreenshot}
                      onPin={pinScreenshot}
                      onToggleFixed={toggleScreenshotFixed}
                      onDelete={deleteScreenshot}
                    />
                  )) : (
                    <div className="source-empty">
                      <ImageIcon size={18} />
                      <span>{copy.screenshot.empty}</span>
                    </div>
                  )}
                </div>
              </div>
            )}

            {activePanel === 'language' && (
              <div className="menu-grid compact">
                {localeOptions.map((code) => (
                  <button
                    key={code}
                    type="button"
                    className={`menu-row ${locale === code ? 'selected' : ''}`}
                    onClick={() => {
                      setLocale(code)
                      setActivePanel(null)
                    }}
                  >
                    <Languages size={16} />
                    <span><strong>{copy.localeNames[code]}</strong></span>
                    {locale === code && <Check size={16} />}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}

        <div className="storage-strip">
          <Gauge size={16} />
          <span>{lastPackage}</span>
          <span>{copy.common.backend}: {lastBackend}</span>
          <span>{copy.common.status}: {lastStatusText}</span>
          {sourceSelectionText && <span>{sourceSelectionText}</span>}
          <Wand2 size={16} />
          <span>{rnnoiseActive ? copy.strip.micEnhancementOn : copy.strip.micEnhancementOff}</span>
          <span>{copy.common.preflight}: {lastPreflight ? copy.preflightLabels[lastPreflight.status] : copy.common.notRun}</span>
          <span>{recoverablePackages > 0 ? copy.strip.recoveryPackages(recoverablePackages) : copy.strip.recoveryClean}</span>
        </div>
      </section>

      {!isWailsDesktopRuntime() && settingsOpen && settingsPanel}
      {closePromptOpen && (
        <section ref={closePromptRef} className="close-confirm-panel" role="dialog" aria-modal="true" aria-label={isRecording ? copy.closeDialog.recordingTitle : copy.closeDialog.idleTitle}>
          <div className="close-confirm-copy">
            <strong>{isRecording ? copy.closeDialog.recordingTitle : copy.closeDialog.idleTitle}</strong>
            <span>{isRecording ? copy.closeDialog.recordingMessage : copy.closeDialog.idleMessage}</span>
          </div>
          <div className="close-confirm-actions">
            <button type="button" className="close-confirm-secondary" disabled={closeBusy} onClick={() => setClosePromptOpen(false)}>
              {copy.common.cancel}
            </button>
            <button type="button" className="close-confirm-primary" disabled={closeBusy} onClick={() => void confirmCloseApplication()}>
              {isRecording ? copy.closeDialog.confirmRecording : copy.closeDialog.confirmIdle}
            </button>
          </div>
        </section>
      )}
    </main>
  )
}

function FloatingPanelWindow() {
  const panelRef = useRef<HTMLElement | null>(null)
  const [panelState, setPanelState] = useState<FloatingPanelState>(() => ({
    visible: false,
    anchor: {x: 0, y: 0, width: 0, height: 0},
    bounds: {x: 0, y: 0, width: 0, height: 0},
    token: 0,
  }))
  const [locale, setLocale] = useState<LocaleCode>('zh-CN')
  const [theme, setTheme] = useState<ThemeCode>('night-teal')
  const [settings, setSettings] = useState<AppSettings>(defaultSettings)
  const [appData, setAppData] = useState<AppDataInfo>(fallbackAppData)
  const [storageStatus, setStorageStatus] = useState<AppStorageStatus>(fallbackStorageStatus)
  const [storageRootDraft, setStorageRootDraft] = useState(fallbackAppData.rootDir)
  const [storageBusy, setStorageBusy] = useState(false)
  const [storageMessage, setStorageMessage] = useState<StorageMessageState | null>(null)
  const [availableSources, setAvailableSources] = useState<CaptureSource[]>(sources)
  const [selectedSource, setSelectedSource] = useState<CaptureSource>(sources[0])
  const [sourcePickerView, setSourcePickerView] = useState<'overview' | 'windows'>('overview')
  const [sourceSelectionMessage, setSourceSelectionMessage] = useState<SourceSelectionMessageState | null>(null)
  const [recordingMode, setRecordingMode] = useState<RecordingMode>('video')
  const [state, setState] = useState<RecordingState>('idle')
  const [availableSystemAudio, setAvailableSystemAudio] = useState<MediaDevice[]>(systemAudioDevices)
  const [availableMicrophones, setAvailableMicrophones] = useState<MediaDevice[]>([])
  const [availableCameras, setAvailableCameras] = useState<MediaDevice[]>(cameraDevices)
  const [audioState, setAudioState] = useState<AudioControlState>({
    system: defaultSettings.audio.system,
    systemDeviceId: defaultSettings.audio.systemDeviceId,
    microphone: defaultSettings.audio.microphone,
    microphoneDeviceId: defaultSettings.audio.microphoneDeviceId,
    noiseSuppression: defaultSettings.audio.noiseSuppression,
    microphoneGain: defaultSettings.audio.microphoneGain,
  })
  const [micLevel, setMicLevel] = useState(0)
  const [micPeak, setMicPeak] = useState(0)
  const [micMonitorActive, setMicMonitorActive] = useState(false)
  const [micMonitorError, setMicMonitorError] = useState<string | null>(null)
  const [screenshots, setScreenshots] = useState<ScreenshotItem[]>([])
  const [screenshotMessage, setScreenshotMessage] = useState('')
  const [ocrResult, setOcrResult] = useState<OcrResult | null>(null)
  const [ocrResultLoading, setOcrResultLoading] = useState(false)
  const [ocrResultError, setOcrResultError] = useState('')
  const [ocrResultExpanded, setOcrResultExpanded] = useState(false)
  const [shortcutCapture, setShortcutCapture] = useState<ShortcutAction | null>(null)
  const [shortcutError, setShortcutError] = useState('')
  const [lastBackend, setLastBackend] = useState('ffmpeg-desktop-capture')
  const [closeBusy, setCloseBusy] = useState(false)
  const copy = copyByLocale[locale]
  const parsedOcrPanelContext = useMemo(
    () => parseOcrPanelContext(panelState.kind === 'ocr-result' ? panelState.contextId : ''),
    [panelState.contextId, panelState.kind],
  )
  const isRecording = state === 'recording' || state === 'paused' || state === 'preparing' || state === 'stopping'
  const shortcuts = settings.shortcuts ?? defaultSettings.shortcuts
  const systemAudio = audioState.system
  const microphone = audioState.microphone
  const selectedSystemAudio = audioState.systemDeviceId || availableSystemAudio[0]?.id || ''
  const selectedMic = audioState.microphoneDeviceId || availableMicrophones[0]?.id || ''
  const noiseSuppression = microphone && audioState.noiseSuppression
  const selectedMicrophoneDevice = availableMicrophones.find((device) => device.id === selectedMic)
  const hasAvailableMicrophone = availableMicrophones.some((device) => device.available !== false)
  const hasUsableCamera = availableCameras.some(isUsableCameraDevice)
  const selectedCamera = settings.camera.deviceId || availableCameras[0]?.id || ''
  const selectedCameraDevice = availableCameras.find((device) => device.id === selectedCamera)
  const selectedCameraUsable = selectedCameraDevice ? isUsableCameraDevice(selectedCameraDevice) : hasUsableCamera
  const fallbackUsableCameraDevice = availableCameras.find(isUsableCameraDevice)
  const camera = settings.camera.enabled && hasUsableCamera
  const currentPipConfig = normalizePipConfig(settings.camera.pip, normalizePipPreset(settings.camera.pipPreset))
  const cameraStatusText = !hasUsableCamera || !selectedCameraUsable
    ? selectedCameraDevice?.unavailableReason || selectedCameraDevice?.meta || copy.pipOverlay.cameraUnavailable
    : camera
      ? copy.panels.cameraEnabled
      : copy.panels.cameraOff
  const micMonitorStatusText = micMonitorError
    ? copy.panels.microphoneLevelError
    : !microphone
      ? copy.panels.microphoneLevelOff
      : selectedMicrophoneDevice?.available === false || !hasAvailableMicrophone
        ? copy.panels.microphoneLevelUnavailable
        : micMonitorActive
          ? copy.panels.microphoneLevelLive
          : copy.panels.microphoneLevelWaiting
  const micMeterLevel = microphone && micMonitorActive ? micLevel : 0
  const micMeterBars = useMemo(() => Array.from({length: 18}, (_, index) => {
    const threshold = (index + 1) / 18
    const active = micMeterLevel >= threshold
    const height = active ? Math.max(14, Math.min(100, micMeterLevel * 100 + index * 0.9)) : 8
    return {active, height: `${height}%`}
  }), [micMeterLevel])
  const allScreensSource = availableSources.find((source) => source.type === 'all-screens')
  const screenSources = availableSources.filter((source) => source.type === 'screen')
  const regionSource = availableSources.find((source) => source.type === 'region')
  const windowSources = availableSources.filter((source) => source.type === 'window')
  const selectedWindowSource = selectedSource.type === 'window' ? selectedSource : windowSources.find((source) => source.id === selectedSource.id)
  const sourceSelectionText = sourceSelectionMessage ? formatSourceSelectionMessage(sourceSelectionMessage, copy) : ''

  useEffect(() => {
    document.body.classList.add('rf-floating-panel-window')
    return () => document.body.classList.remove('rf-floating-panel-window')
  }, [])

  useEffect(() => {
    document.documentElement.lang = locale
    document.documentElement.dataset.theme = theme
  }, [locale, theme])

  useEffect(() => {
    let cancelled = false
    void Promise.all([loadBootstrap(), getFloatingPanelState(), getSourceState(), listScreenshots()])
      .then(([bootstrap, panel, sourceState, screenshotItems]) => {
        if (cancelled) return
        applyFloatingBootstrap(bootstrap)
        applyFloatingSourceState(sourceState, bootstrap.sources.length > 0 ? bootstrap.sources : sources)
        setPanelState(panel)
        setScreenshots(screenshotItems)
      })
      .catch((error) => console.info('Floating panel bootstrap fallback:', error))
    const unsubscribePanel = subscribeFloatingPanelChanged(setPanelState)
    const unsubscribeSettings = subscribeSettingsChanged((next) => applyFloatingSettings(next))
    const unsubscribeAudio = subscribeAudioState(setAudioState)
    const unsubscribeStatus = subscribeRecordingStatus((update) => {
      setState(update.status as RecordingState)
      if (update.status === 'recording' || update.status === 'paused' || update.status === 'preparing' || update.status === 'stopping') {
        void hideFloatingPanel()
      }
    })
    const unsubscribeSource = subscribeSourceStateChanged((sourceState) => applyFloatingSourceState(sourceState, availableSources))
    const unsubscribeRegion = subscribeRegionSelection((result) => {
      if (result.cancelled) {
        setSourceSelectionMessage(result.error ? {key: 'regionTooSmall', fallback: result.error} : {key: 'regionCancelled'})
        return
      }
      if (!result.source) return
      setAvailableSources((current) => [result.source!, ...current.filter((source) => source.id !== result.source!.id)])
      setSelectedSource(result.source)
      setRecordingMode('video')
      setSourceSelectionMessage({
        key: 'regionSelected',
        width: result.geometry?.width ?? result.source.width,
        height: result.geometry?.height ?? result.source.height,
      })
      void patchSourceState({
        recordingMode: 'video',
        sourceId: result.source.id,
        sourceType: result.source.type,
        sourceGeometry: {
          x: result.geometry?.x ?? result.source.x ?? 0,
          y: result.geometry?.y ?? result.source.y ?? 0,
          width: result.geometry?.width ?? result.source.width ?? 0,
          height: result.geometry?.height ?? result.source.height ?? 0,
          displayIndex: result.source.displayIndex,
          nativeId: result.source.nativeId,
        },
      })
    })
    const unsubscribeScreenshot = subscribeScreenshotCaptured((item) => {
      setScreenshots((current) => [item, ...current.filter((entry) => entry.id !== item.id)].slice(0, 200))
      setScreenshotMessage(copy.screenshot.captured(item.width, item.height))
    })
    const unsubscribeScreenshotHistory = subscribeScreenshotHistoryChanged(setScreenshots)
    const unsubscribeOcr = subscribeOcrJobEvents((event) => {
      if (!event.sourceId) return
      if (event.status === 'queued') setScreenshotMessage(copy.screenshot.ocrQueued)
      if (event.status === 'running') setScreenshotMessage(copy.screenshot.ocrStatusRunning)
      if (event.status === 'ready') setScreenshotMessage(copy.screenshot.ocrStatusReady)
      if (event.status === 'failed') setScreenshotMessage(event.error || copy.screenshot.ocrStatusFailed)
    })
    return () => {
      cancelled = true
      unsubscribePanel()
      unsubscribeSettings()
      unsubscribeAudio()
      unsubscribeStatus()
      unsubscribeSource()
      unsubscribeRegion()
      unsubscribeScreenshot()
      unsubscribeScreenshotHistory()
      unsubscribeOcr()
    }
  }, [])

  useEffect(() => {
    if (panelState.kind !== 'ocr-result' || !panelState.visible) setOcrResultExpanded(false)
  }, [panelState.kind, panelState.visible])

  useEffect(() => {
    const resultId = parsedOcrPanelContext.resultId
    if (!panelState.visible || !resultId) {
      setOcrResult(null)
      setOcrResultError('')
      setOcrResultLoading(false)
      return
    }
    let cancelled = false
    setOcrResultLoading(true)
    setOcrResultError('')
    void openOcrResult(resultId)
      .then((result) => {
        if (!cancelled) setOcrResult(result)
      })
      .catch((error) => {
        if (cancelled) return
        setOcrResult(null)
        setOcrResultError(readableError(error) || copy.screenshot.ocrStatusFailed)
      })
      .finally(() => {
        if (!cancelled) setOcrResultLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [copy, panelState.kind, panelState.visible, parsedOcrPanelContext.resultId])

  useEffect(() => subscribeAudioLevel((update: AudioLevelUpdate) => {
    if (update.deviceId && selectedMic && update.deviceId !== selectedMic) return
    if (update.error) {
      setMicMonitorError(update.error)
      setMicMonitorActive(false)
      setMicLevel(0)
      setMicPeak(0)
      return
    }
    setMicMonitorError(null)
    setMicMonitorActive(update.active)
    setMicLevel(update.active ? update.level : 0)
    setMicPeak(update.active ? update.peak : 0)
  }), [selectedMic])

  useEffect(() => {
    const shouldMonitor = panelState.visible &&
      panelState.kind === 'audio' &&
      microphone &&
      !isRecording &&
      selectedMic !== '' &&
      selectedMicrophoneDevice?.available !== false &&
      hasAvailableMicrophone
    if (!shouldMonitor) {
      setMicMonitorActive(false)
      setMicLevel(0)
      setMicPeak(0)
      void stopMicrophoneLevelMonitor()
      return
    }
    let cancelled = false
    setMicMonitorError(null)
    void startMicrophoneLevelMonitor(selectedMic)
      .then(() => {
        if (!cancelled) setMicMonitorActive(true)
      })
      .catch((error) => {
        if (cancelled) return
        setMicMonitorError(readableError(error))
        setMicMonitorActive(false)
      })
    return () => {
      cancelled = true
      void stopMicrophoneLevelMonitor()
    }
  }, [hasAvailableMicrophone, isRecording, microphone, panelState.kind, panelState.visible, selectedMic, selectedMicrophoneDevice?.available])

  useLayoutEffect(() => {
    let frame = 0
    const publish = () => {
      const viewportWidth = window.innerWidth || document.documentElement.clientWidth || 0
      const viewportHeight = window.innerHeight || document.documentElement.clientHeight || 0
      const region = elementHitRegion(panelRef.current, viewportWidth, viewportHeight, 'round-rect', 22)
      void setFloatingPanelHitRegions({
        enabled: Boolean(region),
        force: true,
        viewportWidth,
        viewportHeight,
        devicePixelRatio: window.devicePixelRatio || 1,
        regions: region ? [region] : [],
      })
    }
    frame = window.requestAnimationFrame(publish)
    const observer = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(() => {
      if (frame) window.cancelAnimationFrame(frame)
      frame = window.requestAnimationFrame(publish)
    })
    if (panelRef.current) observer?.observe(panelRef.current)
    window.addEventListener('resize', publish)
    return () => {
      if (frame) window.cancelAnimationFrame(frame)
      observer?.disconnect()
      window.removeEventListener('resize', publish)
    }
  }, [panelState.kind, panelState.visible])

  useEffect(() => {
    if (!panelState.visible) return
    const token = panelState.token
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        void hideFloatingPanel(token)
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [panelState.token, panelState.visible])

  const applyFloatingBootstrap = (bootstrap: {
    appData: AppDataInfo
    storage: AppStorageStatus
    state: string
    backend?: string
    sources: CaptureSource[]
    media: MediaInventory
    settings: AppSettings
  }) => {
    const nextSources = bootstrap.sources.length > 0 ? bootstrap.sources : sources
    setAppData(bootstrap.appData)
    setStorageStatus(bootstrap.storage)
    setStorageRootDraft(bootstrap.appData.rootDir)
    setLastBackend(bootstrap.backend || lastBackend)
    setAvailableSources(nextSources)
    setState(bootstrap.state as RecordingState)
    applyFloatingSettings(bootstrap.settings, bootstrap.media, nextSources)
  }

  const applyFloatingSettings = (next: AppSettings, media?: MediaInventory, nextSources = availableSources) => {
    setSettings(next)
    setLocale(next.locale)
    setTheme(next.window.theme)
    setAudioState({
      system: next.audio.system,
      systemDeviceId: next.audio.systemDeviceId,
      microphone: next.audio.microphone,
      microphoneDeviceId: next.audio.microphoneDeviceId,
      noiseSuppression: next.audio.microphone && next.audio.noiseSuppression,
      microphoneGain: next.audio.microphoneGain,
    })
    if (media) {
      setAvailableSystemAudio(media.systemAudio)
      setAvailableMicrophones(media.microphones)
      setAvailableCameras(media.cameras)
    }
    setSelectedSource(selectVisibleInitialSource(nextSources, next.source.lastSourceId, next.source.lastSourceType))
  }

  const applyFloatingSourceState = (sourceState: SourceControlState, sourceList = availableSources) => {
    setRecordingMode(sourceState.recordingMode)
    const picked = sourceState.sourceId
      ? sourceList.find((source) => source.id === sourceState.sourceId)
      : sourceState.sourceType
        ? sourceList.find((source) => source.type === sourceState.sourceType)
        : undefined
    if (!picked) return
    setSelectedSource(picked.type === 'region' && sourceState.sourceGeometry
      ? {
          ...picked,
          x: sourceState.sourceGeometry.x,
          y: sourceState.sourceGeometry.y,
          width: sourceState.sourceGeometry.width,
          height: sourceState.sourceGeometry.height,
          displayIndex: sourceState.sourceGeometry.displayIndex,
          nativeId: sourceState.sourceGeometry.nativeId,
        }
      : picked)
  }

  const commitSettingsPreferencePatch = (patch: SettingsPreferencesPatch) => {
    setSettings((current) => ({
      ...current,
      recording: {
        ...current.recording,
        quality: patch.recordingQuality ?? current.recording.quality,
        fps: patch.recordingFps ?? current.recording.fps,
        captureCursor: patch.captureCursor ?? current.recording.captureCursor,
        countdownSeconds: patch.countdownSeconds ?? current.recording.countdownSeconds,
      },
      window: {
        ...current.window,
        theme: patch.theme ?? current.window.theme,
        startAtLogin: patch.startAtLogin ?? current.window.startAtLogin,
      },
      ocr: {
        ...current.ocr,
        autoRecognizeScreenshots: patch.autoOcr ?? current.ocr.autoRecognizeScreenshots,
        translation: normalizeOcrTranslationSettings({
          ...current.ocr.translation,
          ...(patch.ocrTranslation ?? {}),
        }),
      },
    }))
    if (patch.theme !== undefined) setTheme(normalizeTheme(patch.theme))
    void patchSettingsPreferences(patch)
      .then((saved) => applyFloatingSettings(saved))
      .catch((error) => console.error('Failed to patch floating settings:', error))
  }

  const commitFloatingShortcutSettingsPatch = (action: ShortcutAction, accelerator: string) => {
    const conflict = shortcutActions.find((candidate) => (
      candidate !== action &&
      shortcutIdentity(shortcuts[candidate]) === shortcutIdentity(accelerator)
    ))
    if (conflict) {
      setShortcutError(copy.settings.shortcutConflict(copy.settings.shortcutActionLabels[conflict]))
      return
    }
    setShortcutCapture(null)
    setShortcutError('')
    setSettings((current) => ({
      ...current,
      shortcuts: {
        ...current.shortcuts,
        [action]: accelerator,
      },
    }))
    void patchShortcutSettings({[action]: accelerator} as ShortcutSettingsPatch)
      .then((saved) => {
        setSettings(saved)
        setLocale(saved.locale)
        setTheme(saved.window.theme)
      })
      .catch((error) => {
        setSettings((current) => ({...current, shortcuts}))
        setShortcutError(readableError(error) || copy.settings.shortcutInvalid)
        console.error('Failed to patch floating shortcut settings:', error)
      })
  }

  const commitAudioStatePatch = (patch: AudioStatePatch) => {
    setAudioState((current) => ({...current, ...patch, noiseSuppression: patch.noiseSuppression ?? current.noiseSuppression}))
    void patchAudioState(patch)
      .then(setAudioState)
      .catch((error) => console.error('Failed to patch floating audio state:', error))
  }

  const commitCameraStatePatch = (patch: Partial<AppSettings['camera']>) => {
    const nextCamera = {
      ...settings.camera,
      ...patch,
      pip: patch.pip ?? settings.camera.pip,
      pipPreset: patch.pipPreset ?? patch.pip?.preset ?? settings.camera.pipPreset,
    }
    setSettings((current) => ({...current, camera: nextCamera}))
    void patchCameraState({
      enabled: nextCamera.enabled,
      deviceId: nextCamera.deviceId,
      pipPreset: nextCamera.pipPreset,
      pip: nextCamera.pip,
    })
      .then((saved) => applyFloatingSettings(saved))
      .catch((error) => console.error('Failed to patch floating camera state:', error))
  }

  const setCameraEnabled = (enabled: boolean) => {
    const nextEnabled = enabled && hasUsableCamera
    const deviceId = selectedCamera || fallbackUsableCameraDevice?.id || ''
    const nextPip = nextEnabled
      ? ensureVisiblePipConfig(currentPipConfig.preset === 'off'
          ? normalizePipConfig({...currentPipConfig, preset: 'bottom-right', position: defaultPipPosition('bottom-right')}, 'bottom-right')
          : currentPipConfig)
      : normalizePipConfig({...currentPipConfig, preset: 'off'}, 'off')
    commitCameraStatePatch({
      enabled: nextEnabled,
      deviceId,
      pipPreset: nextPip.preset,
      pip: nextPip,
    })
    if (!nextEnabled) void hidePipOverlay()
  }

  const commitPipConfigFromPanel = (patch: Partial<PIPConfig>) => {
    if (!camera || !hasUsableCamera || recordingMode === 'audio') return
    const nextConfig = ensureVisiblePipConfig(normalizePipConfig({
      ...currentPipConfig,
      ...patch,
      position: patch.position ?? currentPipConfig.position,
    }, (patch.preset as PIPPreset | undefined) ?? currentPipConfig.preset))
    commitCameraStatePatch({enabled: true, pipPreset: nextConfig.preset, pip: nextConfig})
    const target = {
      deviceId: selectedCameraDevice?.id || selectedCamera,
      nativeId: selectedCameraDevice?.nativeId,
      name: selectedCameraDevice?.name || selectedCamera,
    }
    void updatePipOverlay(nextConfig, 'edit', target).catch(() => undefined)
  }

  const chooseCameraDevice = (deviceId: string) => {
    if (isRecording) return
    commitCameraStatePatch({deviceId})
  }

  const chooseSource = (source: CaptureSource) => {
    if (isRecording) return
    if (source.type !== 'region') void hideRegionFrame()
    setSelectedSource(source)
    setSourcePickerView('overview')
    setSourceSelectionMessage(source.available === false ? {key: 'sourceQueued'} : null)
    void patchSourceState({
      recordingMode,
      sourceId: source.id,
      sourceType: source.type,
      sourceGeometry: source.type === 'region' && source.width && source.height
        ? {
            x: source.x ?? 0,
            y: source.y ?? 0,
            width: source.width,
            height: source.height,
            displayIndex: source.displayIndex,
            nativeId: source.nativeId,
          }
        : undefined,
      clearGeometry: source.type !== 'region',
    })
    void hideFloatingPanel(panelState.token)
  }

  const chooseRegion = async () => {
    if (isRecording) return
    setSourcePickerView('overview')
    setSourceSelectionMessage({key: 'regionSelecting'})
    await hideFloatingPanel(panelState.token)
    try {
      await showRegionSelector()
    } catch (error) {
      console.error('Failed to show region selector:', error)
      setSourceSelectionMessage({key: 'sourceQueued'})
    }
  }

  const setRecordingModeFromPanel = (mode: RecordingMode) => {
    setRecordingMode(mode)
    if (mode === 'audio') void hideRegionFrame()
    void patchSourceState({recordingMode: mode})
  }

  const showScreenMarker = (source: CaptureSource) => {
    if (source.type === 'screen') void showScreenIndicator(source.id)
  }

  const beginScreenshotCapture = () => {
    if (isRecording) return
    void hideFloatingPanel(panelState.token)
    setScreenshotMessage(copy.screenshot.selecting)
    void showScreenshotRegionSelector().catch((error) => {
      setScreenshotMessage(readableError(error) || copy.screenshot.captureFailed)
    })
  }

  const beginScrollingScreenshot = () => {
    if (isRecording) return
    setScreenshotMessage(copy.screenshot.scrollingPreparing)
    void startScrollingScreenshot()
      .then(() => setScreenshotMessage(copy.screenshot.scrollingStarted))
      .catch((error) => setScreenshotMessage(readableError(error) || copy.screenshot.scrollingUnavailable))
  }

  const captureScreenshotMode = (mode: 'full' | 'screen' | 'window' | 'focused-window') => {
    if (isRecording) return
    void hideFloatingPanel(panelState.token)
    void captureScreenshot({mode})
      .then((item) => {
        setScreenshots((current) => [item, ...current.filter((entry) => entry.id !== item.id)].slice(0, 200))
        setScreenshotMessage(copy.screenshot.captured(item.width, item.height))
      })
      .catch((error) => setScreenshotMessage(readableError(error) || copy.screenshot.captureFailed))
  }

  const openWhiteboard = () => {
    void hideFloatingPanel(0)
    void showWhiteboardWindow()
  }

  const queueScreenshotOcr = (item: ScreenshotItem) => {
    if (screenshotOcrBusy(item)) return
    setScreenshotMessage(copy.screenshot.ocrQueued)
    void queueRecognizeScreenshot(item.id)
      .catch((error) => {
        console.error('Failed to queue floating screenshot OCR:', error)
        setScreenshotMessage(readableError(error) || copy.screenshot.ocrStatusFailed)
      })
  }

  const resizeOcrResultPanel = (expanded: boolean) => {
    setOcrResultExpanded(expanded)
    if (panelState.kind !== 'ocr-result' || !panelState.visible) return
    const size = expanded ? floatingPanelOcrResultExpandedSize : floatingPanelOcrResultSize
    void showFloatingPanel({
      ...panelState,
      kind: 'ocr-result',
      bounds: {
        ...panelState.bounds,
        width: size.width,
        height: size.height,
      },
      width: size.width,
      height: size.height,
      minWidth: size.minWidth,
      maxHeight: size.maxHeight,
      token: panelState.token + 1,
    })
  }

  const copyScreenshotOcrText = (item: ScreenshotItem) => {
    if (!item.ocrResultId || item.ocrStatus !== 'ready') {
      queueScreenshotOcr(item)
      return
    }
    void openOcrResult(item.ocrResultId)
      .then((result) => copyOcrResultText(result, copy))
      .then((message) => setScreenshotMessage(message))
      .catch((error) => {
        console.error('Failed to copy floating screenshot OCR text:', error)
        setScreenshotMessage(readableError(error) || copy.screenshot.copyTextEmpty)
      })
  }

  const translateScreenshotOcrText = (item: ScreenshotItem) => {
    if (!item.ocrResultId || item.ocrStatus !== 'ready') {
      queueScreenshotOcr(item)
      return
    }
    setOcrResultExpanded(false)
    const size = floatingPanelSizes['ocr-result']
    const token = panelState.token + 1
    void showFloatingPanel({
      kind: 'ocr-result',
      anchor: panelState.anchor,
      bounds: {
        ...panelState.bounds,
        width: size.width,
        height: size.height,
      },
      dockSide: panelState.dockSide,
      width: size.width,
      height: size.height,
      minWidth: size.minWidth,
      maxHeight: size.maxHeight,
      token,
      screenId: panelState.screenId,
      direction: panelState.direction,
      contextId: ocrPanelContext(item.ocrResultId, true),
    }).catch((error) => {
      console.error('Failed to open translated OCR floating panel:', error)
      setScreenshotMessage(`${copy.screenshot.translationFailed}: ${readableError(error)}`)
    })
  }

  const openScreenshotPreview = (item: ScreenshotItem) => {
    void hideFloatingPanel(panelState.token)
    void showPinnedScreenshot(item.id)
      .catch((error) => {
        console.error('Failed to preview floating screenshot:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const openScreenshotFolder = (item: ScreenshotItem) => {
    void openScreenshotDirectory(item.id)
      .then((opened) => {
        if (opened) setScreenshotMessage(copy.screenshot.openFolder)
      })
      .catch((error) => {
        console.error('Failed to open floating screenshot directory:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const annotateScreenshot = (item: ScreenshotItem) => {
    void hideFloatingPanel(0)
    void openScreenshotInWhiteboard(item.id)
      .catch((error) => {
        console.error('Failed to open floating screenshot in whiteboard:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const pinScreenshot = (item: ScreenshotItem) => {
    void showPinnedScreenshot(item.id)
      .then(() => listScreenshots())
      .then(setScreenshots)
      .catch((error) => {
        console.error('Failed to pin floating screenshot:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const deleteScreenshot = (item: ScreenshotItem) => {
    void deleteScreenshotItem(item.id)
      .then((items) => {
        setScreenshots(items)
        setScreenshotMessage(copy.screenshot.deleted)
      })
      .catch((error) => {
        console.error('Failed to delete floating screenshot:', error)
        setScreenshotMessage(readableError(error) || copy.screenshot.deleteFailed)
      })
  }

  const toggleScreenshotFixed = (item: ScreenshotItem) => {
    void patchScreenshotItem(item.id, {fixed: !item.fixed})
      .then(setScreenshots)
      .then(() => {
        if (!item.fixed) return showPinnedScreenshot(item.id).then(() => undefined)
        return undefined
      })
      .catch((error) => {
        console.error('Failed to toggle floating screenshot fixed state:', error)
        setScreenshotMessage(readableError(error))
      })
  }

  const applyDataRoot = async () => {
    const nextRoot = storageRootDraft.trim()
    if (!nextRoot || storageBusy) return
    setStorageBusy(true)
    setStorageMessage({key: 'applying'})
    try {
      const info = await setDataRoot(nextRoot)
      setAppData(info)
      setStorageRootDraft(info.rootDir)
      setStorageMessage({key: 'changed', path: info.videoDir})
    } catch {
      setStorageMessage({key: 'failed'})
    } finally {
      setStorageBusy(false)
    }
  }

  const openRecordingsDirectory = async () => {
    try {
      const info = await openVideoDirectory()
      setAppData(info)
      setStorageRootDraft(info.rootDir)
    } catch {
      setStorageMessage({key: 'failed'})
    }
  }

  const confirmFloatingCloseApplication = async () => {
    if (closeBusy) return
    setCloseBusy(true)
    try {
      if (state === 'recording' || state === 'paused') {
        await stopRecording()
      }
      await hideRegionFrame()
      await hideScreenIndicator()
      await stopMicrophoneLevelMonitor()
      await hideFloatingPanel(panelState.token)
      await quitApplication()
    } finally {
      setCloseBusy(false)
    }
  }

  if (!panelState.visible) {
    return <main className="floating-panel-shell empty" aria-hidden="true" />
  }

  const panel = panelState.kind
  return (
    <main ref={panelRef} className={`floating-panel-shell popover panel-${panel ?? 'empty'} drop-${panelState.direction ?? 'down'}`} role="dialog" aria-label={panel === 'close' ? copy.aria.closeApplication : panel ? copy.aria.menu(panel === 'settings' ? 'language' : panel) : undefined}>
      {panel === 'source' && (
        <div className="menu-grid source-menu">
          <div className="mode-toggle" role="group" aria-label={copy.aria.recordingMode}>
            {(['video', 'audio'] as RecordingMode[]).map((mode) => {
              const ModeIcon = mode === 'video' ? Video : Volume2
              const selected = recordingMode === mode
              return (
                <button key={mode} type="button" className={selected ? 'selected' : ''} aria-pressed={selected} disabled={isRecording} onClick={() => setRecordingModeFromPanel(mode)}>
                  <ModeIcon size={16} />
                  <span>{copy.recordingModes[mode]}</span>
                </button>
              )
            })}
          </div>
          {recordingMode === 'video' ? (
            sourcePickerView === 'windows' ? (
              <div className="source-window-picker">
                <div className="source-panel-header">
                  <button type="button" className="source-back-button" onClick={() => setSourcePickerView('overview')} aria-label={copy.sourceActions.backToSources}>
                    <ChevronLeft size={16} />
                  </button>
                  <span>
                    <strong>{copy.sourceGroups.window}</strong>
                    <small>{copy.sourceActions.lockedWindowHint}</small>
                  </span>
                </div>
                <div className="source-list-scroll">
                  {windowSources.length > 0 ? windowSources.map((source) => (
                    <SourceMenuRow key={source.id} source={source} copy={copy} selected={selectedSource.id === source.id} disabled={isRecording} onSelect={() => chooseSource(source)} />
                  )) : (
                    <div className="source-empty">
                      <AppWindow size={18} />
                      <span>{copy.sourceActions.noWindows}</span>
                    </div>
                  )}
                </div>
              </div>
            ) : (
              <div className="source-list-scroll">
                <SourceGroup title={copy.sourceGroups.screen}>
                  {allScreensSource && allScreensSource.available !== false && (
                    <SourceMenuRow source={allScreensSource} copy={copy} selected={selectedSource.id === allScreensSource.id} disabled={isRecording} onSelect={() => chooseSource(allScreensSource)} />
                  )}
                  {screenSources.map((source) => (
                    <SourceMenuRow key={source.id} source={source} copy={copy} selected={selectedSource.id === source.id} disabled={isRecording} onSelect={() => chooseSource(source)} onPreviewStart={() => showScreenMarker(source)} onPreviewEnd={() => void hideScreenIndicator()} />
                  ))}
                </SourceGroup>
                <SourceGroup title={copy.sourceGroups.region}>
                  {regionSource ? (
                    <SourceMenuRow source={regionSource} copy={copy} selected={selectedSource.id === regionSource.id} actionLabel={copy.sourceActions.chooseRegion} disabled={isRecording} onSelect={() => void chooseRegion()} />
                  ) : (
                    <div className="source-empty">
                      <Crosshair size={18} />
                      <span>{copy.sourceActions.regionUnavailable}</span>
                    </div>
                  )}
                </SourceGroup>
                <SourceGroup title={copy.sourceGroups.window}>
                  <button className={`menu-row ${selectedWindowSource ? 'selected' : ''}`} type="button" disabled={isRecording} onClick={() => setSourcePickerView('windows')}>
                    <AppWindow size={18} />
                    <span>
                      <strong>{copy.sourceActions.chooseLockedWindow}</strong>
                      <small>{selectedWindowSource ? sourceName(selectedWindowSource, copy) : copy.sourceActions.lockedWindowHint}</small>
                    </span>
                    <ChevronDown size={16} />
                  </button>
                </SourceGroup>
                {sourceSelectionText && <div className="source-selection-note">{sourceSelectionText}</div>}
              </div>
            )
          ) : (
            <div className="menu-row selected audio-mode-summary" aria-live="polite">
              <Volume2 size={18} />
              <span>
                <strong>{copy.sourceAudioOnly.name}</strong>
                <small>{audioOnlySourceMeta(systemAudio, microphone, copy)}</small>
              </span>
              <Check size={16} />
            </div>
          )}
        </div>
      )}

      {panel === 'audio' && (
        <div className="menu-stack">
          <SwitchRow label={copy.panels.systemAudio} checked={systemAudio} disabled={isRecording} onChange={(value) => commitAudioStatePatch({system: value})} />
          <label className="field-label" htmlFor="floating-system-audio-device">{copy.panels.systemAudioDevice}</label>
          <SelectMenu id="floating-system-audio-device" value={selectedSystemAudio} disabled={isRecording} options={availableSystemAudio.map((device) => ({value: device.id, label: mediaDeviceName(device, copy), disabled: device.available === false}))} onChange={(value) => commitAudioStatePatch({systemDeviceId: value})} />
          <SwitchRow label={copy.panels.microphone} checked={microphone && hasAvailableMicrophone} disabled={isRecording || !hasAvailableMicrophone} onChange={(value) => commitAudioStatePatch(value ? {microphone: true, microphoneDeviceId: selectedMic || availableMicrophones.find((device) => device.available !== false)?.id} : {microphone: false, noiseSuppression: false})} />
          <SwitchRow label={copy.panels.rnnoise} checked={noiseSuppression} disabled={isRecording || !microphone || selectedMicrophoneDevice?.rnnoiseEligible === false} onChange={(value) => commitAudioStatePatch({noiseSuppression: value && microphone})} />
          <label className="field-label" htmlFor="floating-mic-device">{copy.panels.microphoneDevice}</label>
          <SelectMenu id="floating-mic-device" value={selectedMic} disabled={isRecording || !microphone || !hasAvailableMicrophone} options={availableMicrophones.length === 0 ? [{value: '', label: copy.panels.noMicrophones, disabled: true}] : availableMicrophones.map((device) => ({value: device.id, label: mediaDeviceName(device, copy), disabled: device.available === false}))} onChange={(value) => commitAudioStatePatch({microphoneDeviceId: value})} />
          <div className={`meter ${micMonitorActive ? 'live' : ''} ${micMonitorError ? 'error' : ''}`} aria-label={copy.aria.microphoneLevel} title={micMonitorError ?? micMonitorStatusText}>
            {micMeterBars.map((bar, index) => <span key={index} className={bar.active ? 'active' : ''} style={{height: bar.height}} />)}
          </div>
          <div className="meter-status">
            <span>{micMonitorStatusText}</span>
            <b>{Math.round(micPeak * 100)}%</b>
          </div>
        </div>
      )}

      {panel === 'camera' && (
        <div className="menu-stack">
          <SwitchRow label={copy.panels.cameraSidecar} checked={camera} disabled={isRecording || !hasUsableCamera} onChange={setCameraEnabled} />
          <label className="field-label" htmlFor="floating-camera-device">{copy.panels.cameraDevice}</label>
          <SelectMenu id="floating-camera-device" value={selectedCamera} disabled={isRecording || availableCameras.length === 0} options={availableCameras.map((device) => ({value: device.id, label: mediaDeviceName(device, copy), disabled: !isUsableCameraDevice(device)}))} onChange={chooseCameraDevice} />
          <div className={`meter-status ${!hasUsableCamera || !selectedCameraUsable ? 'error' : ''}`}><span>{cameraStatusText}</span></div>
          <label className="field-label" htmlFor="floating-pip-preset">{copy.panels.pipPreset}</label>
          <SelectMenu id="floating-pip-preset" value={currentPipConfig.preset} disabled={isRecording || !camera || !hasUsableCamera} options={pipPresetOptions.map((preset) => ({value: preset, label: copy.pipPresetLabels[preset]}))} onChange={(value) => commitPipConfigFromPanel({preset: value as PIPPreset, position: value !== 'free' ? defaultPipPosition(value as PIPPreset) : currentPipConfig.position})} />
          <span className="field-label">{copy.panels.pipShape}</span>
          <div className="mode-toggle">
            {pipShapeOptions.map((shape) => {
              const ShapeIcon = shape === 'circle' ? CircleDot : Square
              return (
                <button key={shape} type="button" className={currentPipConfig.shape === shape ? 'selected' : ''} disabled={isRecording || !camera || !hasUsableCamera} onClick={() => commitPipConfigFromPanel({shape})}>
                  <ShapeIcon size={15} />
                  <span>{copy.pipShapeLabels[shape]}</span>
                </button>
              )
            })}
          </div>
          <SwitchRow label={copy.panels.pipMirror} checked={currentPipConfig.mirror} disabled={isRecording || !camera || !hasUsableCamera} onChange={(value) => commitPipConfigFromPanel({mirror: value})} />
          <label className="field-label" htmlFor="floating-pip-size">{copy.panels.pipSize}</label>
          <div className="pip-slider-row">
            <input id="floating-pip-size" type="range" min={pipMinimumScale} max={pipMaximumScale} step="0.001" value={currentPipConfig.scale} disabled={isRecording || !camera || !hasUsableCamera} onChange={(event) => commitPipConfigFromPanel({scale: Number(event.currentTarget.value)})} />
            <b>{formatPipScalePercent(currentPipConfig.scale)}</b>
          </div>
          <label className="field-label" htmlFor="floating-pip-edge">{copy.panels.pipEdge}</label>
          <div className="pip-slider-row">
            <input id="floating-pip-edge" type="range" min="0.02" max="0.42" step="0.01" value={currentPipConfig.edgeFeather} disabled={isRecording || !camera || !hasUsableCamera} onChange={(event) => commitPipConfigFromPanel({edgeFeather: Number(event.currentTarget.value)})} />
            <b>{Math.round(currentPipConfig.edgeFeather * 100)}%</b>
          </div>
        </div>
      )}

      {panel === 'board' && (
        <div className="menu-stack board-tools-menu">
          <div className="board-tool-actions">
            <button type="button" className="menu-row selected" onClick={beginScreenshotCapture}>
              <ImageIcon size={18} />
              <span><strong>{copy.screenshot.region}</strong><small>{formatShortcutForDisplay(settings.shortcuts.openScreenshot)}</small></span>
            </button>
            <button type="button" className="menu-row" onClick={() => captureScreenshotMode('full')}>
              <Maximize2 size={18} />
              <span><strong>{copy.screenshot.full}</strong><small>{copy.screenshot.fullDetail}</small></span>
            </button>
            <button type="button" className="menu-row" onClick={beginScrollingScreenshot}>
              <ScrollText size={18} />
              <span><strong>{copy.screenshot.scrolling}</strong><small>{copy.screenshot.scrollingDetail}</small></span>
            </button>
            <button type="button" className="menu-row" onClick={openWhiteboard}>
              <PenLine size={18} />
              <span><strong>{copy.whiteboard.open}</strong><small>{formatShortcutForDisplay(settings.shortcuts.openWhiteboard)}</small></span>
            </button>
          </div>
          <div className="screenshot-history-header">
            <span><History size={15} /><strong>{copy.screenshot.history}</strong></span>
            <small>{screenshotMessage || copy.screenshot.historyDetail}</small>
          </div>
          <div className="screenshot-history-list">
            {screenshots.length > 0 ? screenshots.slice(0, 6).map((item) => (
              <ScreenshotHistoryRow
                key={item.id}
                item={item}
                copy={copy}
                onOpen={openScreenshotPreview}
                onCopyOcr={copyScreenshotOcrText}
                onTranslateOcr={translateScreenshotOcrText}
                onOpenFolder={openScreenshotFolder}
                onAnnotate={annotateScreenshot}
                onPin={pinScreenshot}
                onToggleFixed={toggleScreenshotFixed}
                onDelete={deleteScreenshot}
              />
            )) : (
              <div className="source-empty"><ImageIcon size={18} /><span>{copy.screenshot.empty}</span></div>
            )}
          </div>
        </div>
      )}

      {panel === 'language' && (
        <div className="menu-grid compact">
          {localeOptions.map((code) => (
            <button key={code} type="button" className={`menu-row ${locale === code ? 'selected' : ''}`} onClick={() => {
              const nextSettings = {...settings, locale: code}
              setLocale(code)
              setSettings(nextSettings)
              void saveSettings(nextSettings).then(applyFloatingSettings)
              void hideFloatingPanel(panelState.token)
            }}>
              <Languages size={16} />
              <span><strong>{copy.localeNames[code]}</strong></span>
              {locale === code && <Check size={16} />}
            </button>
          ))}
        </div>
      )}

      {panel === 'ocr-result' && (
        <OcrResultPanel
          copy={copy}
          result={ocrResult}
          translation={settings.ocr.translation}
          loading={ocrResultLoading}
          error={ocrResultError}
          expanded={ocrResultExpanded}
          autoTranslate={parsedOcrPanelContext.autoTranslate}
          onToggleExpanded={() => resizeOcrResultPanel(!ocrResultExpanded)}
          onClose={() => void hideFloatingPanel(panelState.token)}
          onCopy={() => {
            if (!ocrResult) return
            void copyOcrResultText(ocrResult, copy).then(setScreenshotMessage)
          }}
        />
      )}

      {panel === 'settings' && (
        <section className="settings-panel settings-sheet floating-settings-panel" role="dialog" aria-label={copy.aria.settingsDialog}>
          <div className="sheet-header">
            <div><strong>RecordingFreedom</strong><span>{copy.settings.title}</span></div>
            <button type="button" className="sheet-close" onClick={() => void hideFloatingPanel(panelState.token)}>{copy.common.close}</button>
          </div>
          <div className="settings-list">
            <SettingLine title={copy.settings.storage} value={appData.videoDir} detail={copy.settings.storageDetail} actionLabel={copy.settings.openRecordings} onAction={() => void openRecordingsDirectory()} />
            <SettingLine title={copy.settings.storageHealth} value={formatStorageStatusValue(storageStatus, copy)} status={storageStatusForBadge(storageStatus.status)} statusLabel={copy.capabilityStatusLabels[storageStatusForBadge(storageStatus.status)]} detail={storageStatusDetail(storageStatus, copy)} />
            <SettingTextAction title={copy.settings.dataRoot} value={storageRootDraft} detail={storageMessage ? formatStorageMessage(storageMessage, copy) : copy.settings.dataRootDetail} actionLabel={storageBusy ? copy.common.applying : copy.common.apply} actionDisabled={storageBusy || isRecording} onChange={setStorageRootDraft} onAction={() => void applyDataRoot()} />
            <SettingLine title={copy.settings.appData} value={appData.rootDir} />
            <SettingLine title={copy.settings.settingsFile} value={joinDisplayPath(appData.rootDir, 'settings.json')} />
            <SettingSelect title={copy.settings.language} value={locale} options={localeOptions.map((code) => ({value: code, label: copy.localeNames[code]}))} onChange={(value) => {
              const nextLocale = normalizeLocale(value)
              const nextSettings = {...settings, locale: nextLocale}
              setLocale(nextLocale)
              setSettings(nextSettings)
              void saveSettings(nextSettings).then(applyFloatingSettings)
            }} />
            <SettingSelect title={copy.settings.theme} value={theme} options={themeOptions.map((option) => ({value: option, label: copy.themeNames[option], swatch: themeSwatches[option]}))} onChange={(value) => commitSettingsPreferencePatch({theme: normalizeTheme(value)})} />
            <SettingToggle title={copy.settings.startAtLogin} checked={settings.window.startAtLogin} detail={copy.settings.startAtLoginDetail} onChange={(value) => commitSettingsPreferencePatch({startAtLogin: value})} />
            <SettingToggle title={copy.settings.autoOcr} checked={settings.ocr.autoRecognizeScreenshots} detail={copy.settings.autoOcrDetail} onChange={(value) => commitSettingsPreferencePatch({autoOcr: value})} />
            <OcrTranslationSettingsPanel copy={copy} translation={settings.ocr.translation} onChange={(ocrTranslation) => commitSettingsPreferencePatch({ocrTranslation})} compact />
            <OcrModelSettings copy={copy} compact />
            <SettingLine title={copy.settings.shortcuts} value={copy.settings.shortcutSummary} detail={shortcutError || copy.settings.shortcutDetail} />
            {shortcutActions.map((action) => (
              <SettingShortcut
                key={action}
                title={copy.settings.shortcutActionLabels[action]}
                value={formatShortcutForDisplay(shortcuts[action])}
                detail={shortcutCapture === action ? copy.settings.shortcutHint : undefined}
                actionLabel={shortcutCapture === action ? copy.settings.shortcutRecording : copy.settings.shortcutRecord}
                capturing={shortcutCapture === action}
                onStart={() => {
                  setShortcutError('')
                  setShortcutCapture(action)
                }}
                onCancel={() => {
                  setShortcutCapture(null)
                  setShortcutError('')
                }}
                onCapture={(accelerator) => commitFloatingShortcutSettingsPatch(action, accelerator)}
              />
            ))}
            <SettingSelect title={copy.settings.quality} value={settings.recording.quality} options={recordingQualityOptions.map((quality) => ({value: quality, label: copy.recordingQualityLabels[quality]}))} detail={copy.settings.qualityDetail} onChange={(value) => commitSettingsPreferencePatch({recordingQuality: normalizeRecordingQuality(value)})} />
            <SettingSelect title={copy.settings.fps} value={String(settings.recording.fps)} options={fpsOptions.map((fps) => ({value: String(fps), label: `${fps} ${copy.settings.fps}`}))} onChange={(value) => commitSettingsPreferencePatch({recordingFps: Number(value)})} />
            <SettingToggle title={copy.settings.captureCursor} checked={settings.recording.captureCursor} onChange={(value) => commitSettingsPreferencePatch({captureCursor: value})} />
            <SettingSelect title={copy.settings.countdown} value={String(settings.recording.countdownSeconds)} options={countdownOptions.map((seconds) => ({value: String(seconds), label: seconds === 0 ? copy.common.off : `${seconds}s`}))} onChange={(value) => commitSettingsPreferencePatch({countdownSeconds: Number(value)})} />
            <SettingLine title={copy.settings.recordingBackend} value={lastBackend} detail={copy.settings.recordingBackendDetail} />
            <SettingLine title={copy.settings.release} value="GitHub Actions Windows portable + setup" />
          </div>
        </section>
      )}

      {panel === 'close' && (
        <section className="close-confirm-panel floating-close-confirm-panel" role="dialog" aria-modal="true" aria-label={isRecording ? copy.closeDialog.recordingTitle : copy.closeDialog.idleTitle}>
          <div className="close-confirm-copy">
            <strong>{isRecording ? copy.closeDialog.recordingTitle : copy.closeDialog.idleTitle}</strong>
            <span>{isRecording ? copy.closeDialog.recordingMessage : copy.closeDialog.idleMessage}</span>
          </div>
          <div className="close-confirm-actions">
            <button type="button" className="close-confirm-secondary" disabled={closeBusy} onClick={() => void hideFloatingPanel(panelState.token)}>
              {copy.common.cancel}
            </button>
            <button type="button" className="close-confirm-primary" disabled={closeBusy} onClick={() => void confirmFloatingCloseApplication()}>
              {isRecording ? copy.closeDialog.confirmRecording : copy.closeDialog.confirmIdle}
            </button>
          </div>
        </section>
      )}
    </main>
  )
}

function OcrResultPanel({
  copy,
  result,
  translation,
  loading,
  error,
  expanded,
  autoTranslate = false,
  onToggleExpanded,
  onClose,
  onCopy,
}: {
  copy: RecorderCopy
  result: OcrResult | null
  translation: AppSettings['ocr']['translation']
  loading: boolean
  error: string
  expanded: boolean
  autoTranslate?: boolean
  onToggleExpanded: () => void
  onClose: () => void
  onCopy: () => void
}) {
  const plainText = result?.plainText.trim() ?? ''
  const [previewDataUrl, setPreviewDataUrl] = useState('')
  const [previewError, setPreviewError] = useState('')
  const [hoveredBlockId, setHoveredBlockId] = useState('')
  const [copiedBlockId, setCopiedBlockId] = useState('')
  const [translationCopied, setTranslationCopied] = useState(false)
  const [translationMessage, setTranslationMessage] = useState('')
  const [translationBusy, setTranslationBusy] = useState(false)
  const [translationResult, setTranslationResult] = useState<OcrTranslationResult | null>(null)
  const previewLogKeyRef = useRef('')
  const renderedLogKeyRef = useRef('')
  const autoTranslateKeyRef = useRef('')
  const translatedText = translationResult?.blocks.map((block) => block.translated.trim()).filter(Boolean).join('\n').trim() ?? ''

  useEffect(() => {
    let cancelled = false
    previewLogKeyRef.current = ''
    renderedLogKeyRef.current = ''
    setPreviewDataUrl('')
    setPreviewError('')
    setHoveredBlockId('')
    setCopiedBlockId('')
    setTranslationCopied(false)
    setTranslationMessage('')
    setTranslationBusy(false)
    setTranslationResult(null)
    autoTranslateKeyRef.current = ''
    if (!result) return
    void readOcrResultImage(result.id)
      .then((image) => {
        if (cancelled) return
        const logKey = `${result.id}:${image.available ? 'available' : 'missing'}:${image.bytes || 0}`
        if (previewLogKeyRef.current !== logKey) {
          previewLogKeyRef.current = logKey
          void logClientEvent('ocr-result', 'preview-loaded', {
            resultId: result.id,
            sourceKind: result.sourceKind,
            sourceId: result.sourceId,
            width: result.width,
            height: result.height,
            blockCount: result.blocks.length,
            available: image.available,
            bytes: image.bytes || 0,
          })
        }
        if (image.available && image.dataUrl) {
          setPreviewDataUrl(image.dataUrl)
        }
      })
      .catch((error) => {
        if (cancelled) return
        setPreviewError(readableError(error))
        void logClientEvent('ocr-result', 'preview-error', {
          resultId: result.id,
          sourceKind: result.sourceKind,
          sourceId: result.sourceId,
        })
      })
    return () => {
      cancelled = true
    }
  }, [result?.id])

  useEffect(() => {
    if (!result || !previewDataUrl) return
    const polygonCount = result.blocks.filter((block) => block.box.length >= 4 && ocrBlockPolygonPoints(block)).length
    const logKey = `${result.id}:${result.width}x${result.height}:${result.blocks.length}:${polygonCount}`
    if (renderedLogKeyRef.current === logKey) return
    let cancelled = false
    const frame = window.requestAnimationFrame(() => {
      if (cancelled || renderedLogKeyRef.current === logKey) return
      renderedLogKeyRef.current = logKey
      void logClientEvent('ocr-result', 'rendered', {
        resultId: result.id,
        sourceKind: result.sourceKind,
        sourceId: result.sourceId,
        width: result.width,
        height: result.height,
        blockCount: result.blocks.length,
        polygonCount,
        hasPreview: true,
      })
    })
    return () => {
      cancelled = true
      window.cancelAnimationFrame(frame)
    }
  }, [previewDataUrl, result])

  const copyBlockText = (block: OcrBlock, blockId = block.id) => {
    const text = block.text.trim()
    if (!text) return
    void writeClipboardText(text)
      .then(() => {
        void logClientEvent('ocr-result', 'copy-block', {
          resultId: result?.id ?? '',
          sourceKind: result?.sourceKind ?? '',
          sourceId: result?.sourceId ?? '',
          blockId,
        })
        setCopiedBlockId(blockId)
        window.setTimeout(() => setCopiedBlockId((current) => current === blockId ? '' : current), 1200)
      })
      .catch(() => undefined)
  }

  const runTranslation = useCallback((force = false) => {
    if (!result || !plainText || translationBusy) return
    const normalized = normalizeOcrTranslationSettings(translation)
    const unavailable = ocrTranslationUnavailableMessage(normalized, copy)
    if (unavailable) {
      setTranslationResult(null)
      setTranslationMessage(unavailable)
      return
    }
    setTranslationBusy(true)
    setTranslationMessage(copy.screenshot.translationWorking)
    void translateOcr({
      ocrResultId: result.id,
      provider: normalized.provider,
      sourceLanguage: normalized.sourceLanguage,
      targetLanguage: normalized.targetLanguage,
      baseUrl: normalized.baseUrl,
      apiKey: normalized.apiKey,
      model: normalized.model,
      force,
    })
      .then((next) => {
        setTranslationResult(next)
        setTranslationMessage(copy.screenshot.translationReady)
        void logClientEvent('ocr-result', 'translation-rendered', {
          resultId: result.id,
          sourceKind: result.sourceKind,
          sourceId: result.sourceId,
          provider: next.provider,
          targetLanguage: next.targetLanguage,
          blockCount: next.blocks.length,
        })
      })
      .catch((error) => {
        setTranslationResult(null)
        setTranslationMessage(`${copy.screenshot.translationFailed}: ${readableError(error)}`)
      })
      .finally(() => setTranslationBusy(false))
  }, [copy, plainText, result, translation, translationBusy])

  useEffect(() => {
    if (!autoTranslate || !result || !plainText || translationBusy || translationResult) return
    const key = `${result.id}:${translation.provider}:${translation.targetLanguage}:${translation.model ?? ''}:${translation.baseUrl ?? ''}`
    if (autoTranslateKeyRef.current === key) return
    autoTranslateKeyRef.current = key
    runTranslation(false)
  }, [autoTranslate, plainText, result, runTranslation, translation.baseUrl, translation.model, translation.provider, translation.targetLanguage, translationBusy, translationResult])

  const copyTranslatedText = () => {
    if (!translatedText) return
    void writeClipboardText(translatedText)
      .then(() => {
        setTranslationCopied(true)
        window.setTimeout(() => setTranslationCopied(false), 1200)
      })
      .catch(() => undefined)
  }

  return (
    <section className={`ocr-result-panel ${expanded ? 'expanded' : ''}`}>
      <div className="ocr-result-header">
        <span>
          <Eye size={16} />
          <strong>{copy.screenshot.ocrResult}</strong>
        </span>
        <div className="ocr-result-header-actions">
          <button
            type="button"
            className="icon-button"
            aria-label={expanded ? copy.screenshot.collapseOcrResult : copy.screenshot.expandOcrResult}
            title={expanded ? copy.screenshot.collapseOcrResult : copy.screenshot.expandOcrResult}
            onClick={onToggleExpanded}
          >
            {expanded ? <Minimize2 size={15} /> : <Maximize2 size={15} />}
          </button>
          <button type="button" className="sheet-close" onClick={onClose}>{copy.common.close}</button>
        </div>
      </div>
      {loading && <div className="source-empty"><FileText size={18} /><span>{copy.screenshot.ocrStatusRunning}</span></div>}
      {!loading && error && <div className="source-empty error"><FileText size={18} /><span>{error}</span></div>}
      {!loading && !error && result && (
        <>
          <div className="ocr-result-summary">
            <span>{result.modelId || copy.screenshot.ocr}</span>
            <b>{copy.screenshot.ocrBlocks(result.blocks.length)}</b>
          </div>
          {previewDataUrl && (
            <div className="ocr-preview-frame">
              <img src={previewDataUrl} alt={copy.screenshot.ocrResult} draggable={false} />
              <svg viewBox={`0 0 ${Math.max(1, result.width)} ${Math.max(1, result.height)}`} preserveAspectRatio="none" aria-hidden="true">
                {result.blocks.map((block, index) => {
                  const blockId = ocrBlockStableId(block, index)
                  return (
                    <polygon
                      key={blockId}
                      points={ocrBlockPolygonPoints(block)}
                      className={blockId === hoveredBlockId ? 'active' : ''}
                    />
                  )
                })}
              </svg>
              <OcrPositionTextLayer
                copy={copy}
                result={result}
                translationResult={translationResult}
                hoveredBlockId={hoveredBlockId}
                copiedBlockId={copiedBlockId}
                onHover={setHoveredBlockId}
                onCopy={copyBlockText}
              />
            </div>
          )}
          {!previewDataUrl && previewError && <div className="ocr-preview-note">{previewError}</div>}
          <div className="ocr-result-text">
            {plainText || copy.screenshot.ocrNoText}
          </div>
          <div className="ocr-result-actions">
            <button type="button" className="menu-row" disabled={!plainText} onClick={onCopy}>
              <CopyIcon size={16} />
              <span><strong>{copy.screenshot.copyText}</strong></span>
            </button>
            <button type="button" className="menu-row" disabled={!plainText || translationBusy} onClick={() => runTranslation(false)}>
              <Languages size={16} />
              <span><strong>{translationBusy ? copy.screenshot.translationWorking : copy.screenshot.translateText}</strong></span>
            </button>
            <button type="button" className="menu-row" disabled={!translatedText} onClick={copyTranslatedText}>
              <CopyIcon size={16} />
              <span><strong>{translationCopied ? copy.screenshot.copiedTranslation : copy.screenshot.copyTranslation}</strong></span>
            </button>
          </div>
          {translationMessage && <div className="ocr-translation-note">{translationMessage}</div>}
          {translationResult && translationResult.blocks.length > 0 && (
            <div className="ocr-translation-list" aria-label={copy.screenshot.translationReady}>
              {translationResult.blocks.map((block) => (
                <div className="ocr-translation-row" key={block.blockId}>
                  <span>{block.source}</span>
                  <strong>{block.translated}</strong>
                </div>
              ))}
            </div>
          )}
          <div className="ocr-block-list" aria-label={copy.screenshot.ocrBlocks(result.blocks.length)}>
            {result.blocks.map((block, index) => {
              const blockId = ocrBlockStableId(block, index)
              return (
                <button
                  type="button"
                  className={`ocr-block-row ${blockId === hoveredBlockId ? 'active' : ''}`}
                  key={blockId}
                  onPointerEnter={() => setHoveredBlockId(blockId)}
                  onPointerLeave={() => setHoveredBlockId('')}
                  onFocus={() => setHoveredBlockId(blockId)}
                  onBlur={() => setHoveredBlockId('')}
                  onClick={() => copyBlockText(block, blockId)}
                  title={copiedBlockId === blockId ? copy.screenshot.copiedText : copy.screenshot.copyText}
                >
                  <span>{block.text || copy.screenshot.ocrNoText}</span>
                  <b>{copiedBlockId === blockId ? copy.screenshot.copiedText : `${Math.round((block.confidence || 0) * 100)}%`}</b>
                </button>
              )
            })}
          </div>
        </>
      )}
      {!loading && !error && !result && <div className="source-empty"><FileText size={18} /><span>{copy.screenshot.ocrNoText}</span></div>}
    </section>
  )
}

function FloatingSelectWindow() {
  const rootRef = useRef<HTMLElement | null>(null)
  const [selectState, setSelectState] = useState<FloatingSelectState>(() => ({
    visible: false,
    anchor: {x: 0, y: 0, width: 0, height: 0},
    bounds: {x: 0, y: 0, width: 0, height: 0},
    options: [],
    token: 0,
  }))

  useEffect(() => {
    document.body.classList.add('rf-floating-select-window')
    return () => document.body.classList.remove('rf-floating-select-window')
  }, [])

  useEffect(() => {
    let cancelled = false
    void getFloatingSelectState()
      .then((state) => {
        if (!cancelled) setSelectState(state)
      })
      .catch((error) => console.info('Floating select state unavailable:', error))
    const unsubscribe = subscribeFloatingSelectChanged(setSelectState)
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  useLayoutEffect(() => {
    const viewportWidth = window.innerWidth || document.documentElement.clientWidth || 0
    const viewportHeight = window.innerHeight || document.documentElement.clientHeight || 0
    const region = elementHitRegion(rootRef.current, viewportWidth, viewportHeight, 'round-rect', 16)
    void setFloatingSelectHitRegions({
      enabled: Boolean(region),
      force: true,
      viewportWidth,
      viewportHeight,
      devicePixelRatio: window.devicePixelRatio || 1,
      regions: region ? [region] : [],
    })
  }, [selectState.visible, selectState.options.length])

  useEffect(() => {
    if (!selectState.visible) return
    const token = selectState.token
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        void hideFloatingSelect(token)
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [selectState.token, selectState.visible])

  if (!selectState.visible) {
    return <main className="floating-select-shell empty" aria-hidden="true" />
  }

  return (
    <main ref={rootRef} className={`floating-select-shell select-menu-list drop-${selectState.direction ?? 'down'}`} role="listbox">
      {selectState.options.map((option) => (
        <button
          key={option.value}
          type="button"
          role="option"
          className={`select-menu-option ${option.value === selectState.value ? 'selected' : ''}`}
          aria-selected={option.value === selectState.value}
          disabled={option.disabled}
          onPointerDown={(event) => {
            event.preventDefault()
            event.stopPropagation()
          }}
          onClick={() => {
            if (option.disabled) return
            void completeFloatingSelect({
              id: selectState.id ?? '',
              value: option.value,
              token: selectState.token,
              panelToken: selectState.panelToken,
            })
          }}
        >
          <span className="select-menu-label">
            {option.swatch && <i className="select-menu-swatch" style={{background: option.swatch}} aria-hidden="true" />}
            <span>{option.label}</span>
          </span>
          {option.value === selectState.value && <Check size={15} />}
        </button>
      ))}
    </main>
  )
}

const screenshotDeleteRevealWidth = 116

function ScreenshotHistoryRow({
  item,
  copy,
  onOpen,
  onCopyOcr,
  onTranslateOcr,
  onOpenFolder,
  onAnnotate,
  onPin,
  onToggleFixed,
  onDelete,
}: {
  item: ScreenshotItem
  copy: RecorderCopy
  onOpen: (item: ScreenshotItem) => void
  onCopyOcr: (item: ScreenshotItem) => void
  onTranslateOcr: (item: ScreenshotItem, anchorElement?: Element) => void
  onOpenFolder: (item: ScreenshotItem) => void
  onAnnotate: (item: ScreenshotItem) => void
  onPin: (item: ScreenshotItem) => void
  onToggleFixed: (item: ScreenshotItem) => void
  onDelete: (item: ScreenshotItem) => void
}) {
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [dragOffset, setDragOffset] = useState<number | null>(null)
  const [previewDataUrl, setPreviewDataUrl] = useState('')
  const [previewError, setPreviewError] = useState('')
  const dragRef = useRef<{pointerId: number; startX: number; startY: number; base: number; captured: boolean} | null>(null)
  const suppressClickUntilRef = useRef(0)
  const previewRequestRef = useRef('')
  const offset = dragOffset ?? (deleteOpen ? screenshotDeleteRevealWidth : 0)

  const loadPreview = () => {
    if (previewDataUrl || previewRequestRef.current === item.id) return
    previewRequestRef.current = item.id
    setPreviewError('')
    void readScreenshotImage(item.id, true)
      .then((image) => {
        if (previewRequestRef.current !== item.id) return
        if (image.available && image.dataUrl) {
          setPreviewDataUrl(image.dataUrl)
          return
        }
        setPreviewError(screenshotDisplayName(item))
      })
      .catch((error) => {
        if (previewRequestRef.current !== item.id) return
        setPreviewError(readableError(error) || screenshotDisplayName(item))
      })
  }

  const suppressNextClick = () => {
    suppressClickUntilRef.current = Date.now() + 260
  }

  const beginSwipe = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (event.pointerType === 'mouse' && event.button !== 0) return
    const target = event.target as HTMLElement | null
    if (target?.closest('.screenshot-history-actions, .screenshot-history-delete-actions')) return
    dragRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      base: deleteOpen ? screenshotDeleteRevealWidth : 0,
      captured: false,
    }
  }

  const moveSwipe = (event: ReactPointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current
    if (!drag || drag.pointerId !== event.pointerId) return
    const deltaX = event.clientX - drag.startX
    const deltaY = event.clientY - drag.startY
    if (Math.abs(deltaX) < 4 && Math.abs(deltaY) < 4) return
    if (Math.abs(deltaX) >= Math.abs(deltaY)) {
      event.preventDefault()
      if (!drag.captured) {
        event.currentTarget.setPointerCapture?.(event.pointerId)
        drag.captured = true
      }
      setDragOffset(Math.max(0, Math.min(screenshotDeleteRevealWidth, drag.base + deltaX)))
    }
  }

  const finishSwipe = (event: ReactPointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current
    if (!drag || drag.pointerId !== event.pointerId) return
    const deltaX = event.clientX - drag.startX
    const finalOffset = Math.max(0, Math.min(screenshotDeleteRevealWidth, drag.base + deltaX))
    const moved = Math.abs(deltaX) > 7 || Math.abs(event.clientY - drag.startY) > 7
    if (moved) suppressNextClick()
    setDeleteOpen(deltaX < -32 ? false : finalOffset > screenshotDeleteRevealWidth * 0.44)
    setDragOffset(null)
    dragRef.current = null
    if (drag.captured) event.currentTarget.releasePointerCapture?.(event.pointerId)
  }

  const cancelSwipe = (event: ReactPointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current
    if (drag?.pointerId === event.pointerId) {
      setDragOffset(null)
      dragRef.current = null
      if (drag.captured) event.currentTarget.releasePointerCapture?.(event.pointerId)
      suppressNextClick()
    }
  }

  const handleOpen = () => {
    if (Date.now() < suppressClickUntilRef.current) return
    if (deleteOpen) {
      setDeleteOpen(false)
      return
    }
    onOpen(item)
  }

  const handleAction = (action: (item: ScreenshotItem) => void) => {
    if (Date.now() < suppressClickUntilRef.current) return
    action(item)
  }

  return (
    <div
      className={`screenshot-history-row ${deleteOpen ? 'delete-open' : ''} ${deleteOpen || dragOffset !== null ? 'delete-revealed' : ''}`.trim()}
      onPointerDown={beginSwipe}
      onPointerMove={moveSwipe}
      onPointerUp={finishSwipe}
      onPointerCancel={cancelSwipe}
    >
      <div className="screenshot-history-delete-actions" aria-hidden={!deleteOpen && dragOffset === null}>
        <button
          type="button"
          className="delete-back"
          tabIndex={deleteOpen ? 0 : -1}
          onPointerDown={(event) => event.stopPropagation()}
          onClick={(event) => {
            event.stopPropagation()
            setDeleteOpen(false)
          }}
        >
          <X size={14} />
          <span>{copy.screenshot.deleteReturn}</span>
        </button>
        <button
          type="button"
          className="delete-confirm"
          tabIndex={deleteOpen ? 0 : -1}
          onPointerDown={(event) => event.stopPropagation()}
          onClick={(event) => {
            event.stopPropagation()
            setDeleteOpen(false)
            onDelete(item)
          }}
        >
          <Trash2 size={14} />
          <span>{copy.screenshot.delete}</span>
        </button>
      </div>
      <div className="screenshot-history-slide" style={{transform: `translateX(${offset}px)`}}>
        <button
          type="button"
          className="screenshot-history-main"
          onPointerEnter={loadPreview}
          onFocus={loadPreview}
          onClick={handleOpen}
        >
          <span className="screenshot-history-icon">
            <ImageIcon size={17} />
            <span className={`screenshot-history-thumbnail ${previewDataUrl ? 'loaded' : ''}`} aria-hidden="true">
              {previewDataUrl ? (
                <img src={previewDataUrl} alt="" draggable={false} />
              ) : (
                <span>{previewError || screenshotDisplayName(item)}</span>
              )}
            </span>
          </span>
          <span>
            <strong>{screenshotDisplayName(item)}</strong>
            <small>{screenshotMeta(item, copy)}</small>
            <small className={`screenshot-ocr-status ${item.ocrStatus}`}>{screenshotOcrStatusText(item, copy)}</small>
          </span>
        </button>
        <div
          className="screenshot-history-actions"
          onPointerDown={(event) => event.stopPropagation()}
          onClick={(event) => event.stopPropagation()}
        >
          <button type="button" disabled={item.ocrStatus !== 'ready'} aria-label={copy.screenshot.copyText} title={copy.screenshot.copyText} onClick={() => handleAction(onCopyOcr)}>
            <CopyIcon size={15} />
          </button>
          <button type="button" disabled={item.ocrStatus !== 'ready'} aria-label={copy.screenshot.translateText} title={copy.screenshot.translateText} onClick={(event) => handleAction(() => onTranslateOcr(item, event.currentTarget))}>
            <Languages size={15} />
          </button>
          <button type="button" aria-label={copy.screenshot.openFolder} title={copy.screenshot.openFolder} onClick={() => handleAction(onOpenFolder)}>
            <FolderOpen size={15} />
          </button>
          <button type="button" aria-label={copy.screenshot.annotate} title={copy.screenshot.annotate} onClick={() => handleAction(onAnnotate)}>
            <PenLine size={15} />
          </button>
          <button type="button" aria-label={copy.screenshot.pin} title={copy.screenshot.pin} onClick={() => handleAction(onPin)}>
            <Pin size={15} />
          </button>
          <button
            type="button"
            className={item.fixed ? 'selected' : ''}
            aria-label={item.fixed ? copy.screenshot.unfix : copy.screenshot.fix}
            title={item.fixed ? copy.screenshot.unfix : copy.screenshot.fix}
            onClick={() => handleAction(onToggleFixed)}
          >
            {item.fixed ? <Lock size={15} /> : <Unlock size={15} />}
          </button>
        </div>
      </div>
    </div>
  )
}

function screenshotOcrBusy(item: ScreenshotItem) {
  return item.ocrStatus === 'queued' || item.ocrStatus === 'running'
}

function screenshotOcrStatusText(item: ScreenshotItem, copy: RecorderCopy) {
  switch (item.ocrStatus) {
    case 'queued':
      return copy.screenshot.ocrStatusQueued
    case 'running':
      return copy.screenshot.ocrStatusRunning
    case 'ready':
      return copy.screenshot.ocrStatusReady
    case 'failed':
      return item.ocrError || copy.screenshot.ocrStatusFailed
    default:
      return copy.screenshot.ocrStatusNone
  }
}

function screenshotOcrPrimaryTitle(item: ScreenshotItem, copy: RecorderCopy) {
  if (item.ocrStatus === 'ready') return copy.screenshot.openOcrResult
  if (item.ocrStatus === 'failed') return copy.screenshot.retryOcr
  if (screenshotOcrBusy(item)) return screenshotOcrStatusText(item, copy)
  return copy.screenshot.recognizeText
}

function shouldUseFloatingPanelWindows() {
  return isWailsDesktopRuntime() || (window as Window & {__RF_FORCE_FLOATING_PANEL_WINDOWS__?: boolean}).__RF_FORCE_FLOATING_PANEL_WINDOWS__ === true
}

async function copyOcrResultText(result: OcrResult, copy: RecorderCopy) {
  const text = result.plainText.trim()
  if (!text) return copy.screenshot.copyTextEmpty
  await writeClipboardText(text)
  return copy.screenshot.copiedText
}

async function translateAndCopyOcrResultText(resultId: string, translation: AppSettings['ocr']['translation'], copy: RecorderCopy) {
  const normalized = normalizeOcrTranslationSettings(translation)
  const unavailable = ocrTranslationUnavailableMessage(normalized, copy)
  if (unavailable) return unavailable
  const result = await translateOcr({
    ocrResultId: resultId,
    provider: normalized.provider,
    sourceLanguage: normalized.sourceLanguage,
    targetLanguage: normalized.targetLanguage,
    baseUrl: normalized.baseUrl,
    apiKey: normalized.apiKey,
    model: normalized.model,
    force: false,
  })
  const text = ocrTranslationPlainText(result)
  if (!text) return copy.screenshot.copyTextEmpty
  await writeClipboardText(text)
  return copy.screenshot.copiedTranslation
}

function ocrTranslationPlainText(result: OcrTranslationResult | null) {
  return result?.blocks.map((block) => block.translated.trim()).filter(Boolean).join('\n').trim() ?? ''
}

function useContainedImageLayerStyle(containerRef: RefObject<HTMLElement | null>, imageWidth: number, imageHeight: number) {
  const [style, setStyle] = useState<CSSProperties>({
    left: 0,
    top: 0,
    width: '100%',
    height: '100%',
  })

  useLayoutEffect(() => {
    const element = containerRef.current
    if (!element) return
    const safeWidth = Number.isFinite(imageWidth) && imageWidth > 0 ? imageWidth : 1
    const safeHeight = Number.isFinite(imageHeight) && imageHeight > 0 ? imageHeight : 1
    const update = () => {
      const rect = element.getBoundingClientRect()
      if (rect.width <= 0 || rect.height <= 0) return
      const imageRatio = safeWidth / safeHeight
      const containerRatio = rect.width / rect.height
      let width = rect.width
      let height = rect.height
      let left = 0
      let top = 0
      if (containerRatio > imageRatio) {
        width = rect.height * imageRatio
        left = (rect.width - width) / 2
      } else {
        height = rect.width / imageRatio
        top = (rect.height - height) / 2
      }
      const next: CSSProperties = {
        left: `${left}px`,
        top: `${top}px`,
        width: `${width}px`,
        height: `${height}px`,
      }
      setStyle((current) => {
        if (current.left === next.left && current.top === next.top && current.width === next.width && current.height === next.height) {
          return current
        }
        return next
      })
    }
    update()
    const observer = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(update) : null
    observer?.observe(element)
    window.addEventListener('resize', update)
    return () => {
      observer?.disconnect()
      window.removeEventListener('resize', update)
    }
  }, [containerRef, imageHeight, imageWidth])

  return style
}

function ScreenIndicatorWindow() {
  const indicatorWindow = window as Window & {__RF_SCREEN_INDICATOR__?: {label?: string}}
  const [label, setLabel] = useState(indicatorWindow.__RF_SCREEN_INDICATOR__?.label ?? '')

  useEffect(() => {
    document.body.classList.add('rf-screen-indicator-window')
    return () => document.body.classList.remove('rf-screen-indicator-window')
  }, [])

  useEffect(() => {
    const onIndicator = (event: Event) => {
      const next = (event as CustomEvent<{label?: string}>).detail
      setLabel(next?.label ?? '')
    }
    window.addEventListener('rf-screen-indicator', onIndicator)
    return () => window.removeEventListener('rf-screen-indicator', onIndicator)
  }, [])

  return (
    <main className="screen-indicator-shell" aria-hidden="true">
      <span>{label || '1'}</span>
    </main>
  )
}

function screenshotPinItems(state: ScreenshotPinState | undefined): ScreenshotPinnedItem[] {
  const sourcePins = state?.pins?.length
    ? state.pins
    : state?.item
      ? [{item: state.item, dataUrl: state.dataUrl, fixed: state.fixed}]
      : []
  const seen = new Set<string>()
  const pins: ScreenshotPinnedItem[] = []
  for (const pin of sourcePins) {
    if (!pin.item?.id || seen.has(pin.item.id)) continue
    seen.add(pin.item.id)
    pins.push({
      item: pin.item,
      dataUrl: pin.dataUrl,
      fixed: pin.fixed === true || pin.item.fixed === true,
    })
  }
  return pins
}

function screenshotPinStateWithPins(current: ScreenshotPinState | undefined, pins: ScreenshotPinnedItem[]): ScreenshotPinState {
  const normalized = screenshotPinItems({visible: pins.length > 0, fixed: false, pins})
  if (normalized.length === 0) return {visible: false, fixed: false, pins: []}
  const active = normalized[normalized.length - 1]
  return {
    ...(current ?? {}),
    visible: true,
    item: active.item,
    dataUrl: active.dataUrl,
    fixed: active.fixed,
    pins: normalized,
  }
}

function ScreenshotPinWindow() {
  const pinWindow = window as Window & {__RF_SCREENSHOT_PIN__?: ScreenshotPinState}
  const [pinState, setPinState] = useState<ScreenshotPinState | undefined>(pinWindow.__RF_SCREENSHOT_PIN__)
  const [selectedPinId, setSelectedPinId] = useState('')
  const [locale, setLocale] = useState<LocaleCode>(navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en')
  const [theme, setTheme] = useState<ThemeCode>('night-teal')
  const [translationSettings, setTranslationSettings] = useState<AppSettings['ocr']['translation']>(defaultSettings.ocr.translation)
  const [ocrResult, setOcrResult] = useState<OcrResult | null>(null)
  const [ocrMessage, setOcrMessage] = useState('')
  const [highlightOcr, setHighlightOcr] = useState(false)
  const [hoveredBlockId, setHoveredBlockId] = useState('')
  const [copiedBlockId, setCopiedBlockId] = useState('')
  const pins = useMemo(() => screenshotPinItems(pinState), [pinState])
  const activePin = pins.find((pin) => pin.item.id === selectedPinId) ?? pins[pins.length - 1]
  const itemRef = useRef<ScreenshotItem | undefined>(activePin?.item)
  const copy = copyByLocale[locale]
  const item = activePin?.item
  const activeDataUrl = activePin?.dataUrl || pinState?.dataUrl || ''
  const activeFixed = activePin?.fixed === true
  const ocrBusy = item ? screenshotOcrBusy(item) : false
  const ocrReady = item?.ocrStatus === 'ready' && Boolean(item.ocrResultId) && Boolean(ocrResult)
  const ocrStatusText = item ? screenshotOcrStatusText(item, copy) : ''
  const pinCanvasRef = useRef<HTMLDivElement | null>(null)
  const pinImageLayerStyle = useContainedImageLayerStyle(
    pinCanvasRef,
    ocrResult?.width || item?.width || 1,
    ocrResult?.height || item?.height || 1,
  )

  useEffect(() => {
    document.body.classList.add('rf-screenshot-pin-window')
    return () => document.body.classList.remove('rf-screenshot-pin-window')
  }, [])

  useEffect(() => {
    if (pins.length === 0) {
      if (selectedPinId) setSelectedPinId('')
      return
    }
    if (!selectedPinId || !pins.some((pin) => pin.item.id === selectedPinId)) {
      setSelectedPinId(pins[pins.length - 1].item.id)
    }
  }, [pins, selectedPinId])

  useEffect(() => {
    void Promise.all([loadSettings(), loadPinnedScreenshot()])
      .then(([settings, state]) => {
        setLocale(normalizeLocale(settings.locale))
        setTheme(normalizeTheme(settings.window.theme))
        setTranslationSettings(normalizeOcrTranslationSettings(settings.ocr.translation))
        setPinState(state)
      })
      .catch((error) => console.info('Using screenshot pin fallback:', error))
    const unsubscribeSettings = subscribeSettingsChanged((settings) => {
      setLocale(normalizeLocale(settings.locale))
      setTheme(normalizeTheme(settings.window.theme))
      setTranslationSettings(normalizeOcrTranslationSettings(settings.ocr.translation))
    })
    const unsubscribePin = subscribeScreenshotPin((state) => {
      setPinState(state)
    })
    const unsubscribeHistory = subscribeScreenshotHistoryChanged((items) => {
      setPinState((current) => {
        const currentPins = screenshotPinItems(current)
        if (currentPins.length === 0) return current
        let changed = false
        const nextPins = currentPins.map((pin) => {
          const updated = items.find((entry) => entry.id === pin.item.id)
          if (!updated) return pin
          changed = true
          return {...pin, item: updated, fixed: updated.fixed}
        })
        return changed ? screenshotPinStateWithPins(current, nextPins) : current
      })
    })
    const unsubscribeOcr = subscribeOcrJobEvents((event) => {
      const currentItem = itemRef.current
      if (!currentItem || event.sourceId !== currentItem.id) return
      if (event.status === 'queued') setOcrMessage(copy.screenshot.ocrQueued)
      if (event.status === 'running') setOcrMessage(copy.screenshot.ocrStatusRunning)
      if (event.status === 'ready') setOcrMessage(copy.screenshot.ocrStatusReady)
      if (event.status === 'failed') setOcrMessage(event.error || copy.screenshot.ocrStatusFailed)
    })
    return () => {
      unsubscribeSettings()
      unsubscribePin()
      unsubscribeHistory()
      unsubscribeOcr()
    }
  }, [copy])

  useEffect(() => {
    document.documentElement.lang = locale
    document.documentElement.dataset.theme = theme
  }, [locale, theme])

  useEffect(() => {
    itemRef.current = item
  }, [item])

  useEffect(() => {
    setHoveredBlockId('')
    setCopiedBlockId('')
    if (!item?.ocrResultId || item.ocrStatus !== 'ready') {
      setOcrResult(null)
      setHighlightOcr(false)
      return
    }
    let cancelled = false
    void openOcrResult(item.ocrResultId)
      .then((result) => {
        if (cancelled) return
        setOcrResult(result)
        setOcrMessage(result.plainText ? copy.screenshot.ocrBlocks(result.blocks.length) : copy.screenshot.ocrNoText)
      })
      .catch((error) => {
        if (cancelled) return
        setOcrResult(null)
        setOcrMessage(readableError(error) || copy.screenshot.ocrStatusFailed)
      })
    return () => {
      cancelled = true
    }
  }, [copy, item?.id, item?.ocrResultId, item?.ocrStatus])

  const queuePinnedOcr = () => {
    if (!item || ocrBusy) return
    setOcrMessage(copy.screenshot.ocrQueued)
    void queueRecognizePinnedScreenshot(item.id)
      .catch((error) => {
        console.error('Failed to queue pinned screenshot OCR:', error)
        setOcrMessage(readableError(error) || copy.screenshot.ocrStatusFailed)
      })
  }

  const openPinnedOcrResult = async () => {
    if (!item?.ocrResultId || item.ocrStatus !== 'ready') {
      queuePinnedOcr()
      return
    }
    await hideFloatingPanel(0)
    setHighlightOcr(true)
    if (ocrResult?.blocks.length) {
      setOcrMessage(copy.screenshot.ocrBlocks(ocrResult.blocks.length))
    }
  }

  const togglePinnedFixed = () => {
    if (!item) return
    const nextFixed = !activeFixed
    void patchScreenshotItem(item.id, {fixed: nextFixed})
      .then((items) => {
        const updated = items.find((entry) => entry.id === item.id) ?? {...item, fixed: nextFixed}
        setPinState((current) => screenshotPinStateWithPins(current, screenshotPinItems(current).map((pin) => pin.item.id === item.id
          ? {...pin, item: updated, fixed: updated.fixed}
          : pin)))
      })
      .catch((error) => setOcrMessage(readableError(error) || copy.screenshot.ocrStatusFailed))
  }

  const copyPinnedOcrText = () => {
    if (!ocrResult) return
    void copyOcrResultText(ocrResult, copy).then(setOcrMessage).catch(() => setOcrMessage(copy.screenshot.copyTextEmpty))
  }

  const translatePinnedOcrText = () => {
    if (!item?.ocrResultId || item.ocrStatus !== 'ready') {
      queuePinnedOcr()
      return
    }
    setOcrMessage(copy.screenshot.translationWorking)
    void translateAndCopyOcrResultText(item.ocrResultId, translationSettings, copy)
      .then(setOcrMessage)
      .catch((error) => {
        console.error('Failed to translate pinned screenshot OCR text:', error)
        setOcrMessage(`${copy.screenshot.translationFailed}: ${readableError(error)}`)
      })
  }

  const copyPinnedBlockText = (block: OcrBlock, blockId: string) => {
    const text = block.text.trim()
    if (!text) return
    void writeClipboardText(text)
      .then(() => {
        setCopiedBlockId(blockId)
        setOcrMessage(copy.screenshot.copiedText)
        window.setTimeout(() => setCopiedBlockId((current) => current === blockId ? '' : current), 1200)
      })
      .catch(() => setOcrMessage(copy.screenshot.copyTextEmpty))
  }

  if (!pinState?.visible || !activeDataUrl || !item) {
    return (
      <main className="screenshot-pin-shell empty" data-theme={theme}>
        <span>{copy.screenshot.empty}</span>
      </main>
    )
  }

  return (
    <main className={`screenshot-pin-shell ${activeFixed ? 'fixed' : ''} ${pins.length > 1 ? 'multi' : ''}`} data-theme={theme}>
      <section className="screenshot-pin-toolbar">
        <span className="screenshot-pin-title">
          <ImageIcon size={15} />
          <span>
            <strong>{screenshotDisplayName(item)}</strong>
            <small>{ocrMessage || ocrStatusText}</small>
          </span>
        </span>
        <div>
          <button type="button" disabled={ocrBusy} aria-label={screenshotOcrPrimaryTitle(item, copy)} title={screenshotOcrPrimaryTitle(item, copy)} onClick={() => void openPinnedOcrResult()}>
            <FileText size={15} />
          </button>
          <button type="button" disabled={!ocrReady} className={highlightOcr ? 'selected' : ''} aria-label={copy.screenshot.openOcrResult} title={copy.screenshot.openOcrResult} onClick={() => setHighlightOcr((visible) => !visible)}>
            <Eye size={15} />
          </button>
          <button type="button" disabled={!ocrReady || !ocrResult?.plainText.trim()} aria-label={copy.screenshot.copyText} title={copy.screenshot.copyText} onClick={copyPinnedOcrText}>
            <CopyIcon size={15} />
          </button>
          <button type="button" disabled={!ocrReady || !ocrResult?.plainText.trim()} aria-label={copy.screenshot.translateText} title={copy.screenshot.translateText} onClick={translatePinnedOcrText}>
            <Languages size={15} />
          </button>
          <button type="button" aria-label={activeFixed ? copy.screenshot.unfix : copy.screenshot.fix} title={activeFixed ? copy.screenshot.unfix : copy.screenshot.fix} onClick={togglePinnedFixed}>
            {activeFixed ? <Lock size={15} /> : <Pin size={15} />}
          </button>
          <button type="button" aria-label={copy.common.close} title={copy.common.close} onClick={() => void hidePinnedScreenshot()}>
            <X size={16} />
          </button>
        </div>
      </section>
      <div className="screenshot-pin-content">
        {pins.length > 1 && (
          <aside className="screenshot-pin-stack" aria-label={copy.screenshot.pinned}>
            {pins.map((pin) => (
              <button
                key={pin.item.id}
                type="button"
                className={pin.item.id === item.id ? 'selected' : ''}
                title={screenshotDisplayName(pin.item)}
                onClick={() => setSelectedPinId(pin.item.id)}
              >
                {pin.dataUrl ? <img src={pin.dataUrl} alt="" draggable={false} /> : <ImageIcon size={18} />}
                <span>{screenshotDisplayName(pin.item)}</span>
              </button>
            ))}
          </aside>
        )}
        <div className="screenshot-pin-canvas" ref={pinCanvasRef}>
          <img src={activeDataUrl} alt={copy.screenshot.pinned} draggable={false} />
          {highlightOcr && ocrResult && ocrReady && ocrResult.blocks.length > 0 && (
            <svg viewBox={`0 0 ${Math.max(1, ocrResult.width)} ${Math.max(1, ocrResult.height)}`} preserveAspectRatio="xMidYMid meet" aria-label={copy.screenshot.ocrBlocks(ocrResult.blocks.length)}>
              {ocrResult.blocks.map((block, index) => {
                const blockId = ocrBlockStableId(block, index)
                return (
                  <polygon
                    key={blockId}
                    points={ocrBlockPolygonPoints(block)}
                    className={`${blockId === hoveredBlockId ? 'active' : ''} ${blockId === copiedBlockId ? 'copied' : ''}`.trim()}
                    tabIndex={0}
                    role="button"
                    aria-label={block.text || copy.screenshot.ocrNoText}
                    onPointerEnter={() => setHoveredBlockId(blockId)}
                    onPointerLeave={() => setHoveredBlockId('')}
                    onFocus={() => setHoveredBlockId(blockId)}
                    onBlur={() => setHoveredBlockId('')}
                    onClick={() => copyPinnedBlockText(block, blockId)}
                    onKeyDown={(event) => {
                      if (event.key !== 'Enter' && event.key !== ' ') return
                      event.preventDefault()
                      copyPinnedBlockText(block, blockId)
                    }}
                  />
                )
              })}
            </svg>
          )}
          {highlightOcr && ocrResult && ocrReady && ocrResult.blocks.length > 0 && (
            <OcrPositionTextLayer
              copy={copy}
              result={ocrResult}
              hoveredBlockId={hoveredBlockId}
              copiedBlockId={copiedBlockId}
              onHover={setHoveredBlockId}
              onCopy={copyPinnedBlockText}
              className="pin"
              style={pinImageLayerStyle}
            />
          )}
        </div>
      </div>
    </main>
  )
}

type PIPEditAction = 'move' | 'n' | 'e' | 's' | 'w' | 'ne' | 'nw' | 'se' | 'sw'

const pipResizeActions: PIPEditAction[] = ['n', 'e', 's', 'w', 'ne', 'nw', 'se', 'sw']
const pipMinimumContentSize = 24

type PIPOverlayWindowGlobal = Window & {
  __RF_PIP_OVERLAY__?: PIPOverlayState
  __RF_STOP_PIP_CAMERA__?: () => void
  __RF_PIP_STOP_TOKEN__?: number
}

function PIPOverlayWindow() {
  const pipWindow = window as PIPOverlayWindowGlobal
  const [overlayState, setOverlayState] = useState<PIPOverlayState | undefined>(pipWindow.__RF_PIP_OVERLAY__)
  const overlayStateRef = useRef<PIPOverlayState | undefined>(overlayState)
  const editRef = useRef<{
    action: PIPEditAction
    startX: number
    startY: number
    content: {x: number; y: number; width: number; height: number}
    state: PIPOverlayState
    latest: PIPConfig
  } | null>(null)
  const pendingPreviewRef = useRef<PIPConfig | null>(null)
  const previewFrameRef = useRef<number | null>(null)
  const videoRef = useRef<HTMLVideoElement | null>(null)
  const activeCameraStreamRef = useRef<MediaStream | null>(null)
  const cameraStreamsRef = useRef<Set<MediaStream>>(new Set())
  const cameraRequestTokenRef = useRef(0)
  const overlayOperationIdRef = useRef(overlayState?.clientOperationId ?? 0)
  const overlayClosedRef = useRef(!overlayState || overlayState.config.preset === 'off')
  const previewImageModifiedRef = useRef(0)
  const previewImageDataUrlRef = useRef<string | null>(null)
  const previewImageReadyLoggedRef = useRef(false)
  const previewImageWaitingLoggedRef = useRef(false)
  const [cameraReady, setCameraReady] = useState(false)
  const [cameraError, setCameraError] = useState<string | null>(null)
  const [previewImageDataUrl, setPreviewImageDataUrl] = useState<string | null>(null)
  const [overlayLocale, setOverlayLocale] = useState<LocaleCode>(navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en')
  const [overlayTheme, setOverlayTheme] = useState<ThemeCode>('night-teal')
  const copy = copyByLocale[overlayLocale]
  const logPipCameraEvent = (event: string, fields: Record<string, unknown> = {}) => {
    void logClientEvent('pip-camera', event, fields)
  }

  const pipStopTokenChanged = (stopToken: number) => (pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0) !== stopToken

  const nextPipOverlayOperationId = () => {
    overlayOperationIdRef.current += 1
    return overlayOperationIdRef.current
  }

  const cancelPendingPipOverlayEdits = () => {
    if (previewFrameRef.current !== null) {
      window.cancelAnimationFrame(previewFrameRef.current)
      previewFrameRef.current = null
    }
    pendingPreviewRef.current = null
    editRef.current = null
  }

  const pipOverlayStateIsStale = (state: PIPOverlayState | undefined) => {
    const operationId = state?.clientOperationId ?? 0
    return operationId > 0 && operationId < overlayOperationIdRef.current
  }

  const setPipPreviewImage = (dataUrl: string | null) => {
    previewImageDataUrlRef.current = dataUrl
    setPreviewImageDataUrl(dataUrl)
  }

  const clearPipPreviewImage = () => {
    previewImageModifiedRef.current = 0
    previewImageReadyLoggedRef.current = false
    previewImageWaitingLoggedRef.current = false
    setPipPreviewImage(null)
  }

  const trackPipCameraStream = (stream: MediaStream, requestToken: number, stopToken: number, isCancelled: () => boolean) => {
    cameraStreamsRef.current.add(stream)
    logPipCameraEvent('stream-opened', {
      requestToken,
      stopToken,
      tracks: describeMediaStream(stream),
    })
    if (isCancelled() || cameraRequestTokenRef.current !== requestToken || pipStopTokenChanged(stopToken)) {
      logPipCameraEvent('stream-opened-cancelled', {requestToken, stopToken})
      stopAndForgetPipCameraStream(stream)
    }
  }

  const stopAndForgetPipCameraStream = (stream: MediaStream) => {
    logPipCameraEvent('stream-stop', {tracks: describeMediaStream(stream)})
    cameraStreamsRef.current.delete(stream)
    stopMediaStream(stream)
  }

  const stopActivePipCameraStream = () => {
    if (videoRef.current) {
      videoRef.current.pause()
      videoRef.current.srcObject = null
      videoRef.current.removeAttribute('src')
      videoRef.current.load()
    }
    activeCameraStreamRef.current = null
    const streams = Array.from(cameraStreamsRef.current)
    cameraStreamsRef.current.clear()
    if (streams.length > 0) {
      logPipCameraEvent('stop-active-streams', {count: streams.length})
    }
    streams.forEach(stopMediaStream)
  }

  const cancelPipCameraStream = () => {
    logPipCameraEvent('cancel', {
      nextStopToken: (pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0) + 1,
      nextRequestToken: cameraRequestTokenRef.current + 1,
    })
    pipWindow.__RF_PIP_STOP_TOKEN__ = (pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0) + 1
    cameraRequestTokenRef.current += 1
    stopActivePipCameraStream()
    clearPipPreviewImage()
    setCameraReady(false)
  }

  useEffect(() => {
    overlayStateRef.current = overlayState
  }, [overlayState])

  useEffect(() => {
    document.body.classList.add('rf-pip-overlay-window')
    pipWindow.__RF_PIP_STOP_TOKEN__ = pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0
    pipWindow.__RF_STOP_PIP_CAMERA__ = cancelPipCameraStream
    const stopOnWindowLifecycle = () => cancelPipCameraStream()
    const stopOnHidden = () => {
      logPipCameraEvent('visibility-change', {visibilityState: document.visibilityState})
    }
    window.addEventListener('pagehide', stopOnWindowLifecycle)
    window.addEventListener('beforeunload', stopOnWindowLifecycle)
    document.addEventListener('visibilitychange', stopOnHidden)
    return () => {
      cancelPipCameraStream()
      window.removeEventListener('pagehide', stopOnWindowLifecycle)
      window.removeEventListener('beforeunload', stopOnWindowLifecycle)
      document.removeEventListener('visibilitychange', stopOnHidden)
      delete pipWindow.__RF_STOP_PIP_CAMERA__
      document.body.classList.remove('rf-pip-overlay-window')
    }
  }, [])

  useEffect(() => {
    void loadSettings()
      .then((settings) => {
        setOverlayLocale(normalizeLocale(settings.locale))
        setOverlayTheme(normalizeTheme(settings.window.theme))
      })
      .catch((error) => console.info('Using PIP overlay language fallback:', error))
  }, [])

  useEffect(() => subscribeSettingsChanged((settings) => {
    setOverlayLocale(normalizeLocale(settings.locale))
    setOverlayTheme(normalizeTheme(settings.window.theme))
  }), [])

  useEffect(() => {
    document.documentElement.dataset.theme = overlayTheme
  }, [overlayTheme])

  useEffect(() => {
    const onState = (event: Event) => {
      const next = (event as CustomEvent<PIPOverlayState>).detail
      if (next) {
        if (pipOverlayStateIsStale(next)) {
          logPipCameraEvent('overlay-state-stale', {
            incomingOperationId: next.clientOperationId ?? 0,
            currentOperationId: overlayOperationIdRef.current,
            preset: next.config.preset,
          })
          if (overlayClosedRef.current && next.config.preset !== 'off') {
            cancelPipCameraStream()
            void hidePipOverlay()
          }
          return
        }
        if (next.clientOperationId && next.clientOperationId > overlayOperationIdRef.current) {
          overlayOperationIdRef.current = next.clientOperationId
        }
        if (next.config.preset === 'off') {
          logPipCameraEvent('overlay-state-off')
          overlayClosedRef.current = true
          cancelPipCameraStream()
          overlayStateRef.current = undefined
          setOverlayState(undefined)
          setCameraError(null)
          return
        }
        logPipCameraEvent('overlay-state', {
          mode: next.mode,
          preset: next.config.preset,
          cameraId: next.camera?.deviceId ?? '',
          cameraName: next.camera?.name ?? next.cameraName ?? '',
          clientOperationId: next.clientOperationId ?? 0,
        })
        overlayClosedRef.current = false
        overlayStateRef.current = next
        setOverlayState(next)
      }
    }
    window.addEventListener('rf-pip-overlay', onState)
    return () => window.removeEventListener('rf-pip-overlay', onState)
  }, [])

  useEffect(() => {
    if (!overlayState || overlayState.config.preset === 'off') {
      cancelPipCameraStream()
      setCameraError(null)
      return
    }
    const recordingPreviewImagePath = overlayState.mode === 'recording' ? overlayState.previewImagePath?.trim() ?? '' : ''
    if (recordingPreviewImagePath) {
      const requestToken = cameraRequestTokenRef.current + 1
      cameraRequestTokenRef.current = requestToken
      const stopToken = pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0
      let cancelled = false
      let timer: number | null = null
      const startedAt = Date.now()
      stopActivePipCameraStream()
      clearPipPreviewImage()
      setCameraReady(false)
      setCameraError(null)
      const isCancelled = () => cancelled || cameraRequestTokenRef.current !== requestToken || pipStopTokenChanged(stopToken)
      logPipCameraEvent('preview-image-poll-start', {
        requestToken,
        stopToken,
        mode: overlayState.mode,
      })
      const scheduleNextPoll = () => {
        if (!isCancelled()) {
          timer = window.setTimeout(pollPreviewImage, 125)
        }
      }
      async function pollPreviewImage() {
        if (isCancelled()) return
        try {
          const result = await readPipPreviewImage(recordingPreviewImagePath, previewImageModifiedRef.current)
          if (isCancelled()) return
          if (result.modifiedUnixNano && result.modifiedUnixNano > previewImageModifiedRef.current) {
            previewImageModifiedRef.current = result.modifiedUnixNano
          }
          if (result.available && result.dataUrl) {
            setPipPreviewImage(result.dataUrl)
            setCameraReady(true)
            setCameraError(null)
            if (!previewImageReadyLoggedRef.current) {
              previewImageReadyLoggedRef.current = true
              logPipCameraEvent('preview-image-ready', {
                requestToken,
                modifiedUnixNano: result.modifiedUnixNano ?? 0,
              })
            }
          } else if (!previewImageWaitingLoggedRef.current && Date.now() - startedAt > 1800) {
            previewImageWaitingLoggedRef.current = true
            logPipCameraEvent('preview-image-waiting', {
              requestToken,
              path: recordingPreviewImagePath,
            })
          }
        } catch (error) {
          if (!isCancelled()) {
            logPipCameraEvent('preview-image-error', {requestToken, error: readableError(error)})
            setCameraError(readableError(error))
            setCameraReady(Boolean(previewImageDataUrlRef.current))
          }
        } finally {
          scheduleNextPoll()
        }
      }
      void pollPreviewImage()
      return () => {
        cancelled = true
        if (timer !== null) {
          window.clearTimeout(timer)
        }
        logPipCameraEvent('preview-image-cleanup', {requestToken})
        if (cameraRequestTokenRef.current === requestToken) {
          cameraRequestTokenRef.current += 1
        }
      }
    }
    if (overlayState.mode === 'recording') {
      cancelPipCameraStream()
      setCameraError(null)
      logPipCameraEvent('preview-image-path-missing', {mode: overlayState.mode})
      return
    }
    clearPipPreviewImage()
    if (!navigator.mediaDevices?.getUserMedia) {
      cancelPipCameraStream()
      setCameraReady(false)
      setCameraError('media-devices-unavailable')
      return
    }
    const cameraTarget = overlayState.camera ?? (overlayState.cameraName ? {name: overlayState.cameraName} : undefined)
    const requestToken = cameraRequestTokenRef.current + 1
    cameraRequestTokenRef.current = requestToken
    const stopToken = pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0
    let cancelled = false
    stopActivePipCameraStream()
    setCameraReady(false)
    setCameraError(null)
    const isCancelled = () => cancelled || cameraRequestTokenRef.current !== requestToken || pipStopTokenChanged(stopToken)
    logPipCameraEvent('stream-request-start', {
      requestToken,
      stopToken,
      targetDeviceId: cameraTarget?.deviceId ?? '',
      targetNativeId: cameraTarget?.nativeId ?? '',
      targetName: cameraTarget?.name ?? '',
      mode: overlayState.mode,
    })
    void openPipCameraStream(
      cameraTarget,
      (stream) => trackPipCameraStream(stream, requestToken, stopToken, isCancelled),
      isCancelled,
    ).then(async (nextStream) => {
      if (isCancelled()) {
        logPipCameraEvent('stream-request-cancelled-before-attach', {requestToken})
        stopAndForgetPipCameraStream(nextStream)
        return
      }
      activeCameraStreamRef.current = nextStream
      const video = videoRef.current
      if (video) {
        video.srcObject = nextStream
        try {
          await video.play()
          logPipCameraEvent('video-play-ready', {requestToken, mode: overlayState.mode})
        } catch (error) {
          logPipCameraEvent('video-play-deferred', {requestToken, mode: overlayState.mode, error: readableError(error)})
          console.info('PIP preview video play was deferred:', error)
        }
      }
      if (isCancelled()) {
        logPipCameraEvent('stream-request-cancelled-after-attach', {requestToken})
        if (activeCameraStreamRef.current === nextStream) activeCameraStreamRef.current = null
        stopAndForgetPipCameraStream(nextStream)
        return
      }
      logPipCameraEvent('stream-ready', {requestToken, tracks: describeMediaStream(nextStream)})
      setCameraReady(true)
      setCameraError(null)
    }).catch((error) => {
      if (isCancelled()) return
      logPipCameraEvent('stream-error', {requestToken, error: readableError(error)})
      setCameraReady(false)
      setCameraError(readableError(error))
    })
    return () => {
      cancelled = true
      logPipCameraEvent('stream-effect-cleanup', {requestToken})
      if (cameraRequestTokenRef.current === requestToken) {
        cameraRequestTokenRef.current += 1
      }
      stopActivePipCameraStream()
      setCameraReady(false)
    }
  }, [overlayState?.camera?.deviceId, overlayState?.camera?.name, overlayState?.camera?.nativeId, overlayState?.cameraName, overlayState?.config.preset, overlayState?.mode, overlayState?.previewImagePath])

  useEffect(() => () => {
    if (previewFrameRef.current !== null) {
      window.cancelAnimationFrame(previewFrameRef.current)
      previewFrameRef.current = null
    }
  }, [])

  const overlayConfig = async (config: PIPConfig, commit: boolean, operationId = nextPipOverlayOperationId()) => {
    const state = overlayStateRef.current
    const mode = state?.mode ?? 'edit'
    const cameraTarget = state?.camera ?? (state?.cameraName ? {name: state.cameraName} : '')
    const previewImagePath = state?.previewImagePath ?? ''
    const nextConfig = normalizePipConfig(config, config.preset)
    if (nextConfig.preset !== 'off') {
      overlayClosedRef.current = false
    }
    const nextState = commit
      ? await updatePipOverlay(nextConfig, mode, cameraTarget, previewImagePath, operationId)
      : await showPipOverlay(nextConfig, mode, cameraTarget, previewImagePath, operationId)
    if (operationId !== overlayOperationIdRef.current || overlayClosedRef.current || pipOverlayStateIsStale(nextState)) {
      logPipCameraEvent('overlay-config-stale', {
        operationId,
        currentOperationId: overlayOperationIdRef.current,
        closed: overlayClosedRef.current,
        preset: nextState.config.preset,
      })
      if (overlayClosedRef.current && nextState.config.preset !== 'off') {
        cancelPipCameraStream()
        void hidePipOverlay()
      }
      return
    }
    overlayClosedRef.current = nextState.config.preset === 'off'
    overlayStateRef.current = nextState
    setOverlayState(nextState)
  }

  const previewConfig = (config: PIPConfig) => {
    if (overlayClosedRef.current) return
    pendingPreviewRef.current = config
    if (previewFrameRef.current !== null) return
    previewFrameRef.current = window.requestAnimationFrame(() => {
      previewFrameRef.current = null
      const pending = pendingPreviewRef.current
      pendingPreviewRef.current = null
      if (pending && !overlayClosedRef.current) {
        const operationId = nextPipOverlayOperationId()
        void overlayConfig(pending, false, operationId).catch((error) => console.info('PIP preview update failed:', error))
      }
    })
  }

  const commitConfig = async (config: PIPConfig) => {
    cancelPendingPipOverlayEdits()
    const operationId = nextPipOverlayOperationId()
    await overlayConfig(config, true, operationId)
  }

  const beginEdit = (event: ReactPointerEvent<HTMLElement>, action: PIPEditAction) => {
    const state = overlayStateRef.current
    if (!state || state.config.preset === 'off' || event.button !== 0) return
    event.preventDefault()
    event.stopPropagation()
    event.currentTarget.setPointerCapture(event.pointerId)
    editRef.current = {
      action,
      startX: event.screenX,
      startY: event.screenY,
      content: pipAbsoluteContentRect(state),
      state,
      latest: state.config,
    }
  }

  const updateEdit = (event: ReactPointerEvent<HTMLElement>) => {
    const edit = editRef.current
    if (!edit) return
    event.preventDefault()
    const rect = resizePipRect(edit.state, edit.content, edit.action, event.screenX - edit.startX, event.screenY - edit.startY)
    const next = pipConfigFromAbsoluteRect(edit.state, rect)
    edit.latest = next
    previewConfig(next)
  }

  const completeEdit = (event: ReactPointerEvent<HTMLElement>) => {
    const edit = editRef.current
    if (!edit) return
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    editRef.current = null
    void commitConfig(edit.latest).catch((error) => console.info('PIP commit failed:', error))
  }

  const updateShape = (shape: PIPShape) => {
    const state = overlayStateRef.current
    if (!state) return
    void commitConfig({...state.config, shape})
  }

  const toggleMirror = () => {
    const state = overlayStateRef.current
    if (!state) return
    void commitConfig({...state.config, mirror: !state.config.mirror})
  }

  const closePip = () => {
    const state = overlayStateRef.current ?? overlayState
    const operationId = nextPipOverlayOperationId()
    overlayClosedRef.current = true
    cancelPendingPipOverlayEdits()
    cancelPipCameraStream()
    overlayStateRef.current = undefined
    setOverlayState(undefined)
    setCameraError(null)
    if (!state) {
      void hidePipOverlay()
      return
    }
    const cameraTarget = state.camera ?? (state.cameraName ? {name: state.cameraName} : '')
    void updatePipOverlay({...state.config, preset: 'off'}, state.mode, cameraTarget, state.previewImagePath ?? '', operationId)
      .catch((error) => console.info('PIP close failed:', error))
      .finally(() => {
        if (operationId !== overlayOperationIdRef.current && !overlayClosedRef.current) return
        cancelPipCameraStream()
        void hidePipOverlay()
        for (const delay of [120, 500]) {
          window.setTimeout(() => {
            if (operationId !== overlayOperationIdRef.current || !overlayClosedRef.current) return
            cancelPipCameraStream()
            overlayStateRef.current = undefined
            setOverlayState(undefined)
            void hidePipOverlay()
          }, delay)
        }
      })
  }

  const content = overlayState?.config.preset !== 'off' ? overlayState?.contentBounds : undefined
  const isRecordingPipOverlay = overlayState?.mode === 'recording'
  const usesBackendPreviewImage = isRecordingPipOverlay && Boolean(overlayState.previewImagePath?.trim())
  const cameraName = overlayState?.camera?.name || overlayState?.cameraName || copy.panels.cameraSidecar
  const showCameraPlaceholder = !isRecordingPipOverlay && !cameraReady
  const cameraPlaceholderTitle = cameraError
      ? copy.pipOverlay.cameraUnavailable
      : copy.pipOverlay.cameraPreparing
  const cameraPlaceholderDetail = cameraError || cameraName
  const featherPx = content && overlayState ? Math.max(2, Math.round(content.width * overlayState.config.edgeFeather)) : 12
  const frameStyle = content ? {
    left: content.x,
    top: content.y,
    width: content.width,
    height: content.height,
    '--pip-feather': `${featherPx}px`,
  } as CSSProperties : undefined

  return (
    <main
      className={`pip-overlay-shell ${overlayState?.mode ?? 'edit'}`}
      aria-label={copy.pipOverlay.label}
      onPointerMove={updateEdit}
      onPointerUp={completeEdit}
      onPointerCancel={completeEdit}
    >
      {overlayState && content && frameStyle && (
        <div className="pip-live-frame" style={frameStyle}>
          <div className={`pip-live-media ${overlayState.config.shape} ${overlayState.config.mirror ? 'mirrored' : ''}`}>
            {usesBackendPreviewImage ? (
              previewImageDataUrl ? (
                <img
                  className={`pip-camera-preview-image ${cameraReady ? 'ready' : ''}`}
                  src={previewImageDataUrl}
                  alt=""
                  draggable={false}
                />
              ) : null
            ) : (
              <video ref={videoRef} autoPlay muted playsInline className={cameraReady ? 'ready' : ''} />
            )}
            {showCameraPlaceholder && (
              <div className={`pip-camera-placeholder ${overlayState.mode} ${cameraError ? 'error' : 'pending'}`}>
                <Camera size={24} />
                <strong>{cameraPlaceholderTitle}</strong>
                <span>{cameraPlaceholderDetail}</span>
              </div>
            )}
          </div>
          <button className="pip-drag-anchor" type="button" aria-label={copy.pipOverlay.move} title={copy.pipOverlay.move} onPointerDown={(event) => beginEdit(event, 'move')}>
            <Move size={18} />
          </button>
          {pipResizeActions.map((action) => (
            <button
              key={action}
              className={`pip-resize-handle ${action}`}
              type="button"
              aria-label={copy.pipOverlay.resize}
              title={copy.pipOverlay.resize}
              onPointerDown={(event) => beginEdit(event, action)}
            />
          ))}
          <div className="pip-overlay-tools">
            <button type="button" className={overlayState.config.shape === 'circle' ? 'selected' : ''} aria-label={copy.pipShapeLabels.circle} title={copy.pipShapeLabels.circle} onClick={() => updateShape('circle')}>
              <CircleDot size={15} />
            </button>
            <button type="button" className={overlayState.config.shape === 'square' ? 'selected' : ''} aria-label={copy.pipShapeLabels.square} title={copy.pipShapeLabels.square} onClick={() => updateShape('square')}>
              <Square size={15} />
            </button>
            <button type="button" className={overlayState.config.mirror ? 'selected' : ''} aria-label={copy.pipOverlay.mirror} title={copy.pipOverlay.mirror} onClick={toggleMirror}>
              <FlipHorizontal size={15} />
            </button>
            <button type="button" aria-label={copy.pipOverlay.close} title={copy.pipOverlay.close} onClick={closePip}>
              <X size={15} />
            </button>
          </div>
        </div>
      )}
    </main>
  )
}

async function openPipCameraStream(
  target?: PIPOverlayCamera,
  onStreamOpened: (stream: MediaStream) => void = () => undefined,
  isCancelled: () => boolean = () => false,
): Promise<MediaStream> {
  const videoConstraints = {
    width: {ideal: 1280},
    height: {ideal: 720},
  }
  const device = await selectPipPreviewDevice(target)
  if (device?.deviceId) {
    if (isCancelled()) throw new Error('pip camera request cancelled')
    try {
      void logClientEvent('pip-camera', 'get-user-media-exact-start', {
        deviceLabel: device.label,
        deviceId: device.deviceId,
      })
      const exactStream = await navigator.mediaDevices.getUserMedia({
        video: {
          ...videoConstraints,
          deviceId: {exact: device.deviceId},
        },
        audio: false,
      })
      onStreamOpened(exactStream)
      if (isCancelled()) {
        stopMediaStream(exactStream)
        throw new Error('pip camera request cancelled')
      }
      return exactStream
    } catch (error) {
      if (isCancelled()) throw error
      void logClientEvent('pip-camera', 'get-user-media-exact-failed', {
        deviceLabel: device.label,
        error: readableError(error),
      })
      console.info('PIP preview selected camera unavailable, using default camera:', error)
    }
  }
  if (isCancelled()) throw new Error('pip camera request cancelled')
  void logClientEvent('pip-camera', 'get-user-media-default-start', {
    targetName: target?.name ?? '',
    targetNativeId: target?.nativeId ?? '',
  })
  const defaultStream = await navigator.mediaDevices.getUserMedia({
    video: videoConstraints,
    audio: false,
  })
  onStreamOpened(defaultStream)
  if (isCancelled()) {
    stopMediaStream(defaultStream)
    throw new Error('pip camera request cancelled')
  }
  return defaultStream
}

async function selectPipPreviewDevice(target: PIPOverlayCamera | undefined): Promise<MediaDeviceInfo | undefined> {
  const devices = await navigator.mediaDevices.enumerateDevices?.().catch((error) => {
    void logClientEvent('pip-camera', 'enumerate-devices-failed', {error: readableError(error)})
    return []
  })
  const videoDevices = devices?.filter((device) => device.kind === 'videoinput') ?? []
  if (videoDevices.length === 0) return undefined
  const matched = findMatchingPipPreviewDevice(target, videoDevices)
  if (matched) return matched
  return undefined
}

function describeMediaStream(stream: MediaStream) {
  return stream.getTracks().map((track) => {
    const settings = typeof track.getSettings === 'function' ? track.getSettings() : {}
    return [
      track.kind,
      track.readyState,
      track.label,
      settings.deviceId,
      settings.width && settings.height ? `${settings.width}x${settings.height}` : '',
      settings.frameRate ? `${settings.frameRate}fps` : '',
    ].filter(Boolean).join('|')
  }).join(';')
}

function stopMediaStream(stream: MediaStream) {
  stream.getTracks().forEach((track) => track.stop())
}

function findMatchingPipPreviewDevice(target: PIPOverlayCamera | undefined, devices: MediaDeviceInfo[]) {
  const tokens = pipPreviewMatchTokens(target)
  if (tokens.length === 0) return undefined
  return devices.find((device) => {
    const label = normalizePipPreviewText(device.label)
    const id = normalizePipPreviewText(device.deviceId)
    if (!label && !id) return false
    return tokens.some((token) => {
      const normalized = normalizePipPreviewText(token)
      if (!normalized || normalized.length < 3) return false
      const labelMatches = label !== '' && (label === normalized || label.includes(normalized) || normalized.includes(label))
      const idMatches = id !== '' && id === normalized
      return labelMatches || idMatches
    })
  })
}

function pipPreviewMatchTokens(target: PIPOverlayCamera | undefined) {
  const rawTokens = [
    target?.name,
    stripDefaultDevicePrefix(target?.name),
    target?.nativeId,
    target?.deviceId,
  ]
  return Array.from(new Set(rawTokens.map((token) => token?.trim()).filter((token): token is string => Boolean(token && token.length >= 3))))
}

function stripDefaultDevicePrefix(value?: string) {
  return value?.replace(/^\s*default\s+/i, '').replace(/^\s*默认\s*/, '')
}

function normalizePipPreviewText(value?: string) {
  return (value ?? '')
    .toLowerCase()
    .replace(/^\s*default\s+/i, '')
    .replace(/^\s*默认\s*/, '')
    .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function pipAbsoluteContentRect(state: PIPOverlayState) {
  return {
    x: state.windowBounds.x + state.contentBounds.x,
    y: state.windowBounds.y + state.contentBounds.y,
    width: state.contentBounds.width,
    height: state.contentBounds.height,
  }
}

function pipCanvasMargin(bounds: PIPOverlayState['overlayBounds']) {
  return Math.max(16, Math.trunc(bounds.width / 40))
}

function pipMaxContentSize(state: PIPOverlayState) {
  const margin = pipCanvasMargin(state.overlayBounds)
  const canvasMax = Math.min(state.overlayBounds.width, state.overlayBounds.height) - margin * 2
  const scaleMax = Math.round(state.overlayBounds.width * pipMaximumScale)
  return Math.max(pipMinimumContentSize, Math.min(canvasMax, scaleMax))
}

function resizePipRect(state: PIPOverlayState, content: {x: number; y: number; width: number; height: number}, action: PIPEditAction, dx: number, dy: number) {
  if (action === 'move') {
    return {...content, x: content.x + dx, y: content.y + dy}
  }
  let delta = 0
  if (action === 'e') delta = dx
  if (action === 's') delta = dy
  if (action === 'w') delta = -dx
  if (action === 'n') delta = -dy
  if (action === 'se') delta = Math.max(dx, dy)
  if (action === 'sw') delta = Math.max(-dx, dy)
  if (action === 'ne') delta = Math.max(dx, -dy)
  if (action === 'nw') delta = Math.max(-dx, -dy)
  const size = clampNumber(Math.round(content.width + delta), pipMinimumContentSize, pipMaxContentSize(state))
  const next = {...content, width: size, height: size}
  if (action.includes('w')) next.x = content.x + content.width - size
  if (action.includes('n')) next.y = content.y + content.height - size
  return next
}

function pipConfigFromAbsoluteRect(state: PIPOverlayState, rect: {x: number; y: number; width: number; height: number}): PIPConfig {
  const overlay = state.overlayBounds
  const margin = pipCanvasMargin(overlay)
  const size = clampNumber(Math.round(Math.max(rect.width, rect.height)), pipMinimumContentSize, pipMaxContentSize(state))
  const minX = overlay.x + margin
  const minY = overlay.y + margin
  const maxX = overlay.x + overlay.width - margin - size
  const maxY = overlay.y + overlay.height - margin - size
  const x = clampNumber(rect.x, minX, Math.max(minX, maxX))
  const y = clampNumber(rect.y, minY, Math.max(minY, maxY))
  const availableWidth = Math.max(1, maxX - minX)
  const availableHeight = Math.max(1, maxY - minY)
  return normalizePipConfig({
    ...state.config,
    preset: 'free',
    position: {
      x: (x - minX) / availableWidth,
      y: (y - minY) / availableHeight,
    },
    scale: size / Math.max(1, overlay.width),
  }, 'free')
}

type RegionFrameState = {
  bounds: {x: number; y: number; width: number; height: number}
  overlayBounds?: {x: number; y: number; width: number; height: number}
  mode?: 'edit' | 'recording'
  purpose?: 'capture' | 'annotation' | 'screenshot' | 'scrolling-screenshot'
}

type RegionEditAction = 'move' | 'n' | 'e' | 's' | 'w' | 'ne' | 'nw' | 'se' | 'sw'

const regionResizeActions: RegionEditAction[] = ['n', 'e', 's', 'w', 'ne', 'nw', 'se', 'sw']

function useRegionFrameState() {
  const frameWindow = window as Window & {__RF_REGION_FRAME__?: RegionFrameState}
  const [frame, setFrame] = useState<RegionFrameState | undefined>(frameWindow.__RF_REGION_FRAME__)
  const clearFrame = useCallback(() => {
    delete (window as Window & {__RF_REGION_FRAME__?: RegionFrameState}).__RF_REGION_FRAME__
    setFrame(undefined)
  }, [])

  useEffect(() => {
    const onFrame = (event: Event) => {
      const next = (event as CustomEvent<RegionFrameState | undefined>).detail
      if (next?.bounds) {
        ;(window as Window & {__RF_REGION_FRAME__?: RegionFrameState}).__RF_REGION_FRAME__ = next
        setFrame(next)
      } else {
        clearFrame()
      }
    }
    window.addEventListener('rf-region-frame', onFrame)
    return () => window.removeEventListener('rf-region-frame', onFrame)
  }, [clearFrame])

  useEffect(() => {
    window.addEventListener('rf-region-session', clearFrame)
    return () => window.removeEventListener('rf-region-session', clearFrame)
  }, [clearFrame])

  return [frame, clearFrame] as const
}

function useRegionEditorDrag(bounds: RegionFrameState['bounds'] | undefined, updateRegion: (bounds: RegionFrameState['bounds']) => Promise<unknown>) {
  const editRef = useRef<{
    action: RegionEditAction
    startX: number
    startY: number
    bounds: RegionFrameState['bounds']
    latest: RegionFrameState['bounds']
  } | null>(null)

  const beginEdit = (event: ReactPointerEvent<HTMLElement>, action: RegionEditAction) => {
    if (!bounds || event.button !== 0) return
    event.preventDefault()
    event.stopPropagation()
    event.currentTarget.setPointerCapture(event.pointerId)
    editRef.current = {
      action,
      startX: event.screenX,
      startY: event.screenY,
      bounds,
      latest: bounds,
    }
  }

  const updateEdit = (event: ReactPointerEvent<HTMLElement>) => {
    const edit = editRef.current
    if (!edit) return
    event.preventDefault()
    const next = resizeRegionBounds(edit.bounds, edit.action, event.screenX - edit.startX, event.screenY - edit.startY)
    edit.latest = next
    void updateRegion(next)
  }

  const completeEdit = (event: ReactPointerEvent<HTMLElement>) => {
    const edit = editRef.current
    if (!edit) return
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    editRef.current = null
    void updateRegion(edit.latest)
  }

  return {beginEdit, updateEdit, completeEdit}
}

function resizeRegionBounds(bounds: RegionFrameState['bounds'], action: RegionEditAction, dx: number, dy: number) {
  const next = {...bounds}
  if (action === 'move') {
    next.x = Math.round(bounds.x + dx)
    next.y = Math.round(bounds.y + dy)
    return next
  }
  if (action.includes('e')) {
    next.width = Math.round(Math.max(minRegionEditorSize, bounds.width + dx))
  }
  if (action.includes('s')) {
    next.height = Math.round(Math.max(minRegionEditorSize, bounds.height + dy))
  }
  if (action.includes('w')) {
    const width = Math.round(Math.max(minRegionEditorSize, bounds.width - dx))
    next.x = Math.round(bounds.x + bounds.width - width)
    next.width = width
  }
  if (action.includes('n')) {
    const height = Math.round(Math.max(minRegionEditorSize, bounds.height - dy))
    next.y = Math.round(bounds.y + bounds.height - height)
    next.height = height
  }
  return next
}

const minRegionEditorSize = 64
const regionManualDragThreshold = 6

function RegionOverlayWindow() {
  const overlayWindow = window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}
  const initialSession = overlayWindow.__RF_REGION_SESSION__
  const [editFrame, clearEditFrame] = useRegionFrameState()
  const isScreenshotRegionEdit = editFrame?.mode === 'edit' && editFrame.purpose === 'screenshot'
  const editDrag = useRegionEditorDrag(
    editFrame?.mode === 'edit' ? editFrame.bounds : undefined,
    isScreenshotRegionEdit ? updateScreenshotRegionSelection : updateSelectedRegion,
  )
  const [session, setSession] = useState<RegionSelectionSession | undefined>(initialSession)
  const [drag, setDrag] = useState<{startX: number; startY: number; currentX: number; currentY: number} | null>(null)
  const [cursor, setCursor] = useState({x: -1, y: -1})
  const [invalid, setInvalid] = useState(false)
  const [assistCandidate, setAssistCandidate] = useState<RegionSmartCandidate | null>(null)
  const [assistRequestPending, setAssistRequestPending] = useState(false)
  const shellRef = useRef<HTMLElement | null>(null)
  const pointerDownCandidateRef = useRef<RegionSmartCandidate | null>(null)
  const assistRequestSeqRef = useRef(0)
  const assistTimerRef = useRef<number | null>(null)
  const assistCandidateLevelRef = useRef(0)
  const lastAssistPointRef = useRef<{x: number; y: number} | null>(null)
  const handledRightPointerRef = useRef(false)
  const [overlayLocale, setOverlayLocale] = useState<LocaleCode>(navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en')
  const [overlayTheme, setOverlayTheme] = useState<ThemeCode>('night-teal')
  const copy = copyByLocale[overlayLocale]
  const minimumWidth = session?.minimumWidth ?? 64
  const minimumHeight = session?.minimumHeight ?? 64
  const dragDistance = drag ? pointerDistance(drag.startX, drag.startY, drag.currentX, drag.currentY) : 0
  const isManualDrag = Boolean(drag && dragDistance > regionManualDragThreshold)
  const selectedRect = drag && isManualDrag ? normalizedClientRect(drag.startX, drag.startY, drag.currentX, drag.currentY) : null
  const isEditingRegion = editFrame?.mode === 'edit'
  const isRecordingRegion = editFrame?.mode === 'recording'
  const isAnnotationRegionSelection = session?.purpose === 'annotation'
  const isScreenshotRegionSelection = session?.purpose === 'screenshot'
  const isScrollingScreenshotSelection = session?.purpose === 'scrolling-screenshot'
  const sessionCandidates = session?.candidates ?? []
  const visibleAssistCandidate = !isEditingRegion && !isRecordingRegion
    ? selectedRect
      ? bestLocalRegionCandidate(sessionCandidates, selectedRect) ?? assistCandidate
      : assistCandidate
    : null
  const shouldShowCrosshair = !isEditingRegion &&
    !isRecordingRegion &&
    cursor.x >= 0 &&
    !selectedRect &&
    !visibleAssistCandidate &&
    !assistRequestPending
  const overlayOrigin = editFrame?.overlayBounds ?? session?.bounds ?? {x: 0, y: 0, width: 0, height: 0}
  const editableRect = isEditingRegion ? {
    x: editFrame.bounds.x - overlayOrigin.x,
    y: editFrame.bounds.y - overlayOrigin.y,
    width: editFrame.bounds.width,
    height: editFrame.bounds.height,
  } : null
  const recordingRect = isRecordingRegion ? {
    x: editFrame.bounds.x - overlayOrigin.x,
    y: editFrame.bounds.y - overlayOrigin.y,
    width: editFrame.bounds.width,
    height: editFrame.bounds.height,
  } : null

  useEffect(() => {
    document.body.classList.add('rf-region-overlay-window')
    return () => {
      document.body.classList.remove('rf-region-overlay-window')
      if (assistTimerRef.current !== null) {
        window.clearTimeout(assistTimerRef.current)
        assistTimerRef.current = null
      }
    }
  }, [])

  useEffect(() => {
    void loadSettings()
      .then((settings) => {
        setOverlayLocale(normalizeLocale(settings.locale))
        setOverlayTheme(normalizeTheme(settings.window.theme))
      })
      .catch((error) => console.info('Using region overlay language fallback:', error))
  }, [])

  useEffect(() => subscribeSettingsChanged((settings) => {
    setOverlayLocale(normalizeLocale(settings.locale))
    setOverlayTheme(normalizeTheme(settings.window.theme))
  }), [])

  useEffect(() => {
    document.documentElement.dataset.theme = overlayTheme
  }, [overlayTheme])

  useEffect(() => {
    const onSession = (event: Event) => {
      const next = (event as CustomEvent<RegionSelectionSession>).detail
      if (next) {
        assistRequestSeqRef.current += 1
        if (assistTimerRef.current !== null) {
          window.clearTimeout(assistTimerRef.current)
          assistTimerRef.current = null
        }
        delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
        setSession(next)
        setDrag(null)
        setCursor({x: -1, y: -1})
        setInvalid(false)
        setAssistCandidate(null)
        setAssistRequestPending(false)
        pointerDownCandidateRef.current = null
        assistCandidateLevelRef.current = 0
        lastAssistPointRef.current = null
      }
    }
    window.addEventListener('rf-region-session', onSession)
    return () => window.removeEventListener('rf-region-session', onSession)
  }, [])

  const clearRegionSelectionRuntimeState = () => {
    assistRequestSeqRef.current += 1
    if (assistTimerRef.current !== null) {
      window.clearTimeout(assistTimerRef.current)
      assistTimerRef.current = null
    }
    delete (window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}).__RF_REGION_SESSION__
    delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
    clearEditFrame()
    setSession(undefined)
    setDrag(null)
    setCursor({x: -1, y: -1})
    setInvalid(false)
    setAssistCandidate(null)
    setAssistRequestPending(false)
    pointerDownCandidateRef.current = null
    assistCandidateLevelRef.current = 0
    lastAssistPointRef.current = null
  }

  const cancelSelection = async () => {
    clearRegionSelectionRuntimeState()
    await cancelRegionSelector()
    if (!window.navigator.userAgent.includes('Wails')) {
      window.close()
    }
  }

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        if (isEditingRegion) {
          clearEditFrame()
          void (isScreenshotRegionEdit ? cancelRegionSelector() : cancelSelectedRegion())
          return
        }
        void cancelSelection()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [isEditingRegion, isScreenshotRegionEdit])

  const cancelPendingHoverAssist = () => {
    assistRequestSeqRef.current += 1
    setAssistRequestPending(false)
    if (assistTimerRef.current !== null) {
      window.clearTimeout(assistTimerRef.current)
      assistTimerRef.current = null
    }
  }

  const completeSelection = async (rect: RegionSelectionSession['bounds']) => {
    if (rect.width < minimumWidth || rect.height < minimumHeight) {
      setInvalid(true)
      window.setTimeout(() => setInvalid(false), 360)
      return
    }
    try {
      if (isAnnotationRegionSelection) {
        await completeAnnotationRegionSelection(rect)
        return
      }
      if (isScreenshotRegionSelection) {
        await beginScreenshotAnnotationOverlay(rect)
        return
      }
      if (isScrollingScreenshotSelection) {
        await completeScrollingScreenshotSelection(rect)
        return
      }
      await completeRegionSelection(rect)
    } catch (error) {
      console.error('Failed to complete region selection:', error)
      setInvalid(true)
      window.setTimeout(() => setInvalid(false), 360)
      window.alert(readableError(error) || copy.screenshot.captureFailed)
    }
  }

  const assistedSelection = async (rect: RegionSelectionSession['bounds'], point: {x: number; y: number}) => {
    if (!session) return rect
    try {
      const result = await assistRegionSelection({
        sessionId: session.id,
        purpose: session.purpose,
        pointerX: point.x,
        pointerY: point.y,
        selection: rect,
        candidates: sessionCandidates,
      })
      ;(window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__ = result
      return result.best?.bounds ?? rect
    } catch (error) {
      console.info('Region assist failed; using manual selection:', error)
      return rect
    }
  }

  const requestHoverAssistCandidateForSession = (
    activeSession: RegionSelectionSession | undefined,
    point: {x: number; y: number},
    immediate = false,
    allowEditing = false,
  ) => {
    if (!activeSession || isRecordingRegion || (isEditingRegion && !allowEditing)) {
      setAssistRequestPending(false)
      return
    }
    const activeCandidates = activeSession.candidates ?? []
    const requestSeq = assistRequestSeqRef.current + 1
    assistRequestSeqRef.current = requestSeq
    const lastPoint = lastAssistPointRef.current
    if (!lastPoint || pointerDistance(lastPoint.x, lastPoint.y, point.x, point.y) > 10) {
      assistCandidateLevelRef.current = 0
    }
    lastAssistPointRef.current = point
    setAssistRequestPending(true)
    const run = () => {
      assistTimerRef.current = null
      const level = assistCandidateLevelRef.current
      void assistRegionSelection({
        sessionId: activeSession.id,
        purpose: activeSession.purpose,
        pointerX: point.x,
        pointerY: point.y,
        candidateLevel: level,
        candidates: activeCandidates,
      }).then((result) => {
        if (assistRequestSeqRef.current !== requestSeq) return
        ;(window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__ = result
        setAssistCandidate(result.best ?? bestLocalRegionCandidateForPoint(activeCandidates, point))
        setAssistRequestPending(false)
      }).catch((error) => {
        if (assistRequestSeqRef.current !== requestSeq) return
        console.info('Region hover assist failed; using local candidates:', error)
        setAssistCandidate(bestLocalRegionCandidateForPoint(activeCandidates, point))
        setAssistRequestPending(false)
      })
    }
    if (assistTimerRef.current !== null) {
      window.clearTimeout(assistTimerRef.current)
      assistTimerRef.current = null
    }
    if (immediate) {
      run()
      return
    }
    assistTimerRef.current = window.setTimeout(run, 70)
  }

  const requestHoverAssistCandidate = (point: {x: number; y: number}, immediate = false) => {
    requestHoverAssistCandidateForSession(session, point, immediate)
  }

  useEffect(() => {
    if (!session?.initialPointer || isEditingRegion || isRecordingRegion || lastAssistPointRef.current) return
    const point = clampedClientPoint(session.initialPointer.x, session.initialPointer.y)
    setCursor(point)
    setAssistCandidate(bestLocalRegionCandidateForPoint(session.candidates ?? [], point))
    requestHoverAssistCandidateForSession(session, point, true)
  }, [session?.id, isEditingRegion, isRecordingRegion])

  const returnToAutoSelection = (point: {x: number; y: number}, activeSession = session, allowEditing = false) => {
    cancelPendingHoverAssist()
    const activeCandidates = activeSession?.candidates ?? []
    pointerDownCandidateRef.current = null
    setDrag(null)
    setInvalid(false)
    assistCandidateLevelRef.current = 0
    setCursor(point)
    setAssistCandidate(bestLocalRegionCandidateForPoint(activeCandidates, point))
    requestHoverAssistCandidateForSession(activeSession, point, true, allowEditing)
  }

  const showSelectionSessionForPurpose = (purpose: RegionFrameState['purpose'] | undefined) => {
    if (purpose === 'screenshot') return showScreenshotRegionSelector()
    if (purpose === 'scrolling-screenshot') return startScrollingScreenshot()
    if (purpose === 'annotation') return showAnnotationRegionSelector()
    return showRegionSelector()
  }

  const returnEditingRegionToAutoSelection = (point: {x: number; y: number}) => {
    const purpose = editFrame?.purpose
    clearEditFrame()
    cancelPendingHoverAssist()
    pointerDownCandidateRef.current = null
    setDrag(null)
    setInvalid(false)
    assistCandidateLevelRef.current = 0
    lastAssistPointRef.current = null
    setAssistCandidate(null)
    delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
    void showSelectionSessionForPurpose(purpose).then((nextSession) => {
      setSession(nextSession)
      returnToAutoSelection(point, nextSession, true)
    }).catch((error) => {
      console.error('Failed to return region editor to auto selection:', error)
      setInvalid(true)
      window.setTimeout(() => setInvalid(false), 360)
    })
  }

  const cancelRegionFromPointer = (point: {x: number; y: number}) => {
    if (isRecordingRegion) return
    if (isEditingRegion) {
      returnEditingRegionToAutoSelection(point)
      return
    }
    if (drag) {
      returnToAutoSelection(point)
      return
    }
    cancelPendingHoverAssist()
    pointerDownCandidateRef.current = null
    setAssistCandidate(null)
    delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
    void cancelSelection()
  }

  const shellMode = isRecordingRegion ? 'recording' : isEditingRegion ? 'editing' : 'selecting'

  return (
    <main
      ref={shellRef}
      className={`region-overlay-shell ${shellMode}`}
      aria-label={copy.aria.regionOverlay}
      onPointerMove={isEditingRegion ? editDrag.updateEdit : undefined}
      onPointerCancel={isEditingRegion ? editDrag.completeEdit : undefined}
      onPointerMoveCapture={(event) => {
        if (isEditingRegion || isRecordingRegion) return
        const point = clampedClientPoint(event.clientX, event.clientY)
        setCursor(point)
        if (drag) {
          const nextDrag = {...drag, currentX: point.x, currentY: point.y}
          setDrag(nextDrag)
          const nextDistance = pointerDistance(nextDrag.startX, nextDrag.startY, nextDrag.currentX, nextDrag.currentY)
          if (nextDistance > regionManualDragThreshold) {
            setAssistCandidate(bestLocalRegionCandidate(sessionCandidates, normalizedClientRect(nextDrag.startX, nextDrag.startY, nextDrag.currentX, nextDrag.currentY)))
          }
        } else {
          setAssistCandidate(bestLocalRegionCandidateForPoint(sessionCandidates, point))
          requestHoverAssistCandidate(point)
        }
      }}
      onWheel={(event) => {
        if (isEditingRegion || isRecordingRegion || drag) return
        event.preventDefault()
        const point = cursor.x >= 0 ? cursor : clampedClientPoint(event.clientX, event.clientY)
        const direction = event.deltaY > 0 ? -1 : 1
        assistCandidateLevelRef.current = Math.max(0, assistCandidateLevelRef.current + direction)
        requestHoverAssistCandidate(point, true)
      }}
      onMouseDownCapture={(event) => {
        if (isRecordingRegion || event.button !== 2) return
        event.preventDefault()
        event.stopPropagation()
        if (handledRightPointerRef.current) return
        handledRightPointerRef.current = true
        window.setTimeout(() => {
          handledRightPointerRef.current = false
        }, 240)
        cancelRegionFromPointer(clampedClientPoint(event.clientX, event.clientY))
      }}
      onPointerDown={(event) => {
        if (isRecordingRegion) return
        if (event.button === 2) {
          event.preventDefault()
          event.stopPropagation()
          handledRightPointerRef.current = true
          window.setTimeout(() => {
            handledRightPointerRef.current = false
          }, 240)
          cancelRegionFromPointer(clampedClientPoint(event.clientX, event.clientY))
          return
        }
        if (isEditingRegion) return
        if (event.button !== 0) return
        cancelPendingHoverAssist()
        event.currentTarget.setPointerCapture(event.pointerId)
        const point = clampedClientPoint(event.clientX, event.clientY)
        const candidate = assistCandidate && rectContainsPoint(assistCandidate.bounds, point)
          ? assistCandidate
          : bestLocalRegionCandidateForPoint(sessionCandidates, point)
        pointerDownCandidateRef.current = candidate
        setAssistCandidate(candidate)
        setDrag({startX: point.x, startY: point.y, currentX: point.x, currentY: point.y})
        setInvalid(false)
      }}
      onPointerUp={(event) => {
        if (isRecordingRegion) return
        if (isEditingRegion) {
          editDrag.completeEdit(event)
          return
        }
        if (!drag) return
        if (event.currentTarget.hasPointerCapture(event.pointerId)) {
          event.currentTarget.releasePointerCapture(event.pointerId)
        }
        const point = clampedClientPoint(event.clientX, event.clientY)
        const rect = normalizedClientRect(drag.startX, drag.startY, point.x, point.y)
        const distance = pointerDistance(drag.startX, drag.startY, point.x, point.y)
        const clickCandidate = pointerDownCandidateRef.current && distance <= regionManualDragThreshold
          ? pointerDownCandidateRef.current
          : null
        pointerDownCandidateRef.current = null
        setDrag(null)
        setAssistCandidate(null)
        setAssistRequestPending(false)
        assistCandidateLevelRef.current = 0
        if (clickCandidate) {
          void completeSelection(clickCandidate.bounds)
          return
        }
        if (distance <= regionManualDragThreshold) {
          returnToAutoSelection(point)
          return
        }
        void assistedSelection(rect, point).then((nextRect) => completeSelection(nextRect))
      }}
      onContextMenu={(event) => {
        event.preventDefault()
        event.stopPropagation()
        if (handledRightPointerRef.current) {
          handledRightPointerRef.current = false
          return
        }
        cancelRegionFromPointer(clampedClientPoint(event.clientX, event.clientY))
      }}
      onPointerLeave={(event) => {
        if (isEditingRegion || isRecordingRegion) return
        setCursor({x: -1, y: -1})
        cancelPendingHoverAssist()
        pointerDownCandidateRef.current = null
        assistCandidateLevelRef.current = 0
        lastAssistPointRef.current = null
        setAssistCandidate(null)
        delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
      }}
    >
      <div className="region-overlay-scrim" />
      {shouldShowCrosshair && (
        <>
          <div className="region-crosshair horizontal" style={{top: cursor.y}} />
          <div className="region-crosshair vertical" style={{left: cursor.x}} />
        </>
      )}
      {visibleAssistCandidate && (
        <div
          className={`region-smart-candidate ${visibleAssistCandidate.kind}`}
          style={{
            left: visibleAssistCandidate.bounds.x,
            top: visibleAssistCandidate.bounds.y,
            width: visibleAssistCandidate.bounds.width,
            height: visibleAssistCandidate.bounds.height,
          }}
        >
          <span>{regionCandidateLabel(visibleAssistCandidate)}</span>
        </div>
      )}
      {!isEditingRegion && !isRecordingRegion && selectedRect && (
        <div
          className={`region-selection-rect ${invalid ? 'invalid' : ''}`}
          style={{
            left: selectedRect.x,
            top: selectedRect.y,
            width: selectedRect.width,
            height: selectedRect.height,
          }}
        >
          <b className="region-size-badge">
            {selectedRect.width} x {selectedRect.height}
          </b>
          <span className="corner top-left" />
          <span className="corner top-right" />
          <span className="corner bottom-left" />
          <span className="corner bottom-right" />
        </div>
      )}
      {recordingRect && (
        <div
          className="region-recording-frame"
          style={{
            left: recordingRect.x,
            top: recordingRect.y,
            width: recordingRect.width,
            height: recordingRect.height,
          }}
        />
      )}
      {editableRect && (
        <div
          className="region-edit-rect"
          style={{
            left: editableRect.x,
            top: editableRect.y,
            width: editableRect.width,
            height: editableRect.height,
          }}
        >
          <button className="region-edit-move" type="button" aria-label="Move selected region" onPointerDown={(event) => editDrag.beginEdit(event, 'move')}>
            <span />
          </button>
          {regionResizeActions.map((action) => (
            <button
              key={action}
              className={`region-edit-resize ${action}`}
              type="button"
              aria-label={`Resize ${action}`}
              onPointerDown={(event) => editDrag.beginEdit(event, action)}
            />
          ))}
          <div className="region-edit-controls">
            <b>{editableRect.width} x {editableRect.height}</b>
            {isScreenshotRegionEdit && (
              <button
                className="region-edit-confirm"
                type="button"
                aria-label={copy.regionOverlay.saveScreenshot}
                title={copy.regionOverlay.saveScreenshot}
                onPointerDown={(event) => event.stopPropagation()}
                onClick={() => void completeScreenshotRegionSelection(editFrame.bounds)}
              >
                <Check size={16} />
                <span>{copy.regionOverlay.saveScreenshot}</span>
              </button>
            )}
            <button
              type="button"
              aria-label={copy.regionOverlay.cancel}
              title={copy.regionOverlay.cancel}
              onPointerDown={(event) => event.stopPropagation()}
              onClick={() => void (isScreenshotRegionEdit ? cancelRegionSelector() : cancelSelectedRegion())}
            >
              <X size={17} />
            </button>
          </div>
        </div>
      )}
      {!isRecordingRegion && <button
        className="region-cancel-button"
        type="button"
        aria-label={copy.regionOverlay.cancel}
        title={copy.regionOverlay.cancel}
        onPointerDown={(event) => event.stopPropagation()}
        onClick={() => void (isEditingRegion ? (isScreenshotRegionEdit ? cancelRegionSelector() : cancelSelectedRegion()) : cancelSelection())}
      >
        <X size={22} />
      </button>}
      {!isEditingRegion && !isRecordingRegion && <div className="region-overlay-badge" aria-hidden="true">
        <MousePointer2 size={16} />
        <span>{copy.regionOverlay.esc}</span>
      </div>}
    </main>
  )
}

function bestLocalRegionCandidateForPoint(candidates: RegionSmartCandidate[], point: {x: number; y: number}) {
  let best: RegionSmartCandidate | null = null
  let bestScore = -1
  for (const candidate of candidates) {
    if (!rectContainsPoint(candidate.bounds, point)) continue
    const area = Math.max(1, candidate.bounds.width * candidate.bounds.height)
    const score = (candidate.score ?? 0) + localRegionKindWeight(candidate.kind) + 1000000 / area
    if (score > bestScore) {
      best = candidate
      bestScore = score
    }
  }
  return best
}

function bestLocalRegionCandidate(candidates: RegionSmartCandidate[], selection: RegionSelectionSession['bounds']) {
  let best: RegionSmartCandidate | null = null
  let bestScore = -1
  for (const candidate of candidates) {
    const score = localRegionCandidateScore(candidate.bounds, selection) + localRegionKindWeight(candidate.kind)
    if (score > bestScore) {
      best = candidate
      bestScore = score
    }
  }
  return bestScore >= 0.5 ? best : null
}

function localRegionCandidateScore(candidate: RegionSelectionSession['bounds'], selection: RegionSelectionSession['bounds']) {
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

function localRegionKindWeight(kind: RegionSmartCandidate['kind']) {
  if (kind === 'element') return 0.36
  if (kind === 'edge') return 0.16
  if (kind === 'window') return 0.08
  return 0
}

function rectContainsPoint(rect: RegionSelectionSession['bounds'], point: {x: number; y: number}) {
  return rect.width > 0 &&
    rect.height > 0 &&
    point.x >= rect.x &&
    point.x < rect.x + rect.width &&
    point.y >= rect.y &&
    point.y < rect.y + rect.height
}

function pointerDistance(startX: number, startY: number, currentX: number, currentY: number) {
  return Math.hypot(currentX - startX, currentY - startY)
}

function regionCandidateLabel(candidate: RegionSmartCandidate) {
  const label = candidate.label?.trim()
  if (label) return label
  if (candidate.kind === 'window') return 'Window'
  if (candidate.kind === 'screen') return 'Screen'
  if (candidate.kind === 'element') return 'Element'
  if (candidate.kind === 'edge') return 'Edge'
  return 'Target'
}

function SwitchRow({label, checked, disabled = false, onChange}: {label: string; checked: boolean; disabled?: boolean; onChange: (value: boolean) => void}) {
  return (
    <label className={`switch-row ${disabled ? 'is-disabled' : ''}`}>
      <span>{label}</span>
      <input type="checkbox" checked={checked} disabled={disabled} onChange={(event) => onChange(event.target.checked)} />
      <i aria-hidden="true" />
    </label>
  )
}

type SelectMenuOption = {
  value: string
  label: string
  disabled?: boolean
  swatch?: string
}

function SelectMenu({
  id,
  value,
  options,
  disabled = false,
  className = '',
  onChange,
}: {
  id?: string
  value: string
  options: SelectMenuOption[]
  disabled?: boolean
  className?: string
  onChange: (value: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [dropDirection, setDropDirection] = useState<'down' | 'up'>('down')
  const rootRef = useRef<HTMLDivElement | null>(null)
  const openedAtRef = useRef(0)
  const pointerInsideAtRef = useRef(0)
  const pointerInsideRef = useRef(false)
  const floatingSelectTokenRef = useRef(0)
  const selected = options.find((option) => option.value === value) ?? options.find((option) => !option.disabled) ?? options[0]
  const selectOption = (option: SelectMenuOption) => {
    if (option.disabled) return
    onChange(option.value)
    setOpen(false)
  }
  const updateDropDirection = () => {
    const rect = rootRef.current?.getBoundingClientRect()
    if (!rect) return
    const estimatedMenuHeight = Math.min(220, Math.max(44, options.length * 44 + 12))
    const spaceBelow = window.innerHeight - rect.bottom
    const spaceAbove = rect.top
    setDropDirection(spaceBelow < estimatedMenuHeight + 10 && spaceAbove > spaceBelow ? 'up' : 'down')
  }

  useEffect(() => {
    if (!open) return
    if (isWailsDesktopRuntime()) return
    openedAtRef.current = Date.now()
    updateDropDirection()
    const close = () => setOpen(false)
    const markPointerInside = (inside: boolean) => {
      pointerInsideRef.current = inside
      if (inside) pointerInsideAtRef.current = Date.now()
    }
    const onPointerDown = (event: PointerEvent) => {
      if (eventPathContains(event, rootRef.current)) {
        markPointerInside(true)
        return
      }
      markPointerInside(false)
      close()
    }
    const onPointerMove = (event: PointerEvent) => {
      markPointerInside(eventPathContains(event, rootRef.current))
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        close()
      }
    }
    const onVisibilityChange = () => {
      if (document.visibilityState !== 'visible' && !pointerInsideRef.current) close()
    }
    const onWindowBlur = () => {
      window.setTimeout(() => {
        const now = Date.now()
        if (
          pointerInsideRef.current ||
          now - pointerInsideAtRef.current < 900 ||
          now - openedAtRef.current < 360
        ) {
          return
        }
        close()
      }, 140)
    }
    document.addEventListener('pointerdown', onPointerDown, true)
    document.addEventListener('pointermove', onPointerMove, true)
    document.addEventListener('keydown', onKeyDown)
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('resize', updateDropDirection)
    window.addEventListener('blur', onWindowBlur)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown, true)
      document.removeEventListener('pointermove', onPointerMove, true)
      document.removeEventListener('keydown', onKeyDown)
      document.removeEventListener('visibilitychange', onVisibilityChange)
      window.removeEventListener('resize', updateDropDirection)
      window.removeEventListener('blur', onWindowBlur)
      pointerInsideRef.current = false
    }
  }, [open, options.length])

  useEffect(() => subscribeFloatingSelectChosen((event) => {
    if (!isWailsDesktopRuntime()) return
    if (event.token !== floatingSelectTokenRef.current) return
    const option = options.find((candidate) => candidate.value === event.value)
    if (!option || option.disabled) return
    onChange(option.value)
    setOpen(false)
  }), [onChange, options])

  useEffect(() => {
    if (!disabled) return
    setOpen(false)
    if (floatingSelectTokenRef.current) void hideFloatingSelect(floatingSelectTokenRef.current)
  }, [disabled])

  const toggleFloatingSelect = async () => {
    if (!isWailsDesktopRuntime()) {
      if (!open) updateDropDirection()
      setOpen((value) => !value)
      return
    }
    if (open) {
      await hideFloatingSelect(floatingSelectTokenRef.current)
      setOpen(false)
      return
    }
    const root = rootRef.current
    if (!root) return
    const token = floatingSelectTokenRef.current + 1
    floatingSelectTokenRef.current = token
    const anchorWidth = root.getBoundingClientRect().width
    const placement = await resolveFloatingSelectPlacement(root, {
      width: clampNumber(anchorWidth, floatingSelectMinWidth, floatingSelectMaxWidth),
      minWidth: floatingSelectMinWidth,
      maxWidth: floatingSelectMaxWidth,
      maxHeight: floatingSelectMaxHeight,
      optionCount: options.length,
    })
    const parentPanel = await getFloatingPanelState().catch(() => undefined)
    await showFloatingSelect({
      id: id ?? `select-${token}`,
      anchor: placement.anchor,
      bounds: placement.bounds,
      value,
      options: options as FloatingSelectOption[],
      token,
      panelToken: parentPanel?.visible ? parentPanel.token : undefined,
      width: placement.bounds.width,
      maxHeight: placement.bounds.height,
      screenId: placement.screenId,
      direction: placement.direction,
    })
    setOpen(true)
  }

  return (
    <div
      ref={rootRef}
      className={`select-menu ${open ? 'open' : ''} drop-${dropDirection} ${disabled ? 'disabled' : ''} ${className}`}
      onPointerDownCapture={() => {
        pointerInsideRef.current = true
        pointerInsideAtRef.current = Date.now()
      }}
      onPointerMoveCapture={() => {
        pointerInsideRef.current = true
        pointerInsideAtRef.current = Date.now()
      }}
    >
      <button
        id={id}
        type="button"
        className="select-menu-button"
        disabled={disabled || options.length === 0}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => void toggleFloatingSelect()}
      >
        <span className="select-menu-label">
          {selected?.swatch && <i className="select-menu-swatch" style={{background: selected.swatch}} aria-hidden="true" />}
          <span>{selected?.label ?? ''}</span>
        </span>
        <ChevronDown size={16} />
      </button>
      {open && !isWailsDesktopRuntime() && (
        <div className="select-menu-list" role="listbox" aria-labelledby={id}>
          {options.map((option) => (
            <button
              key={option.value}
              type="button"
              role="option"
              className={`select-menu-option ${option.value === value ? 'selected' : ''}`}
              aria-selected={option.value === value}
              disabled={option.disabled}
              onPointerDown={(event) => {
                event.preventDefault()
                event.stopPropagation()
                selectOption(option)
              }}
              onClick={() => {
                selectOption(option)
              }}
            >
              <span className="select-menu-label">
                {option.swatch && <i className="select-menu-swatch" style={{background: option.swatch}} aria-hidden="true" />}
                <span>{option.label}</span>
              </span>
              {option.value === value && <Check size={15} />}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function SourceGroup({title, children}: {title: string; children: ReactNode}) {
  return (
    <section className="source-group" aria-label={title}>
      <div className="source-group-label">{title}</div>
      {children}
    </section>
  )
}

function SourceMenuRow({
  source,
  copy,
  selected,
  actionLabel,
  disabled = false,
  onSelect,
  onPreviewStart,
  onPreviewEnd,
}: {
  source: CaptureSource
  copy: RecorderCopy
  selected: boolean
  actionLabel?: string
  disabled?: boolean
  onSelect: () => void
  onPreviewStart?: () => void
  onPreviewEnd?: () => void
}) {
  const Icon = sourceIcon[source.type]
  return (
    <button
      className={`menu-row ${selected ? 'selected' : ''} ${source.available === false ? 'queued' : ''}`}
      type="button"
      disabled={disabled}
      onClick={onSelect}
      onPointerEnter={disabled ? undefined : onPreviewStart}
      onPointerLeave={disabled ? undefined : onPreviewEnd}
      onFocus={disabled ? undefined : onPreviewStart}
      onBlur={onPreviewEnd}
    >
      <Icon size={18} />
      <span>
        <strong>{sourceName(source, copy)}</strong>
        <small>{sourceMeta(source, copy)}</small>
      </span>
      {actionLabel && <b className="row-action-label">{actionLabel}</b>}
      {selected && <Check size={16} />}
    </button>
  )
}

function statusMessageFromBackend(message: string): StatusMessageState {
  switch (message) {
    case 'Preparing recording package':
      return {key: 'preparing'}
    case 'Preparing audio-only recording package':
      return {key: 'preparing'}
    case 'Recording started':
      return {key: 'started'}
    case 'Audio-only recording started':
      return {key: 'started'}
    case 'Recording paused':
      return {key: 'paused'}
    case 'Recording resumed':
      return {key: 'resumed'}
    case 'Finalizing recording package':
      return {key: 'finalizing'}
    case 'Recording package ready':
      return {key: 'ready'}
    default:
      return {key: 'backendMessage', fallback: message}
  }
}

function formatRecoveryMessage(message: RecoveryMessageState, copy: RecorderCopy) {
  if (message.key === 'recovered' && message.count) return copy.recoveryMessages.recoveredCount(message.count)
  return copy.recoveryMessages[message.key]
}

function formatExportMessage(message: ExportMessageState, copy: RecorderCopy) {
  if (message.fallback) return message.fallback
  if (message.key === 'ready' && message.path) return copy.settings.exportReady(message.path)
  return copy.settings.exportFailed
}

function ExportPlanTimelinePreview({plan, copy}: {plan: RecordingExportPlan | null; copy: RecorderCopy}) {
  const visibleSegments = plan?.annotationsVisible ? plan.annotationSnapshots?.slice(0, 4) ?? [] : []
  const previewKey = visibleSegments.map((segment) => `${segment.relativePath || segment.inputPath}:${segment.startOffsetMs}`).join('|')
  const [previewImages, setPreviewImages] = useState<Record<string, string>>({})

  useEffect(() => {
    setPreviewImages({})
    if (!plan?.packageDir || visibleSegments.length === 0) return
    let cancelled = false
    void (async () => {
      const entries = await Promise.all(visibleSegments.map(async (segment) => {
        const key = annotationSegmentKey(segment)
        const snapshotPath = segment.relativePath || segment.inputPath
        if (!snapshotPath) return null
        const preview = await readAnnotationPreviewImage(plan.packageDir, snapshotPath)
        if (!preview.available || !preview.dataUrl) return null
        return [key, preview.dataUrl] as const
      }))
      if (cancelled) return
      const nextImages: Record<string, string> = {}
      for (const entry of entries) {
        if (entry) nextImages[entry[0]] = entry[1]
      }
      setPreviewImages(nextImages)
    })()
    return () => {
      cancelled = true
    }
  }, [plan?.packageDir, previewKey])

  if (!plan?.annotationsVisible || visibleSegments.length === 0) return null
  const remaining = (plan.annotationSnapshots?.length ?? visibleSegments.length) - visibleSegments.length
  const summary = plan.annotationSummary
  const eventBytes = formatBytes(summary?.eventFileBytes ?? 0)
  const snapshotBytes = formatBytes(summary?.snapshotBytes ?? 0)
  const skipped = summary?.skippedSnapshotCount ?? 0
  return (
    <div className="export-plan-timeline">
      <div className="export-plan-timeline-header">
        <strong>{copy.settings.exportPlanTimelineTitle}</strong>
        <span>{copy.settings.exportPlanTimelineStats(eventBytes, snapshotBytes, skipped)}</span>
        {(summary?.elementKeyframeCount ?? 0) > 0 && (
          <span>
            {copy.settings.exportPlanElementTimelineStats(
              summary?.elementKeyframeCount ?? 0,
              summary?.finalElementCount ?? 0,
              summary?.missingElementPayloads ?? 0,
            )}
          </span>
        )}
      </div>
      <div className="export-plan-segments" aria-label={copy.settings.exportPlanTimelineTitle}>
        {visibleSegments.map((segment, index) => {
          const start = formatMilliseconds(segment.startOffsetMs)
          const end = segment.endOffsetMs && segment.endOffsetMs > segment.startOffsetMs
            ? formatMilliseconds(segment.endOffsetMs)
            : copy.settings.exportPlanOpenEnded
          const size = formatBytes(segment.bytes ?? 0)
          const label = copy.settings.exportPlanSegmentLabel(index + 1, start, end, size)
          const segmentKey = annotationSegmentKey(segment)
          const previewUrl = previewImages[segmentKey]
          return (
            <div className={`export-plan-segment ${previewUrl ? 'has-preview' : ''}`} key={segmentKey}>
              {previewUrl && <img src={previewUrl} alt="" />}
              <span style={{'--segment-start': `${Math.min(92, Math.max(0, segment.startOffsetMs / 1000))}%`} as CSSProperties} />
              <small>{label}</small>
              {segment.relativePath && <em>{segment.relativePath}</em>}
            </div>
          )
        })}
      </div>
      {remaining > 0 && <p>{copy.settings.exportPlanSegmentMore(remaining)}</p>}
    </div>
  )
}

function annotationSegmentKey(segment: NonNullable<RecordingExportPlan['annotationSnapshots']>[number]) {
  return `${segment.relativePath || segment.inputPath}:${segment.startOffsetMs}`
}

function formatExportPlanValue(plan: RecordingExportPlan, copy: RecorderCopy) {
  const pip = plan.pipVisible ? copy.settings.exportPlanPip : copy.settings.exportPlanNoPip
  if (!plan.annotationsVisible) {
    return `${pip} · ${copy.settings.exportPlanAnnotationsOff}`
  }
  const events = plan.annotationSummary?.eventCount ?? 0
  if (plan.annotationTimeline === 'element-pngs' && plan.annotationSnapshots?.length) {
    return `${pip} · ${copy.settings.exportPlanRenderedSegments(plan.annotationSnapshots.length, events)}`
  }
  if (plan.annotationTimeline === 'snapshot-segments' && plan.annotationSnapshots?.length) {
    return `${pip} · ${copy.settings.exportPlanSnapshotSegments(plan.annotationSnapshots.length, events)}`
  }
  return `${pip} · ${copy.settings.exportPlanSnapshotFallback(plan.annotationTimeline || plan.annotationSummary?.mode || '', events)}`
}

function formatExportPlanDetail(plan: RecordingExportPlan, copy: RecorderCopy) {
  const details = [copy.settings.exportPlanOutput(plan.outputPath)]
  if (!plan.annotationsVisible) {
    details.push(copy.settings.exportPlanNoAnnotations)
  } else {
    const start = plan.annotationSummary?.startOffsetMs ?? plan.annotationStartMs ?? 0
    const end = plan.annotationSummary?.endOffsetMs ?? 0
    details.push(copy.settings.exportPlanRange(formatMilliseconds(start), end > 0 ? formatMilliseconds(end) : copy.settings.exportPlanOpenEnded))
  }
  if (plan.warnings.length > 0) {
    details.push(copy.settings.exportPlanWarnings(plan.warnings.length))
  }
  return details.join(' · ')
}

function formatMilliseconds(value: number) {
  const numeric = Number.isFinite(value) ? Math.max(0, value) : 0
  return `${(numeric / 1000).toFixed(1)}s`
}

function formatStorageMessage(message: StorageMessageState, copy: RecorderCopy) {
  if (message.key === 'changed' && message.path) return copy.storageMessages.changedTo(message.path)
  return copy.storageMessages[message.key]
}

function formatSourceSelectionMessage(message: SourceSelectionMessageState, copy: RecorderCopy) {
  if (message.fallback) return message.fallback
  if (message.key === 'regionSelected' && message.width && message.height) {
    return copy.sourceSelectionMessages.regionSelectedSize(message.width, message.height)
  }
  return copy.sourceSelectionMessages[message.key]
}

function sourceTypeLabel(source: CaptureSource, copy: RecorderCopy) {
  return copy.sourceTypes[source.type]
}

function sourceName(source: CaptureSource, copy: RecorderCopy) {
  if (source.type === 'screen') {
    return copy.sourceActions.screenLabel(screenIndex(source))
  }
  return copy.sourceNames[source.id] ?? source.name
}

function sourceMeta(source: CaptureSource, copy: RecorderCopy) {
  if (source.available === false) {
    const reason = source.unavailableReason || copy.sourceUnavailable
    return source.meta ? `${source.meta} · ${reason}` : reason
  }
  return copy.sourceMeta[source.id] ?? source.meta
}

function audioOnlySourceMeta(systemAudio: boolean, microphone: boolean, copy: RecorderCopy) {
  if (systemAudio && microphone) return copy.sourceAudioOnly.systemAndMic
  if (systemAudio) return copy.sourceAudioOnly.systemOnly
  if (microphone) return copy.sourceAudioOnly.micOnly
  return copy.sourceAudioOnly.noAudio
}

function mediaDeviceName(device: MediaDevice, copy: RecorderCopy) {
  return copy.mediaDeviceNames[device.id] ?? device.name
}

function screenshotDisplayName(item: ScreenshotItem) {
  const date = new Date(item.createdAt)
  if (Number.isNaN(date.getTime())) return item.id
  return `${date.toLocaleDateString()} ${date.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'})}`
}

function screenshotMeta(item: ScreenshotItem, copy: RecorderCopy) {
  const size = item.width > 0 && item.height > 0 ? `${item.width} x ${item.height}` : copy.common.status
  const mode = item.mode === 'scrolling'
    ? copy.screenshot.scrolling
    : item.mode === 'full'
      ? copy.screenshot.full
      : item.mode === 'window'
        ? copy.screenshot.window
        : item.mode === 'focused-window'
          ? copy.screenshot.focusedWindow
          : item.mode === 'whiteboard'
            ? copy.whiteboard.open
            : copy.screenshot.region
  const flags = [
    item.fixed ? copy.screenshot.fixed : '',
  ].filter(Boolean)
  return [mode, size, ...flags].join(' · ')
}

function selectPreferredCameraDevice(devices: MediaDevice[] | undefined, preferredId?: string): MediaDevice | undefined {
  if (!devices || devices.length === 0) return undefined
  const preferred = preferredId ? devices.find((device) => device.id === preferredId) : undefined
  if (preferred && isUsableCameraDevice(preferred)) return preferred
  return devices.find(isUsableCameraDevice)
}

function isUsableCameraDevice(device: MediaDevice): boolean {
  return device.available !== false && device.sidecarEligible !== false && Boolean(device.nativeId?.trim())
}

function screenIndex(source: CaptureSource) {
  return source.displayIndex && source.displayIndex > 0 ? source.displayIndex : 1
}

function selectVisibleInitialSource(nextSources: CaptureSource[], lastSourceId: string | undefined, lastSourceType: CaptureSource['type']) {
  const visibleSources = nextSources.filter((source) => source.type !== 'application')
  const fallback = visibleSources.find((source) => source.type === 'screen') ?? visibleSources[0] ?? nextSources[0] ?? sources[0]
  if (lastSourceId) {
    const byID = visibleSources.find((source) => source.id === lastSourceId)
    if (byID) return byID
  }
  if (lastSourceType !== 'application') {
    const byType = visibleSources.find((source) => source.type === lastSourceType)
    if (byType) return byType
  }
  return fallback
}

function fallbackVisibleSource(nextSources: CaptureSource[]) {
  const visibleSources = nextSources.filter((source) => source.type !== 'application' && source.type !== 'region')
  return visibleSources.find((source) => source.type === 'screen') ?? visibleSources[0] ?? sources[0]
}

function capabilityTitle(capability: CaptureCapability, copy: RecorderCopy) {
  return copy.capabilityLabels[capability.id] ?? capability.label
}

function capabilityDetail(capability: CaptureCapability, copy: RecorderCopy) {
  return copy.capabilityDetails[capability.id] ?? capability.reason
}

function formatCapabilityValue(capability: CaptureCapability, copy: RecorderCopy) {
  const permission = copy.capabilityPermissionLabels[capability.permission] ?? capability.permission
  return `${capability.backend} · ${permission}`
}

function preflightStatusForBadge(status: RecordingPreflight['status']): CaptureCapability['status'] {
  if (status === 'ready') return 'available'
  if (status === 'warning') return 'queued'
  return 'blocked'
}

function storageStatusForBadge(status: AppStorageStatus['status']): CaptureCapability['status'] {
  if (status === 'ready') return 'available'
  if (status === 'warning') return 'queued'
  return 'blocked'
}

function formatStorageStatusValue(status: AppStorageStatus, copy: RecorderCopy) {
  const freeSpace = status.freeSpaceKnown ? formatBytes(status.availableBytes) : copy.common.unknownSpace
  return `${copy.storageStatusLabels[status.status]} · ${freeSpace}`
}

function storageStatusDetail(status: AppStorageStatus, copy: RecorderCopy) {
  const minimum = formatBytes(status.minimumRecommendedBytes)
  const writable = status.writable ? copy.storage.writable : copy.storage.notWritable
  const statusDetail = status.status === 'blocked'
    ? copy.storage.blockedDetail
    : status.status === 'warning'
      ? copy.storage.warningDetail
      : copy.storage.readyDetail
  return `${writable}. ${copy.storage.recommendedFreeSpace(minimum)}. ${statusDetail}`
}

function preflightDetail(preflight: RecordingPreflight, copy: RecorderCopy) {
  const issue = preflight.checks.find((check) => check.status === 'blocked') ?? preflight.checks.find((check) => check.status === 'warning')
  const message = copy.preflightMessages[preflight.status]
  if (!issue) return message
  const label = copy.preflightCheckLabels[issue.id] ?? issue.label
  const detail = copy.preflightCheckDetails[issue.id] ?? issue.reason
  return detail ? `${message} ${label}: ${detail}` : message
}

function OcrTranslationSettingsPanel({
  copy,
  translation,
  compact = false,
  onChange,
}: {
  copy: RecorderCopy
  translation: AppSettings['ocr']['translation']
  compact?: boolean
  onChange: (patch: Partial<AppSettings['ocr']['translation']>) => void
}) {
  const normalized = normalizeOcrTranslationSettings(translation)
  const enabled = normalized.provider !== 'disabled'
  const providerOptions = ocrTranslationProviders.map((provider) => ({
    value: provider,
    label: copy.settings.ocrTranslationProviderLabels[provider] ?? provider,
  }))
  const languageOptions = ocrTranslationLanguageOptions.map((language) => ({value: language, label: language === 'auto' ? 'Auto' : language}))
  return (
    <div className={`setting-line ocr-translation-settings ${compact ? 'compact' : ''}`}>
      <span>{copy.settings.ocrTranslation}</span>
      <div className="ocr-translation-grid">
        <SettingSelect
          title={copy.settings.ocrTranslationProvider}
          value={normalized.provider}
          options={providerOptions}
          detail={copy.settings.ocrTranslationDetail}
          onChange={(provider) => {
            const nextProvider = provider === 'deepl' || provider === 'openai-compatible' ? provider : 'disabled'
            onChange({
              provider: nextProvider,
              privacyConfirmed: nextProvider !== 'disabled' ? normalized.privacyConfirmed : false,
              privacyConfirmedAt: nextProvider !== 'disabled' ? normalized.privacyConfirmedAt : '',
            })
          }}
        />
        {enabled && (
          <>
            <SettingTextInput
              title={copy.settings.ocrTranslationBaseUrl}
              value={normalized.baseUrl ?? ''}
              detail={copy.settings.ocrTranslationBaseUrlDetail}
              placeholder={normalized.provider === 'deepl' ? 'https://api-free.deepl.com/v2/translate' : 'https://api.example.com/v1'}
              onCommit={(baseUrl) => onChange({baseUrl})}
            />
            <SettingTextInput
              title={copy.settings.ocrTranslationApiKey}
              value={normalized.apiKey ?? ''}
              detail={copy.settings.ocrTranslationApiKeyDetail}
              inputType="password"
              placeholder={normalized.apiKeySet ? copy.settings.ocrTranslationApiKeySaved : 'sk-...'}
              onCommit={(apiKey) => onChange({apiKey, apiKeySet: Boolean(apiKey.trim())})}
            />
            {normalized.provider === 'openai-compatible' && (
              <SettingTextInput
                title={copy.settings.ocrTranslationModel}
                value={normalized.model ?? ''}
                placeholder="gpt-4o-mini"
                onCommit={(model) => onChange({model})}
              />
            )}
            <SettingSelect
              title={copy.settings.ocrTranslationSourceLanguage}
              value={normalized.sourceLanguage}
              options={languageOptions}
              onChange={(sourceLanguage) => onChange({sourceLanguage})}
            />
            <SettingSelect
              title={copy.settings.ocrTranslationTargetLanguage}
              value={normalized.targetLanguage}
              options={languageOptions.filter((option) => option.value !== 'auto')}
              onChange={(targetLanguage) => onChange({targetLanguage})}
            />
            <SettingToggle
              title={copy.settings.ocrTranslationPrivacy}
              checked={normalized.privacyConfirmed}
              detail={copy.settings.ocrTranslationPrivacyDetail}
              onChange={(privacyConfirmed) => onChange({
                privacyConfirmed,
                privacyConfirmedAt: privacyConfirmed ? new Date().toISOString() : '',
              })}
            />
          </>
        )}
      </div>
    </div>
  )
}

function OcrModelSettings({copy, compact = false}: {copy: RecorderCopy; compact?: boolean}) {
  const [status, setStatus] = useState<OcrStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState('')
  const [packagePath, setPackagePath] = useState('')
  const [message, setMessage] = useState('')
  const [confirmModelId, setConfirmModelId] = useState('')
  const [downloads, setDownloads] = useState<Record<string, OcrModelDownloadSnapshot>>({})

  const refresh = useCallback(async (quiet = false) => {
    if (!quiet) setLoading(true)
    try {
      const [next, nextDownloads] = await Promise.all([
        getOcrStatus(),
        getOcrModelDownloads().catch(() => [] as OcrModelDownloadSnapshot[]),
      ])
      setStatus(next)
      setDownloads(Object.fromEntries(nextDownloads.map((download) => [download.modelId, download])))
      setMessage(next.message || '')
    } catch (error) {
      setMessage(readableError(error) || copy.settings.ocrModelsUnavailable)
    } finally {
      if (!quiet) setLoading(false)
    }
  }, [copy.settings.ocrModelsUnavailable])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    Promise.all([
      getOcrStatus(),
      getOcrModelDownloads().catch(() => [] as OcrModelDownloadSnapshot[]),
    ])
      .then(([next, nextDownloads]) => {
        if (cancelled) return
        setStatus(next)
        setDownloads(Object.fromEntries(nextDownloads.map((download) => [download.modelId, download])))
        setMessage(next.message || '')
      })
      .catch((error) => {
        if (!cancelled) setMessage(readableError(error) || copy.settings.ocrModelsUnavailable)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [copy.settings.ocrModelsUnavailable])

  useEffect(() => subscribeOcrModelDownloadEvents((snapshot) => {
    setDownloads((current) => ({...current, [snapshot.modelId]: snapshot}))
    if (snapshot.status === 'installed') {
      void refresh(true).finally(() => setMessage(copy.settings.ocrModelDownloadInstalled))
    } else if (snapshot.status === 'failed') {
      void refresh(true).finally(() => setMessage(snapshot.error || copy.settings.ocrModelDownloadFailed))
    }
  }), [copy.settings.ocrModelDownloadFailed, copy.settings.ocrModelDownloadInstalled, refresh])

  const runAction = async (key: string, action: () => Promise<void>, success: string) => {
    setBusy(key)
    setMessage('')
    try {
      await action()
      await refresh(true)
      setMessage(success)
    } catch (error) {
      await refresh(true).catch(() => undefined)
      setMessage(readableError(error) || copy.settings.ocrModelActionFailed)
    } finally {
      setBusy('')
    }
  }

  const importPackage = () => {
    const trimmed = packagePath.trim()
    if (!trimmed) {
      setMessage(copy.settings.ocrModelPackageEmpty)
      return
    }
    void runAction('import', async () => {
      await installOcrModelPackage(trimmed)
    }, copy.settings.ocrModelImported)
  }

  const downloadModel = (model: OcrModelInfo) => {
    void runAction(`download:${model.id}`, async () => {
      let downloadable = model
      if (!downloadable.downloadAvailable) {
        const nextStatus = await refreshOcrModelCatalog('')
        setStatus(nextStatus)
        downloadable = nextStatus.models.find((candidate) => candidate.id === model.id) ?? downloadable
      }
      if (!downloadable.downloadAvailable) {
        throw new Error(copy.settings.ocrModelDownloadUnavailable)
      }
      const snapshot = await startOcrModelDownload(downloadable.id)
      setDownloads((current) => ({...current, [model.id]: snapshot}))
    }, copy.settings.ocrModelDownloadQueued)
  }

  const refreshCatalog = () => {
    void runAction('catalog', async () => {
      const next = await refreshOcrModelCatalog('')
      setStatus(next)
    }, copy.settings.ocrModelCatalogRefreshed)
  }

  const cancelDownload = (model: OcrModelInfo) => {
    void runAction(`cancel-download:${model.id}`, async () => {
      const snapshot = await cancelOcrModelDownload(model.id)
      setDownloads((current) => ({...current, [model.id]: snapshot}))
    }, copy.settings.ocrModelDownloadCancelled)
  }

  const models = status?.models ?? []
  const statusValue = loading ? copy.settings.ocrModelsLoading : ocrStatusText(status, copy)
  const disabled = loading || busy !== ''

  return (
    <div className={`setting-line ocr-model-settings ${compact ? 'compact' : ''}`}>
      <span>{copy.settings.ocrModels}</span>
      <div className="setting-value">
        <strong>{statusValue}</strong>
        <button className="setting-action" type="button" disabled={disabled} onClick={() => void refresh()}>
          {copy.settings.ocrModelRefresh}
        </button>
        <button className="setting-action" type="button" disabled={disabled} onClick={refreshCatalog}>
          {busy === 'catalog' ? copy.settings.ocrModelCatalogRefreshing : copy.settings.ocrModelCatalogRefresh}
        </button>
      </div>
      <small>{message || `${copy.settings.ocrModelsDetail} ${copy.settings.ocrModelCatalogDetail}`}</small>
      <div className="ocr-model-package-row">
        <input
          className="setting-control-input"
          value={packagePath}
          placeholder={copy.settings.ocrModelPackagePath}
          onChange={(event) => setPackagePath(event.target.value)}
        />
        <button className="setting-action" type="button" disabled={disabled || packagePath.trim() === ''} onClick={importPackage}>
          {busy === 'import' ? copy.settings.ocrModelImporting : copy.settings.ocrModelPackageImport}
        </button>
      </div>
      <div className="ocr-model-list">
        {models.map((model) => (
          <OcrModelRow
            key={model.id}
            model={model}
            copy={copy}
            busy={busy}
            disabled={disabled}
            download={downloads[model.id]}
            confirming={confirmModelId === model.id}
            onUse={() => {
              setConfirmModelId(model.id)
              setMessage(copy.settings.ocrModelSwitchConfirm(model.name))
            }}
            onConfirmUse={() => void runAction(`use:${model.id}`, async () => {
              await setActiveOcrModel(model.id)
            }, copy.settings.ocrModelActivated).finally(() => setConfirmModelId(''))}
            onCancelUse={() => {
              setConfirmModelId('')
              setMessage(status?.message || '')
            }}
            onRemove={() => void runAction(`remove:${model.id}`, async () => {
              await removeOcrModel(model.id)
            }, copy.settings.ocrModelRemoved)}
            onDownload={() => downloadModel(model)}
            onCancelDownload={() => cancelDownload(model)}
          />
        ))}
      </div>
      {status && (
        <small>
          {copy.settings.ocrModelWorkerStatus}: {status.workerPath || copy.common.notRun}
          {status.runtimeDir ? ` · ${copy.settings.ocrModelRuntime}: ${status.runtimeDir}` : ''}
        </small>
      )}
    </div>
  )
}

function OcrModelRow({
  model,
  copy,
  busy,
  disabled,
  download,
  confirming,
  onUse,
  onConfirmUse,
  onCancelUse,
  onRemove,
  onDownload,
  onCancelDownload,
}: {
  model: OcrModelInfo
  copy: RecorderCopy
  busy: string
  disabled: boolean
  download?: OcrModelDownloadSnapshot
  confirming: boolean
  onUse: () => void
  onConfirmUse: () => void
  onCancelUse: () => void
  onRemove: () => void
  onDownload: () => void
  onCancelDownload: () => void
}) {
  const downloadActive = download?.status === 'queued' || download?.status === 'running'
  const downloadFinished = download?.status === 'installed'
  const downloadFailed = download?.status === 'failed'
  const state = downloadActive ? copy.settings.ocrModelDownloading : ocrModelStateText(model, copy)
  const channel = copy.settings.ocrModelChannelLabels[model.channel] ?? model.channel
  const detail = ocrModelDetail(model, copy)
  const progressText = downloadActive ? ocrModelDownloadProgressText(download) : ''
  return (
    <div className={`ocr-model-row ${model.active ? 'active' : ''} ${downloadActive ? 'downloading' : ''}`}>
      <div>
        <strong>{model.name}</strong>
        <span>{channel} · {model.id}</span>
        {detail && <small>{detail}</small>}
        {!model.installed && !model.downloadAvailable && <small>{copy.settings.ocrModelDownloadUnavailable}</small>}
        {progressText && <small>{progressText}</small>}
        {downloadFinished && !model.installed && <small>{copy.settings.ocrModelDownloadInstalled}</small>}
        {downloadFailed && <small>{download?.error || copy.settings.ocrModelDownloadFailed}</small>}
      </div>
      <div className="ocr-model-actions">
        <b className={`status-badge ${ocrModelBadge(model)}`}>{state}</b>
        {!model.installed && !downloadActive && (
          <button className="setting-action" type="button" disabled={disabled || busy === `download:${model.id}`} onClick={onDownload}>
            {busy === `download:${model.id}` ? copy.settings.ocrModelDownloading : copy.settings.ocrModelDownload}
          </button>
        )}
        {downloadActive && (
          <button className="setting-action" type="button" disabled={busy === `cancel-download:${model.id}`} onClick={onCancelDownload}>
            {busy === `cancel-download:${model.id}` ? copy.settings.ocrModelCancellingDownload : copy.settings.ocrModelCancelDownload}
          </button>
        )}
        {model.installed && model.verified && !model.active && (
          <button className="setting-action" type="button" disabled={disabled || busy === `use:${model.id}`} onClick={onUse}>
            {copy.settings.ocrModelUse}
          </button>
        )}
        {model.installed && !model.active && (
          <button className="setting-action danger" type="button" disabled={disabled || busy === `remove:${model.id}`} onClick={onRemove}>
            {busy === `remove:${model.id}` ? copy.settings.ocrModelRemoving : copy.settings.ocrModelRemove}
          </button>
        )}
      </div>
      {confirming && (
        <div className="ocr-model-confirm" role="alert">
          <small>{copy.settings.ocrModelSwitchRisk}</small>
          <div>
            <button className="setting-action" type="button" disabled={disabled || busy === `use:${model.id}`} onClick={onConfirmUse}>
              {busy === `use:${model.id}` ? copy.settings.ocrModelActivating : copy.settings.ocrModelConfirmUse}
            </button>
            <button className="setting-action" type="button" disabled={busy !== ''} onClick={onCancelUse}>
              {copy.settings.ocrModelCancelUse}
            </button>
          </div>
        </div>
      )}
      {downloadActive && (
        <div className="ocr-model-progress" aria-label={progressText}>
          <span style={{width: `${Math.max(3, Math.min(100, Math.round(download?.percent ?? 0)))}%`}} />
        </div>
      )}
    </div>
  )
}

function ocrStatusText(status: OcrStatus | null, copy: RecorderCopy) {
  if (!status) return copy.settings.ocrModelsUnavailable
  return copy.settings.ocrStatusLabels[status.status] ?? status.status
}

function ocrModelStateText(model: OcrModelInfo, copy: RecorderCopy) {
  if (model.active && model.verified) return copy.settings.ocrModelActive
  if (model.installed && model.verified) return copy.settings.ocrModelVerified
  if (model.installed) return copy.settings.ocrModelInvalid
  return copy.settings.ocrModelMissing
}

function ocrModelBadge(model: OcrModelInfo): CaptureCapability['status'] {
  if (model.active && model.verified) return 'available'
  if (model.installed && model.verified) return 'queued'
  if (model.installed) return 'blocked'
  return 'unsupported'
}

function ocrModelDetail(model: OcrModelInfo, copy: RecorderCopy) {
  const details = [
    model.version,
    model.language.length > 0 ? model.language.join('/') : '',
    !model.installed && model.downloadAvailable && model.downloadBytes ? `${copy.settings.ocrModelDownloadSize}: ${formatBytes(model.downloadBytes)}` : '',
    model.smokeAssetReady ? copy.settings.ocrModelSmokeReady : '',
    model.smokeError || model.verificationError || (model.missingFiles.length > 0 ? `${copy.settings.ocrModelMissing}: ${model.missingFiles.join(', ')}` : ''),
  ].filter(Boolean)
  return details.join(' · ')
}

function ocrModelDownloadProgressText(download?: OcrModelDownloadSnapshot) {
  if (!download) return ''
  const percent = Math.max(0, Math.min(100, Math.round(download.percent || 0)))
  return `${percent}% · ${formatBytes(download.downloadedBytes)} / ${formatBytes(download.totalBytes)}`
}

function SettingLine({
  title,
  value,
  status,
  statusLabel,
  detail,
  actionLabel,
  actionDisabled,
  onAction,
}: {
  title: string
  value: string
  status?: CaptureCapability['status']
  statusLabel?: string
  detail?: string
  actionLabel?: string
  actionDisabled?: boolean
  onAction?: () => void
}) {
  return (
    <div className={`setting-line ${status ? `status-${status}` : ''}`}>
      <span>{title}</span>
      <div className="setting-value">
        <strong>{value}</strong>
        {status && <b className={`status-badge ${status}`}>{statusLabel ?? status}</b>}
        {actionLabel && (
          <button className="setting-action" type="button" disabled={actionDisabled} onClick={onAction}>
            {actionLabel}
          </button>
        )}
      </div>
      {detail && <small>{detail}</small>}
    </div>
  )
}

function SettingSelect({
  title,
  value,
  options,
  detail,
  onChange,
}: {
  title: string
  value: string
  options: SelectMenuOption[]
  detail?: string
  onChange: (value: string) => void
}) {
  return (
    <div className="setting-line setting-control">
      <span>{title}</span>
      <SelectMenu className="setting-control-select" value={value} options={options} onChange={onChange} />
      {detail && <small>{detail}</small>}
    </div>
  )
}

function SettingTextAction({
  title,
  value,
  detail,
  actionLabel,
  actionDisabled,
  onChange,
  onAction,
}: {
  title: string
  value: string
  detail?: string
  actionLabel: string
  actionDisabled?: boolean
  onChange: (value: string) => void
  onAction: () => void
}) {
  return (
    <label className="setting-line setting-control">
      <span>{title}</span>
      <div className="setting-input-row">
        <input
          className="setting-control-input"
          value={value}
          onChange={(event) => onChange(event.target.value)}
        />
        <button className="setting-action" type="button" disabled={actionDisabled} onClick={onAction}>
          {actionLabel}
        </button>
      </div>
      {detail && <small>{detail}</small>}
    </label>
  )
}

function SettingTextInput({
  title,
  value,
  detail,
  placeholder,
  inputType = 'text',
  onCommit,
}: {
  title: string
  value: string
  detail?: string
  placeholder?: string
  inputType?: 'text' | 'password'
  onCommit: (value: string) => void
}) {
  const [draft, setDraft] = useState(value)
  useEffect(() => setDraft(value), [value])
  const commit = () => {
    if (draft.trim() !== value.trim()) onCommit(draft.trim())
  }
  return (
    <label className="setting-line setting-control">
      <span>{title}</span>
      <input
        className="setting-control-input"
        value={draft}
        type={inputType}
        placeholder={placeholder}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={commit}
        onKeyDown={(event) => {
          if (event.key === 'Enter') {
            event.preventDefault()
            event.currentTarget.blur()
          }
        }}
      />
      {detail && <small>{detail}</small>}
    </label>
  )
}

function SettingShortcut({
  title,
  value,
  detail,
  actionLabel,
  capturing,
  onStart,
  onCancel,
  onCapture,
}: {
  title: string
  value: string
  detail?: string
  actionLabel: string
  capturing: boolean
  onStart: () => void
  onCancel: () => void
  onCapture: (accelerator: string) => void
}) {
  const buttonRef = useRef<HTMLButtonElement | null>(null)
  useEffect(() => {
    if (capturing) buttonRef.current?.focus()
  }, [capturing])
  return (
    <div className={`setting-line setting-shortcut ${capturing ? 'is-capturing' : ''}`}>
      <span>{title}</span>
      <div className="setting-shortcut-row">
        <kbd>{capturing ? '...' : value}</kbd>
        <button
          ref={buttonRef}
          className="setting-action shortcut-capture-button"
          type="button"
          onClick={capturing ? onCancel : onStart}
          onKeyDown={(event) => {
            if (!capturing) return
            event.preventDefault()
            event.stopPropagation()
            if (event.key === 'Escape') {
              onCancel()
              return
            }
            const accelerator = shortcutFromKeyboardEvent(event.nativeEvent)
            if (accelerator) onCapture(accelerator)
          }}
        >
          {actionLabel}
        </button>
      </div>
      {detail && <small>{detail}</small>}
    </div>
  )
}

function SettingToggle({title, checked, detail, onChange}: {title: string; checked: boolean; detail?: string; onChange: (value: boolean) => void}) {
  return (
    <div className="setting-line setting-control">
      <SwitchRow label={title} checked={checked} onChange={onChange} />
      {detail && <small>{detail}</small>}
    </div>
  )
}

function shortcutFromKeyboardEvent(event: KeyboardEvent): string | null {
  const key = normalizeKeyboardShortcutKey(event.key)
  if (!key) return null
  const modifiers: string[] = []
  const macLike = isMacLikePlatform()
  if (event.metaKey || (!macLike && event.ctrlKey)) modifiers.push('CmdOrCtrl')
  if (macLike && event.ctrlKey) modifiers.push('Ctrl')
  if (event.altKey) modifiers.push('OptionOrAlt')
  if (event.shiftKey) modifiers.push('Shift')
  const uniqueModifiers = Array.from(new Set(modifiers))
  if (uniqueModifiers.length === 0) return null
  if (uniqueModifiers.length === 1 && uniqueModifiers[0] === 'Shift' && isPrintableShortcutKey(key)) return null
  return [...uniqueModifiers, key].join('+')
}

function normalizeKeyboardShortcutKey(key: string): string | null {
  if (!key || key === 'Control' || key === 'Shift' || key === 'Alt' || key === 'Meta') return null
  if (key === ' ') return 'Space'
  if (key === '+') return 'Plus'
  if (key.length === 1) return key.toUpperCase()
  if (key.startsWith('Arrow')) return key.replace('Arrow', '')
  if (/^F\d{1,2}$/i.test(key)) return key.toUpperCase()
  const aliases: Record<string, string> = {
    Escape: 'Escape',
    Esc: 'Escape',
    Enter: 'Enter',
    Return: 'Enter',
    Backspace: 'Backspace',
    Delete: 'Delete',
    Tab: 'Tab',
    Home: 'Home',
    End: 'End',
    PageUp: 'Page Up',
    PageDown: 'Page Down',
  }
  return aliases[key] ?? null
}

function isMacLikePlatform(): boolean {
  const platform = navigator.platform?.toLowerCase() ?? ''
  return platform.includes('mac') || /mac os|iphone|ipad/.test(navigator.userAgent.toLowerCase())
}

function isPrintableShortcutKey(key: string): boolean {
  return key.length === 1 || key === 'Space' || key === 'Plus'
}

function formatShortcutForDisplay(shortcut: string): string {
  return shortcut
    .split('+')
    .filter(Boolean)
    .map((part) => {
      if (part === 'CmdOrCtrl') return isMacLikePlatform() ? 'Cmd' : 'Ctrl'
      if (part === 'OptionOrAlt') return isMacLikePlatform() ? 'Option' : 'Alt'
      return part
    })
    .join(' + ')
}

function shortcutIdentity(shortcut: string): string {
  return shortcut
    .split('+')
    .filter(Boolean)
    .map((part, index, parts) => {
      if (index === parts.length - 1) return part.toLowerCase()
      if (part === 'CmdOrCtrl') return isMacLikePlatform() ? 'cmd' : 'ctrl'
      if (part === 'OptionOrAlt') return 'alt'
      return part.toLowerCase()
    })
    .sort()
    .join('+')
}

export default App
