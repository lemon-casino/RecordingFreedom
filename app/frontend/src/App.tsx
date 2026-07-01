import {
  AppWindow,
  Camera,
  Check,
  ChevronDown,
  ChevronLeft,
  CircleDot,
  Crosshair,
  Gauge,
  Languages,
  Maximize2,
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
import {Window as WailsWindow} from '@wailsio/runtime'
import {useEffect, useMemo, useRef, useState, type PointerEvent as ReactPointerEvent, type ReactNode} from 'react'
import {copyByLocale, type RecorderCopy, type RecoveryMessageKey, type SourceSelectionMessageKey, type StatusMessageKey, type StorageMessageKey} from './i18n'
import {
  cameraDevices,
  fallbackAppData,
  localeOptions,
  normalizeLocale,
  sources,
  systemAudioDevices,
  type AppDataInfo,
  type AppSettings,
  type AppStorageStatus,
  type CaptureCapabilities,
  type CaptureCapability,
  type CaptureSource,
  type LocaleCode,
  type MediaDevice,
  type MediaInventory,
  type PIPPreset,
  type RecordingMode,
  type RecordingPreflight,
  type RecordingQuality,
  type RecordingState,
  fallbackCapabilities,
  fallbackStorageStatus,
} from './services/mockBackend'
import {cancelRegionSelector, cancelSelectedRegion, completeRegionSelection, hideRegionFrame, hideScreenIndicator, hideSettingsWindow, loadBootstrap, loadSettings, pauseRecording, preflightAudioOnlyRecording, preflightRecording, quitApplication, recoverRecordingPackage, resumeRecording, saveSettings, setCapsuleWindowExpanded, setDataRoot, showRegionSelector, showScreenIndicator, startAudioOnlyRecording, startMicrophoneLevelMonitor, startRecording, stopMicrophoneLevelMonitor, stopRecording, subscribeAudioLevel, subscribeRecordingStatus, subscribeRegionSelection, subscribeSettingsChanged, updateSelectedRegion, type AudioLevelUpdate, type RecordingRecovery, type RecordingStatusUpdate, type RegionSelectionSession} from './services/recorderBackend'

const sourceIcon = {
  screen: Monitor,
  'all-screens': Maximize2,
  region: Crosshair,
  window: AppWindow,
  application: Radio,
}

const pipPresetOptions: PIPPreset[] = ['bottom-right', 'bottom-left', 'free', 'off']

const recordingQualityOptions: RecordingQuality[] = ['standard', 'balanced', 'high']
const fpsOptions = [24, 30, 60]
const countdownOptions = [0, 3, 5, 10]
type ActivePanel = 'source' | 'audio' | 'camera' | 'language'

function normalizePipPreset(value: PIPPreset): PIPPreset {
  return pipPresetOptions.includes(value) ? value : 'bottom-right'
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

function App() {
  const route = currentWindowRoute()
  const isSettingsWindow = route === '/settings'
  const isRegionOverlayWindow = route === '/region-overlay'
  const isScreenIndicatorWindow = route === '/screen-indicator'
  const isRegionFrameEdgeWindow = route === '/region-frame-edge'
  if (isScreenIndicatorWindow) {
    return <ScreenIndicatorWindow />
  }
  if (isRegionFrameEdgeWindow) {
    return <RegionFrameEdgeWindow />
  }
  if (isRegionOverlayWindow) {
    return <RegionOverlayWindow />
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
  const [locale, setLocale] = useState<LocaleCode>('zh-CN')
  const [lastPackage, setLastPackage] = useState<string>('data/video/recording-preview.rfrec')
  const [lastBackend, setLastBackend] = useState<string>('ui-preview')
  const [lastStatusMessage, setLastStatusMessage] = useState<StatusMessageState>({key: 'waiting'})
  const [lastPreflight, setLastPreflight] = useState<RecordingPreflight | null>(null)
  const [recoveries, setRecoveries] = useState<RecordingRecovery[]>([])
  const [recoveryBusy, setRecoveryBusy] = useState(false)
  const [recoveryMessage, setRecoveryMessage] = useState<RecoveryMessageState | null>(null)
  const [settingsLoaded, setSettingsLoaded] = useState(false)
  const [capabilities, setCapabilities] = useState<CaptureCapabilities>(fallbackCapabilities)
  const [appData, setAppData] = useState<AppDataInfo>(fallbackAppData)
  const [storageStatus, setStorageStatus] = useState<AppStorageStatus>(fallbackStorageStatus)
  const [storageRootDraft, setStorageRootDraft] = useState(fallbackAppData.rootDir)
  const [storageBusy, setStorageBusy] = useState(false)
  const [storageMessage, setStorageMessage] = useState<StorageMessageState | null>(null)
  const [sourceSelectionMessage, setSourceSelectionMessage] = useState<SourceSelectionMessageState | null>(null)
  const selectedMicRef = useRef(selectedMic)
  const rnnoiseActive = microphone && noiseSuppression

  const copy = copyByLocale[locale]
  const lastStatusText = lastStatusMessage.fallback ?? copy.statusMessages[lastStatusMessage.key]
  const recoveryText = recoveryMessage ? formatRecoveryMessage(recoveryMessage, copy) : ''
  const storageText = storageMessage ? formatStorageMessage(storageMessage, copy) : ''
  const sourceSelectionText = sourceSelectionMessage ? formatSourceSelectionMessage(sourceSelectionMessage, copy) : ''
  const isRecording = state === 'recording' || state === 'paused' || state === 'preparing' || state === 'stopping'
  const recordingConfigLocked = isRecording
  const capsuleExpanded = activePanel !== null || settingsOpen || closePromptOpen
  const capsuleExpandedHeight = settingsOpen
    ? 590
    : closePromptOpen
      ? 330
      : activePanel === 'audio'
        ? 520
        : activePanel === 'camera'
          ? 470
          : activePanel === 'language'
            ? 300
            : 520
  const SourceIcon = recordingMode === 'audio' ? Volume2 : sourceIcon[selectedSource.type]
  const sourceTitle = recordingMode === 'audio' ? copy.recordingModes.audio : sourceTypeLabel(selectedSource, copy)
  const sourceSubtitle = recordingMode === 'audio' ? audioOnlySourceMeta(systemAudio, microphone, copy) : sourceName(selectedSource, copy)
  const currentSettings = useMemo<AppSettings>(() => ({
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
    audio: {
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
      pipPreset,
    },
    window: {
      minimizeToTray: true,
    },
  }), [appData.rootDir, camera, captureCursor, countdownSeconds, locale, microphone, noiseSuppression, pipPreset, recordingFPS, recordingQuality, selectedCamera, selectedMic, selectedSource.id, selectedSource.type, selectedSystemAudio, systemAudio])
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
  const selectedMicrophoneDevice = useMemo(
    () => availableMicrophones.find((device) => device.id === selectedMic),
    [availableMicrophones, selectedMic],
  )
  const hasAvailableMicrophone = useMemo(
    () => availableMicrophones.some((device) => device.available !== false),
    [availableMicrophones],
  )
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
  const applyRecordingStatus = (update: RecordingStatusUpdate) => {
    setState(update.status as RecordingState)
    if (update.session?.packagePath) setLastPackage(update.session.packagePath)
    const backend = update.session?.backend || update.backend
    if (backend) setLastBackend(backend)
    if (update.message) setLastStatusMessage(statusMessageFromBackend(update.message))
  }
  const applySettingsState = (nextSettings: AppSettings, nextMedia?: MediaInventory, nextSources?: CaptureSource[]) => {
    const systemAudioList = nextMedia?.systemAudio
    const microphoneList = nextMedia?.microphones
    const cameraList = nextMedia?.cameras
    setLocale(normalizeLocale(nextSettings.locale))
    setRecordingQuality(normalizeRecordingQuality(nextSettings.recording.quality))
    setRecordingFPS(fpsOptions.includes(nextSettings.recording.fps) ? nextSettings.recording.fps : 30)
    setCaptureCursor(nextSettings.recording.captureCursor)
    setCountdownSeconds(countdownOptions.includes(nextSettings.recording.countdownSeconds) ? nextSettings.recording.countdownSeconds : 0)
    setSystemAudio(nextSettings.audio.system)
    const nextHasAvailableMicrophone = !microphoneList || microphoneList.some((device) => device.available !== false)
    setMicrophone(nextSettings.audio.microphone && nextHasAvailableMicrophone)
    setNoiseSuppression(nextSettings.audio.microphone && nextHasAvailableMicrophone && nextSettings.audio.noiseSuppression)
    setCamera(nextSettings.camera.enabled)
    setPipPreset(normalizePipPreset(nextSettings.camera.pipPreset))

    if (systemAudioList) setAvailableSystemAudio(systemAudioList)
    if (microphoneList) setAvailableMicrophones(microphoneList)
    if (cameraList) setAvailableCameras(cameraList)
    if (nextSettings.audio.systemDeviceId && (!systemAudioList || systemAudioList.some((device) => device.id === nextSettings.audio.systemDeviceId))) {
      setSelectedSystemAudio(nextSettings.audio.systemDeviceId)
    } else if (systemAudioList?.[0]) {
      setSelectedSystemAudio(systemAudioList[0].id)
    }
    if (nextSettings.audio.microphoneDeviceId && (!microphoneList || microphoneList.some((device) => device.id === nextSettings.audio.microphoneDeviceId))) {
      setSelectedMic(nextSettings.audio.microphoneDeviceId)
    } else if (microphoneList?.[0]) {
      setSelectedMic(microphoneList[0].id)
    } else if (microphoneList) {
      setSelectedMic('')
    }
    if (nextSettings.camera.deviceId && (!cameraList || cameraList.some((device) => device.id === nextSettings.camera.deviceId))) {
      setSelectedCamera(nextSettings.camera.deviceId)
    } else if (cameraList?.[0]) {
      setSelectedCamera(cameraList[0].id)
    }
    if (nextSources) {
      setSelectedSource(selectVisibleInitialSource(nextSources, nextSettings.source.lastSourceId, nextSettings.source.lastSourceType))
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
    selectedMicRef.current = selectedMic
  }, [selectedMic])

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
    void setCapsuleWindowExpanded(capsuleExpanded, capsuleExpandedHeight)
  }, [capsuleExpanded, capsuleExpandedHeight, isSettingsWindow])

  useEffect(() => {
    if (recordingMode === 'audio' && activePanel === 'camera') {
      setActivePanel(null)
    }
  }, [activePanel, recordingMode])

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
      void saveSettings(currentSettings).catch((error) => console.error('Failed to save settings:', error))
    }, 300)
    return () => window.clearTimeout(saveTimer)
  }, [currentSettings, settingsLoaded])

  useEffect(() => {
    if (state !== 'recording') {
      return
    }
    const timer = window.setInterval(() => setElapsed((value) => value + 1), 1000)
    return () => window.clearInterval(timer)
  }, [state])

  useEffect(() => subscribeRecordingStatus(applyRecordingStatus), [])

  useEffect(() => subscribeSettingsChanged((settings) => {
    applySettingsState(settings)
  }), [])

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

  const statusLabel = useMemo(() => {
    return copy.statusChips[state] ?? copy.statusChips.idle
  }, [copy, state])

  const beginRecording = async () => {
    setActivePanel(null)
    setElapsed(0)
    setState('preparing')
    try {
      const recording = {
        quality: recordingQuality,
        fps: recordingFPS,
        captureCursor,
        countdownSeconds,
      }
      if (recordingMode === 'audio') {
        const request = {
          recording,
          systemAudio,
          systemAudioDeviceId: selectedSystemAudio || undefined,
          microphone,
          microphoneDeviceId: selectedMic || undefined,
          noiseSuppression,
        }
        const preflight = await preflightAudioOnlyRecording(request)
        setLastPreflight(preflight)
        if (preflight.status === 'blocked') {
          setState('failed')
          setLastStatusMessage({key: 'preflightBlocked'})
          return
        }
        const session = await startAudioOnlyRecording(request)
        applyRecordingStatus({
          status: session.status ?? 'recording',
          message: 'Audio-only recording started',
          backend: session.backend,
          session,
        })
        return
      }

      const request = {
        source: selectedSource,
        recording,
        systemAudio,
        systemAudioDeviceId: selectedSystemAudio || undefined,
        microphone,
        microphoneDeviceId: selectedMic || undefined,
        noiseSuppression,
        camera,
        cameraDeviceId: selectedCamera,
        cameraDeviceNativeId: selectedCameraDevice?.nativeId,
        pipPreset,
      }
      const preflight = await preflightRecording(request)
      setLastPreflight(preflight)
      if (preflight.status === 'blocked') {
        setState('failed')
        setLastStatusMessage({key: 'preflightBlocked'})
        return
      }
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
      setState('failed')
    }
  }

  const finishRecording = async () => {
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
      } else {
        setState('ready')
        setLastStatusMessage({key: 'ready'})
      }
      await hideRegionFrame()
      return true
    } catch (error) {
      console.error('Failed to stop recording:', error)
      setLastStatusMessage({key: 'failedToStop'})
      setState('failed')
      return false
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
    <section className={`settings-panel ${isSettingsWindow ? 'settings-window-panel' : 'settings-sheet'}`} role="dialog" aria-modal={!isSettingsWindow} aria-label={copy.aria.settingsDialog}>
      <div className="sheet-header">
        <div>
          <strong>RecordingFreedom</strong>
          <span>{copy.settings.title}</span>
        </div>
        <button type="button" className="sheet-close" onClick={closeSettings}>{copy.common.close}</button>
      </div>
      <div className="settings-list">
        <SettingLine title={copy.settings.storage} value={appData.videoDir} detail={copy.settings.storageDetail} />
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
          actionDisabled={storageBusy || storageRootDraft.trim() === '' || storageRootDraft.trim() === appData.rootDir}
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
        />
        <SettingLine title={copy.settings.recordingPackage} value="recording-*.rfrec" />
        <SettingSelect
          title={copy.settings.quality}
          value={recordingQuality}
          options={recordingQualityOptions.map((quality) => ({value: quality, label: copy.recordingQualityLabels[quality]}))}
          detail={copy.settings.qualityDetail}
          onChange={(value) => setRecordingQuality(normalizeRecordingQuality(value))}
        />
        <SettingSelect
          title={copy.settings.fps}
          value={String(recordingFPS)}
          options={fpsOptions.map((fps) => ({value: String(fps), label: `${fps} ${copy.settings.fps}`}))}
          onChange={(value) => setRecordingFPS(Number(value))}
        />
        <SettingToggle title={copy.settings.captureCursor} checked={captureCursor} onChange={setCaptureCursor} />
        <SettingSelect
          title={copy.settings.countdown}
          value={String(countdownSeconds)}
          options={countdownOptions.map((seconds) => ({value: String(seconds), label: seconds === 0 ? copy.common.off : `${seconds}s`}))}
          onChange={(value) => setCountdownSeconds(Number(value))}
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
        <SettingLine title={copy.settings.release} value="GitHub Actions macOS, Windows, Linux" />
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
    <main className={`rf-shell ${capsuleExpanded ? 'is-expanded' : 'is-collapsed'}`} aria-label={copy.aria.recorderShell}>
      <section className="recorder-stage" aria-label={copy.aria.recorderControls}>
        <div className={`capsule ${isRecording ? 'capsule-active' : ''}`}>
          <button
            className="grabber"
            type="button"
            aria-label={copy.aria.moveRecorder}
            title={copy.aria.moveRecorder}
          >
            <span />
            <span />
          </button>

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
            <strong>{statusLabel}</strong>
            <span>{formatTime(elapsed)}</span>
          </div>

          <button
            className="icon-button soft"
            type="button"
            aria-label={state === 'paused' ? copy.aria.resumeRecording : copy.aria.pauseRecording}
            title={state === 'paused' ? copy.aria.resumeRecording : copy.aria.pauseRecording}
            disabled={state !== 'recording' && state !== 'paused'}
            onClick={togglePause}
          >
            {state === 'paused' ? <Play size={18} /> : <Pause size={18} />}
          </button>

          <button
            className="icon-button soft"
            type="button"
            aria-label={copy.aria.selectLanguage}
            title={copy.localeNames[locale]}
            aria-expanded={activePanel === 'language'}
            onClick={() => togglePanel('language')}
          >
            <Languages size={18} />
          </button>

          <button
            className="icon-button soft"
            type="button"
            aria-label={copy.aria.openSettings}
            title={copy.settings.title}
            onClick={openSettings}
          >
            <Settings size={18} />
          </button>

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
          <div className={`popover panel-${activePanel}`} role="dialog" aria-label={copy.aria.menu(activePanel)}>
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
                <SwitchRow label={copy.panels.systemAudio} checked={systemAudio} disabled={recordingConfigLocked} onChange={setSystemAudio} />
                <label className="field-label" htmlFor="system-audio-device">{copy.panels.systemAudioDevice}</label>
                <SelectMenu
                  id="system-audio-device"
                  value={selectedSystemAudio}
                  disabled={recordingConfigLocked}
                  options={availableSystemAudio.map((device) => ({value: device.id, label: mediaDeviceName(device, copy), disabled: device.available === false}))}
                  onChange={setSelectedSystemAudio}
                />
                <SwitchRow
                  label={copy.panels.microphone}
                  checked={microphone && hasAvailableMicrophone}
                  disabled={recordingConfigLocked || !hasAvailableMicrophone}
                  onChange={(value) => {
                    setMicrophone(value)
                    if (!value) {
                      setNoiseSuppression(false)
                      setMicMonitorError(null)
                    }
                  }}
                />
                <SwitchRow
                  label={copy.panels.rnnoise}
                  checked={rnnoiseActive}
                  disabled={recordingConfigLocked || !microphone || selectedMicrophoneDevice?.rnnoiseEligible === false}
                  onChange={setNoiseSuppression}
                />
                <label className="field-label" htmlFor="mic-device">{copy.panels.microphoneDevice}</label>
                <SelectMenu
                  id="mic-device"
                  value={selectedMic}
                  disabled={recordingConfigLocked || !microphone || !hasAvailableMicrophone}
                  options={availableMicrophones.length === 0
                    ? [{value: '', label: copy.panels.noMicrophones, disabled: true}]
                    : availableMicrophones.map((device) => ({value: device.id, label: mediaDeviceName(device, copy), disabled: device.available === false}))}
                  onChange={setSelectedMic}
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
                <SwitchRow label={copy.panels.cameraSidecar} checked={camera} disabled={recordingConfigLocked} onChange={setCamera} />
                <label className="field-label" htmlFor="camera-device">{copy.panels.cameraDevice}</label>
                <SelectMenu
                  id="camera-device"
                  value={selectedCamera}
                  disabled={recordingConfigLocked}
                  options={availableCameras.map((device) => ({value: device.id, label: mediaDeviceName(device, copy), disabled: device.available === false}))}
                  onChange={setSelectedCamera}
                />
                <label className="field-label" htmlFor="pip-preset">{copy.panels.pipPreset}</label>
                <SelectMenu
                  id="pip-preset"
                  value={pipPreset}
                  disabled={recordingConfigLocked}
                  options={pipPresetOptions.map((preset) => ({value: preset, label: copy.pipPresetLabels[preset]}))}
                  onChange={(value) => setPipPreset(value as PIPPreset)}
                />
                <div className="camera-preview">
                  <Video size={26} />
                  <span>{copy.panels.pipPresetPreview(copy.pipPresetLabels[pipPreset])}</span>
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

      {settingsOpen && settingsPanel}
      {closePromptOpen && (
        <section className="close-confirm-panel" role="dialog" aria-modal="true" aria-label={isRecording ? copy.closeDialog.recordingTitle : copy.closeDialog.idleTitle}>
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

function RegionFrameEdgeWindow() {
  const [edge, setEdge] = useState<RegionFrameEdge>('top')
  const frame = useRegionFrameState()

  useEffect(() => {
    document.body.classList.add('rf-region-frame-window')
    return () => document.body.classList.remove('rf-region-frame-window')
  }, [])

  useEffect(() => {
    void WailsWindow.Name()
      .then((name) => setEdge(regionFrameEdgeFromName(name)))
      .catch(() => setEdge('top'))
  }, [])

  const mode = frame?.mode ?? 'recording'

  return (
    <main
      className={`region-frame-edge-shell edge-${edge} ${mode}`}
      aria-label={`Selected recording region edge ${edge}`}
    >
      <span />
    </main>
  )
}

type RegionFrameState = {
  bounds: {x: number; y: number; width: number; height: number}
  mode?: 'edit' | 'recording'
}

type RegionEditAction = 'move' | 'n' | 'e' | 's' | 'w' | 'ne' | 'nw' | 'se' | 'sw'
type RegionFrameEdge = 'top' | 'right' | 'bottom' | 'left'

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

function regionFrameEdgeFromName(name: string): RegionFrameEdge {
  if (name.endsWith('right')) return 'right'
  if (name.endsWith('bottom')) return 'bottom'
  if (name.endsWith('left')) return 'left'
  return 'top'
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
  const copy = copyByLocale[overlayLocale]
  const minimumWidth = session?.minimumWidth ?? 64
  const minimumHeight = session?.minimumHeight ?? 64
  const selectedRect = drag ? normalizedClientRect(drag.startX, drag.startY, drag.currentX, drag.currentY) : null
  const isEditingRegion = editFrame?.mode === 'edit'
  const overlayOrigin = session?.bounds ?? {x: 0, y: 0, width: 0, height: 0}
  const editableRect = isEditingRegion ? {
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
      .then((settings) => setOverlayLocale(normalizeLocale(settings.locale)))
      .catch((error) => console.info('Using region overlay language fallback:', error))
  }, [])

  useEffect(() => subscribeSettingsChanged((settings) => {
    setOverlayLocale(normalizeLocale(settings.locale))
  }), [])

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

  return (
    <main
      ref={shellRef}
      className="region-overlay-shell"
      aria-label={copy.aria.regionOverlay}
      onPointerMove={isEditingRegion ? editDrag.updateEdit : undefined}
      onPointerCancel={isEditingRegion ? editDrag.completeEdit : undefined}
      onPointerMoveCapture={(event) => {
        if (isEditingRegion) return
        setCursor({x: event.clientX, y: event.clientY})
        if (drag) {
          setDrag({...drag, currentX: event.clientX, currentY: event.clientY})
        }
      }}
      onPointerDown={(event) => {
        if (isEditingRegion) return
        if (event.button !== 0) return
        event.currentTarget.setPointerCapture(event.pointerId)
        setDrag({startX: event.clientX, startY: event.clientY, currentX: event.clientX, currentY: event.clientY})
        setInvalid(false)
      }}
      onPointerUp={(event) => {
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
        if (isEditingRegion) return
        setCursor({x: event.clientX, y: event.clientY})
      }}
    >
      <div className="region-overlay-scrim" />
      {!isEditingRegion && cursor.x >= 0 && (
        <>
          <div className="region-crosshair horizontal" style={{top: cursor.y}} />
          <div className="region-crosshair vertical" style={{left: cursor.x}} />
        </>
      )}
      {!isEditingRegion && selectedRect && (
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
      <button
        className="region-cancel-button"
        type="button"
        aria-label={copy.regionOverlay.cancel}
        title={copy.regionOverlay.cancel}
        onPointerDown={(event) => event.stopPropagation()}
        onClick={() => void (isEditingRegion ? cancelSelectedRegion() : cancelSelection())}
      >
        <X size={22} />
      </button>
      {!isEditingRegion && <div className="region-overlay-badge" aria-hidden="true">
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
  const rootRef = useRef<HTMLDivElement | null>(null)
  const selected = options.find((option) => option.value === value) ?? options.find((option) => !option.disabled) ?? options[0]

  useEffect(() => {
    if (!open) return
    const onPointerDown = (event: PointerEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false)
      }
    }
    document.addEventListener('pointerdown', onPointerDown)
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('pointerdown', onPointerDown)
      document.removeEventListener('keydown', onKeyDown)
    }
  }, [open])

  useEffect(() => {
    if (disabled) setOpen(false)
  }, [disabled])

  return (
    <div ref={rootRef} className={`select-menu ${open ? 'open' : ''} ${disabled ? 'disabled' : ''} ${className}`}>
      <button
        id={id}
        type="button"
        className="select-menu-button"
        disabled={disabled || options.length === 0}
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
      >
        <span>{selected?.label ?? ''}</span>
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
              onClick={() => {
                if (option.disabled) return
                onChange(option.value)
                setOpen(false)
              }}
            >
              <span>{option.label}</span>
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
  options: Array<{value: string; label: string}>
  detail?: string
  onChange: (value: string) => void
}) {
  return (
    <label className="setting-line setting-control">
      <span>{title}</span>
      <SelectMenu className="setting-control-select" value={value} options={options} onChange={onChange} />
      {detail && <small>{detail}</small>}
    </label>
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
