import {
  AppWindow,
  Camera,
  Check,
  ChevronDown,
  ChevronLeft,
  CircleDot,
  Crosshair,
  FlipHorizontal,
  Gauge,
  Languages,
  Maximize2,
  Move,
  MousePointer2,
  Monitor,
  Pause,
  Play,
  Radio,
  Settings,
  Square,
  Video,
  Volume2,
  Wand2,
  X,
} from 'lucide-react'
import {useEffect, useLayoutEffect, useMemo, useRef, useState, type CSSProperties, type PointerEvent as ReactPointerEvent, type ReactNode} from 'react'
import {copyByLocale, type RecorderCopy, type RecoveryMessageKey, type SourceSelectionMessageKey, type StatusMessageKey, type StorageMessageKey} from './i18n'
import {
  cameraDevices,
  fallbackAppData,
  localeOptions,
  normalizeLocale,
  normalizeTheme,
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
  type ThemeCode,
  fallbackCapabilities,
  fallbackStorageStatus,
} from './services/mockBackend'
import {cancelRegionSelector, cancelSelectedRegion, completeRegionSelection, exportRecordingPackage, hidePipOverlay, hideRegionFrame, hideScreenIndicator, hideSettingsWindow, loadBootstrap, loadSettings, logClientEvent, openRecordingPackage, openVideoDirectory, patchAudioState, patchSettingsPreferences, pauseRecording, preflightAudioOnlyRecording, preflightRecording, quitApplication, readPipPreviewImage, recoverRecordingPackage, restoreCapsuleWindow, resumeRecording, saveSettings, setCapsuleWindowExpanded, setCapsuleWindowHitRegions, setDataRoot, showPipOverlay, showRegionSelector, showScreenIndicator, startAudioOnlyRecording, startMicrophoneLevelMonitor, startRecording, stopMicrophoneLevelMonitor, stopRecording, subscribeAudioLevel, subscribeAudioState, subscribeRecordingStatus, subscribeRegionSelection, subscribeSettingsChanged, updatePipOverlay, updateSelectedRegion, type AudioControlState, type AudioLevelUpdate, type AudioStatePatch, type CapsuleWindowExpandDirection, type CapsuleWindowHitRegion, type PIPOverlayCamera, type PIPOverlayState, type RecordingRecovery, type RecordingStatusUpdate, type RegionSelectionSession, type SettingsPreferencesPatch} from './services/recorderBackend'

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
const pipMaximumScale = 0.08
const pipMinimumScale = pipMaximumScale * 0.2
const pipDefaultScale = pipMaximumScale
const pipMinimumDisplayPercent = 20
const pipMaximumDisplayPercent = 100

const recordingQualityOptions: RecordingQuality[] = ['standard', 'balanced', 'high']
const fpsOptions = [24, 30, 60]
const countdownOptions = [0, 3, 5, 10]
const previewPackagePath = 'data/video/recording-preview.rfrec'
type ActivePanel = 'source' | 'audio' | 'camera' | 'language'

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
  preserveAudioEnabled?: boolean
  preserveAudioSelection?: boolean
  preserveCameraEnabled?: boolean
  preserveCameraSelection?: boolean
  preservePipConfig?: boolean
}

