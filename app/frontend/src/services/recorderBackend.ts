import {Application, Events, Window as WailsWindow} from '@wailsio/runtime'
import {RecordingFreedomService} from '../../bindings/github.com/lemon-casino/RecordingFreedom/app'
import {
  type AudioState as BoundAudioState,
  type AudioStatePatchRequest as BoundAudioStatePatchRequest,
  type BootstrapState as BoundBootstrapState,

  type CameraStatePatchRequest as BoundCameraStatePatchRequest,
  type ExportRecordingPlanResult as BoundExportRecordingPlanResult,
  type ExportRecordingResult as BoundExportRecordingResult,

  type PIPPreviewImageRequest as BoundPIPPreviewImageRequest,
  type PIPPreviewImageResult as BoundPIPPreviewImageResult,
  type PIPOverlayRequest as BoundPIPOverlayRequest,
  type PIPOverlayState as BoundPIPOverlayState,
  type ScreenIndicatorRequest as BoundScreenIndicatorRequest,
  type ScreenIndicatorResult as BoundScreenIndicatorResult,
  type SettingsPreferencesPatchRequest as BoundSettingsPreferencesPatchRequest,
  type ShortcutSettingsPatchRequest as BoundShortcutSettingsPatchRequest,
  type SourceControlState as BoundSourceControlState,
  type SourceGeometry as BoundSourceGeometry,
  type SourceStatePatchRequest as BoundSourceStatePatchRequest,
  type WhiteboardSettingsPatchRequest as BoundWhiteboardSettingsPatchRequest } from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/models'
import {
  type Capabilities as BoundCaptureCapabilities,
  type Capability as BoundCaptureCapability } from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/capture/models'
import {
  type Plan as BoundExportPlan } from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/exportplan/models'
import {
  CaptureSourceType as BoundCaptureSourceType,
  type MediaDevice as BoundMediaDevice,
  type MediaInventory as BoundMediaInventory } from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/devices/models'
import {
  type AudioOnlyRequest,
  type Session as BoundSession,
  type StartRequest,
  type StatusEvent as BoundStatusEvent } from '../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/recording/models'
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
  type RecordingMode,
  type RecordingPreflight,
  type ShortcutAction,
  type ShortcutSettings } from './mockBackend'
import {fromBoundOcrSettings, normalizeOcrTranslationSettings} from './backend/ocr'
import {fromBoundRegionRect, fromBoundSource, isWailsDesktopRuntime, reportMockFallback, themedPopupURL} from './backend/shared'
export {isWailsDesktopRuntime, readableError, reportMockFallback, type ScreenshotImage} from './backend/shared'

export * from './backend/windows'
export * from './backend/ocr'
export * from './backend/capture'

// Browser preview (vite dev server) legitimately falls back to mock data when
// the Go backend is absent. A packaged desktop build must surface the failure
// instead of silently showing placeholder data.

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
  locale?: AppSettings['locale']
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
  preserveCapsuleHidden: boolean
}

export type WhiteboardSettingsPatch = Partial<AppSettings['whiteboard']>

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
const browserSourceStateEvent = 'rf-source-state'
const browserShortcutTriggeredEvent = 'rf-shortcut-triggered'
const settingsSchemaVersion = 4
const legacyPipMinimumScale = 0.016
const legacyPipMaximumScale = 0.08
const pipMinimumScale = 0.08
const pipMaximumScale = 0.15


