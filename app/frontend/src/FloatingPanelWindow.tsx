import {copyOcrResultText} from './components/screenshotShared'
import {captureScreenshot, deleteScreenshotItem, getFloatingPanelState, getSourceState, hideFloatingPanel, hidePipOverlay, hideRegionFrame, hideScreenIndicator, listScreenshots, loadBootstrap, openOcrResult, openScreenshotDirectory, openScreenshotInWhiteboard, openVideoDirectory, patchAudioState, patchCameraState, patchScreenshotItem, patchSettingsPreferences, patchShortcutSettings, patchSourceState, queueRecognizeScreenshot, quitApplication, saveSettings, setDataRoot, setFloatingPanelHitRegions, showFloatingPanel, showPinnedScreenshot, showRegionSelector, showScreenIndicator, showScreenshotRegionSelector, showWhiteboardWindow, startMicrophoneLevelMonitor, startScrollingScreenshot, stopMicrophoneLevelMonitor, stopRecording, updatePipOverlay} from './services/recorderBackend'
import {OcrResultPanel} from './components/OcrResultPanel'
import {OcrModelSettings, OcrTranslationSettingsPanel} from './components/OcrSettingsPanels'
import {ScreenshotHistoryRow} from './components/ScreenshotHistoryRow'
import {SelectMenu, SettingLine, SettingSelect, SettingShortcut, SettingTextAction, SettingToggle, SourceGroup, SourceMenuRow, SwitchRow, formatShortcutForDisplay, shortcutIdentity} from './components/controls'
import {ocrPanelContext, parseOcrPanelContext} from './components/floating/ocrResultPanel'
import {elementHitRegion} from './components/hitRegion'
import {SourceSelectionMessageState, StorageMessageState, audioOnlySourceMeta, countdownOptions, ensureVisiblePipConfig, floatingPanelOcrResultExpandedSize, floatingPanelOcrResultSize, floatingPanelSizes, formatPipScalePercent, formatSourceSelectionMessage, formatStorageMessage, formatStorageStatusValue, fpsOptions, isUsableCameraDevice, mediaDeviceName, normalizeRecordingQuality, pipPresetOptions, pipShapeOptions, recordingQualityOptions, selectVisibleInitialSource, sourceName, storageStatusDetail, storageStatusForBadge} from './components/panelShared'
import {screenshotOcrBusy} from './components/screenshotShared'
import {joinDisplayPath} from './components/panelShared'
import {applyTheme, themeSelectOptions} from './components/themeOptions'
import {AppDataInfo, AppStorageStatus, CaptureSource, MediaDevice, MediaInventory, RecordingMode, ScreenshotItem, ShortcutAction, cameraDevices, fallbackAppData, fallbackStorageStatus, localeOptions, pipMaximumScale, pipMinimumScale, shortcutActions, sources, systemAudioDevices} from './services/mockBackend'
import {AudioControlState, AudioLevelUpdate, AudioStatePatch, FloatingPanelState, OcrResult, SettingsPreferencesPatch, ShortcutSettingsPatch, SourceControlState, subscribeAudioLevel, subscribeAudioState, subscribeFloatingPanelChanged, subscribeOcrJobEvents, subscribeRecordingStatus, subscribeRegionSelection, subscribeScreenshotCaptured, subscribeScreenshotHistoryChanged, subscribeSettingsChanged, subscribeSourceStateChanged} from './services/recorderBackend'
import {AppWindow, Check, ChevronDown, ChevronLeft, CircleDot, Crosshair, Globe2, History, ImageIcon, Maximize2, PenLine, ScrollText, Square, Video, Volume2} from 'lucide-react'
import {useEffect, useLayoutEffect, useMemo, useRef, useState} from 'react'
import {copyByLocale} from './i18n'
import {
  defaultSettings,

  normalizeLocale,
  normalizeOcrTranslationSettings,
  normalizePipConfig,
  normalizePipPreset,
  normalizeTheme,

  defaultPipPosition,
  type AppSettings,
  type LocaleCode,
  type PIPConfig,
  type PIPPreset,
  type RecordingState,
  type ThemeCode } from './services/mockBackend'
import {readableError} from './services/recorderBackend'

