import {
  AppWindow,
  Camera,
  Check,
  ChevronDown,
  CircleDot,
  Gauge,
  Languages,
  Monitor,
  Pause,
  Play,
  Radio,
  Settings,
  Square,
  Video,
  Volume2,
  Wand2,
} from 'lucide-react'
import {useEffect, useMemo, useState} from 'react'
import {copyByLocale, type RecorderCopy, type RecoveryMessageKey, type StatusMessageKey, type StorageMessageKey} from './i18n'
import {
  cameraDevices,
  fallbackAppData,
  microphoneDevices,
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
  type PIPPreset,
  type RecordingPreflight,
  type RecordingQuality,
  type RecordingState,
  fallbackCapabilities,
  fallbackStorageStatus,
} from './services/mockBackend'
import {hideSettingsWindow, loadBootstrap, pauseRecording, preflightRecording, recoverRecordingPackage, resumeRecording, saveSettings, setCapsuleWindowExpanded, setDataRoot, showSettingsWindow, startRecording, stopRecording, subscribeRecordingStatus, type RecordingRecovery, type RecordingStatusUpdate} from './services/recorderBackend'

const sourceIcon = {
  screen: Monitor,
  window: AppWindow,
  application: Radio,
}

const pipPresetOptions: PIPPreset[] = ['bottom-right', 'bottom-left', 'free', 'off']

const recordingQualityOptions: RecordingQuality[] = ['standard', 'balanced', 'high']
const fpsOptions = [24, 30, 60]
const countdownOptions = [0, 3, 5, 10]

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

