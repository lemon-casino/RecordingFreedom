// Cross-domain helpers shared by the backend service modules.

import type {ScreenshotImageResult as BoundScreenshotImageResult} from '../../../bindings/github.com/lemon-casino/RecordingFreedom/app'
import type {CaptureSource as BoundCaptureSource} from '../../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/devices/models'

import type {CaptureSource, ScreenshotItem} from '../mockBackend'
export function normalizeOcrStatus(status: unknown): ScreenshotItem['ocrStatus'] {
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

const allowMockFallback = import.meta.env.DEV

export function readableError(error: unknown): string {
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  try {
    return JSON.stringify(error)
  } catch {
    return String(error)
  }
}

export function reportMockFallback(operation: string, error: unknown): void {
  if (allowMockFallback) {
    console.info(`Using browser mock ${operation}:`, error)
    return
  }
  console.error(`Backend ${operation} failed; showing placeholder data.`, error)
  window.dispatchEvent(new CustomEvent('rf-backend-error', {
    detail: {operation, message: readableError(error)} }))
}

export const browserScreenshotHistoryKey = 'recordingfreedom.screenshots.history.v1'

export function isWailsDesktopRuntime(): boolean {
  if (window.navigator.userAgent.includes('Wails')) return true
  return window.location.hostname === 'wails.localhost'
}

export function finitePositiveInteger(value: unknown, fallback: number) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric <= 0) return fallback
  return Math.round(numeric)
}

export function finiteNonNegativeNumber(value: unknown, fallback: number) {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric < 0) return fallback
  return numeric
}

export function fromBrowserScreenshotItem(value: unknown): ScreenshotItem | null {
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
    ocrError: typeof record.ocrError === 'string' ? record.ocrError : undefined }
}

export function safeJSON(value: string | null | undefined): unknown {
  if (!value) return null
  try {
    return JSON.parse(value)
  } catch {
    return null
  }
}

export function trimOptionalString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

export type ScreenshotImage = {
  available: boolean
  dataUrl?: string
  path?: string
  bytes?: number
}

export function browserScreenshotDataUrl(item: ScreenshotItem | undefined | null): string | undefined {
  if (!item) return undefined
  const width = Math.max(64, Math.min(800, item.width || 480))
  const height = Math.max(64, Math.min(600, item.height || 320))
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#151b24"/><stop offset="1" stop-color="#263445"/></linearGradient></defs><rect width="100%" height="100%" fill="url(#g)"/><rect x="18" y="18" width="${width - 36}" height="${height - 36}" rx="18" fill="none" stroke="#ef4444" stroke-width="4"/><text x="32" y="54" fill="#f8fafc" font-family="Inter, Arial" font-size="18">RecordingFreedom screenshot preview</text><text x="32" y="84" fill="#9ca3af" font-family="Inter, Arial" font-size="13">${item.id}</text></svg>`
  return `data:image/svg+xml;base64,${window.btoa(unescape(encodeURIComponent(svg)))}`
}

export function loadBrowserScreenshotHistory(): ScreenshotItem[] {
  const parsed = safeJSON(window.localStorage?.getItem(browserScreenshotHistoryKey))
  if (!Array.isArray(parsed)) return []
  return parsed.map(fromBrowserScreenshotItem).filter((item): item is ScreenshotItem => item !== null)
}

export function fromBoundScreenshotImage(result: BoundScreenshotImageResult): ScreenshotImage {
  return {
    available: result.available === true,
    dataUrl: result.dataUrl,
    path: result.path,
    bytes: result.bytes }
}

// Opens a secondary window with the currently applied theme in the URL so it
// can paint correctly on its very first frame.
export function themedPopupURL(hashRoute: string): string {
  const theme = document.documentElement.dataset.theme || 'night-teal'
  return `/?theme=${encodeURIComponent(theme)}${hashRoute.replace(/^\//, '')}`
}

export function fromBoundRegionRect(rect: {x: number; y: number; width: number; height: number}) {
  return {
    x: rect.x,
    y: rect.y,
    width: rect.width,
    height: rect.height }
}

function sourceDimensions(source: BoundCaptureSource) {
  if (source.width && source.height) {
    return `${source.width} x ${source.height}`
  }
  return 'Ready'
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

function sourceLabel(type: CaptureSource['type']) {
  if (type === 'screen') return 'Screen'
  if (type === 'all-screens') return 'All Screens'
  if (type === 'region') return 'Region'
  if (type === 'window') return 'Window'
  return 'Program'
}

export function fromBoundSource(source: BoundCaptureSource): CaptureSource {
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
    unavailableReason: source.unavailableReason }
}