export async function patchSourceState(patch: SourceStatePatch): Promise<SourceControlState> {
  try {
    return fromBoundSourceControlState(await RecordingFreedomService.PatchSourceState(toBoundSourceStatePatch(patch)))
  } catch (error) {
    console.info('Using browser source state patch fallback:', error)
    const current = (window as Window & {__RF_SOURCE_STATE__?: SourceControlState}).__RF_SOURCE_STATE__ ?? {
      recordingMode: 'video' as RecordingMode,
      sourceType: 'screen' as CaptureSource['type'] }
    const next: SourceControlState = {
      ...current,
      ...patch,
      sourceGeometry: patch.clearGeometry ? undefined : patch.sourceGeometry ?? current.sourceGeometry,
      recordingMode: patch.recordingMode ?? current.recordingMode }
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
      sourceType: 'screen' }
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

export async function logClientEvent(component: string, event: string, fields: Record<string, unknown> = {}, message = ''): Promise<void> {
  try {
    await RecordingFreedomService.LogClientEvent({
      component,
      event,
      message,
      fields: Object.fromEntries(Object.entries(fields).map(([key, value]) => [key, String(value ?? '')])) })
  } catch (error) {
    console.info('Using browser client log fallback:', error)
  }
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



export async function quitApplication(): Promise<void> {
  try {
    await Application.Quit()
  } catch (error) {
    console.info('Using browser quit fallback:', error)
    window.close()
  }
}

export async function minimizeApplication(): Promise<void> {
  if (!isWailsDesktopRuntime()) {
    const browserWindow = window as Window & {__RF_WINDOW_MINIMIZED__?: boolean}
    browserWindow.__RF_WINDOW_MINIMIZED__ = true
    window.dispatchEvent(new Event('rf-window-minimized'))
    return
  }
  try {
    await WailsWindow.Minimise()
  } catch (error) {
    console.info('Using browser minimize fallback:', error)
    const browserWindow = window as Window & {__RF_WINDOW_MINIMIZED__?: boolean}
    browserWindow.__RF_WINDOW_MINIMIZED__ = true
    window.dispatchEvent(new Event('rf-window-minimized'))
  }
}

export async function loadBootstrap(): Promise<RecorderBootstrap> {
  try {
    return fromBoundBootstrap(await RecordingFreedomService.Bootstrap())
  } catch (error) {
    reportMockFallback('bootstrap', error)
    return {
      appData: fallbackAppData,
      storage: fallbackStorageStatus,
      state: 'idle',
      backend: 'browser-mock',
      sources: fallbackSources,
      media: fallbackMediaInventory,
      recoveries: [],
      settings: loadBrowserSettings(),
      capabilities: fallbackCapabilities }
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
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('shortcut.triggered', (event) => {
      handler(fromShortcutTriggeredEvent(event.data))
    })
  } catch (error) {
    console.info('Desktop shortcut events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler(fromShortcutTriggeredEvent((event as CustomEvent<ShortcutTriggeredUpdate>).detail))
  }
  window.addEventListener(browserShortcutTriggeredEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserShortcutTriggeredEvent, onBrowserEvent)
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
      microphoneGain: nextAudio.microphoneGain }
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
        pip: nextPip },
      updatedAt: new Date().toISOString() }
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
    const popup = window.open(themedPopupURL('/#/settings'), 'recordingfreedom-settings', 'width=920,height=720')
    popup?.focus()
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
      knownModifiedUnixNano } as BoundPIPPreviewImageRequest)
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
    reportMockFallback('sources', error)
    return fallbackSources
  }
}

export async function loadMediaDevices(): Promise<MediaInventory> {
  try {
    return fromBoundMediaInventory(await RecordingFreedomService.ListMediaDevices())
  } catch (error) {
    reportMockFallback('media devices', error)
    return fallbackMediaInventory
  }
}

export async function loadCaptureCapabilities(): Promise<CaptureCapabilities> {
  try {
    return fromBoundCapabilities(await RecordingFreedomService.GetCaptureCapabilities())
  } catch (error) {
    reportMockFallback('capture capabilities', error)
    return fallbackCapabilities
  }
}

export async function loadSettings(): Promise<AppSettings> {
  try {
    return fromBoundSettings(await RecordingFreedomService.GetSettings())
  } catch (error) {
    reportMockFallback('settings', error)
    return loadBrowserSettings()
  }
}

