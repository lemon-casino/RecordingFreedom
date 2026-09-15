import {AppWindow, Crosshair} from 'lucide-react'
import {PIPShape, defaultPipPosition, normalizePipConfig, sources} from '../services/mockBackend'
import {FloatingPanelKind} from '../services/recorderBackend'
import {ocrResultPanelExpandedSize, ocrResultPanelSize} from './floating/ocrResultPanel'
import {Monitor, Radio} from 'lucide-react'
import type {} from 'react'
import {Maximize2} from 'lucide-react'
import {isWailsDesktopRuntime} from '../services/recorderBackend'
import {type AppStorageStatus, type CaptureCapability, type CaptureSource, type MediaDevice, type RecordingPreflight, type ScreenshotItem} from '../services/mockBackend'
import {pipMaximumScale, pipMinimumScale, pipMinimumDisplayPercent, pipMaximumDisplayPercent, clampNumber, type AppSettings, type PIPConfig, type PIPPreset, type RecordingQuality} from '../services/mockBackend'
import type {RecorderCopy, RecoveryMessageKey, StorageMessageKey, SourceSelectionMessageKey, StatusMessageKey} from '../i18n'
import type {RecordingExportPlan} from '../services/recorderBackend'

export type StatusMessageState = {
  key: StatusMessageKey
  fallback?: string
}

export type RecoveryMessageState = {
  key: RecoveryMessageKey
  count?: number
}

export type ExportMessageState = {
  key: 'ready' | 'failed'
  path?: string
  fallback?: string
}

export type StorageMessageState = {
  key: StorageMessageKey
  path?: string
}

export type SourceSelectionMessageState = {
  key: SourceSelectionMessageKey
  width?: number
  height?: number
  fallback?: string
}

export const floatingPanelStandardSize = {width: 320, height: 340, maxHeight: 340, minWidth: 300}
export const floatingPanelCompactSize = {width: 260, height: 120, maxHeight: 120, minWidth: 240}
export const floatingPanelSettingsSize = {width: 340, height: 340, maxHeight: 340, minWidth: 320}
export const floatingPanelOcrResultSize = ocrResultPanelSize
export const floatingPanelOcrResultExpandedSize = ocrResultPanelExpandedSize
export const floatingSelectMinWidth = 180
export const floatingSelectMaxWidth = 280
export const floatingSelectMaxHeight = 220

export const floatingPanelSizes: Record<FloatingPanelKind, {width: number; height: number; maxHeight: number; minWidth?: number}> = {
  source: floatingPanelStandardSize,
  audio: floatingPanelStandardSize,
  camera: floatingPanelStandardSize,
  board: floatingPanelStandardSize,
  language: floatingPanelCompactSize,
  settings: floatingPanelSettingsSize,
  close: {width: 340, height: 170, maxHeight: 170, minWidth: 320},
  'ocr-result': floatingPanelOcrResultSize }


export const pipPresetOptions: PIPPreset[] = ['bottom-right', 'bottom-left', 'free']
export const pipShapeOptions: PIPShape[] = ['circle', 'square']
export const recordingQualityOptions: RecordingQuality[] = ['standard', 'balanced', 'high']
export const fpsOptions = [24, 30, 60]
export const countdownOptions = [0, 3, 5, 10]
export const ocrTranslationProviders: AppSettings['ocr']['translation']['provider'][] = ['disabled', 'deepl', 'openai-compatible']
export const ocrTranslationLanguageOptions = ['auto', 'zh-CN', 'en', 'ja', 'ko', 'fr', 'de', 'es']

export function formatPipScalePercent(scale: number) {
  const normalized = (clampNumber(scale, pipMinimumScale, pipMaximumScale) - pipMinimumScale) / (pipMaximumScale - pipMinimumScale)
  return `${Math.round(pipMinimumDisplayPercent + normalized * (pipMaximumDisplayPercent - pipMinimumDisplayPercent))}%`
}