function App() {
  const route = currentWindowRoute()
  const isSettingsWindow = route === '/settings'
  const isRegionOverlayWindow = route === '/region-overlay'
  const isScreenIndicatorWindow = route === '/screen-indicator'
  const isPipOverlayWindow = route === '/pip-overlay'
  if (isScreenIndicatorWindow) {
    return <ScreenIndicatorWindow />
  }
  if (isRegionOverlayWindow) {
    return <RegionOverlayWindow />
  }
  if (isPipOverlayWindow) {
    return <PIPOverlayWindow />
  }

  const [selectedSource, setSelectedSource] = useState<CaptureSource>(sources[0])
  const [availableSources, setAvailableSources] = useState<CaptureSource[]>(sources)
  const [activePanel, setActivePanel] = useState<ActivePanel | null>(null)
  const [sourcePickerView, setSourcePickerView] = useState<'overview' | 'windows'>('overview')
  const [settingsOpen, setSettingsOpen] = useState(false)
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
  const [settingsLoaded, setSettingsLoaded] = useState(false)
  const [capsuleExpandDirection, setCapsuleExpandDirection] = useState<CapsuleWindowExpandDirection>('down')
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
  const floatingPointerInsideAtRef = useRef(0)
  const floatingPointerInsideRef = useRef(false)
  const countdownTimerRef = useRef<number | null>(null)
  const countdownTokenRef = useRef(0)
  const cameraPreviewGenerationRef = useRef(0)
  const audioPatchTokenRef = useRef(0)
  const preferencePatchTokenRef = useRef(0)
  const localAudioIntentUntilRef = useRef(0)
  const localPreferenceIntentUntilRef = useRef(0)
  const localCameraIntentUntilRef = useRef(0)
  const localPipIntentUntilRef = useRef(0)
  const selectedSystemAudioRef = useRef(selectedSystemAudio)
  const selectedMicRef = useRef(selectedMic)
  const systemAudioRef = useRef(systemAudio)
  const microphoneRef = useRef(microphone)
  const noiseSuppressionRef = useRef(noiseSuppression)
  const cameraRef = useRef(camera)
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
  const capsuleExpanded = activePanel !== null || settingsOpen || closePromptOpen
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
      schemaVersion: 1,
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
      window: {
        minimizeToTray: true,
        theme,
      },
    }
    if (!isSettingsWindow || !persistedSettingsRef.current) return nextSettings
    return {
      ...persistedSettingsRef.current,
      schemaVersion: 1,
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
      window: {
        minimizeToTray: true,
        theme,
      },
    }
  }, [appData.rootDir, camera, captureCursor, countdownSeconds, isSettingsWindow, locale, microphone, noiseSuppression, pipEdgeFeather, pipMirror, pipPosition, pipPreset, pipScale, pipShape, recordingFPS, recordingQuality, selectedCamera, selectedMic, selectedSource.id, selectedSource.type, selectedSystemAudio, systemAudio, theme])
  const settingsAutosaveKey = useMemo(() => JSON.stringify({
    locale,
    sourceId: selectedSource.id,
    sourceType: selectedSource.type,
    dataRootDir: appData.rootDir,
    camera,
    selectedCamera,
    pipPreset,
    pipShape,
    pipMirror,
    pipPosition,
    pipScale,
    pipEdgeFeather,
  }), [appData.rootDir, camera, locale, pipEdgeFeather, pipMirror, pipPosition, pipPreset, pipScale, pipShape, selectedCamera, selectedSource.id, selectedSource.type])
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
  const persistCameraSettings = (enabled: boolean, pipConfig: PIPConfig, deviceId = selectedCamera) => {
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
    void saveSettings(nextSettings)
      .then((saved) => {
        persistedSettingsRef.current = saved
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
    if (!nextEnabled) markLocalPipIntent()
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
  const applyRecordingStatus = (update: RecordingStatusUpdate) => {
    const nextStatus = update.status as RecordingState
    setState(nextStatus)
    if (update.status !== 'preparing') setCountdownRemaining(0)
    if (nextStatus === 'idle' || nextStatus === 'ready' || nextStatus === 'failed') {
      setElapsed(0)
    }
    if (!isSettingsWindow && (nextStatus === 'ready' || nextStatus === 'failed')) {
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
  const markLocalPipIntent = () => {
    localPipIntentUntilRef.current = Date.now() + 5000
  }
  const hasLocalAudioIntent = () => Date.now() < localAudioIntentUntilRef.current
  const hasLocalPreferenceIntent = () => Date.now() < localPreferenceIntentUntilRef.current
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
      },
    }
  }
  const applyLocalPreferencePatch = (patch: SettingsPreferencesPatch) => {
    if (patch.theme !== undefined) setTheme(normalizeTheme(patch.theme))
    if (patch.recordingQuality !== undefined) setRecordingQuality(normalizeRecordingQuality(patch.recordingQuality))
    if (patch.recordingFps !== undefined) setRecordingFPS(fpsOptions.includes(patch.recordingFps) ? patch.recordingFps : 30)
    if (patch.captureCursor !== undefined) setCaptureCursor(patch.captureCursor)
    if (patch.countdownSeconds !== undefined) setCountdownSeconds(countdownOptions.includes(patch.countdownSeconds) ? patch.countdownSeconds : 0)
    currentSettingsRef.current = settingsWithPreferencePatch(currentSettingsRef.current, patch)
    persistedSettingsRef.current = settingsWithPreferencePatch(persistedSettingsRef.current, patch)
  }
  const commitSettingsPreferencePatch = (patch: SettingsPreferencesPatch) => {
    markLocalPreferenceIntent()
    const token = preferencePatchTokenRef.current + 1
    preferencePatchTokenRef.current = token
    applyLocalPreferencePatch(patch)
    void logClientEvent('settings-preferences', 'patch-request', {
      theme: patch.theme ?? '',
      recordingQuality: patch.recordingQuality ?? '',
      recordingFps: patch.recordingFps ?? '',
      captureCursor: patch.captureCursor ?? '',
      countdownSeconds: patch.countdownSeconds ?? '',
    })
    void patchSettingsPreferences(patch)
      .then((settings) => {
        if (token !== preferencePatchTokenRef.current) return
        localPreferenceIntentUntilRef.current = Date.now() + 3000
        void logClientEvent('settings-preferences', 'patch-success', {
          theme: settings.window.theme,
          recordingQuality: settings.recording.quality,
          recordingFps: settings.recording.fps,
          captureCursor: settings.recording.captureCursor,
          countdownSeconds: settings.recording.countdownSeconds,
        })
        applySettingsState(settings, undefined, undefined, {
          preserveRecordingSettings: hasLocalPreferenceIntent(),
          preserveTheme: hasLocalPreferenceIntent(),
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
    if ((options.preserveRecordingSettings || options.preserveTheme || options.preservePipConfig) && currentSettingsRef.current) {
      effectiveSettings = {
        ...effectiveSettings,
        recording: options.preserveRecordingSettings
          ? currentSettingsRef.current.recording
          : effectiveSettings.recording,
        window: options.preserveTheme
          ? {
              ...effectiveSettings.window,
              theme: currentSettingsRef.current.window.theme,
            }
          : effectiveSettings.window,
        camera: options.preservePipConfig
          ? {
              ...effectiveSettings.camera,
              pipPreset: currentSettingsRef.current.camera.pipPreset,
              pip: currentSettingsRef.current.camera.pip,
            }
          : effectiveSettings.camera,
      }
    }
    persistedSettingsRef.current = effectiveSettings
    const systemAudioList = nextMedia?.systemAudio
    const microphoneList = nextMedia?.microphones
    const cameraList = nextMedia?.cameras
    setLocale(normalizeLocale(effectiveSettings.locale))
    if (!options.preserveTheme) {
      setTheme(normalizeTheme(effectiveSettings.window.theme))
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
    document.body.classList.toggle('rf-recorder-window', !isSettingsWindow)
    return () => {
      document.body.classList.remove('rf-settings-window', 'rf-recorder-window')
    }
  }, [isSettingsWindow])

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

  useEffect(() => {
    if (isSettingsWindow) return
    void setCapsuleWindowExpanded(capsuleExpanded, capsuleExpandedHeight, 'auto', capsuleWindowCompact)
      .then(setCapsuleExpandDirection)
  }, [capsuleExpanded, capsuleExpandedHeight, capsuleWindowCompact, isSettingsWindow])

  useLayoutEffect(() => {
    if (isSettingsWindow) return
    let disposed = false
    let frame = 0

    const publish = () => {
      if (disposed) return
      const viewportWidth = window.innerWidth || document.documentElement.clientWidth || 0
      const viewportHeight = window.innerHeight || document.documentElement.clientHeight || 0
      const regions = [
        elementHitRegion(capsuleRef.current, viewportWidth, viewportHeight, 'pill', 999),
        elementHitRegion(popoverRef.current, viewportWidth, viewportHeight, 'round-rect', 22),
        settingsOpen ? elementHitRegion(settingsPanelRef.current, viewportWidth, viewportHeight, 'round-rect', 24) : null,
        closePromptOpen ? elementHitRegion(closePromptRef.current, viewportWidth, viewportHeight, 'round-rect', 22) : null,
        ...Array.from(shellRef.current?.querySelectorAll('.select-menu-list') ?? [])
          .map((element) => elementHitRegion(element, viewportWidth, viewportHeight, 'round-rect', 16)),
      ].filter((region): region is CapsuleWindowHitRegion => region !== null)
      const request = {
        enabled: regions.length > 0,
        viewportWidth,
        viewportHeight,
        devicePixelRatio: window.devicePixelRatio || 1,
        regions,
      }
      const signature = capsuleHitRegionRequestSignature(request)
      if (signature === capsuleHitRegionSignatureRef.current) return
      capsuleHitRegionSignatureRef.current = signature
      void setCapsuleWindowHitRegions(request)
    }

    const schedule = () => {
      if (frame) window.cancelAnimationFrame(frame)
      frame = window.requestAnimationFrame(publish)
    }

    schedule()
    const resizeObserver = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(schedule)
    ;[
      shellRef.current,
      capsuleRef.current,
      popoverRef.current,
      settingsPanelRef.current,
      closePromptRef.current,
    ].forEach((element) => {
      if (element) resizeObserver?.observe(element)
    })
    const shellElement = shellRef.current
    const mutationObserver = typeof MutationObserver === 'undefined' || !shellElement
      ? null
      : new MutationObserver(schedule)
    if (shellElement) mutationObserver?.observe(shellElement, {
      childList: true,
      subtree: true,
      attributes: true,
      attributeFilter: ['aria-expanded', 'class', 'style'],
    })
    window.addEventListener('resize', schedule)

    return () => {
      disposed = true
      if (frame) window.cancelAnimationFrame(frame)
      resizeObserver?.disconnect()
      mutationObserver?.disconnect()
      window.removeEventListener('resize', schedule)
    }
  }, [activePanel, capsuleExpanded, capsuleExpandedHeight, closePromptOpen, isSettingsWindow, settingsOpen, sourcePickerView])

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
    if (activePanel === 'source' || activePanel === 'audio' || activePanel === 'camera') {
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
      preservePreferences,
      preserveAudioEnabled,
      preserveAudioSelection,
      cameraEnabled: settings.camera.enabled,
      currentCamera: cameraRef.current,
      pipPreset: settings.camera.pipPreset,
      preserveCameraEnabled,
      preservePipConfig,
    })
    if (incomingCameraOff && cameraRef.current && !preserveCameraEnabled) {
      stopCameraPreview('settings-camera-off')
    }
    applySettingsState(settings, undefined, undefined, {
      preserveRecordingSettings: preservePreferences,
      preserveTheme: preservePreferences,
      preserveAudioEnabled,
      preserveAudioSelection,
      preserveCameraEnabled,
      preserveCameraSelection: preserveCameraEnabled,
      preservePipConfig,
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
    if (isSettingsWindow || (!activePanel && !settingsOpen && !closePromptOpen)) return
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
  }, [activePanel, closePromptOpen, isSettingsWindow, settingsOpen])

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

  const buildVideoRequest = (): MockRecordingRequest => ({
    source: selectedSource,
    recording: currentRecordingProfile(),
    systemAudio,
    systemAudioDeviceId: selectedSystemAudio || undefined,
    microphone,
    microphoneDeviceId: selectedMic || undefined,
    noiseSuppression,
    camera,
    cameraDeviceId: selectedCamera,
    cameraDeviceNativeId: selectedCameraDevice?.nativeId,
    pipPreset: camera ? ensureVisiblePipConfig(currentPipConfig).preset : 'off',
    pip: camera ? ensureVisiblePipConfig(currentPipConfig) : currentPipConfig,
  })

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
      const result = await exportRecordingPackage(packagePath)
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
      const result = await exportRecordingPackage(lastPackage)
      setExportMessage({key: 'ready', path: result.outputPath})
    } catch (error) {
      console.error('Failed to export recording package:', error)
      setExportMessage({key: 'failed', fallback: error instanceof Error ? error.message : undefined})
    } finally {
      setExportBusy(false)
    }
  }

  const togglePanel = (panel: ActivePanel) => {
    setSettingsOpen(false)
    setClosePromptOpen(false)
    setActivePanel(activePanel === panel ? null : panel)
  }

  const openSettings = () => {
    setActivePanel(null)
    setClosePromptOpen(false)
    setSettingsOpen((open) => !open)
  }

  const requestCloseApplication = () => {
    setActivePanel(null)
    setSettingsOpen(false)
    setClosePromptOpen(true)
  }

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
    setSourcePickerView('overview')
  }

  const chooseRegion = async (source: CaptureSource) => {
    if (recordingConfigLocked) return
    await hideScreenIndicator()
    setActivePanel(null)
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
        <SettingLine title={copy.settings.release} value="GitHub Actions Windows portable" />
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
    <main ref={shellRef} className={`rf-shell ${capsuleExpanded ? 'is-expanded' : 'is-collapsed'} ${recordingConfigLocked ? 'is-recording-compact' : ''} drop-${capsuleExpandDirection}`} aria-label={copy.aria.recorderShell}>
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
              onClick={() => togglePanel('source')}
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
                onClick={() => togglePanel('audio')}
              >
                <Volume2 size={18} />
              </button>
              <button
                className={`icon-button ${recordingMode === 'video' && camera ? 'is-on' : ''}`}
                type="button"
                aria-label={copy.aria.openCameraSettings}
                title={copy.panels.cameraSidecar}
                disabled={recordingConfigLocked || recordingMode === 'audio'}
                onClick={() => togglePanel('camera')}
              >
                <Camera size={18} />
              </button>
            </div>
          </div>

          <button
            className={`record-button ${state}`}
            type="button"
            aria-label={state === 'recording' || state === 'paused' ? copy.aria.stopRecording : copy.aria.startRecording}
            onClick={toggleRecord}
          >
            {state === 'recording' || state === 'paused' ? <Square size={20} /> : <CircleDot size={22} />}
          </button>

          <div className="time-chip" aria-live="polite">
            <span className={`status-dot ${state}`} />
            <strong>{timeChipLabel}</strong>
            <span>{timeChipValue}</span>
          </div>

          <button
            className="icon-button soft pause-button"
            type="button"
            aria-label={state === 'paused' ? copy.aria.resumeRecording : copy.aria.pauseRecording}
            title={state === 'paused' ? copy.aria.resumeRecording : copy.aria.pauseRecording}
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
              onClick={() => togglePanel('language')}
            >
              <Languages size={18} />
            </button>
            <button
              className="icon-button soft"
              type="button"
              aria-label={copy.aria.openSettings}
              title={copy.settings.title}
              disabled={recordingConfigLocked}
              onClick={openSettings}
            >
              <Settings size={18} />
            </button>
          </div>

          <button
            className="icon-button close-app-button"
            type="button"
            aria-label={copy.aria.closeApplication}
            title={copy.aria.closeApplication}
            onClick={requestCloseApplication}
          >
            <X size={18} />
          </button>
        </div>

        {activePanel && (
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
                  onChange={setSelectedCamera}
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

      {settingsOpen && settingsPanel}
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
  const previewImageModifiedRef = useRef(0)
  const previewImageDataUrlRef = useRef<string | null>(null)
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

  const setPipPreviewImage = (dataUrl: string | null) => {
    previewImageDataUrlRef.current = dataUrl
    setPreviewImageDataUrl(dataUrl)
  }

  const clearPipPreviewImage = () => {
    previewImageModifiedRef.current = 0
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
        if (next.config.preset === 'off') {
          logPipCameraEvent('overlay-state-off')
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
        })
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

  const overlayConfig = async (config: PIPConfig, commit: boolean) => {
    const state = overlayStateRef.current
    const mode = state?.mode ?? 'edit'
    const cameraTarget = state?.camera ?? (state?.cameraName ? {name: state.cameraName} : '')
    const previewImagePath = state?.previewImagePath ?? ''
    const nextConfig = normalizePipConfig(config, config.preset)
    const nextState = commit
      ? await updatePipOverlay(nextConfig, mode, cameraTarget, previewImagePath)
      : await showPipOverlay(nextConfig, mode, cameraTarget, previewImagePath)
    overlayStateRef.current = nextState
    setOverlayState(nextState)
  }

  const previewConfig = (config: PIPConfig) => {
    pendingPreviewRef.current = config
    if (previewFrameRef.current !== null) return
    previewFrameRef.current = window.requestAnimationFrame(() => {
      previewFrameRef.current = null
      const pending = pendingPreviewRef.current
      pendingPreviewRef.current = null
      if (pending) {
        void overlayConfig(pending, false).catch((error) => console.info('PIP preview update failed:', error))
      }
    })
  }

  const commitConfig = async (config: PIPConfig) => {
    if (previewFrameRef.current !== null) {
      window.cancelAnimationFrame(previewFrameRef.current)
      previewFrameRef.current = null
      pendingPreviewRef.current = null
    }
    await overlayConfig(config, true)
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
    cancelPipCameraStream()
    if (!overlayState) {
      void hidePipOverlay()
      return
    }
    void commitConfig({...overlayState.config, preset: 'off'})
      .then(() => hidePipOverlay())
      .catch((error) => console.info('PIP close failed:', error))
  }

  const content = overlayState?.config.preset !== 'off' ? overlayState?.contentBounds : undefined
  const usesBackendPreviewImage = overlayState?.mode === 'recording' && Boolean(overlayState.previewImagePath?.trim())
  const cameraName = overlayState?.camera?.name || overlayState?.cameraName || copy.panels.cameraSidecar
  const cameraPlaceholderTitle = overlayState?.mode === 'recording'
    ? copy.pipOverlay.cameraRecording
    : cameraError
      ? copy.pipOverlay.cameraUnavailable
      : copy.pipOverlay.cameraPreparing
  const cameraPlaceholderDetail = overlayState?.mode === 'recording'
    ? cameraName
    : cameraError || cameraName
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
            {!cameraReady && !usesBackendPreviewImage && (
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
}

type RegionEditAction = 'move' | 'n' | 'e' | 's' | 'w' | 'ne' | 'nw' | 'se' | 'sw'

const regionResizeActions: RegionEditAction[] = ['n', 'e', 's', 'w', 'ne', 'nw', 'se', 'sw']

function useRegionFrameState() {
  const frameWindow = window as Window & {__RF_REGION_FRAME__?: RegionFrameState}
  const [frame, setFrame] = useState<RegionFrameState | undefined>(frameWindow.__RF_REGION_FRAME__)

  useEffect(() => {
    const onFrame = (event: Event) => {
      const next = (event as CustomEvent<RegionFrameState>).detail
      if (next) setFrame(next)
    }
    window.addEventListener('rf-region-frame', onFrame)
    return () => window.removeEventListener('rf-region-frame', onFrame)
  }, [])

  return frame
}

function useRegionEditorDrag(bounds: RegionFrameState['bounds'] | undefined) {
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
    void updateSelectedRegion(next)
  }

  const completeEdit = (event: ReactPointerEvent<HTMLElement>) => {
    const edit = editRef.current
    if (!edit) return
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    editRef.current = null
    void updateSelectedRegion(edit.latest)
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

function RegionOverlayWindow() {
  const overlayWindow = window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}
  const initialSession = overlayWindow.__RF_REGION_SESSION__
  const editFrame = useRegionFrameState()
  const editDrag = useRegionEditorDrag(editFrame?.mode === 'edit' ? editFrame.bounds : undefined)
  const [session, setSession] = useState<RegionSelectionSession | undefined>(initialSession)
  const [drag, setDrag] = useState<{startX: number; startY: number; currentX: number; currentY: number} | null>(null)
  const [cursor, setCursor] = useState({x: -1, y: -1})
  const [invalid, setInvalid] = useState(false)
  const shellRef = useRef<HTMLElement | null>(null)
  const [overlayLocale, setOverlayLocale] = useState<LocaleCode>(navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en')
  const [overlayTheme, setOverlayTheme] = useState<ThemeCode>('night-teal')
  const copy = copyByLocale[overlayLocale]
  const minimumWidth = session?.minimumWidth ?? 64
  const minimumHeight = session?.minimumHeight ?? 64
  const selectedRect = drag ? normalizedClientRect(drag.startX, drag.startY, drag.currentX, drag.currentY) : null
  const isEditingRegion = editFrame?.mode === 'edit'
  const isRecordingRegion = editFrame?.mode === 'recording'
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
    return () => document.body.classList.remove('rf-region-overlay-window')
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
      if (next) setSession(next)
    }
    window.addEventListener('rf-region-session', onSession)
    return () => window.removeEventListener('rf-region-session', onSession)
  }, [])

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        void (isEditingRegion ? cancelSelectedRegion() : cancelRegionSelector())
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [isEditingRegion])

  const cancelSelection = async () => {
    await cancelRegionSelector()
    if (!window.navigator.userAgent.includes('Wails')) {
      window.close()
    }
  }

  const completeSelection = async (rect: RegionSelectionSession['bounds']) => {
    if (rect.width < minimumWidth || rect.height < minimumHeight) {
      setInvalid(true)
      window.setTimeout(() => setInvalid(false), 360)
      return
    }
    await completeRegionSelection(rect)
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
        setCursor({x: event.clientX, y: event.clientY})
        if (drag) {
          setDrag({...drag, currentX: event.clientX, currentY: event.clientY})
        }
      }}
      onPointerDown={(event) => {
        if (isEditingRegion || isRecordingRegion) return
        if (event.button !== 0) return
        event.currentTarget.setPointerCapture(event.pointerId)
        setDrag({startX: event.clientX, startY: event.clientY, currentX: event.clientX, currentY: event.clientY})
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
        const rect = normalizedClientRect(drag.startX, drag.startY, event.clientX, event.clientY)
        setDrag(null)
        void completeSelection(rect)
      }}
      onPointerLeave={(event) => {
        if (isEditingRegion || isRecordingRegion) return
        setCursor({x: event.clientX, y: event.clientY})
      }}
    >
      <div className="region-overlay-scrim" />
      {!isEditingRegion && !isRecordingRegion && cursor.x >= 0 && (
        <>
          <div className="region-crosshair horizontal" style={{top: cursor.y}} />
          <div className="region-crosshair vertical" style={{left: cursor.x}} />
        </>
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
            <button
              type="button"
              aria-label={copy.regionOverlay.cancel}
              title={copy.regionOverlay.cancel}
              onPointerDown={(event) => event.stopPropagation()}
              onClick={() => void cancelSelectedRegion()}
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
        onClick={() => void (isEditingRegion ? cancelSelectedRegion() : cancelSelection())}
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
  const pointerInsideAtRef = useRef(0)
  const pointerInsideRef = useRef(false)
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
        if (pointerInsideRef.current || Date.now() - pointerInsideAtRef.current < 650) return
        close()
      }, 120)
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

  useEffect(() => {
    if (disabled) setOpen(false)
  }, [disabled])

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
        onClick={() => {
          if (!open) updateDropDirection()
          setOpen((value) => !value)
        }}
      >
        <span className="select-menu-label">
          {selected?.swatch && <i className="select-menu-swatch" style={{background: selected.swatch}} aria-hidden="true" />}
          <span>{selected?.label ?? ''}</span>
        </span>
        <ChevronDown size={16} />
      </button>
      {open && (
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

function SettingToggle({title, checked, onChange}: {title: string; checked: boolean; onChange: (value: boolean) => void}) {
  return (
    <div className="setting-line setting-control">
      <SwitchRow label={title} checked={checked} onChange={onChange} />
    </div>
  )
}

export default App