export async function saveSettings(settings: AppSettings): Promise<AppSettings> {
  try {
    return fromBoundSettings(await RecordingFreedomService.SaveSettings(toBoundSettings(settings)))
  } catch (error) {
    reportMockFallback('settings save', error)
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
        startAtLogin: current.window.startAtLogin },
      updatedAt: new Date().toISOString() }
    window.localStorage?.setItem(browserSettingsKey, JSON.stringify(next))
    return next
  }
}

export async function setDataRoot(rootDir: string): Promise<AppDataInfo> {
  try {
    const info = await RecordingFreedomService.SetDataRoot(rootDir)
    return {
      rootDir: info.rootDir,
      videoDir: info.videoDir }
  } catch (error) {
    reportMockFallback('data root apply', error)
    const cleanRoot = rootDir.trim() || fallbackAppData.rootDir
    const separator = cleanRoot.includes('\\') ? '\\' : '/'
    return {
      rootDir: cleanRoot,
      videoDir: `${cleanRoot.replace(/[\\/]+$/, '')}${separator}data${separator}video` }
  }
}

export async function openVideoDirectory(): Promise<AppDataInfo> {
  try {
    const info = await RecordingFreedomService.OpenVideoDirectory()
    return {
      rootDir: info.rootDir,
      videoDir: info.videoDir }
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
      includeAnnotations: options.includeAnnotations }))
  } catch (error) {
    console.info('Desktop recording export unavailable:', error)
    throw error
  }
}

export async function previewExportRecordingPackage(packagePath: string, options: RecordingExportOptions = {}): Promise<RecordingExportPlan> {
  try {
    const result: BoundExportRecordingPlanResult = await RecordingFreedomService.PreviewExportRecordingPackage({
      packageDir: packagePath,
      includeAnnotations: options.includeAnnotations })
    return fromBoundExportPlan(result.plan)
  } catch (error) {
    console.info('Desktop recording export preview unavailable:', error)
    throw error
  }
}

export async function scanRecordingPackages(): Promise<RecordingRecovery[]> {
  try {
    const recoveries = await RecordingFreedomService.ScanRecordingPackages()
    return (recoveries ?? []).map(fromBoundRecovery)
  } catch (error) {
    reportMockFallback('recovery scan', error)
    return []
  }
}

export async function recoverRecordingPackage(packagePath: string): Promise<RecordingRecovery | null> {
  try {
    return fromBoundRecovery(await RecordingFreedomService.RecoverRecordingPackage(packagePath))
  } catch (error) {
    reportMockFallback('package recovery', error)
    return null
  }
}

export async function preflightRecording(request: MockRecordingRequest): Promise<RecordingPreflight> {
  try {
    return fromBoundPreflight(await RecordingFreedomService.PreflightRecording(toStartRequest(request)))
  } catch (error) {
    reportMockFallback('preflight', error)
    return {
      status: 'ready',
      backend: 'browser-mock',
      message: 'Browser UI shell is ready to create a mock recording package.',
      checks: [
        {
          id: 'browser-mock',
          label: 'Browser Preview',
          status: 'ready',
          reason: 'Preview mode validates UI flow only; desktop runtime performs native preflight.' },
      ] }
  }
}

export async function preflightAudioOnlyRecording(request: AudioOnlyRecordingRequest): Promise<RecordingPreflight> {
  try {
    return fromBoundPreflight(await RecordingFreedomService.PreflightAudioOnlyRecording(toAudioOnlyRequest(request)))
  } catch (error) {
    reportMockFallback('audio-only preflight', error)
    return {
      status: request.systemAudio || request.microphone ? 'ready' : 'blocked',
      backend: 'browser-mock',
      message: 'Browser UI shell is ready to create a mock audio-only package.',
      checks: [
        {
          id: 'browser-mock',
          label: 'Browser Preview',
          status: request.systemAudio || request.microphone ? 'ready' : 'blocked',
          reason: 'Preview mode validates UI flow only; desktop runtime performs native preflight.' },
      ] }
  }
}

