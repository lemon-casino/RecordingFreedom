import {Events, Window as WailsWindow} from '@wailsio/runtime'
import {RecordingFreedomService} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app'
import {type BootstrapState as BoundBootstrapState} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/models'
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
  type Session as BoundSession,
  type StartRequest,
  type StatusEvent as BoundStatusEvent,
} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/recording/models'
import {type Summary as BoundPreflightSummary} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/preflight/models'
import {type RecoverySummary as BoundRecoverySummary} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/recpackage/models'
import {type Settings as BoundSettings} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/settings/models'
import {
  createMockRecordingPackage,
  defaultSettings,
  fallbackAppData,
  fallbackCapabilities,
  fallbackStorageStatus,
  mediaInventory as fallbackMediaInventory,
  normalizeLocale,
  sources as fallbackSources,
  type AppSettings,
  type AppDataInfo,
  type AppStorageStatus,
  type CaptureCapabilities,
  type CaptureCapability,
  type CaptureSource,
  type MediaDevice,
  type MediaInventory,
  type MockRecordingRequest,
  type RecordingPreflight,
} from './mockBackend'

export type RecordingSession = {
  id: string
  packagePath: string
  manifestPath?: string
  backend?: string
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

const browserSettingsKey = 'recordingfreedom.settings.v1'
const capsuleWindowWidth = 860
const capsuleWindowCollapsedHeight = 166
const capsuleWindowExpandedHeight = 420

export async function setCapsuleWindowExpanded(expanded: boolean): Promise<void> {
  try {
    await WailsWindow.SetSize(capsuleWindowWidth, expanded ? capsuleWindowExpandedHeight : capsuleWindowCollapsedHeight)
  } catch (error) {
    console.info('Using browser capsule window size fallback:', error)
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

export async function showSettingsWindow(): Promise<void> {
  try {
    await RecordingFreedomService.ShowSettingsWindow()
  } catch (error) {
    console.info('Using browser settings window fallback:', error)
    const popup = window.open('/settings', 'recordingfreedom-settings', 'width=920,height=720')
    popup?.focus()
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
    available: source.available,
    capability: source.capability,
    unavailableReason: source.unavailableReason,
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
    return {...next, locale: normalizeLocale(next.locale)}
  } catch {
    return defaultSettings
  }
}

function toStartRequest(request: MockRecordingRequest): StartRequest {
  return {
    sourceId: request.source.id,
    sourceType: toBoundSourceType(request.source.type),
    sourceName: request.source.name,
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
      pipPreset: request.camera ? request.pipPreset : 'off',
    },
  }
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
  if (type === 'window') return BoundCaptureSourceType.SourceWindow
  return BoundCaptureSourceType.SourceApplication
}

function fromBoundSession(session: BoundSession): RecordingSession {
  return {
    id: session.id,
    packagePath: session.packageDir,
    manifestPath: session.manifest,
    backend: session.backend,
    status: session.status,
  }
}

function fromBoundStatusEvent(event: BoundStatusEvent): RecordingStatusUpdate {
  const session = event.sessionId && event.packageDir ? {
    id: event.sessionId,
    packagePath: event.packageDir,
    manifestPath: event.manifest,
    backend: event.backend,
    status: event.status,
  } : undefined
  return {
    status: event.status,
    message: event.message,
    backend: event.backend,
    session,
  }
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
