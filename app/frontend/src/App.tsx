import {joinDisplayPath} from './components/panelShared'
import {eventPathContains, shouldUseFloatingPanelWindows, sourceIcon} from './components/panelShared'
import {ExportPlanTimelinePreview} from './components/ExportPlanTimelinePreview'
import {OcrModelSettings, OcrTranslationSettingsPanel} from './components/OcrSettingsPanels'
import {ScreenshotHistoryRow} from './components/ScreenshotHistoryRow'
import {SelectMenu, SettingLine, SettingSelect, SettingShortcut, SettingTextAction, SettingToggle, SourceGroup, SourceMenuRow, SwitchRow, formatShortcutForDisplay, shortcutIdentity} from './components/controls'
import {ExportMessageState, RecoveryMessageState, SourceSelectionMessageState, StatusMessageState, StorageMessageState, audioOnlySourceMeta, capabilityDetail, capabilityTitle, countdownOptions, ensureVisiblePipConfig, fallbackVisibleSource, floatingPanelSizes, formatCapabilityValue, formatExportMessage, formatExportPlanDetail, formatExportPlanValue, formatPipScalePercent, formatRecoveryMessage, formatSourceSelectionMessage, formatStorageMessage, formatStorageStatusValue, fpsOptions, isUsableCameraDevice, mediaDeviceName, normalizeRecordingQuality, pipPresetOptions, pipShapeOptions, preflightDetail, preflightStatusForBadge, recordingQualityOptions, selectPreferredCameraDevice, selectVisibleInitialSource, sourceName, sourceTypeLabel, statusMessageFromBackend, storageStatusDetail, storageStatusForBadge} from './components/panelShared'
import {
  AppWindow,
  Camera,
  Check,

  ChevronDown,
  ChevronLeft,
  CircleDot,
  Copy as 
  Crosshair,

  FolderOpen,
  Gauge,
  Globe2,
  History,
  Image as ImageIcon,


  Maximize2,
  Minus,


  Pause,
  PenLine,
  Play,

  Settings,
  ScrollText,
  Square,

  Video,
  Volume2,
  Wand2,
  X,

} from 'lucide-react'
import {Suspense, lazy, useEffect, useLayoutEffect, useMemo, useRef, useState} from 'react'
import {copyByLocale} from './i18n'
import {
  cameraDevices,
  defaultSettings,
  fallbackAppData,
  localeOptions,
  normalizeLocale,
  defaultPipPosition, normalizeOcrTranslationSettings, normalizePipConfig, normalizePipPreset, 
  pipDefaultScale, pipMaximumScale, pipMinimumScale,
  normalizeTheme,
  shortcutActions,
  sources,
  systemAudioDevices,
      type AppDataInfo,
  type AppSettings,
  type AppStorageStatus,
  type AudioOnlyRecordingRequest,
  type CaptureCapabilities,
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
  fallbackStorageStatus } from './services/mockBackend'