export async function startRecording(request: MockRecordingRequest): Promise<RecordingSession> {
  try {
    const session = await RecordingFreedomService.StartRecording(toStartRequest(request))
    return fromBoundSession(session)
  } catch (error) {
    reportMockFallback('recording package', error)
    const session = createMockRecordingPackage(request)
    return {id: session.id, packagePath: session.packagePath, backend: 'browser-mock'}
  }
}

export async function startAudioOnlyRecording(request: AudioOnlyRecordingRequest): Promise<RecordingSession> {
  try {
    const session = await RecordingFreedomService.StartAudioOnlyRecording(toAudioOnlyRequest(request))
    return fromBoundSession(session)
  } catch (error) {
    reportMockFallback('audio-only recording package', error)
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
      reason: check.reason })) }
}

export async function pauseRecording(): Promise<RecordingSession | null> {
  try {
    return fromBoundSession(await RecordingFreedomService.PauseRecording())
  } catch (error) {
    reportMockFallback('pause', error)
    return null
  }
}

export async function resumeRecording(): Promise<RecordingSession | null> {
  try {
    return fromBoundSession(await RecordingFreedomService.ResumeRecording())
  } catch (error) {
    reportMockFallback('resume', error)
    return null
  }
}

export async function stopRecording(): Promise<RecordingSession | null> {
  try {
    return fromBoundSession(await RecordingFreedomService.StopRecording())
  } catch (error) {
    reportMockFallback('stop', error)
    return null
  }
}

function fromBoundBootstrap(bootstrap: BoundBootstrapState): RecorderBootstrap {
  const boundSources = bootstrap.sources ?? []
  const recoveries = bootstrap.recoveries ?? []
  return {
    appData: {
      rootDir: bootstrap.appData.rootDir,
      videoDir: bootstrap.appData.videoDir },
    storage: fromBoundStorageStatus(bootstrap.storage),
    state: bootstrap.state,
    backend: bootstrap.backend,
    sources: boundSources.length > 0 ? boundSources.map(fromBoundSource) : fallbackSources,
    media: fromBoundMediaInventory(bootstrap.media),
    recoveries: recoveries.map(fromBoundRecovery),
    settings: fromBoundSettings(bootstrap.settings),
    capabilities: fromBoundCapabilities(bootstrap.capabilities) }
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
    reason: storage.reason }
}

function fromBoundScreenIndicatorResult(result: BoundScreenIndicatorResult): ScreenIndicatorResult {
  return {
    sourceId: result.sourceId,
    displayIndex: result.displayIndex,
    label: result.label,
    sourceBounds: fromBoundRegionRect(result.sourceBounds),
    windowBounds: fromBoundRegionRect(result.windowBounds) }
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
        visible: state.placement.rect.visible },
      shape: state.placement.shape as PIPConfig['shape'],
      mirror: state.placement.mirror,
      edgeFeather: state.placement.edgeFeather },
    overlayBounds: fromBoundRegionRect(state.overlayBounds),
    windowBounds: fromBoundRegionRect(state.windowBounds),
    contentBounds: fromBoundRegionRect(state.contentBounds),
    mode: state.mode === 'recording' ? 'recording' : 'edit',
    cameraName: state.cameraName,
    camera: fromBoundPipCamera(state.camera),
    previewImagePath: state.previewImagePath,
    captureExcluded: state.captureExcluded,
    clientOperationId: state.clientOperationId }
}

function fromBoundPipPreviewImage(result: BoundPIPPreviewImageResult): PIPPreviewImage {
  return {
    available: result.available,
    dataUrl: result.dataUrl,
    modifiedUnixNano: result.modifiedUnixNano }
}

function fromBoundPipCamera(camera: BoundPIPOverlayState['camera']): PIPOverlayCamera | undefined {
  if (!camera) return undefined
  return normalizePipOverlayCamera({
    deviceId: camera.deviceId,
    nativeId: camera.nativeId,
    name: camera.name })
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
      unavailableReason: inventory.enhancement.unavailableReason } }
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
    sidecarEligible: device.sidecarEligible }
}

