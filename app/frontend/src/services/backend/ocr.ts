import {RecordingFreedomService, type OcrJobEvent as BoundOcrJobEvent} from '../../../bindings/github.com/lemon-casino/RecordingFreedom/app'
import {
  type Block as BoundOcrBlock,
  type JobSnapshot as BoundOcrJobSnapshot,
  type ModelDownloadEvent as BoundOcrModelDownloadEvent,
  type ModelDownloadSnapshot as BoundOcrModelDownloadSnapshot,
  type ModelInfo as BoundOcrModelInfo,
  type RecognizeRequest as BoundOcrRecognizeRequest,
  type Result as BoundOcrResult,
  SourceKind as BoundOcrSourceKind,
  type Status as BoundOcrStatus,
  type TranslateRequest as BoundOcrTranslateRequest,
  type TranslationBlock as BoundOcrTranslationBlock,
  type TranslationResult as BoundOcrTranslationResult,
  type WhiteboardRequest as BoundOcrWhiteboardRequest,
  type WorkerCapabilities as BoundOcrWorkerCapabilities,
} from '../../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/ocr/models'
import {Events} from '@wailsio/runtime'
import {type AppSettings} from '../mockBackend'
import {browserScreenshotDataUrl, finiteNonNegativeNumber, finitePositiveInteger, fromBoundScreenshotImage, isWailsDesktopRuntime, loadBrowserScreenshotHistory, trimOptionalString, type ScreenshotImage} from './shared'

export type OcrSourceKind = 'region-screenshot' | 'full-screenshot' | 'window-screenshot' | 'focused-window-screenshot' | 'scrolling-screenshot' | 'pinned-screenshot' | 'whiteboard' | 'whiteboard-selection' | 'image'

export type OcrModelInfo = {
  id: string
  name: string
  channel: string
  engine: string
  language: string[]
  version?: string
  sourceUrl?: string
  license?: string
  downloadAvailable: boolean
  downloadBytes?: number
  installed: boolean
  verified: boolean
  active: boolean
  modelDir?: string
  smokeImage?: string
  smokeExpected?: string
  smokeAssetReady: boolean
  smokeError?: string
  missingFiles: string[]
  verificationError?: string
}

export type OcrStatus = {
  status: string
  activeModelId?: string
  models: OcrModelInfo[]
  workerPath?: string
  runtimeDir?: string
  workerCapabilities?: OcrWorkerCapabilities
  message?: string
}

export type OcrModelDownloadSnapshot = {
  id: string
  modelId: string
  status: 'queued' | 'running' | 'installed' | 'failed' | 'cancelled' | string
  downloadedBytes: number
  totalBytes: number
  percent: number
  error?: string
  model?: OcrModelInfo
  startedAt: string
  updatedAt: string
}

export type OcrJobSnapshot = {
  jobId: string
  status: string
  cacheKey?: string
  request: OcrRecognizeRequest
  merged: boolean
  createdAt: string
  updatedAt: string
}

export type OcrJobUpdate = {
  jobId: string
  sourceKind: OcrSourceKind
  sourceId: string
  status: string
  cacheKey?: string
  merged?: boolean
  error?: string
  result?: OcrResult
}

export type OcrWorkerCapabilities = {
  schemaVersion: number
  name: string
  version: string
  protocolVersion: string
  engine: string
  modelFormats: string[]
  supportsRecognize: boolean
  runtimeDir?: string
  runtimeLibrary?: string
  runtimeAvailable: boolean
  runtimeVersion?: string
  runtimeApiVersion?: number
  runtimeError?: string
  message?: string
}

export type OcrPoint = {x: number; y: number}

export type OcrBlock = {
  id: string
  text: string
  confidence: number
  box: OcrPoint[]
  lineIndex: number
  languageHint?: string
}

export type OcrResult = {
  id: string
  sourceKind: OcrSourceKind
  sourceId: string
  imagePath: string
  imageSha256: string
  modelId: string
  language: string
  width: number
  height: number
  blocks: OcrBlock[]
  plainText: string
  createdAt: string
  durationMs: number
}

export type OcrRecognizeRequest = {
  imagePath: string
  sourceKind: OcrSourceKind
  sourceId: string
  language?: string
  modelId?: string
  force?: boolean
  priority?: 'interactive' | 'normal' | 'background' | string
}

export type OcrWhiteboardRequest = {
  imagePath: string
  elementId?: string
  sceneId?: string
  language?: string
  force?: boolean
  priority?: 'interactive' | 'normal' | 'background' | string
}

export type OcrTranslateRequest = {
  ocrResultId: string
  blockIds?: string[]
  provider: string
  sourceLanguage: string
  targetLanguage: string
  baseUrl?: string
  apiKey?: string
  model?: string
  force?: boolean
}

export type OcrTranslationBlock = {
  blockId: string
  source: string
  translated: string
}

export type OcrTranslationResult = {
  ocrResultId: string
  provider: string
  sourceLanguage: string
  targetLanguage: string
  model?: string
  promptVersion?: string
  blocks: OcrTranslationBlock[]
  createdAt: string
}

const browserOcrJobEvent = 'rf-ocr-job'

const browserOcrModelDownloadEvent = 'rf-ocr-model-download'