export function ensureVisiblePipConfig(value: PIPConfig): PIPConfig {
  if (value.preset !== 'off') return value
  return normalizePipConfig({
    ...value,
    preset: 'bottom-right',
    position: defaultPipPosition('bottom-right') }, 'bottom-right')
}

export function normalizeRecordingQuality(value: string): RecordingQuality {
  return recordingQualityOptions.includes(value as RecordingQuality) ? value as RecordingQuality : 'balanced'
}

export function statusMessageFromBackend(message: string): StatusMessageState {
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

export function formatRecoveryMessage(message: RecoveryMessageState, copy: RecorderCopy) {
  if (message.key === 'recovered' && message.count) return copy.recoveryMessages.recoveredCount(message.count)
  return copy.recoveryMessages[message.key]
}

export function formatExportMessage(message: ExportMessageState, copy: RecorderCopy) {
  if (message.fallback) return message.fallback
  if (message.key === 'ready' && message.path) return copy.settings.exportReady(message.path)
  return copy.settings.exportFailed
}



export function formatExportPlanValue(plan: RecordingExportPlan, copy: RecorderCopy) {
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

export function formatExportPlanDetail(plan: RecordingExportPlan, copy: RecorderCopy) {
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

export function formatMilliseconds(value: number) {
  const numeric = Number.isFinite(value) ? Math.max(0, value) : 0
  return `${(numeric / 1000).toFixed(1)}s`
}

export function formatStorageMessage(message: StorageMessageState, copy: RecorderCopy) {
  if (message.key === 'changed' && message.path) return copy.storageMessages.changedTo(message.path)
  return copy.storageMessages[message.key]
}

export function formatSourceSelectionMessage(message: SourceSelectionMessageState, copy: RecorderCopy) {
  if (message.fallback) return message.fallback
  if (message.key === 'regionSelected' && message.width && message.height) {
    return copy.sourceSelectionMessages.regionSelectedSize(message.width, message.height)
  }
  return copy.sourceSelectionMessages[message.key]
}

export function sourceTypeLabel(source: CaptureSource, copy: RecorderCopy) {
  return copy.sourceTypes[source.type]
}

export function sourceName(source: CaptureSource, copy: RecorderCopy) {
  if (source.type === 'screen') {
    return copy.sourceActions.screenLabel(screenIndex(source))
  }
  return copy.sourceNames[source.id] ?? source.name
}

export function sourceMeta(source: CaptureSource, copy: RecorderCopy) {
  if (source.available === false) {
    const reason = source.unavailableReason || copy.sourceUnavailable
    return source.meta ? `${source.meta} · ${reason}` : reason
  }
  return copy.sourceMeta[source.id] ?? source.meta
}

export function audioOnlySourceMeta(systemAudio: boolean, microphone: boolean, copy: RecorderCopy) {
  if (systemAudio && microphone) return copy.sourceAudioOnly.systemAndMic
  if (systemAudio) return copy.sourceAudioOnly.systemOnly
  if (microphone) return copy.sourceAudioOnly.micOnly
  return copy.sourceAudioOnly.noAudio
}

export function mediaDeviceName(device: MediaDevice, copy: RecorderCopy) {
  return copy.mediaDeviceNames[device.id] ?? device.name
}

export function screenshotMeta(item: ScreenshotItem, copy: RecorderCopy) {
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

export function selectPreferredCameraDevice(devices: MediaDevice[] | undefined, preferredId?: string): MediaDevice | undefined {
  if (!devices || devices.length === 0) return undefined
  const preferred = preferredId ? devices.find((device) => device.id === preferredId) : undefined
  if (preferred && isUsableCameraDevice(preferred)) return preferred
  return devices.find(isUsableCameraDevice)
}

export function isUsableCameraDevice(device: MediaDevice): boolean {
  return device.available !== false && device.sidecarEligible !== false && Boolean(device.nativeId?.trim())
}

export function screenIndex(source: CaptureSource) {
  return source.displayIndex && source.displayIndex > 0 ? source.displayIndex : 1
}

export function selectVisibleInitialSource(nextSources: CaptureSource[], lastSourceId: string | undefined, lastSourceType: CaptureSource['type']) {
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

export function fallbackVisibleSource(nextSources: CaptureSource[]) {
  const visibleSources = nextSources.filter((source) => source.type !== 'application' && source.type !== 'region')
  return visibleSources.find((source) => source.type === 'screen') ?? visibleSources[0] ?? sources[0]
}

export function capabilityTitle(capability: CaptureCapability, copy: RecorderCopy) {
  return copy.capabilityLabels[capability.id] ?? capability.label
}

export function capabilityDetail(capability: CaptureCapability, copy: RecorderCopy) {
  return copy.capabilityDetails[capability.id] ?? capability.reason
}

export function formatCapabilityValue(capability: CaptureCapability, copy: RecorderCopy) {
  const permission = copy.capabilityPermissionLabels[capability.permission] ?? capability.permission
  return `${capability.backend} · ${permission}`
}

export function preflightStatusForBadge(status: RecordingPreflight['status']): CaptureCapability['status'] {
  if (status === 'ready') return 'available'
  if (status === 'warning') return 'queued'
  return 'blocked'
}

export function storageStatusForBadge(status: AppStorageStatus['status']): CaptureCapability['status'] {
  if (status === 'ready') return 'available'
  if (status === 'warning') return 'queued'
  return 'blocked'
}

export function formatStorageStatusValue(status: AppStorageStatus, copy: RecorderCopy) {
  const freeSpace = status.freeSpaceKnown ? formatBytes(status.availableBytes) : copy.common.unknownSpace
  return `${copy.storageStatusLabels[status.status]} · ${freeSpace}`
}

export function storageStatusDetail(status: AppStorageStatus, copy: RecorderCopy) {
  const minimum = formatBytes(status.minimumRecommendedBytes)
  const writable = status.writable ? copy.storage.writable : copy.storage.notWritable
  const statusDetail = status.status === 'blocked'
    ? copy.storage.blockedDetail
    : status.status === 'warning'
      ? copy.storage.warningDetail
      : copy.storage.readyDetail
  return `${writable}. ${copy.storage.recommendedFreeSpace(minimum)}. ${statusDetail}`
}

export function preflightDetail(preflight: RecordingPreflight, copy: RecorderCopy) {
  const issue = preflight.checks.find((check) => check.status === 'blocked') ?? preflight.checks.find((check) => check.status === 'warning')
  const message = copy.preflightMessages[preflight.status]
  if (!issue) return message
  const label = copy.preflightCheckLabels[issue.id] ?? issue.label
  const detail = copy.preflightCheckDetails[issue.id] ?? issue.reason
  return detail ? `${message} ${label}: ${detail}` : message
}


export function eventPathContains(event: Event, element: Element | null) {
  if (!element) return false
  if (typeof event.composedPath === 'function' && event.composedPath().includes(element)) return true
  return event.target instanceof Node && element.contains(event.target)
}
export function formatBytes(bytes: number) {
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
export const sourceIcon = {
  screen: Monitor,
  'all-screens': Maximize2,
  region: Crosshair,
  window: AppWindow,
  application: Radio }

export function shouldUseFloatingPanelWindows() {
  return isWailsDesktopRuntime() || (window as Window & {__RF_FORCE_FLOATING_PANEL_WINDOWS__?: boolean}).__RF_FORCE_FLOATING_PANEL_WINDOWS__ === true
}

export function joinDisplayPath(root: string, leaf: string) {
  if (!root || root === 'browser-preview') return leaf
  const separator = root.includes('\\') ? '\\' : '/'
  return `${root.replace(/[\\/]+$/, '')}${separator}${leaf}`
}

export function formatTime(totalSeconds: number) {
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return hours > 0
    ? `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
    : `${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
}

export function packageDisplayName(packagePath: string) {
  const parts = packagePath.split(/[\\/]/).filter(Boolean)
  return parts[parts.length - 1] || packagePath
}

export function isRecordingPackagePath(packagePath: string) {
  return /(^|[\\/])[^\\/]+\.rfrec$/.test(packagePath)
}