function fromBoundAudioState(state: BoundAudioState): AudioControlState {
  return {
    system: state.system,
    systemDeviceId: state.systemDeviceId,
    microphone: state.microphone,
    microphoneDeviceId: state.microphoneDeviceId,
    noiseSuppression: state.noiseSuppression,
    microphoneGain: state.microphoneGain || 1 }
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
    clearMicrophoneDevice: patch.clearMicrophoneDevice }
}

function toBoundSettingsPreferencesPatch(patch: SettingsPreferencesPatch): BoundSettingsPreferencesPatchRequest {
  return {
    locale: patch.locale as BoundSettingsPreferencesPatchRequest['locale'],
    theme: patch.theme as BoundSettingsPreferencesPatchRequest['theme'],
    recordingQuality: patch.recordingQuality,
    recordingFps: patch.recordingFps,
    captureCursor: patch.captureCursor,
    countdownSeconds: patch.countdownSeconds,
    startAtLogin: patch.startAtLogin,
    autoOcr: patch.autoOcr,
    ocrTranslation: patch.ocrTranslation as BoundSettingsPreferencesPatchRequest['ocrTranslation'] }
}

function applyBrowserSettingsPreferencesPatch(settings: AppSettings, patch: SettingsPreferencesPatch): AppSettings {
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
        ...(patch.ocrTranslation ?? {}) }) },
    updatedAt: new Date().toISOString() }
}

function applyBrowserShortcutPatch(settings: AppSettings, patch: ShortcutSettingsPatch): AppSettings {
  const nextShortcuts = normalizeShortcutSettings({
    ...settings.shortcuts,
    ...patch })
  return {
    ...settings,
    shortcuts: nextShortcuts,
    updatedAt: new Date().toISOString() }
}

function applyBrowserWhiteboardPatch(settings: AppSettings, patch: WhiteboardSettingsPatch): AppSettings {
  return {
    ...settings,
    whiteboard: fromBoundWhiteboardSettings({
      ...settings.whiteboard,
      ...patch }),
    updatedAt: new Date().toISOString() }
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
    packageRecovery: fromBoundCapability(capabilities.packageRecovery) }
}

function fromBoundCapability(capability: BoundCaptureCapability): CaptureCapability {
  return {
    id: capability.id,
    label: capability.label,
    status: capability.status as CaptureCapability['status'],
    backend: capability.backend,
    permission: capability.permission as CaptureCapability['permission'],
    reason: capability.reason }
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
      nativeId: state.sourceGeometry.nativeId } : undefined }
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
      nativeId: patch.sourceGeometry.nativeId } as BoundSourceGeometry : undefined,
    clearGeometry: patch.clearGeometry }
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
      lastSourceType: settings.source.lastSourceType as CaptureSource['type'] },
    storage: {
      dataRootDir: settings.storage?.dataRootDir },
    recording: {
      quality: normalizeRecordingQuality(settings.recording.quality),
      fps: normalizeRecordingFPS(settings.recording.fps),
      captureCursor: settings.recording.captureCursor,
      countdownSeconds: normalizeCountdown(settings.recording.countdownSeconds) },
    audio: {
      system: settings.audio.system,
      systemDeviceId: settings.audio.systemDeviceId,
      microphone: settings.audio.microphone,
      microphoneDeviceId: settings.audio.microphoneDeviceId,
      noiseSuppression: settings.audio.noiseSuppression,
      microphoneGain: settings.audio.microphoneGain },
    camera: {
      enabled: settings.camera.enabled,
      deviceId: settings.camera.deviceId,
      pipPreset: settings.camera.pipPreset as AppSettings['camera']['pipPreset'],
      pip: fromBoundPipConfig((settings.camera as BoundSettings['camera'] & {pip?: PIPConfig}).pip, settings.camera.pipPreset as AppSettings['camera']['pipPreset']) },
    whiteboard: fromBoundWhiteboardSettings((settings as BoundSettings & {whiteboard?: Partial<AppSettings['whiteboard']>}).whiteboard),
    ocr: fromBoundOcrSettings((settings as BoundSettings & {ocr?: Partial<AppSettings['ocr']>}).ocr),
    shortcuts: normalizeShortcutSettings((settings as BoundSettings & {shortcuts?: Partial<ShortcutSettings>}).shortcuts),
    window: {
      minimizeToTray: settings.window.minimizeToTray,
      theme: normalizeTheme(settings.window.theme),
      startAtLogin: Boolean((settings.window as BoundSettings['window'] & {startAtLogin?: boolean}).startAtLogin) },
    updatedAt: typeof settings.updatedAt === 'string' ? settings.updatedAt : undefined }
}