export function subscribeOcrJobEvents(handler: (event: OcrJobUpdate) => void): () => void {
  const eventNames = ['ocr.job.queued', 'ocr.job.started', 'ocr.job.finished', 'ocr.job.failed', 'ocr.job.cancelled'] as const
  const disposers: Array<() => void> = []
  for (const eventName of eventNames) {
    try {
      disposers.push(Events.On(eventName, (event) => {
        handler(fromBoundOcrJobEvent(event.data as BoundOcrJobEvent))
      }))
    } catch (error) {
      console.info('Using browser OCR job event fallback:', error)
    }
  }
  const onBrowserEvent = (event: Event) => {
    if (isWailsDesktopRuntime()) return
    handler(fromBrowserOcrJobEvent((event as CustomEvent<Partial<OcrJobUpdate>>).detail))
  }
  window.addEventListener(browserOcrJobEvent, onBrowserEvent)
  return () => {
    for (const dispose of disposers) dispose()
    window.removeEventListener(browserOcrJobEvent, onBrowserEvent)
  }
}

export function subscribeOcrModelDownloadEvents(handler: (snapshot: OcrModelDownloadSnapshot) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('ocr.model.download.changed', (event) => {
      handler(fromBoundOcrModelDownloadEvent(event.data as BoundOcrModelDownloadEvent).snapshot)
    })
  } catch (error) {
    console.info('Using browser OCR model download event fallback:', error)
  }
  const onBrowserEvent = (event: Event) => {
    if (isWailsDesktopRuntime()) return
    handler(fromBrowserOcrModelDownloadEvent((event as CustomEvent<Partial<OcrModelDownloadSnapshot>>).detail).snapshot)
  }
  window.addEventListener(browserOcrModelDownloadEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserOcrModelDownloadEvent, onBrowserEvent)
  }
}

export async function listOcrModels(): Promise<OcrModelInfo[]> {
  try {
    return (await RecordingFreedomService.ListOcrModels() ?? []).map(fromBoundOcrModelInfo)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model fallback:', error)
    return browserOcrStatus().models
  }
}

export async function getOcrStatus(): Promise<OcrStatus> {
  try {
    return fromBoundOcrStatus(await RecordingFreedomService.GetOcrStatus())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR status fallback:', error)
    return browserOcrStatus()
  }
}