import {captureScreenshot, deleteScreenshotItem, exportRecordingPackage, getFloatingPanelState, getSourceState, hideAnnotationOverlay, hideCapsuleWindow, hideFloatingPanel, hidePipOverlay, hideRegionFrame, hideScreenIndicator, hideSettingsWindow, hideWhiteboardWindow, isWailsDesktopRuntime, listScreenshots, loadBootstrap, loadSettings, logClientEvent, minimizeApplication, openOcrResult, openRecordingPackage, openScreenshotDirectory, openScreenshotInWhiteboard, openVideoDirectory, patchAudioState, patchCameraState, patchScreenshotItem, patchSettingsPreferences, patchShortcutSettings, patchSourceState, patchWhiteboardSettings, pauseRecording, preflightAudioOnlyRecording, preflightRecording, previewExportRecordingPackage, queueRecognizeScreenshot, quitApplication, recoverRecordingPackage, restoreCapsuleWindow, resumeRecording, saveSettings, setCapsuleWindowExpanded, setCapsuleWindowHitRegions, setDataRoot, showAnnotationOverlay, showAnnotationRegionSelector, showFloatingPanel, showPinnedScreenshot, showPipOverlay, showRegionSelector, showScreenIndicator, showScreenshotRegionSelector, showWhiteboardWindow, snapCapsuleWindowToEdge, startAudioOnlyRecording, startRecording, startScrollingScreenshot, stopMicrophoneLevelMonitor, stopRecording, subscribeAudioState, subscribeCapsuleDockSide, subscribeCapsuleWindowMoveEnded, subscribeFloatingPanelChanged, subscribeOcrJobEvents, subscribeRecordingStatus, subscribeRegionSelection, subscribeScreenshotCaptured, subscribeScreenshotHistoryChanged, subscribeSettingsChanged, subscribeShortcutTriggered, subscribeSourceStateChanged, subscribeWhiteboardVisibility, updatePipOverlay, type AudioControlState, type AudioStatePatch, type CapsuleWindowDockSide, type CapsuleWindowExpandDirection, type CapsuleWindowHitRegion, type FloatingPanelKind, type FloatingPanelState, type PIPOverlayCamera, type RecordingExportPlan, type RecordingRecovery, type RecordingStatusUpdate, type SettingsPreferencesPatch, type ShortcutSettingsPatch, type SourceControlState, type WhiteboardSettingsPatch, type WhiteboardVisibilityUpdate} from './services/recorderBackend'
import {resolveFloatingPanelPlacement} from './components/floating/floatingPosition'
import {elementHitRegion} from './components/hitRegion'
import {isRecordingPackagePath, packageDisplayName} from './components/panelShared'
import {applyTheme, themeSelectOptions} from './components/themeOptions'
import {copyOcrResultText, translateAndCopyOcrResultText} from './components/screenshotShared'
import {readableError} from './services/recorderBackend'
import {MicMeter} from './components/MicMeter'
import {RecordingClock} from './components/RecordingClock'
import {ocrPanelContext} from './components/floating/ocrResultPanel'
import {} from './utils/clipboard'

const AnnotationOverlayWindow = lazy(() => import('./AnnotationOverlayWindow'))
const AnnotationRenderWindow = lazy(() => import('./AnnotationRenderWindow'))
const WhiteboardWindow = lazy(() => import('./WhiteboardWindow'))
const ScreenIndicatorWindow = lazy(() => import('./ScreenIndicatorWindow'))
const FloatingSelectWindow = lazy(() => import('./FloatingSelectWindow'))
const ScreenshotPinWindow = lazy(() => import('./ScreenshotPinWindow'))
const PIPOverlayWindow = lazy(() => import('./PIPOverlayWindow'))
const RegionOverlayWindow = lazy(() => import('./RegionOverlayWindow'))
const FloatingPanelWindow = lazy(() => import('./FloatingPanelWindow'))

// Warm the per-window lazy chunks once the primary shell is up, so opening a
// secondary window (region overlay, floating panel, ...) never waits on the
// network. Code splitting still keeps the first paint small.
const windowChunkWarmups: Array<() => Promise<unknown>> = [
  () => import('./ScreenIndicatorWindow'),
  () => import('./FloatingSelectWindow'),
  () => import('./ScreenshotPinWindow'),
  () => import('./PIPOverlayWindow'),
  () => import('./RegionOverlayWindow'),
  () => import('./FloatingPanelWindow'),
]

const previewPackagePath = 'data/video/recording-preview.rfrec'
type ActivePanel = 'source' | 'audio' | 'camera' | 'language' | 'board' | 'screenshot-paste'

function annotationCapturePolicy(includeAnnotations: boolean): AppSettings['whiteboard']['capturePolicy'] {
  return includeAnnotations ? 'export-compose' : 'preview-only'
}