function toBoundSettings(settings: AppSettings): BoundSettings {
  return {
    schemaVersion: settings.schemaVersion,
    locale: settings.locale as BoundSettings['locale'],
    source: {
      lastSourceId: settings.source.lastSourceId,
      lastSourceType: settings.source.lastSourceType },
    storage: {
      dataRootDir: settings.storage.dataRootDir },
    recording: {
      quality: settings.recording.quality,
      fps: settings.recording.fps,
      captureCursor: settings.recording.captureCursor,
      countdownSeconds: settings.recording.countdownSeconds },
    audio: {
      system: settings.audio.system,
      systemDeviceId: settings.audio.systemDeviceId,
      microphone: settings.audio.microphone,
      microphoneDeviceId: settings.audio.microphoneDeviceId,
      noiseSuppression: settings.audio.noiseSuppression,
      microphoneGain: settings.audio.microphoneGain },
    camera: {
      enabled: settings.camera.enabled,
      deviceId: settings.camera.deviceId,
      pipPreset: settings.camera.pipPreset,
      pip: settings.camera.pip as unknown as BoundSettings['camera']['pip'] },
    whiteboard: settings.whiteboard as unknown as BoundSettings['whiteboard'],
    ocr: settings.ocr as unknown as BoundSettings['ocr'],
    shortcuts: settings.shortcuts as unknown as BoundSettings['shortcuts'],
    window: {
      minimizeToTray: settings.window.minimizeToTray,
      theme: settings.window.theme as BoundSettings['window']['theme'],
      startAtLogin: settings.window.startAtLogin },
    updatedAt: settings.updatedAt ?? new Date(0).toISOString() }
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
      window: {...defaultSettings.window, ...parsed.window} }
    const camera = {
      ...next.camera,
      pip: fromBoundPipConfig(next.camera.pip, next.camera.pipPreset) }
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
        scale: typeof scale === 'number' && Number.isFinite(scale) ? migrateLegacyPipScale(scale) : scale } }
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
    capturePolicy: next.capturePolicy === 'preview-only' ? 'preview-only' : 'export-compose' }
}

function normalizeWhiteboardTool(value: unknown): AppSettings['whiteboard']['lastTool'] {
  return value === 'selection' ||
    value === 'hand' ||
    value === 'freedraw' ||
    value === 'laser' ||
    value === 'arrow' ||
    value === 'line' ||
    value === 'rectangle' ||
    value === 'diamond' ||
    value === 'ellipse' ||
    value === 'text' ||
    value === 'image' ||
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
      microphoneGain: 1 },
    camera: {
      enabled: request.camera,
      deviceId: request.cameraDeviceId,
      deviceNativeId: request.cameraDeviceNativeId,
      pipPreset: request.camera ? request.pipPreset : 'off',
      pip: (request.camera ? request.pip : {...request.pip, preset: 'off'}) as unknown as StartRequest['camera']['pip'] } }
}