export async function installOcrModel(modelId: string): Promise<OcrModelInfo> {
  try {
    return fromBoundOcrModelInfo(await RecordingFreedomService.InstallOcrModel(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR install fallback:', error)
    throw new Error(`OCR model ${modelId} is not available in browser preview`)
  }
}

export async function installOcrModelPackage(packagePath: string): Promise<OcrModelInfo> {
  try {
    return fromBoundOcrModelInfo(await RecordingFreedomService.InstallOcrModelPackage(packagePath))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR package install fallback:', error)
    throw new Error('OCR model package import is local-only and is not available in browser preview')
  }
}

export async function removeOcrModel(modelId: string): Promise<OcrStatus> {
  try {
    return fromBoundOcrStatus(await RecordingFreedomService.RemoveOcrModel(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR remove fallback:', error)
    return browserOcrStatus()
  }
}

export async function setActiveOcrModel(modelId: string): Promise<OcrStatus> {
  try {
    return fromBoundOcrStatus(await RecordingFreedomService.SetActiveOcrModel(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR active model fallback:', error)
    const browserWindow = window as Window & {
      __RF_OCR_STATUS__?: OcrStatus
      __RF_SET_ACTIVE_OCR_MODEL_ERROR__?: string
      __RF_LAST_SET_ACTIVE_OCR_MODEL__?: {modelId: string; at: string}
    }
    browserWindow.__RF_LAST_SET_ACTIVE_OCR_MODEL__ = {modelId, at: new Date().toISOString()}
    if (browserWindow.__RF_SET_ACTIVE_OCR_MODEL_ERROR__) {
      throw new Error(browserWindow.__RF_SET_ACTIVE_OCR_MODEL_ERROR__)
    }
    const current = browserOcrStatus()
    const target = current.models.find((model) => model.id === modelId)
    if (!target?.installed || !target.verified) {
      throw new Error(`OCR model ${modelId} is not installed or failed verification`)
    }
    const next: OcrStatus = {
      ...current,
      activeModelId: modelId,
      models: current.models.map((model) => ({...model, active: model.id === modelId})) }
    browserWindow.__RF_OCR_STATUS__ = next
    return next
  }
}

export async function startOcrModelDownload(modelId: string): Promise<OcrModelDownloadSnapshot> {
  try {
    return fromBoundOcrModelDownloadSnapshot(await RecordingFreedomService.StartOcrModelDownload(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model download fallback:', error)
    return browserStartOcrModelDownload(modelId)
  }
}

export async function cancelOcrModelDownload(modelId: string): Promise<OcrModelDownloadSnapshot> {
  try {
    return fromBoundOcrModelDownloadSnapshot(await RecordingFreedomService.CancelOcrModelDownload(modelId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model download cancel fallback:', error)
    return browserCancelOcrModelDownload(modelId)
  }
}

export async function getOcrModelDownloads(): Promise<OcrModelDownloadSnapshot[]> {
  try {
    return (await RecordingFreedomService.GetOcrModelDownloads() ?? []).map(fromBoundOcrModelDownloadSnapshot)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model downloads fallback:', error)
    return Object.values(browserOcrModelDownloads())
  }
}

export async function refreshOcrModelCatalog(catalogUrl = ''): Promise<OcrStatus> {
  try {
    return fromBoundOcrStatus(await RecordingFreedomService.RefreshOcrModelCatalog(catalogUrl))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR model catalog refresh fallback:', error)
    const browserWindow = window as Window & {
      __RF_OCR_STATUS__?: OcrStatus
      __RF_OCR_MODEL_CATALOG_STATUS__?: OcrStatus
      __RF_LAST_OCR_MODEL_CATALOG_REFRESH__?: {catalogUrl: string; at: string}
    }
    browserWindow.__RF_LAST_OCR_MODEL_CATALOG_REFRESH__ = {catalogUrl, at: new Date().toISOString()}
    if (!browserWindow.__RF_OCR_MODEL_CATALOG_STATUS__) {
      throw new Error('OCR model catalog refresh is desktop-only in browser preview')
    }
    browserWindow.__RF_OCR_STATUS__ = browserWindow.__RF_OCR_MODEL_CATALOG_STATUS__
    return browserWindow.__RF_OCR_STATUS__
  }
}

export async function recognizeImage(request: OcrRecognizeRequest): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.RecognizeImage(toBoundOcrRecognizeRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR recognize fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function queueRecognizeImage(request: OcrRecognizeRequest): Promise<OcrJobSnapshot> {
  try {
    return fromBoundOcrJobSnapshot(await RecordingFreedomService.QueueRecognizeImage(toBoundOcrRecognizeRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR queue fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function queueRecognizeScreenshot(id: string): Promise<OcrJobSnapshot> {
  try {
    return fromBoundOcrJobSnapshot(await RecordingFreedomService.QueueRecognizeScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot OCR queue fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function recognizeScreenshot(id: string): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.RecognizeScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot OCR fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function queueRecognizePinnedScreenshot(id: string): Promise<OcrJobSnapshot> {
  try {
    return fromBoundOcrJobSnapshot(await RecordingFreedomService.QueueRecognizePinnedScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser pinned OCR queue fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function recognizePinnedScreenshot(id: string): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.RecognizePinnedScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser pinned OCR fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function queueRecognizeWhiteboard(request: OcrWhiteboardRequest): Promise<OcrJobSnapshot> {
  try {
    return fromBoundOcrJobSnapshot(await RecordingFreedomService.QueueRecognizeWhiteboard({
      ...request,
      force: request.force === true } as BoundOcrWhiteboardRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard OCR queue fallback:', error)
    return browserQueuedWhiteboardOcrSnapshot(request)
  }
}

export async function recognizeWhiteboard(request: OcrWhiteboardRequest): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.RecognizeWhiteboard({
      ...request,
      force: request.force === true } as BoundOcrWhiteboardRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard OCR fallback:', error)
    throw new Error('OCR is local-only and is not available in browser preview')
  }
}

export async function translateOcr(request: OcrTranslateRequest): Promise<OcrTranslationResult> {
  try {
    return fromBoundOcrTranslationResult(await RecordingFreedomService.TranslateOcr(request as BoundOcrTranslateRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR translation fallback:', error)
    return browserTranslateOcr(request)
  }
}

export async function openOcrResult(resultId: string): Promise<OcrResult> {
  try {
    return fromBoundOcrResult(await RecordingFreedomService.OpenOcrResult(resultId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR result fallback:', error)
    return browserOcrResult(resultId)
  }
}

export async function readOcrResultImage(resultId: string): Promise<ScreenshotImage> {
  try {
    return fromBoundScreenshotImage(await RecordingFreedomService.ReadOcrResultImage(resultId))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR result image fallback:', error)
    return browserOcrResultImage(resultId)
  }
}

export async function cancelOcrJob(jobId: string): Promise<void> {
  try {
    await RecordingFreedomService.CancelOcrJob(jobId)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser OCR cancel fallback:', error)
  }
}

function fromBoundOcrModelInfo(model: BoundOcrModelInfo): OcrModelInfo {
  return {
    id: model.id,
    name: model.name,
    channel: model.channel,
    engine: model.engine,
    language: model.language ?? [],
    version: model.version,
    sourceUrl: model.sourceUrl,
    license: model.license,
    downloadAvailable: model.downloadAvailable === true,
    downloadBytes: typeof model.downloadBytes === 'number' ? model.downloadBytes : undefined,
    installed: model.installed === true,
    verified: model.verified === true,
    active: model.active === true,
    modelDir: model.modelDir,
    smokeImage: model.smokeImage,
    smokeExpected: model.smokeExpected,
    smokeAssetReady: model.smokeAssetReady === true,
    smokeError: model.smokeError,
    missingFiles: model.missingFiles ?? [],
    verificationError: model.verificationError }
}

function fromBoundOcrModelDownloadSnapshot(snapshot: BoundOcrModelDownloadSnapshot): OcrModelDownloadSnapshot {
  return {
    id: snapshot.id,
    modelId: snapshot.modelId,
    status: snapshot.status,
    downloadedBytes: Number(snapshot.downloadedBytes ?? 0),
    totalBytes: Number(snapshot.totalBytes ?? 0),
    percent: Number(snapshot.percent ?? 0),
    error: snapshot.error,
    model: snapshot.model ? fromBoundOcrModelInfo(snapshot.model) : undefined,
    startedAt: String(snapshot.startedAt ?? ''),
    updatedAt: String(snapshot.updatedAt ?? '') }
}

function fromBoundOcrModelDownloadEvent(event: BoundOcrModelDownloadEvent): {snapshot: OcrModelDownloadSnapshot} {
  return {
    snapshot: fromBoundOcrModelDownloadSnapshot(event.snapshot) }
}

function fromBoundOcrStatus(status: BoundOcrStatus): OcrStatus {
  return {
    status: status.status,
    activeModelId: status.activeModelId,
    models: (status.models ?? []).map(fromBoundOcrModelInfo),
    workerPath: status.workerPath,
    runtimeDir: status.runtimeDir,
    workerCapabilities: status.workerCapabilities ? fromBoundOcrWorkerCapabilities(status.workerCapabilities) : undefined,
    message: status.message }
}

function fromBoundOcrJobSnapshot(snapshot: BoundOcrJobSnapshot): OcrJobSnapshot {
  return {
    jobId: snapshot.jobId,
    status: snapshot.status,
    cacheKey: snapshot.cacheKey,
    request: fromBoundOcrRecognizeRequest(snapshot.request),
    merged: snapshot.merged === true,
    createdAt: String(snapshot.createdAt ?? ''),
    updatedAt: String(snapshot.updatedAt ?? '') }
}

function fromBoundOcrJobEvent(event: BoundOcrJobEvent): OcrJobUpdate {
  return {
    jobId: event.jobId,
    sourceKind: fromBoundOcrSourceKind(event.sourceKind),
    sourceId: event.sourceId,
    status: event.status,
    cacheKey: event.cacheKey,
    merged: event.merged,
    error: event.error,
    result: event.result ? fromBoundOcrResult(event.result) : undefined }
}

function fromBrowserOcrJobEvent(event: Partial<OcrJobUpdate>): OcrJobUpdate {
  const result = event.result ? fromBrowserOcrResult(event.result) : undefined
  return {
    jobId: String(event.jobId || result?.id || `browser-ocr-${Date.now()}`),
    sourceKind: fromOcrSourceKindString(String(event.sourceKind || result?.sourceKind || 'image')),
    sourceId: String(event.sourceId || result?.sourceId || ''),
    status: String(event.status || (result ? 'ready' : 'queued')),
    cacheKey: event.cacheKey,
    merged: event.merged,
    error: event.error,
    result }
}

function fromBrowserOcrResult(result: Partial<OcrResult>): OcrResult {
  const sourceKind = fromOcrSourceKindString(String(result.sourceKind || 'image'))
  return {
    id: String(result.id || `browser-ocr-result-${Date.now()}`),
    sourceKind,
    sourceId: String(result.sourceId || ''),
    imagePath: String(result.imagePath || ''),
    imageSha256: String(result.imageSha256 || ''),
    modelId: String(result.modelId || 'browser-ocr-e2e'),
    language: String(result.language || 'zh-en'),
    width: finitePositiveInteger(result.width, 1),
    height: finitePositiveInteger(result.height, 1),
    blocks: Array.isArray(result.blocks) ? result.blocks.map(fromBrowserOcrBlock) : [],
    plainText: String(result.plainText || ''),
    createdAt: String(result.createdAt || new Date().toISOString()),
    durationMs: finiteNonNegativeNumber(result.durationMs, 0) }
}

function fromBrowserOcrBlock(block: Partial<OcrBlock>, index: number): OcrBlock {
  return {
    id: String(block.id || `browser-block-${index}`),
    text: String(block.text || ''),
    confidence: finiteNonNegativeNumber(block.confidence, 0),
    box: Array.isArray(block.box) ? block.box.map(fromBrowserOcrPoint).filter((point): point is OcrPoint => Boolean(point)) : [],
    lineIndex: Number.isFinite(block.lineIndex) ? Number(block.lineIndex) : index,
    languageHint: block.languageHint }
}

function fromBrowserOcrPoint(point: unknown): OcrPoint | null {
  if (Array.isArray(point) && point.length >= 2) {
    const x = Number(point[0])
    const y = Number(point[1])
    if (Number.isFinite(x) && Number.isFinite(y)) return {x, y}
  }
  if (point && typeof point === 'object') {
    const candidate = point as {x?: unknown; y?: unknown}
    const x = Number(candidate.x)
    const y = Number(candidate.y)
    if (Number.isFinite(x) && Number.isFinite(y)) return {x, y}
  }
  return null
}

function fromBoundOcrWorkerCapabilities(capabilities: BoundOcrWorkerCapabilities): OcrWorkerCapabilities {
  return {
    schemaVersion: capabilities.schemaVersion,
    name: capabilities.name,
    version: capabilities.version,
    protocolVersion: capabilities.protocolVersion,
    engine: capabilities.engine,
    modelFormats: capabilities.modelFormats ?? [],
    supportsRecognize: capabilities.supportsRecognize === true,
    runtimeDir: capabilities.runtimeDir,
    runtimeLibrary: capabilities.runtimeLibrary,
    runtimeAvailable: capabilities.runtimeAvailable === true,
    runtimeVersion: capabilities.runtimeVersion,
    runtimeApiVersion: capabilities.runtimeApiVersion,
    runtimeError: capabilities.runtimeError,
    message: capabilities.message }
}

function fromBoundOcrRecognizeRequest(request: BoundOcrRecognizeRequest): OcrRecognizeRequest {
  return {
    imagePath: request.imagePath,
    sourceKind: fromBoundOcrSourceKind(request.sourceKind),
    sourceId: request.sourceId,
    language: request.language,
    modelId: request.modelId,
    force: request.force,
    priority: request.priority }
}

function fromBoundOcrResult(result: BoundOcrResult): OcrResult {
  return {
    id: result.id,
    sourceKind: fromBoundOcrSourceKind(result.sourceKind),
    sourceId: result.sourceId,
    imagePath: result.imagePath,
    imageSha256: result.imageSha256,
    modelId: result.modelId,
    language: result.language,
    width: result.width,
    height: result.height,
    blocks: (result.blocks ?? []).map(fromBoundOcrBlock),
    plainText: result.plainText,
    createdAt: String(result.createdAt ?? ''),
    durationMs: result.durationMs }
}

function fromBoundOcrBlock(block: BoundOcrBlock): OcrBlock {
  return {
    id: block.id,
    text: block.text,
    confidence: block.confidence,
    box: (block.box ?? []).map((point) => ({x: point.x, y: point.y})),
    lineIndex: block.lineIndex,
    languageHint: block.languageHint }
}

function fromBoundOcrTranslationResult(result: BoundOcrTranslationResult): OcrTranslationResult {
  return {
    ocrResultId: result.ocrResultId,
    provider: result.provider,
    sourceLanguage: result.sourceLanguage,
    targetLanguage: result.targetLanguage,
    model: result.model,
    promptVersion: result.promptVersion,
    blocks: (result.blocks ?? []).map(fromBoundOcrTranslationBlock),
    createdAt: String(result.createdAt ?? '') }
}

function fromBoundOcrTranslationBlock(block: BoundOcrTranslationBlock): OcrTranslationBlock {
  return {
    blockId: block.blockId,
    source: block.source,
    translated: block.translated }
}

function toBoundOcrRecognizeRequest(request: OcrRecognizeRequest): BoundOcrRecognizeRequest {
  return {
    imagePath: request.imagePath,
    sourceKind: toBoundOcrSourceKind(request.sourceKind),
    sourceId: request.sourceId,
    language: request.language ?? 'zh-en',
    modelId: request.modelId,
    force: request.force === true,
    priority: request.priority ?? 'normal' }
}

function fromBoundOcrSourceKind(sourceKind: BoundOcrSourceKind): OcrSourceKind {
  return fromOcrSourceKindString(String(sourceKind))
}

function fromOcrSourceKindString(sourceKind: string): OcrSourceKind {
  if (isOcrSourceKind(sourceKind)) return sourceKind
  return 'image'
}

function toBoundOcrSourceKind(sourceKind: OcrSourceKind): BoundOcrSourceKind {
  switch (sourceKind) {
    case 'region-screenshot':
      return BoundOcrSourceKind.SourceRegionScreenshot
    case 'full-screenshot':
      return BoundOcrSourceKind.SourceFullScreenshot
    case 'window-screenshot':
      return BoundOcrSourceKind.SourceWindowScreenshot
    case 'focused-window-screenshot':
      return BoundOcrSourceKind.SourceFocusedWindowScreenshot
    case 'scrolling-screenshot':
      return BoundOcrSourceKind.SourceScrollingScreenshot
    case 'pinned-screenshot':
      return BoundOcrSourceKind.SourcePinnedScreenshot
    case 'whiteboard':
      return BoundOcrSourceKind.SourceWhiteboard
    case 'whiteboard-selection':
      return BoundOcrSourceKind.SourceWhiteboardSelection
    default:
      return BoundOcrSourceKind.SourceImage
  }
}

function isOcrSourceKind(value: string): value is OcrSourceKind {
  return value === 'region-screenshot' ||
    value === 'full-screenshot' ||
    value === 'window-screenshot' ||
    value === 'focused-window-screenshot' ||
    value === 'scrolling-screenshot' ||
    value === 'pinned-screenshot' ||
    value === 'whiteboard' ||
    value === 'whiteboard-selection' ||
    value === 'image'
}

type BrowserOcrModelDownloadWindow = Window & {
  __RF_OCR_STATUS__?: OcrStatus
  __RF_OCR_MODEL_DOWNLOADS__?: Record<string, OcrModelDownloadSnapshot>
  __RF_LAST_OCR_MODEL_DOWNLOAD__?: {modelId: string; at: string}
}

function browserOcrModelDownloads(): Record<string, OcrModelDownloadSnapshot> {
  const browserWindow = window as BrowserOcrModelDownloadWindow
  if (!browserWindow.__RF_OCR_MODEL_DOWNLOADS__) browserWindow.__RF_OCR_MODEL_DOWNLOADS__ = {}
  return browserWindow.__RF_OCR_MODEL_DOWNLOADS__
}

function browserStoreOcrModelDownload(snapshot: OcrModelDownloadSnapshot) {
  browserOcrModelDownloads()[snapshot.modelId] = snapshot
  window.dispatchEvent(new CustomEvent(browserOcrModelDownloadEvent, {detail: snapshot}))
}

function fromBrowserOcrModelDownloadEvent(snapshot: Partial<OcrModelDownloadSnapshot>): {snapshot: OcrModelDownloadSnapshot} {
  const now = new Date().toISOString()
  return {
    snapshot: {
      id: snapshot.id || `browser-ocr-model-download-${snapshot.modelId || 'unknown'}`,
      modelId: snapshot.modelId || '',
      status: snapshot.status || 'queued',
      downloadedBytes: Number(snapshot.downloadedBytes ?? 0),
      totalBytes: Number(snapshot.totalBytes ?? 0),
      percent: Number(snapshot.percent ?? 0),
      error: snapshot.error,
      model: snapshot.model,
      startedAt: snapshot.startedAt || now,
      updatedAt: snapshot.updatedAt || now } }
}

function browserStartOcrModelDownload(modelId: string): OcrModelDownloadSnapshot {
  const browserWindow = window as BrowserOcrModelDownloadWindow
  const status = browserOcrStatus()
  const target = status.models.find((model) => model.id === modelId)
  if (!target?.downloadAvailable) {
    throw new Error(`OCR model ${modelId} does not have a verified RecordingFreedom package download`)
  }
  browserWindow.__RF_LAST_OCR_MODEL_DOWNLOAD__ = {modelId, at: new Date().toISOString()}
  const now = new Date().toISOString()
  const totalBytes = target.downloadBytes || 1
  const queued: OcrModelDownloadSnapshot = {
    id: `browser-ocr-model-download-${modelId}-${Date.now()}`,
    modelId,
    status: 'queued',
    downloadedBytes: 0,
    totalBytes,
    percent: 0,
    startedAt: now,
    updatedAt: now }
  browserStoreOcrModelDownload(queued)
  window.setTimeout(() => {
    const running: OcrModelDownloadSnapshot = {
      ...queued,
      status: 'running',
      downloadedBytes: Math.max(1, Math.round(totalBytes * 0.55)),
      percent: 55,
      updatedAt: new Date().toISOString() }
    browserStoreOcrModelDownload(running)
  }, 10)
  window.setTimeout(() => {
    const current = browserOcrStatus()
    const installedModel = current.models.find((model) => model.id === modelId) || target
    const verifiedModel: OcrModelInfo = {
      ...installedModel,
      installed: true,
      verified: true,
      active: installedModel.active === true,
      smokeAssetReady: true,
      missingFiles: [],
      verificationError: undefined,
      smokeError: undefined }
    browserWindow.__RF_OCR_STATUS__ = {
      ...current,
      models: current.models.map((model) => model.id === modelId ? verifiedModel : model) }
    browserStoreOcrModelDownload({
      ...queued,
      status: 'installed',
      downloadedBytes: totalBytes,
      percent: 100,
      model: verifiedModel,
      updatedAt: new Date().toISOString() })
  }, 25)
  return queued
}

function browserCancelOcrModelDownload(modelId: string): OcrModelDownloadSnapshot {
  const current = browserOcrModelDownloads()[modelId]
  const now = new Date().toISOString()
  const cancelled: OcrModelDownloadSnapshot = {
    ...(current || {
      id: `browser-ocr-model-download-${modelId}-${Date.now()}`,
      modelId,
      downloadedBytes: 0,
      totalBytes: 0,
      percent: 0,
      startedAt: now }),
    status: 'cancelled',
    updatedAt: now }
  browserStoreOcrModelDownload(cancelled)
  return cancelled
}

function browserOcrStatus(): OcrStatus {
  const injected = (window as Window & {__RF_OCR_STATUS__?: OcrStatus}).__RF_OCR_STATUS__
  if (injected) {
    return {
      ...injected,
      models: injected.models ?? [] }
  }
  return {
    status: 'no-model',
    activeModelId: 'ppocrv5-mobile-zh-en',
    models: [
      {
        id: 'ppocrv5-mobile-zh-en',
        name: 'PP-OCRv5 Mobile Chinese/English',
        channel: 'stable',
        engine: 'onnxruntime',
        language: ['zh', 'en'],
        downloadAvailable: false,
        downloadBytes: 0,
        installed: false,
        verified: false,
        active: true,
        smokeAssetReady: false,
        missingFiles: ['manifest.json', 'det.onnx', 'cls.onnx', 'rec.onnx', 'keys.txt'] },
      {
        id: 'ppocrv6-mobile-zh-en',
        name: 'PP-OCRv6 Mobile Chinese/English',
        channel: 'latest',
        engine: 'onnxruntime',
        language: ['zh', 'en'],
        downloadAvailable: false,
        downloadBytes: 0,
        installed: false,
        verified: false,
        active: false,
        smokeAssetReady: false,
        missingFiles: ['manifest.json', 'det.onnx', 'cls.onnx', 'rec.onnx', 'keys.txt'] },
      {
        id: 'ppocrv6-medium-zh-en',
        name: 'PP-OCRv6 Medium Chinese/English',
        channel: 'quality',
        engine: 'onnxruntime',
        language: ['zh', 'en'],
        downloadAvailable: false,
        downloadBytes: 0,
        installed: false,
        verified: false,
        active: false,
        smokeAssetReady: false,
        missingFiles: ['manifest.json', 'det.onnx', 'cls.onnx', 'rec.onnx', 'keys.txt'] },
    ],
    message: 'OCR is local-only and is not available in browser preview.' }
}

function browserOcrResult(resultId: string): OcrResult {
  const injected = browserInjectedOcrResult(resultId)
  if (injected) return injected
  const item = loadBrowserScreenshotHistory().find((entry) => entry.ocrResultId === resultId)
  if (!item || item.ocrStatus !== 'ready') {
    throw new Error(`OCR result ${resultId} is not available in browser preview`)
  }
  const width = Math.max(64, item.width || 480)
  const height = Math.max(64, item.height || 320)
  const titleTop = Math.round(height * 0.17)
  const subtitleTop = Math.round(height * 0.36)
  const titleBottom = Math.min(height - 12, titleTop + Math.max(26, Math.round(height * 0.12)))
  const subtitleBottom = Math.min(height - 10, subtitleTop + Math.max(24, Math.round(height * 0.11)))
  const left = Math.round(width * 0.08)
  const titleRight = Math.min(width - left, left + Math.round(width * 0.56))
  const subtitleRight = Math.min(width - left, left + Math.round(width * 0.42))
  return {
    id: resultId,
    sourceKind: browserScreenshotOcrSourceKind(item.mode),
    sourceId: item.id,
    imagePath: item.path,
    imageSha256: `browser-preview-${item.id}`,
    modelId: item.ocrModelId || 'ppocrv5-mobile-zh-en',
    language: item.ocrLanguage || 'zh-en',
    width,
    height,
    blocks: [
      {
        id: `${resultId}-block-title`,
        text: 'RecordingFreedom',
        confidence: 0.97,
        lineIndex: 0,
        languageHint: 'en',
        box: [
          {x: left, y: titleTop},
          {x: titleRight, y: titleTop},
          {x: titleRight, y: titleBottom},
          {x: left, y: titleBottom},
        ] },
      {
        id: `${resultId}-block-zh`,
        text: '文字识别',
        confidence: 0.94,
        lineIndex: 1,
        languageHint: 'zh',
        box: [
          {x: left, y: subtitleTop},
          {x: subtitleRight, y: subtitleTop},
          {x: subtitleRight, y: subtitleBottom},
          {x: left, y: subtitleBottom},
        ] },
    ],
    plainText: 'RecordingFreedom\n文字识别',
    createdAt: item.ocrUpdatedAt || item.createdAt,
    durationMs: 126 }
}

function browserOcrResultImage(resultId: string): ScreenshotImage {
  const injected = browserInjectedOcrImage(resultId)
  if (injected) return injected
  const item = loadBrowserScreenshotHistory().find((entry) => entry.ocrResultId === resultId)
  if (!item || item.ocrStatus !== 'ready') {
    return {available: false}
  }
  const dataUrl = browserScreenshotDataUrl(item)
  return {
    available: Boolean(dataUrl),
    dataUrl,
    path: item.path,
    bytes: dataUrl?.length ?? 0 }
}

function browserInjectedOcrResult(resultId: string): OcrResult | null {
  const injected = (window as Window & {__RF_OCR_RESULTS__?: Record<string, Partial<OcrResult>>}).__RF_OCR_RESULTS__?.[resultId]
  if (!injected) return null
  return fromBrowserOcrResult(injected)
}

function browserInjectedOcrImage(resultId: string): ScreenshotImage | null {
  const injected = (window as Window & {__RF_OCR_IMAGES__?: Record<string, Partial<ScreenshotImage>>}).__RF_OCR_IMAGES__?.[resultId]
  if (!injected) return null
  const dataUrl = typeof injected.dataUrl === 'string' && injected.dataUrl.startsWith('data:image/') ? injected.dataUrl : undefined
  return {
    available: Boolean(dataUrl),
    dataUrl,
    path: typeof injected.path === 'string' ? injected.path : undefined,
    bytes: typeof injected.bytes === 'number' && Number.isFinite(injected.bytes) ? injected.bytes : dataUrl?.length ?? 0 }
}

function browserTranslateOcr(request: OcrTranslateRequest): OcrTranslationResult {
  if (!request.provider || request.provider === 'disabled') throw new Error('OCR translation provider is disabled')
  if (!request.baseUrl?.trim()) throw new Error('OCR translation base URL is required')
  if (!request.apiKey?.trim()) throw new Error('OCR translation API key is required')
  if (request.provider === 'openai-compatible' && !request.model?.trim()) throw new Error('OpenAI-compatible translation model is required')
  const result = browserOcrResultForTranslation(request.ocrResultId)
  const selected = new Set((request.blockIds ?? []).filter(Boolean))
  const blocks = result.blocks
    .filter((block) => selected.size === 0 || selected.has(block.id))
    .map((block) => ({
      blockId: block.id,
      source: block.text,
      translated: browserTranslatedText(block.text, request.targetLanguage) }))
    .filter((block) => block.source.trim() && block.translated.trim())
  if (blocks.length === 0) throw new Error('No OCR text to translate')
  return {
    ocrResultId: request.ocrResultId,
    provider: request.provider,
    sourceLanguage: request.sourceLanguage || 'auto',
    targetLanguage: request.targetLanguage || 'zh-CN',
    model: request.model,
    promptVersion: 'browser-preview',
    blocks,
    createdAt: new Date().toISOString() }
}

function browserOcrResultForTranslation(resultId: string): OcrResult {
  try {
    return browserOcrResult(resultId)
  } catch (error) {
    if (resultId !== 'whiteboard-selection-result') throw error
    return {
      id: resultId,
      sourceKind: 'whiteboard-selection',
      sourceId: 'browser-preview/data/whiteboards/board-current.excalidraw',
      imagePath: 'browser-preview/data/whiteboards/exports/whiteboard.png',
      imageSha256: 'browser-e2e-selection-image',
      modelId: 'ppocrv5-mobile-zh-en',
      language: 'zh-en',
      width: 320,
      height: 180,
      blocks: [
        {
          id: 'block-recordingfreedom',
          text: 'RecordingFreedom',
          confidence: 0.97,
          lineIndex: 0,
          languageHint: 'en',
          box: [] },
        {
          id: 'block-chinese',
          text: '文字识别',
          confidence: 0.94,
          lineIndex: 1,
          languageHint: 'zh',
          box: [] },
      ],
      plainText: 'RecordingFreedom\n文字识别',
      createdAt: '2026-07-06T10:00:00.000Z',
      durationMs: 126 }
  }
}

function browserTranslatedText(text: string, targetLanguage: string) {
  const trimmed = text.trim()
  if (!trimmed) return ''
  if (targetLanguage.toLowerCase().startsWith('zh')) {
    if (trimmed === 'RecordingFreedom') return '录制自由'
    if (trimmed === '文字识别') return '文字识别'
  }
  if (trimmed === 'RecordingFreedom') return 'RecordingFreedom translated'
  if (trimmed === '文字识别') return 'Text recognition translated'
  return `${trimmed} translated`
}

function browserQueuedWhiteboardOcrSnapshot(request: OcrWhiteboardRequest): OcrJobSnapshot {
  const sourceId = (request.sceneId || request.elementId || 'browser-whiteboard').trim()
  const recognizeRequest: OcrRecognizeRequest = {
    imagePath: request.imagePath,
    sourceKind: request.elementId ? 'whiteboard-selection' : 'whiteboard',
    sourceId,
    language: request.language || 'zh-en',
    force: request.force === true,
    priority: request.priority || 'interactive' }
  const snapshot: OcrJobSnapshot = {
    jobId: `browser-whiteboard-ocr-${Date.now()}`,
    status: 'queued',
    request: recognizeRequest,
    merged: false,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString() }
  ;(window as Window & {
    __RF_LAST_WHITEBOARD_OCR_QUEUE__?: OcrJobSnapshot
    __RF_LAST_WHITEBOARD_OCR_REQUEST__?: OcrWhiteboardRequest
  }).__RF_LAST_WHITEBOARD_OCR_QUEUE__ = snapshot
  ;(window as Window & {__RF_LAST_WHITEBOARD_OCR_REQUEST__?: OcrWhiteboardRequest}).__RF_LAST_WHITEBOARD_OCR_REQUEST__ = request
  return snapshot
}

function browserScreenshotOcrSourceKind(mode: string): OcrSourceKind {
  switch (mode) {
    case 'full':
    case 'screen':
      return 'full-screenshot'
    case 'window':
      return 'window-screenshot'
    case 'focused-window':
      return 'focused-window-screenshot'
    case 'scrolling':
      return 'scrolling-screenshot'
    case 'whiteboard':
      return 'whiteboard'
    default:
      return 'region-screenshot'
  }
}

export function fromBoundOcrSettings(value: Partial<AppSettings['ocr']> | undefined): AppSettings['ocr'] {
  return {
    autoRecognizeScreenshots: value?.autoRecognizeScreenshots === true,
    translation: normalizeOcrTranslationSettings(value?.translation) }
}

export function normalizeOcrTranslationSettings(value: Partial<AppSettings['ocr']['translation']> | undefined): AppSettings['ocr']['translation'] {
  const provider = value?.provider === 'deepl' || value?.provider === 'openai-compatible' ? value.provider : 'disabled'
  const privacyConfirmed = provider !== 'disabled' && value?.privacyConfirmed === true
  return {
    provider,
    baseUrl: trimOptionalString(value?.baseUrl),
    apiKey: trimOptionalString(value?.apiKey),
    apiKeySet: value?.apiKeySet === true || Boolean(trimOptionalString(value?.apiKey)),
    model: trimOptionalString(value?.model),
    sourceLanguage: trimOptionalString(value?.sourceLanguage) || 'auto',
    targetLanguage: trimOptionalString(value?.targetLanguage) || 'zh-CN',
    privacyConfirmed,
    privacyConfirmedAt: privacyConfirmed ? trimOptionalString(value?.privacyConfirmedAt) : undefined }
}