function whiteboardLaunchMode(state: RecordingState, recordingMode: RecordingMode): 'whiteboard' | 'annotation' {
  return (state === 'recording' || state === 'paused') && recordingMode === 'video' ? 'annotation' : 'whiteboard'
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
    return (
      <Suspense fallback={null}>
        <FloatingSelectWindow />
      </Suspense>
    )
  }
  if (isScreenIndicatorWindow) {
    return (
      <Suspense fallback={null}>
        <ScreenIndicatorWindow />
      </Suspense>
    )
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
    return (
      <Suspense fallback={null}>
        <RegionOverlayWindow />
      </Suspense>
    )
  }
  if (isPipOverlayWindow) {
    return (
      <Suspense fallback={null}>
        <PIPOverlayWindow />
      </Suspense>
    )
  }
  if (isScreenshotPinWindow) {
    return (
      <Suspense fallback={null}>
        <ScreenshotPinWindow />
      </Suspense>
    )
  }
  if (isWhiteboardWindow) {
    return (
      <Suspense fallback={<main className="whiteboard-loading"><span>Loading whiteboard</span></main>}>
        <WhiteboardWindow />
      </Suspense>
    )
  }
  if (isFloatingPanelWindow) {
    return (
      <Suspense fallback={null}>
        <FloatingPanelWindow />
      </Suspense>
    )
  }

  const [selectedSource, setSelectedSource] = useState<CaptureSource>(sources[0])
  const [backendNotice, setBackendNotice] = useState<string | null>(null)
  const [availableSources, setAvailableSources] = useState<CaptureSource[]>(sources)
  const [activePanel, setActivePanel] = useState<ActivePanel | null>(null)
  const [sourcePickerView, setSourcePickerView] = useState<'overview' | 'windows'>('overview')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [whiteboardVisibility, setWhiteboardVisibility] = useState<WhiteboardVisibilityUpdate | null>(null)
  const [closePromptOpen, setClosePromptOpen] = useState(false)
  const [closeBusy, setCloseBusy] = useState(false)
  const [recordingMode, setRecordingMode] = useState<RecordingMode>('video')
  const [state, setState] = useState<RecordingState>('idle')
  const [clockEpoch, setClockEpoch] = useState(0)
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
    openScrollingScreenshot: () => undefined,
    pasteImage: () => undefined })
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
      edgeFeather: pipEdgeFeather }, pipPreset)
    const savedPipConfig = camera ? ensureVisiblePipConfig(rawPipConfig) : rawPipConfig
    const nextSettings = {
      schemaVersion: defaultSettings.schemaVersion,
      locale,
      source: {
        lastSourceId: selectedSource.id,
        lastSourceType: selectedSource.type },
      storage: {
        dataRootDir: appData.rootDir },
      recording: {
        quality: recordingQuality,
        fps: recordingFPS,
        captureCursor,
        countdownSeconds },
      audio: persistedSettingsRef.current?.audio ?? {
        system: systemAudio,
        systemDeviceId: selectedSystemAudio || undefined,
        microphone,
        microphoneDeviceId: selectedMic || undefined,
        noiseSuppression,
        microphoneGain: 1 },
      camera: {
        enabled: camera,
        deviceId: selectedCamera,
        pipPreset: savedPipConfig.preset,
        pip: savedPipConfig },
      whiteboard: {
        ...(persistedSettingsRef.current?.whiteboard ?? defaultSettings.whiteboard),
        capturePolicy: annotationCapturePolicy(includeAnnotationsInExport) },
      ocr: {
        ...(persistedSettingsRef.current?.ocr ?? defaultSettings.ocr),
        autoRecognizeScreenshots,
        translation: ocrTranslation },
      shortcuts,
      window: {
        minimizeToTray: true,
        theme,
        startAtLogin } }
    if (!isSettingsWindow || !persistedSettingsRef.current) return nextSettings
    return {
      ...persistedSettingsRef.current,
      schemaVersion: defaultSettings.schemaVersion,
      locale,
      storage: {
        dataRootDir: appData.rootDir },
      recording: {
        quality: recordingQuality,
        fps: recordingFPS,
        captureCursor,
        countdownSeconds },
      whiteboard: {
        ...persistedSettingsRef.current.whiteboard,
        capturePolicy: annotationCapturePolicy(includeAnnotationsInExport) },
      ocr: {
        ...persistedSettingsRef.current.ocr,
        autoRecognizeScreenshots,
        translation: ocrTranslation },
      shortcuts,
      window: {
        minimizeToTray: true,
        theme,
        startAtLogin } }
  }, [appData.rootDir, autoRecognizeScreenshots, camera, captureCursor, countdownSeconds, includeAnnotationsInExport, isSettingsWindow, locale, microphone, noiseSuppression, ocrTranslation, pipEdgeFeather, pipMirror, pipPosition, pipPreset, pipScale, pipShape, recordingFPS, recordingQuality, selectedCamera, selectedMic, selectedSource.id, selectedSource.type, selectedSystemAudio, shortcuts, startAtLogin, systemAudio, theme])
  const settingsAutosaveKey = useMemo(() => JSON.stringify({
    locale,
    sourceId: selectedSource.id,
    sourceType: selectedSource.type,
    dataRootDir: appData.rootDir }), [appData.rootDir, locale, selectedSource.id, selectedSource.type])
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
    edgeFeather: pipEdgeFeather }, pipPreset), [pipEdgeFeather, pipMirror, pipPosition, pipPreset, pipScale, pipShape])
  const pipCameraLabel = selectedCameraDevice?.name || selectedCamera
  const pipCameraTarget = useMemo<PIPOverlayCamera>(() => ({
    deviceId: selectedCameraDevice?.id || selectedCamera,
    nativeId: selectedCameraDevice?.nativeId,
    name: pipCameraLabel }), [pipCameraLabel, selectedCamera, selectedCameraDevice?.id, selectedCameraDevice?.nativeId])

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
        pip: normalizedPip } }
    currentSettingsRef.current = nextSettings
    persistedSettingsRef.current = nextSettings
    void patchCameraState({
      enabled,
      deviceId,
      pipPreset: normalizedPip.preset,
      pip: normalizedPip })
      .then((saved) => {
        currentSettingsRef.current = saved
        persistedSettingsRef.current = saved
        applySettingsState(saved, undefined, undefined, {
          preserveCameraEnabled: hasLocalCameraIntent(),
          preserveCameraSelection: hasLocalCameraIntent(),
          preservePipConfig: hasLocalPipIntent() })
      })
      .catch((error) => console.error('Failed to persist camera settings:', error))
  }
  const commitPipConfigFromPanel = (patch: Partial<PIPConfig>) => {
    if (!cameraRef.current || !hasUsableCamera || recordingMode !== 'video') return
    const nextConfig = ensureVisiblePipConfig(normalizePipConfig({
      ...currentPipConfig,
      ...patch,
      position: patch.position ?? currentPipConfig.position }, (patch.preset as PIPPreset | undefined) ?? currentPipConfig.preset))
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
            pip: state.config } }
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
      selectedCameraName: selectedCameraDevice?.name ?? '' })
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
      setClockEpoch((value) => value + 1)
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
      locale: patch.locale ?? settings.locale,
      recording: {
        ...settings.recording,
        quality: patch.recordingQuality ?? settings.recording.quality,
        fps: patch.recordingFps ?? settings.recording.fps,
        captureCursor: patch.captureCursor ?? settings.recording.captureCursor,
        countdownSeconds: patch.countdownSeconds ?? settings.recording.countdownSeconds },
      window: {
        ...settings.window,
        theme: patch.theme ?? settings.window.theme,
        startAtLogin: patch.startAtLogin ?? settings.window.startAtLogin },
      ocr: {
        ...settings.ocr,
        autoRecognizeScreenshots: patch.autoOcr ?? settings.ocr.autoRecognizeScreenshots,
        translation: normalizeOcrTranslationSettings({
          ...settings.ocr.translation,
          ...(patch.ocrTranslation ?? {}) }) } }
  }
  const applyLocalPreferencePatch = (patch: SettingsPreferencesPatch) => {
    if (patch.locale !== undefined) setLocale(normalizeLocale(patch.locale))
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
        ...patch.ocrTranslation }) }
  }
  const commitSettingsPreferencePatch = (rawPatch: SettingsPreferencesPatch) => {
    const patch = completeSettingsPreferencePatch(rawPatch)
    markLocalPreferenceIntent()
    const token = preferencePatchTokenRef.current + 1
    preferencePatchTokenRef.current = token
    applyLocalPreferencePatch(patch)
    void logClientEvent('settings-preferences', 'patch-request', {
      locale: patch.locale ?? '',
      theme: patch.theme ?? '',
      startAtLogin: patch.startAtLogin ?? '',
      recordingQuality: patch.recordingQuality ?? '',
      recordingFps: patch.recordingFps ?? '',
      captureCursor: patch.captureCursor ?? '',
      countdownSeconds: patch.countdownSeconds ?? '',
      autoOcr: patch.autoOcr ?? '',
      ocrTranslationProvider: patch.ocrTranslation?.provider ?? '',
      ocrTranslationApiKeySet: patch.ocrTranslation?.apiKey !== undefined ? String(Boolean(patch.ocrTranslation.apiKey)) : '' })
    void patchSettingsPreferences(patch)
      .then((settings) => {
        if (token !== preferencePatchTokenRef.current) return
        localPreferenceIntentUntilRef.current = Date.now() + 3000
        void logClientEvent('settings-preferences', 'patch-success', {
          locale: settings.locale,
          theme: settings.window.theme,
          startAtLogin: settings.window.startAtLogin,
          recordingQuality: settings.recording.quality,
          recordingFps: settings.recording.fps,
          captureCursor: settings.recording.captureCursor,
          countdownSeconds: settings.recording.countdownSeconds,
          autoOcr: settings.ocr.autoRecognizeScreenshots,
          ocrTranslationProvider: settings.ocr.translation.provider,
          ocrTranslationApiKeySet: Boolean(settings.ocr.translation.apiKey || settings.ocr.translation.apiKeySet) })
        applySettingsState(settings, undefined, undefined, {
          preserveRecordingSettings: hasLocalPreferenceIntent(),
          preserveTheme: hasLocalPreferenceIntent(),
          preserveOcr: hasLocalPreferenceIntent() })
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
        ...patch } }
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
      lastTool: patch.lastTool ?? '' })
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
        ...patch } }
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
      accelerator })
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
          noiseSuppression: audio.noiseSuppression,
          microphoneGain: audio.microphoneGain || 1 } }
    }
    currentSettingsRef.current = apply(currentSettingsRef.current)
    persistedSettingsRef.current = apply(persistedSettingsRef.current)
  }
  const applyAudioControlState = (audio: AudioControlState) => {
    const nextNoiseSuppression = audio.noiseSuppression
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
    mergeAudioIntoSettingsCache(audio)
  }
  const optimisticAudioState = (patch: AudioStatePatch): AudioControlState => {
    const nextMicrophone = patch.microphone ?? microphoneRef.current
    const nextNoiseSuppression = patch.noiseSuppression ?? noiseSuppressionRef.current
    return {
      system: patch.system ?? systemAudioRef.current,
      systemDeviceId: patch.clearSystemDevice ? undefined : (patch.systemDeviceId ?? selectedSystemAudioRef.current),
      microphone: nextMicrophone,
      microphoneDeviceId: patch.clearMicrophoneDevice ? undefined : (patch.microphoneDeviceId ?? selectedMicRef.current),
      noiseSuppression: nextNoiseSuppression,
      microphoneGain: patch.microphoneGain ?? currentSettingsRef.current?.audio.microphoneGain ?? 1 }
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
            microphoneGain: settings.audio.microphoneGain }))
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
              startAtLogin: currentSettingsRef.current.window.startAtLogin }
          : effectiveSettings.window,
        ocr: options.preserveOcr
          ? currentSettingsRef.current.ocr
          : effectiveSettings.ocr,
        camera: options.preservePipConfig
          ? {
              ...effectiveSettings.camera,
              pipPreset: currentSettingsRef.current.camera.pipPreset,
              pip: currentSettingsRef.current.camera.pip }
          : effectiveSettings.camera,
        whiteboard: options.preserveWhiteboard
          ? currentSettingsRef.current.whiteboard
          : effectiveSettings.whiteboard }
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
      ? noiseSuppressionRef.current
      : effectiveSettings.audio.noiseSuppression
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
    applyTheme(theme)
  }, [theme])

  useEffect(() => {
    let cancelled = false
    const warm = () => {
      if (cancelled) return
      for (const load of windowChunkWarmups) {
        void load().catch(() => {})
      }
    }
    const idle = (window as Window & {requestIdleCallback?: (cb: () => void, opts?: {timeout: number}) => number}).requestIdleCallback
    const handle = idle ? idle(warm, {timeout: 2000}) : window.setTimeout(warm, 1500)
    return () => {
      cancelled = true
      const cancel = (window as Window & {cancelIdleCallback?: (handle: number) => void}).cancelIdleCallback
      if (cancel && typeof handle === 'number') {
        cancel(handle)
      } else {
        window.clearTimeout(handle)
      }
    }
  }, [])

  useEffect(() => {
    const handler = (event: Event) => {
      const detail = (event as CustomEvent<{operation?: string; message?: string}>).detail
      const message = typeof detail?.message === 'string' ? detail.message.trim() : ''
      if (message) {
        setBackendNotice(message)
      }
    }
    window.addEventListener('rf-backend-error', handler)
    return () => window.removeEventListener('rf-backend-error', handler)
  }, [])

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
        // In the browser runtime the floating panel does not exist; applying
        // its initial empty state would close inline panels the user opened
        // while the mock resolution was still in flight.
        if (!isWailsDesktopRuntime()) return
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
        nativeId: sourceState.sourceGeometry.nativeId }
    })
  }

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
        regions }
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
      attributeFilter: ['class', 'style'] })
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
    let pointerFallbackTimer = 0

    const clearTimers = () => {
      if (settleTimer) window.clearTimeout(settleTimer)
      if (forceTimer) window.clearTimeout(forceTimer)
      if (pointerFallbackTimer) window.clearTimeout(pointerFallbackTimer)
      settleTimer = 0
      forceTimer = 0
      pointerFallbackTimer = 0
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
        capsuleDragPendingUntilRef.current = Date.now() + (isWailsDesktopRuntime() ? 900 : 0)
        if (!isWailsDesktopRuntime()) {
          capsuleDragPendingUntilRef.current = 0
          stabilize('pointer-end')
          return
        }
        pointerFallbackTimer = window.setTimeout(() => {
          pointerFallbackTimer = 0
          if (disposed || Date.now() > capsuleDragPendingUntilRef.current) return
          capsuleDragPendingUntilRef.current = 0
          stabilize('pointer-end')
        }, 520)
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
      if (!capsuleDragCandidateRef.current && !capsuleDragObservedMoveRef.current && !dragPending) return
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
      selectedCameraName: selectedCameraDevice?.name ?? '' })
    void showPipOverlay(ensureVisiblePipConfig(currentPipConfig), 'edit', pipCameraTarget)
      .then(() => {
        if (generation !== cameraPreviewGenerationRef.current || !cameraRef.current || recordingMode !== 'video') {
          void logClientEvent('camera', 'preview-show-cancelled-after-show', {
            generation,
            currentGeneration: cameraPreviewGenerationRef.current,
            camera: cameraRef.current,
            recordingMode })
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
      noiseSuppression: audio.noiseSuppression })
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
      preservePipConfig })
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
      preserveWhiteboard })
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
        nativeId: pickedSource.nativeId } })
    setSourceSelectionMessage({
      key: 'regionSelected',
      width: result.geometry?.width ?? pickedSource.width,
      height: result.geometry?.height ?? pickedSource.height })
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

  const currentRecordingProfile = () => ({
    quality: recordingQuality,
    fps: recordingFPS,
    captureCursor,
    countdownSeconds })

  const buildAudioOnlyRequest = (): AudioOnlyRecordingRequest => ({
    recording: currentRecordingProfile(),
    systemAudio,
    systemAudioDeviceId: selectedSystemAudio || undefined,
    microphone,
    microphoneDeviceId: selectedMic || undefined,
    noiseSuppression: microphone && noiseSuppression })

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
      noiseSuppression: microphone && noiseSuppression,
      camera: requestCameraEnabled,
      cameraDeviceId: requestCameraDevice?.id ?? requestedCameraId,
      cameraDeviceNativeId: requestCameraDevice?.nativeId,
      pipPreset: requestCameraEnabled ? requestPip.preset : 'off',
      pip: requestPip }
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
    setClockEpoch((value) => value + 1)
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
          session })
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
        systemAudio: request.systemAudio })
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
        session })
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
          session })
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
      minWidth: size.minWidth })
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
      contextId })
  }

  const togglePanel = (panel: Exclude<ActivePanel, 'screenshot-paste'>, anchorElement?: Element) => {
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
        recordingMode }, message)
    }
    await showAnnotationRegionSelector()
  }

  const launchWhiteboard = () => {
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
          message: error instanceof Error ? error.message : String(error) })
        await showWhiteboardWindow()
      }
    })()
  }

  const toggleWhiteboard = () => {
    if (whiteboardButtonActive) {
      const hide = whiteboardVisibility?.mode === 'annotation' ? hideAnnotationOverlay : hideWhiteboardWindow
      void hide().catch((error) => console.error('Failed to close whiteboard:', error))
      return
    }
    launchWhiteboard()
  }

  const openWhiteboard = toggleWhiteboard

  const openScreenshotPastePicker = () => {
    setSettingsOpen(false)
    setClosePromptOpen(false)
    setActivePanel(null)
    void (async () => {
      await hideFloatingPanel()
      if (isWailsDesktopRuntime() && capsuleRef.current) {
        await openFloatingPanelFromAnchor('board', capsuleRef.current, 'screenshot-paste')
        return
      }
      setActivePanel('screenshot-paste')
    })().catch((error) => console.error('Failed to open screenshot paste picker:', error))
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

  const pasteScreenshot = (item: ScreenshotItem) => {
    setScreenshotMessage(copy.screenshot.paste)
    annotateScreenshot(item)
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
        recordingMode })
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
    openScrollingScreenshot: beginScrollingScreenshot,
    pasteImage: openScreenshotPastePicker }

  useEffect(() => subscribeShortcutTriggered((event) => {
    if (isSettingsWindow || isFloatingPanelWindow || shortcutCaptureRef.current) return
    void (async () => {
      if (event.preserveCapsuleHidden) {
        await hideCapsuleWindow()
      }
      shortcutActionsRef.current[event.action]?.()
      if (event.preserveCapsuleHidden) {
        window.setTimeout(() => void hideCapsuleWindow(), 0)
      }
    })()
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
            nativeId: source.nativeId }
        : undefined,
      clearGeometry: source.type !== 'region' })
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
          onChange={(value) => commitSettingsPreferencePatch({locale: normalizeLocale(value)})}
        />
        <SettingSelect
          title={copy.settings.theme}
          value={theme}
          options={themeSelectOptions(copy)}
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
      {backendNotice && (
        <div className="rf-backend-notice" role="alert">
          <span className="rf-backend-notice-text">{backendNotice}</span>
          <button className="rf-backend-notice-dismiss" type="button" aria-label="Dismiss" onClick={() => setBackendNotice(null)}>
            <X size={14} />
          </button>
        </div>
      )}
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
            <RecordingClock active={state === 'recording'} epoch={clockEpoch} countdown={countdownRemaining > 0 && state === 'preparing' ? countdownRemaining : null} />
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
              <Globe2 size={18} />
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
            className="icon-button soft minimize-app-button"
            type="button"
            aria-label={copy.aria.minimizeApplication}
            title={copy.aria.minimizeApplication}
            onClick={() => void minimizeApplication()}
          >
            <Minus size={18} />
          </button>

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
                      : {microphone: false})
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
                <MicMeter copy={copy} microphone={microphone} deviceId={selectedMic} deviceAvailable={selectedMicrophoneDevice?.available !== false} hasAvailableDevice={hasAvailableMicrophone} isRecording={isRecording} />
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
                      position: nextPreset !== 'free' ? defaultPipPosition(nextPreset) : currentPipConfig.position })
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
                      <small>{formatShortcutForDisplay(shortcuts.openScrollingScreenshot)}</small>
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
                      onPaste={pasteScreenshot}
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

            {activePanel === 'screenshot-paste' && (
              <div className="menu-stack screenshot-paste-menu">
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
                      onPaste={pasteScreenshot}
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
                    <Globe2 size={16} />
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

export default App