function FloatingPanelWindow() {
  const panelRef = useRef<HTMLElement | null>(null)
  const [panelState, setPanelState] = useState<FloatingPanelState>(() => ({
    visible: false,
    anchor: {x: 0, y: 0, width: 0, height: 0},
    bounds: {x: 0, y: 0, width: 0, height: 0},
    token: 0 }))
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
    microphoneGain: defaultSettings.audio.microphoneGain })
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
  const noiseSuppression = audioState.noiseSuppression
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
    applyTheme(theme)
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
        height: result.geometry?.height ?? result.source.height })
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
          nativeId: result.source.nativeId } })
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
        regions: region ? [region] : [] })
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
      noiseSuppression: next.audio.noiseSuppression,
      microphoneGain: next.audio.microphoneGain })
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
          nativeId: sourceState.sourceGeometry.nativeId }
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
        countdownSeconds: patch.countdownSeconds ?? current.recording.countdownSeconds },
      window: {
        ...current.window,
        theme: patch.theme ?? current.window.theme,
        startAtLogin: patch.startAtLogin ?? current.window.startAtLogin },
      ocr: {
        ...current.ocr,
        autoRecognizeScreenshots: patch.autoOcr ?? current.ocr.autoRecognizeScreenshots,
        translation: normalizeOcrTranslationSettings({
          ...current.ocr.translation,
          ...(patch.ocrTranslation ?? {}) }) } }))
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
        [action]: accelerator } }))
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
      pipPreset: patch.pipPreset ?? patch.pip?.preset ?? settings.camera.pipPreset }
    setSettings((current) => ({...current, camera: nextCamera}))
    void patchCameraState({
      enabled: nextCamera.enabled,
      deviceId: nextCamera.deviceId,
      pipPreset: nextCamera.pipPreset,
      pip: nextCamera.pip })
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
      pip: nextPip })
    if (!nextEnabled) void hidePipOverlay()
  }

  const commitPipConfigFromPanel = (patch: Partial<PIPConfig>) => {
    if (!camera || !hasUsableCamera || recordingMode === 'audio') return
    const nextConfig = ensureVisiblePipConfig(normalizePipConfig({
      ...currentPipConfig,
      ...patch,
      position: patch.position ?? currentPipConfig.position }, (patch.preset as PIPPreset | undefined) ?? currentPipConfig.preset))
    commitCameraStatePatch({enabled: true, pipPreset: nextConfig.preset, pip: nextConfig})
    const target = {
      deviceId: selectedCameraDevice?.id || selectedCamera,
      nativeId: selectedCameraDevice?.nativeId,
      name: selectedCameraDevice?.name || selectedCamera }
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
            nativeId: source.nativeId }
        : undefined,
      clearGeometry: source.type !== 'region' })
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
        height: size.height },
      width: size.width,
      height: size.height,
      minWidth: size.minWidth,
      maxHeight: size.maxHeight,
      token: panelState.token + 1 })
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
        height: size.height },
      dockSide: panelState.dockSide,
      width: size.width,
      height: size.height,
      minWidth: size.minWidth,
      maxHeight: size.maxHeight,
      token,
      screenId: panelState.screenId,
      direction: panelState.direction,
      contextId: ocrPanelContext(item.ocrResultId, true) }).catch((error) => {
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

  const pasteScreenshot = (item: ScreenshotItem) => {
    setScreenshotMessage(copy.screenshot.paste)
    annotateScreenshot(item)
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
    <main ref={panelRef} className={`floating-panel-shell popover panel-${panel ?? 'empty'} drop-${panelState.direction ?? 'down'}`} role="dialog" aria-label={panel === 'close' ? copy.aria.closeApplication : panel ? copy.aria.menu(panelState.contextId === 'screenshot-paste' ? 'screenshot-paste' : panel === 'settings' ? 'language' : panel) : undefined}>
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
          <SwitchRow label={copy.panels.microphone} checked={microphone && hasAvailableMicrophone} disabled={isRecording || !hasAvailableMicrophone} onChange={(value) => commitAudioStatePatch(value ? {microphone: true, microphoneDeviceId: selectedMic || availableMicrophones.find((device) => device.available !== false)?.id} : {microphone: false})} />
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

      {panel === 'board' && panelState.contextId !== 'screenshot-paste' && (
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
              <span><strong>{copy.screenshot.scrolling}</strong><small>{formatShortcutForDisplay(settings.shortcuts.openScrollingScreenshot)}</small></span>
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
                onPaste={pasteScreenshot}
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

      {panel === 'board' && panelState.contextId === 'screenshot-paste' && (
        <div className="menu-stack screenshot-paste-menu">
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
                onPaste={pasteScreenshot}
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
              <Globe2 size={16} />
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
            <SettingSelect title={copy.settings.theme} value={theme} options={themeSelectOptions(copy)} onChange={(value) => commitSettingsPreferencePatch({theme: normalizeTheme(value)})} />
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

export default FloatingPanelWindow