function fromBoundPipConfig(config: Partial<PIPConfig> | undefined, fallbackPreset: AppSettings['camera']['pipPreset']): PIPConfig {
  const preset = normalizePipPreset(config?.preset ?? fallbackPreset)
  return {
    preset,
    shape: config?.shape === 'square' ? 'square' : 'circle',
    mirror: config?.mirror !== false,
    position: {
      x: normalizedUnit(config?.position?.x ?? (preset === 'bottom-left' ? 0 : 1)),
      y: normalizedUnit(config?.position?.y ?? 1) },
    scale: normalizedRange(config?.scale, pipMaximumScale, pipMinimumScale, pipMaximumScale),
    edgeFeather: normalizedRange(config?.edgeFeather, 0.16, 0.02, 0.42) }
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
    nativeId: source.nativeId }
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
      microphoneGain: 1 } }
}

function toBoundScreenIndicatorRequest(sourceId: string): BoundScreenIndicatorRequest {
  return {
    sourceId }
}

function toBoundPipOverlayRequest(config: PIPConfig, mode: PIPOverlayMode, camera: string | PIPOverlayCamera, previewImagePath = '', clientOperationId = 0): BoundPIPOverlayRequest {
  const target = normalizePipOverlayCamera(camera)
  return {
    config: config as unknown as BoundPIPOverlayRequest['config'],
    mode,
    cameraName: target.name,
    camera: target as BoundPIPOverlayRequest['camera'],
    previewImagePath,
    clientOperationId }
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
      edgeFeather: normalized.edgeFeather },
    overlayBounds,
    windowBounds: {x: 0, y: 0, width: size + 48, height: size + 48},
    contentBounds,
    mode,
    cameraName: target.name,
    camera: target,
    previewImagePath,
    captureExcluded: false,
    clientOperationId }
}

function normalizePipOverlayCamera(camera: string | PIPOverlayCamera | undefined): PIPOverlayCamera {
  const target = typeof camera === 'string' ? {name: camera} : {...(camera ?? {})}
  const next = {
    deviceId: cleanOptionalString(target.deviceId),
    nativeId: cleanOptionalString(target.nativeId),
    name: cleanOptionalString(target.name) }
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
    status: session.status }
}

function fromBoundStatusEvent(event: BoundStatusEvent): RecordingStatusUpdate {
  const session = event.sessionId && event.packageDir ? {
    id: event.sessionId,
    packagePath: event.packageDir,
    manifestPath: event.manifest,
    backend: event.backend,
    recordingMode: undefined,
    status: event.status } : undefined
  return {
    status: event.status,
    message: event.message,
    backend: event.backend,
    session }
}

function fromShortcutTriggeredEvent(value: unknown): ShortcutTriggeredUpdate {
  const record = value && typeof value === 'object' ? value as Record<string, unknown> : {}
  const action = shortcutActions.includes(record.action as ShortcutAction) ? record.action as ShortcutAction : 'toggleRecording'
  return {
    action,
    accelerator: typeof record.accelerator === 'string' ? record.accelerator : '',
    preserveCapsuleHidden: record.preserveCapsuleHidden === true }
}

function fromBoundAudioLevel(event: unknown): AudioLevelUpdate {
  const data = (event ?? {}) as Partial<AudioLevelUpdate>
  return {
    deviceId: typeof data.deviceId === 'string' ? data.deviceId : '',
    level: normalizedUnit(data.level),
    rms: normalizedUnit(data.rms),
    peak: normalizedUnit(data.peak),
    active: data.active === true,
    error: typeof data.error === 'string' ? data.error : undefined }
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
    reason: recovery.reason }
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
      bytes: snapshot.bytes })),
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
      bytes: scene.bytes })),
    annotationSummary: plan.annotationSummary
      ? {
        ...plan.annotationSummary,
        elementTypeCounts: normalizedNumberRecord(plan.annotationSummary.elementTypeCounts),
        elementPreviewFrames: plan.annotationSummary.elementPreviewFrames ?? undefined }
      : undefined,
    warnings: plan.warnings ?? [] }
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
    outputVerified: result.export.outputVerified === true }
}