function App() {
  const isSettingsWindow = window.location.pathname === '/settings'
  const [selectedSource, setSelectedSource] = useState<CaptureSource>(sources[0])
  const [availableSources, setAvailableSources] = useState<CaptureSource[]>(sources)
  const [activePanel, setActivePanel] = useState<'source' | 'audio' | 'camera' | 'language' | null>(null)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [state, setState] = useState<RecordingState>('idle')
  const [elapsed, setElapsed] = useState(0)
  const [recordingQuality, setRecordingQuality] = useState<RecordingQuality>('balanced')
  const [recordingFPS, setRecordingFPS] = useState(30)
  const [captureCursor, setCaptureCursor] = useState(true)
  const [countdownSeconds, setCountdownSeconds] = useState(0)
  const [systemAudio, setSystemAudio] = useState(true)
  const [availableSystemAudio, setAvailableSystemAudio] = useState<MediaDevice[]>(systemAudioDevices)
  const [selectedSystemAudio, setSelectedSystemAudio] = useState(systemAudioDevices[0].id)
  const [microphone, setMicrophone] = useState(true)
  const [noiseSuppression, setNoiseSuppression] = useState(true)
  const [availableMicrophones, setAvailableMicrophones] = useState<MediaDevice[]>(microphoneDevices)
  const [selectedMic, setSelectedMic] = useState(microphoneDevices[0].id)
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
  const rnnoiseActive = microphone && noiseSuppression

  const copy = copyByLocale[locale]
  const lastStatusText = lastStatusMessage.fallback ?? copy.statusMessages[lastStatusMessage.key]
  const recoveryText = recoveryMessage ? formatRecoveryMessage(recoveryMessage, copy) : ''
  const storageText = storageMessage ? formatStorageMessage(storageMessage, copy) : ''
  const isRecording = state === 'recording' || state === 'paused' || state === 'preparing' || state === 'stopping'
  const SourceIcon = sourceIcon[selectedSource.type]
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
      systemDeviceId: selectedSystemAudio,
      microphone,
      microphoneDeviceId: selectedMic,
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
    capabilities.applicationRecording,
    capabilities.systemAudio,
    capabilities.microphone,
    capabilities.microphoneEnhancement,
    capabilities.cameraSidecar,
    capabilities.pipExport,
    capabilities.packageRecovery,
  ], [capabilities])
  const recoverableRecoveries = useMemo(() => recoveries.filter((recovery) => recovery.recoverable), [recoveries])
  const recoverablePackages = recoverableRecoveries.length
  const applyRecordingStatus = (update: RecordingStatusUpdate) => {
    setState(update.status as RecordingState)
    if (update.session?.packagePath) setLastPackage(update.session.packagePath)
    const backend = update.session?.backend || update.backend
    if (backend) setLastBackend(backend)
    if (update.message) setLastStatusMessage(statusMessageFromBackend(update.message))
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
    if (isSettingsWindow) return
    void setCapsuleWindowExpanded(activePanel !== null)
  }, [activePanel, isSettingsWindow])

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
        setLocale(normalizeLocale(nextSettings.locale))
        setRecordingQuality(normalizeRecordingQuality(nextSettings.recording.quality))
        setRecordingFPS(fpsOptions.includes(nextSettings.recording.fps) ? nextSettings.recording.fps : 30)
        setCaptureCursor(nextSettings.recording.captureCursor)
        setCountdownSeconds(countdownOptions.includes(nextSettings.recording.countdownSeconds) ? nextSettings.recording.countdownSeconds : 0)
        setSystemAudio(nextSettings.audio.system)
        if (bootstrap.media.systemAudio.length > 0) {
          setAvailableSystemAudio(bootstrap.media.systemAudio)
          setSelectedSystemAudio(nextSettings.audio.systemDeviceId && bootstrap.media.systemAudio.some((device) => device.id === nextSettings.audio.systemDeviceId)
            ? nextSettings.audio.systemDeviceId
            : bootstrap.media.systemAudio[0].id)
        } else if (nextSettings.audio.systemDeviceId) {
          setSelectedSystemAudio(nextSettings.audio.systemDeviceId)
        }
        setMicrophone(nextSettings.audio.microphone)
        setNoiseSuppression(nextSettings.audio.noiseSuppression)
        if (bootstrap.media.microphones.length > 0) {
          setAvailableMicrophones(bootstrap.media.microphones)
          setSelectedMic(nextSettings.audio.microphoneDeviceId && bootstrap.media.microphones.some((device) => device.id === nextSettings.audio.microphoneDeviceId)
            ? nextSettings.audio.microphoneDeviceId
            : bootstrap.media.microphones[0].id)
        } else if (nextSettings.audio.microphoneDeviceId) {
          setSelectedMic(nextSettings.audio.microphoneDeviceId)
        }
        setCamera(nextSettings.camera.enabled)
        setPipPreset(normalizePipPreset(nextSettings.camera.pipPreset))
        if (bootstrap.media.cameras.length > 0) {
          setAvailableCameras(bootstrap.media.cameras)
          setSelectedCamera(nextSettings.camera.deviceId && bootstrap.media.cameras.some((device) => device.id === nextSettings.camera.deviceId)
            ? nextSettings.camera.deviceId
            : bootstrap.media.cameras[0].id)
        } else if (nextSettings.camera.deviceId) {
          setSelectedCamera(nextSettings.camera.deviceId)
        }
        setSelectedSource(nextSettings.source.lastSourceId
          ? nextSources.find((source) => source.id === nextSettings.source.lastSourceId) ?? nextSources[0]
          : nextSources.find((source) => source.type === nextSettings.source.lastSourceType) ?? nextSources[0])
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

  const statusLabel = useMemo(() => {
    return copy.statusChips[state] ?? copy.statusChips.idle
  }, [copy, state])

  const beginRecording = async () => {
    setActivePanel(null)
    setElapsed(0)
    setState('preparing')
    try {
      const request = {
        source: selectedSource,
        recording: {
          quality: recordingQuality,
          fps: recordingFPS,
          captureCursor,
          countdownSeconds,
        },
        systemAudio,
        systemAudioDeviceId: selectedSystemAudio,
        microphone,
        microphoneDeviceId: selectedMic,
        noiseSuppression,
        camera,
        cameraDeviceId: selectedCamera,
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
    } catch (error) {
      console.error('Failed to stop recording:', error)
      setLastStatusMessage({key: 'failedToStop'})
      setState('failed')
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

  const openSettings = async () => {
    setActivePanel(null)
    await showSettingsWindow()
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
        <SettingLine title={copy.settings.language} value={copy.localeNames[locale]} />
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
    <main className="rf-shell" aria-label={copy.aria.recorderShell}>
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
            onClick={() => setActivePanel(activePanel === 'source' ? null : 'source')}
          >
            <SourceIcon size={18} />
            <span className="source-text">
              <strong>{sourceTypeLabel(selectedSource, copy)}</strong>
              <small>{sourceName(selectedSource, copy)}</small>
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
              onClick={() => setActivePanel(activePanel === 'audio' ? null : 'audio')}
            >
              <Volume2 size={18} />
            </button>
            <button
              className={`icon-button ${camera ? 'is-on' : ''}`}
              type="button"
              aria-label={copy.aria.openCameraSettings}
              title={copy.panels.cameraSidecar}
              onClick={() => setActivePanel(activePanel === 'camera' ? null : 'camera')}
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
            onClick={() => setActivePanel(activePanel === 'language' ? null : 'language')}
          >
            <Languages size={18} />
          </button>

          <button
            className="icon-button soft"
            type="button"
            aria-label={copy.aria.openSettings}
            title={copy.settings.title}
            onClick={() => void openSettings()}
          >
            <Settings size={18} />
          </button>
        </div>

        {activePanel && (
          <div className="popover" role="dialog" aria-label={copy.aria.menu(activePanel)}>
            {activePanel === 'source' && (
              <div className="menu-grid">
                {availableSources.map((source) => {
                  const Icon = sourceIcon[source.type]
                  const selected = selectedSource.id === source.id
                  return (
                    <button
                      key={source.id}
                      className={`menu-row ${selected ? 'selected' : ''}`}
                      type="button"
                      onClick={() => {
                        setSelectedSource(source)
                        setActivePanel(null)
                      }}
                    >
                      <Icon size={18} />
                      <span>
                        <strong>{sourceName(source, copy)}</strong>
                        <small>{sourceMeta(source, copy)}</small>
                      </span>
                      {selected && <Check size={16} />}
                    </button>
                  )
                })}
              </div>
            )}

            {activePanel === 'audio' && (
              <div className="menu-stack">
                <SwitchRow label={copy.panels.systemAudio} checked={systemAudio} onChange={setSystemAudio} />
                <label className="field-label" htmlFor="system-audio-device">{copy.panels.systemAudioDevice}</label>
                <select id="system-audio-device" value={selectedSystemAudio} onChange={(event) => setSelectedSystemAudio(event.target.value)}>
                  {availableSystemAudio.map((device) => <option key={device.id} value={device.id}>{mediaDeviceName(device, copy)}</option>)}
                </select>
                <SwitchRow label={copy.panels.microphone} checked={microphone} onChange={setMicrophone} />
                <SwitchRow label={copy.panels.rnnoise} checked={rnnoiseActive} disabled={!microphone} onChange={setNoiseSuppression} />
                <label className="field-label" htmlFor="mic-device">{copy.panels.microphoneDevice}</label>
                <select id="mic-device" value={selectedMic} onChange={(event) => setSelectedMic(event.target.value)}>
                  {availableMicrophones.map((device) => <option key={device.id} value={device.id}>{mediaDeviceName(device, copy)}</option>)}
                </select>
                <div className="meter" aria-label={copy.aria.microphoneLevel}>
                  {Array.from({length: 18}, (_, index) => <span key={index} style={{height: `${20 + ((index * 17) % 54)}%`}} />)}
                </div>
              </div>
            )}

            {activePanel === 'camera' && (
              <div className="menu-stack">
                <SwitchRow label={copy.panels.cameraSidecar} checked={camera} onChange={setCamera} />
                <label className="field-label" htmlFor="camera-device">{copy.panels.cameraDevice}</label>
                <select id="camera-device" value={selectedCamera} onChange={(event) => setSelectedCamera(event.target.value)}>
                  {availableCameras.map((device) => <option key={device.id} value={device.id}>{mediaDeviceName(device, copy)}</option>)}
                </select>
                <label className="field-label" htmlFor="pip-preset">{copy.panels.pipPreset}</label>
                <select id="pip-preset" value={pipPreset} onChange={(event) => setPipPreset(event.target.value as PIPPreset)}>
                  {pipPresetOptions.map((preset) => <option key={preset} value={preset}>{copy.pipPresetLabels[preset]}</option>)}
                </select>
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
          <Wand2 size={16} />
          <span>{rnnoiseActive ? copy.strip.micEnhancementOn : copy.strip.micEnhancementOff}</span>
          <span>{copy.common.preflight}: {lastPreflight ? copy.preflightLabels[lastPreflight.status] : copy.common.notRun}</span>
          <span>{recoverablePackages > 0 ? copy.strip.recoveryPackages(recoverablePackages) : copy.strip.recoveryClean}</span>
        </div>
      </section>

      {settingsOpen && settingsPanel}
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

function statusMessageFromBackend(message: string): StatusMessageState {
  switch (message) {
    case 'Preparing recording package':
      return {key: 'preparing'}
    case 'Recording started':
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

function sourceTypeLabel(source: CaptureSource, copy: RecorderCopy) {
  return copy.sourceTypes[source.type]
}

function sourceName(source: CaptureSource, copy: RecorderCopy) {
  return copy.sourceNames[source.id] ?? source.name
}

function sourceMeta(source: CaptureSource, copy: RecorderCopy) {
  if (source.available === false) return copy.sourceUnavailable
  return copy.sourceMeta[source.id] ?? source.meta
}

function mediaDeviceName(device: MediaDevice, copy: RecorderCopy) {
  return copy.mediaDeviceNames[device.id] ?? device.name
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
      <select className="setting-control-select" value={value} onChange={(event) => onChange(event.target.value)}>
        {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>
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
