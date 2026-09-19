import {Events} from '@wailsio/runtime'
import {RecordingFreedomService} from '../../../bindings/github.com/lemon-casino/RecordingFreedom/app'
import {
  type AnnotationCaptureRequest as BoundAnnotationCaptureRequest,
  type AnnotationCaptureResult as BoundAnnotationCaptureResult,
  type AnnotationOverlayState as BoundAnnotationOverlayState,
  type AnnotationPreviewImageRequest as BoundAnnotationPreviewImageRequest,
  type AnnotationPreviewImageResult as BoundAnnotationPreviewImageResult,
  type AnnotationRenderJobClaim as BoundAnnotationRenderJobClaim,
  type AnnotationRenderJobResult as BoundAnnotationRenderJobResult,
  type RegionAssistRequest as BoundRegionAssistRequest,
  type RegionAssistResult as BoundRegionAssistResult,
  type RegionSelectionRequest as BoundRegionSelectionRequest,
  type RegionSelectionResult as BoundRegionSelectionResult,
  type RegionSelectionSession as BoundRegionSelectionSession,
  type RegionSmartCandidate as BoundRegionSmartCandidate,
  type ScreenshotCaptureRequest as BoundScreenshotCaptureRequest,
  type ScreenshotCaptureResult as BoundScreenshotCaptureResult,
  type ScreenshotHistoryResult as BoundScreenshotHistoryResult,
  type ScreenshotImageRequest as BoundScreenshotImageRequest,
  type ScreenshotItem as BoundScreenshotItem,
  type ScreenshotItemPatchRequest as BoundScreenshotItemPatchRequest,
  type ScreenshotPinState as BoundScreenshotPinState,
  type ScreenshotWhiteboardContext as BoundScreenshotWhiteboardContext,
  type WhiteboardExportRequest as BoundWhiteboardExportRequest,
  type WhiteboardExportResult as BoundWhiteboardExportResult,
  type WhiteboardSceneRequest as BoundWhiteboardSceneRequest,
  type WhiteboardSceneResult as BoundWhiteboardSceneResult,
  type WhiteboardSnapshotRequest as BoundWhiteboardSnapshotRequest,
  type WhiteboardSnapshotResult as BoundWhiteboardSnapshotResult } from '../../../bindings/github.com/lemon-casino/RecordingFreedom/app/models'
import {type CaptureSource as BoundCaptureSource} from '../../../bindings/github.com/lemon-casino/RecordingFreedom/app/internal/devices/models'
import {type CaptureSource, type ScreenshotItem} from '../mockBackend'
import {browserScreenshotDataUrl, browserScreenshotHistoryKey, fromBoundRegionRect, fromBoundScreenshotImage, fromBoundSource, fromBrowserScreenshotItem, isWailsDesktopRuntime, loadBrowserScreenshotHistory, normalizeOcrStatus, safeJSON, themedPopupURL, type ScreenshotImage} from './shared'

export type ScreenshotPinState = {
  visible: boolean
  item?: ScreenshotItem
  dataUrl?: string
  fixed: boolean
  pins?: ScreenshotPinnedItem[]
}

export type ScreenshotPinnedItem = {
  item: ScreenshotItem
  dataUrl?: string
  fixed: boolean
}

export type ScreenshotWhiteboardContext = {
  available: boolean
  item?: ScreenshotItem
  dataUrl?: string
}

export type WhiteboardScene = {
  available: boolean
  scenePath: string
  sceneJson?: string
  bytes: number
  updatedAt?: string
  contentType?: string
}

export type WhiteboardExport = {
  format: 'png' | 'svg' | 'excalidraw'
  outputPath: string
  bytes: number
}

export type AnnotationOverlayState = {
  mode?: 'annotation' | 'screenshot'
  packageDir?: string
  manifestPath?: string
  windowBounds: {x: number; y: number; width: number; height: number}
  canvasBounds: {x: number; y: number; width: number; height: number}
  toolbarBounds?: {x: number; y: number; width: number; height: number}
  toolbarPlacement?: 'top' | 'bottom'
  target: {
    type: string
    id: string
    geometry?: {x: number; y: number; width: number; height: number; displayIndex?: number; nativeId?: string}
  }
  captureExcluded: boolean
  sourceImageDataURL?: string
  sourceImageCapturedAt?: string
}

export type AnnotationCapture = {
  packageDir: string
  scenePath: string
  eventsPath: string
  snapshotPath: string
  timelineSnapshotPath?: string
  bytes: number
}

export type AnnotationRenderJob = {
  id: string
  packageDir: string
  scenePath: string
  relativeScenePath: string
  outputPath: string
  relativeOutputPath: string
  sceneJson: string
  canvasWidth: number
  canvasHeight: number
  index: number
  startOffsetMs?: number
  endOffsetMs?: number
}

export type AnnotationRenderJobClaim = {
  available: boolean
  job?: AnnotationRenderJob
}

export type RegionSelectionPurpose = 'capture' | 'annotation' | 'screenshot' | 'scrolling-screenshot'

export type RegionSmartCandidate = {
  id: string
  kind: 'screen' | 'window' | 'element' | 'edge' | string
  label?: string
  bounds: {x: number; y: number; width: number; height: number}
  sourceId?: string
  score?: number
}

export type RegionDisplayBounds = {
  id?: string
  bounds: {x: number; y: number; width: number; height: number}
  captureBounds: {x: number; y: number; width: number; height: number}
  scaleFactor?: number
}

export type RegionSelectionSession = {
  id: string
  bounds: {x: number; y: number; width: number; height: number}
  captureBounds?: {x: number; y: number; width: number; height: number}
  displayBounds?: RegionDisplayBounds[]
  minimumWidth: number
  minimumHeight: number
  displayCount: number
  purpose?: RegionSelectionPurpose
  candidates?: RegionSmartCandidate[]
  initialPointer?: {x: number; y: number}
}

export type RegionAssistRequest = {
  sessionId?: string
  purpose?: RegionSelectionPurpose
  pointerX?: number
  pointerY?: number
  selection?: RegionSelectionSession['bounds']
  candidateLevel?: number
  candidates?: RegionSmartCandidate[]
}

export type RegionAssistResult = {
  candidates: RegionSmartCandidate[]
  best?: RegionSmartCandidate
  source?: 'element' | 'image-hover' | 'selection' | 'static'
}

export type RegionSelectionResult = {
  sessionId?: string
  source?: CaptureSource
  geometry?: {x: number; y: number; width: number; height: number}
  cancelled: boolean
  error?: string
}

export type AnnotationPreviewImage = {
  available: boolean
  dataUrl?: string
  relativePath?: string
  bytes?: number
}

export type WhiteboardVisibilityMode = 'whiteboard' | 'annotation'

export type WhiteboardVisibilityUpdate = {
  visible: boolean
  mode: WhiteboardVisibilityMode
}

const browserWhiteboardSceneKey = 'recordingfreedom.whiteboard.scene.v1'
const browserAnnotationSceneKey = 'recordingfreedom.annotation.scene.v1'
const browserScreenshotPinStateKey = 'recordingfreedom.screenshots.pin.v1'
const browserScreenshotWhiteboardKey = 'recordingfreedom.screenshots.whiteboard.v1'
const browserScreenshotAnnotationKey = 'recordingfreedom.screenshots.annotation.v1'
const browserWhiteboardVisibilityEvent = 'rf-whiteboard-visibility'
const browserScreenshotCapturedEvent = 'rf-screenshot-captured'
const browserScreenshotPinEvent = 'rf-screenshot-pin'
const browserScreenshotWhiteboardEvent = 'rf-screenshot-whiteboard'

export function subscribeRegionSelection(handler: (event: RegionSelectionResult) => void): () => void {
  try {
    return Events.On('capture.region.selected', (event) => {
      handler(fromBoundRegionSelectionResult(event.data as BoundRegionSelectionResult))
    })
  } catch (error) {
    console.info('Using browser region selection event fallback:', error)
    return () => {}
  }
}

export function subscribeWhiteboardVisibility(handler: (event: WhiteboardVisibilityUpdate) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('whiteboard.visibility', (event) => {
      handler(fromWhiteboardVisibilityEvent(event.data))
    })
  } catch (error) {
    console.info('Using browser whiteboard visibility event fallback:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler(fromWhiteboardVisibilityEvent((event as CustomEvent<WhiteboardVisibilityUpdate>).detail))
  }
  window.addEventListener(browserWhiteboardVisibilityEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserWhiteboardVisibilityEvent, onBrowserEvent)
  }
}

export function subscribeScreenshotCaptured(handler: (item: ScreenshotItem) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('screenshot.captured', (event) => {
      const record = event.data && typeof event.data === 'object' ? event.data as {item?: BoundScreenshotItem} : {}
      if (record.item) handler(fromBoundScreenshotItem(record.item))
    })
  } catch (error) {
    console.info('Desktop screenshot captured events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    const item = (event as CustomEvent<ScreenshotItem>).detail
    if (item) handler(item)
  }
  window.addEventListener(browserScreenshotCapturedEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserScreenshotCapturedEvent, onBrowserEvent)
  }
}

export function subscribeScreenshotHistoryChanged(handler: (items: ScreenshotItem[]) => void): () => void {
  try {
    return Events.On('screenshot.history.changed', (event) => {
      handler(fromBoundScreenshotHistory(event.data as BoundScreenshotHistoryResult))
    })
  } catch (error) {
    console.info('Using browser screenshot history event fallback:', error)
    return () => {}
  }
}

export function subscribeScreenshotPin(handler: (state: ScreenshotPinState) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('screenshot.pin', (event) => {
      handler(fromBoundScreenshotPinState(event.data as BoundScreenshotPinState))
    })
  } catch (error) {
    console.info('Desktop screenshot pin events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler(fromBrowserScreenshotPinState((event as CustomEvent<ScreenshotPinState>).detail))
  }
  window.addEventListener(browserScreenshotPinEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserScreenshotPinEvent, onBrowserEvent)
  }
}

export function subscribeScreenshotWhiteboardContext(handler: (context: ScreenshotWhiteboardContext) => void): () => void {
  let disposeDesktop = () => {}
  try {
    disposeDesktop = Events.On('screenshot.whiteboard', (event) => {
      handler(fromBoundScreenshotWhiteboardContext(event.data as BoundScreenshotWhiteboardContext))
    })
  } catch (error) {
    console.info('Desktop screenshot whiteboard events unavailable:', error)
  }
  const onBrowserEvent = (event: Event) => {
    handler(fromBrowserScreenshotWhiteboardContext((event as CustomEvent<ScreenshotWhiteboardContext>).detail))
  }
  window.addEventListener(browserScreenshotWhiteboardEvent, onBrowserEvent)
  return () => {
    disposeDesktop()
    window.removeEventListener(browserScreenshotWhiteboardEvent, onBrowserEvent)
  }
}

export async function showWhiteboardWindow(): Promise<void> {
  try {
    await RecordingFreedomService.ShowWhiteboardWindow()
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard window fallback:', error)
    ;(window as Window & {__RF_LAST_WHITEBOARD_LAUNCH__?: {mode: string; url: string; at: string}}).__RF_LAST_WHITEBOARD_LAUNCH__ = {
      mode: 'whiteboard',
      url: '/#/whiteboard',
      at: new Date().toISOString() }
    emitBrowserWhiteboardVisibility({visible: true, mode: 'whiteboard'})
    const popup = window.open(themedPopupURL('/#/whiteboard'), 'recordingfreedom-whiteboard', 'width=1120,height=760')
    popup?.focus()
  }
}

export async function showAnnotationOverlay(): Promise<AnnotationOverlayState> {
  try {
    return fromBoundAnnotationOverlayState(await RecordingFreedomService.ShowAnnotationOverlay())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation overlay fallback:', error)
    ;(window as Window & {__RF_LAST_WHITEBOARD_LAUNCH__?: {mode: string; url: string; at: string}}).__RF_LAST_WHITEBOARD_LAUNCH__ = {
      mode: 'annotation',
      url: '/#/annotation-overlay',
      at: new Date().toISOString() }
    emitBrowserWhiteboardVisibility({visible: true, mode: 'annotation'})
    const popup = window.open(themedPopupURL('/#/annotation-overlay'), 'recordingfreedom-annotation-overlay', 'width=1280,height=720')
    popup?.focus()
    return browserAnnotationOverlayState()
  }
}

export async function showAnnotationRegionSelector(): Promise<RegionSelectionSession> {
  try {
    return finalizeRegionSelectionSession(
      fromBoundRegionSelectionSession(await RecordingFreedomService.ShowAnnotationRegionSelector()),
      'annotation',
      `browser-annotation-region-${Date.now()}`,
    )
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation region selector fallback:', error)
    return emitBrowserRegionSession(browserRegionSelectionSession('annotation', `browser-annotation-region-${Date.now()}`))
  }
}

export async function completeAnnotationRegionSelection(request: RegionSelectionSession['bounds']): Promise<AnnotationOverlayState> {
  try {
    return fromBoundAnnotationOverlayState(await RecordingFreedomService.CompleteAnnotationRegionSelection(toBoundRegionSelectionRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation region completion fallback:', error)
    return browserAnnotationOverlayState()
  }
}

export async function showScreenshotRegionSelector(): Promise<RegionSelectionSession> {
  try {
    return finalizeRegionSelectionSession(
      fromBoundRegionSelectionSession(await RecordingFreedomService.ShowScreenshotRegionSelector()),
      'screenshot',
      `browser-screenshot-${Date.now()}`,
    )
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot region selector fallback:', error)
    return emitBrowserRegionSession(browserRegionSelectionSession('screenshot', `browser-screenshot-${Date.now()}`))
  }
}

export async function completeScreenshotRegionSelection(request: RegionSelectionSession['bounds']): Promise<ScreenshotItem> {
  try {
    const result = await RecordingFreedomService.CompleteScreenshotRegionSelection(toBoundRegionSelectionRequest(request))
    return fromBoundScreenshotCaptureResult(result as BoundScreenshotCaptureResult).item
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot completion fallback:', error)
    const item = createBrowserScreenshotItem('region', request)
    saveBrowserScreenshotHistory([item, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: item}))
    return item
  }
}

export async function completeScrollingScreenshotSelection(request: RegionSelectionSession['bounds']): Promise<ScreenshotItem> {
  try {
    const result = await RecordingFreedomService.CompleteScrollingScreenshotSelection(toBoundRegionSelectionRequest(request))
    return fromBoundScreenshotCaptureResult(result as BoundScreenshotCaptureResult).item
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser scrolling screenshot completion fallback:', error)
    const item = createBrowserScreenshotItem('region', request)
    saveBrowserScreenshotHistory([item, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: item}))
    return item
  }
}

export async function beginScreenshotRegionEdit(request: RegionSelectionSession['bounds']): Promise<void> {
  try {
    await RecordingFreedomService.BeginScreenshotRegionEdit(toBoundRegionSelectionRequest(request))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot region edit fallback:', error)
    emitBrowserRegionFrame(request, 'screenshot')
  }
}

export async function beginScreenshotAnnotationOverlay(request: RegionSelectionSession['bounds']): Promise<AnnotationOverlayState> {
  try {
    return fromBoundAnnotationOverlayState(await RecordingFreedomService.BeginScreenshotAnnotationOverlay(toBoundRegionSelectionRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot annotation overlay fallback:', error)
    const item = createBrowserScreenshotItem('region', request)
    const context: ScreenshotWhiteboardContext = {
      available: true,
      item,
      dataUrl: browserScreenshotDataUrl(item) }
    window.localStorage?.setItem(browserScreenshotAnnotationKey, JSON.stringify(context))
    const state = browserAnnotationOverlayState('screenshot', request)
    ;(window as Window & {__RF_ANNOTATION_OVERLAY__?: AnnotationOverlayState}).__RF_ANNOTATION_OVERLAY__ = state
    ;(window as Window & {__RF_LAST_WHITEBOARD_LAUNCH__?: {mode: string; url: string; at: string}}).__RF_LAST_WHITEBOARD_LAUNCH__ = {
      mode: 'screenshot',
      url: '/#/annotation-overlay',
      at: new Date().toISOString() }
    emitBrowserWhiteboardVisibility({visible: true, mode: 'annotation'})
    const popup = window.open(themedPopupURL('/#/annotation-overlay'), 'recordingfreedom-screenshot-annotation', 'width=1280,height=720')
    popup?.focus()
    return state
  }
}

export async function updateScreenshotRegionSelection(request: RegionSelectionSession['bounds']): Promise<void> {
  try {
    await RecordingFreedomService.UpdateScreenshotRegionSelection(toBoundRegionSelectionRequest(request))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot region update fallback:', error)
    emitBrowserRegionFrame(request, 'screenshot')
  }
}

export async function captureScreenshot(request: {mode?: string; region?: RegionSelectionSession['bounds']} = {}): Promise<ScreenshotItem> {
  try {
    const result = await RecordingFreedomService.CaptureScreenshot(request as unknown as BoundScreenshotCaptureRequest)
    return fromBoundScreenshotCaptureResult(result as BoundScreenshotCaptureResult).item
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot capture fallback:', error)
    const item = createBrowserScreenshotItem(request.mode ?? 'full', request.region)
    saveBrowserScreenshotHistory([item, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: item}))
    return item
  }
}

export async function reselectAnnotationRegion(): Promise<RegionSelectionSession> {
  try {
    return fromBoundRegionSelectionSession(await RecordingFreedomService.ReselectAnnotationRegion())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation region reselect fallback:', error)
    window.localStorage?.removeItem(browserAnnotationSceneKey)
    const session = await showAnnotationRegionSelector()
    ;(window as Window & {__RF_LAST_ANNOTATION_REGION_RESELECT__?: RegionSelectionSession}).__RF_LAST_ANNOTATION_REGION_RESELECT__ = session
    return session
  }
}

export async function reselectScreenshotAnnotationRegion(): Promise<RegionSelectionSession> {
  try {
    return fromBoundRegionSelectionSession(await RecordingFreedomService.ReselectScreenshotAnnotationRegion())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot annotation region reselect fallback:', error)
    window.localStorage?.removeItem(browserScreenshotAnnotationKey)
    const session = await showScreenshotRegionSelector()
    ;(window as Window & {__RF_LAST_SCREENSHOT_REGION_RESELECT__?: RegionSelectionSession}).__RF_LAST_SCREENSHOT_REGION_RESELECT__ = session
    return session
  }
}

export async function hideAnnotationOverlay(): Promise<void> {
  try {
    await RecordingFreedomService.HideAnnotationOverlay()
  } catch (error) {
    console.info('Using browser annotation overlay hide fallback:', error)
    emitBrowserWhiteboardVisibility({visible: false, mode: 'annotation'})
    window.close()
  }
}

export async function hideScreenshotAnnotationOverlay(): Promise<void> {
  try {
    await RecordingFreedomService.HideScreenshotAnnotationOverlay()
  } catch (error) {
    console.info('Using browser screenshot annotation hide fallback:', error)
    window.localStorage?.removeItem(browserScreenshotAnnotationKey)
    emitBrowserWhiteboardVisibility({visible: false, mode: 'annotation'})
    window.close()
  }
}

export async function loadAnnotationCapture(): Promise<WhiteboardScene> {
  try {
    return fromBoundWhiteboardScene(await RecordingFreedomService.LoadAnnotationCapture())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation capture load fallback:', error)
    const sceneJson = window.localStorage?.getItem(browserAnnotationSceneKey) ?? ''
    return {
      available: sceneJson.trim() !== '',
      scenePath: 'browser-preview/data/video/recording-preview.rfrec/annotations/scene.excalidraw',
      sceneJson,
      bytes: sceneJson.length,
      contentType: 'application/vnd.excalidraw+json' }
  }
}

export async function loadScreenshotAnnotationCapture(): Promise<ScreenshotWhiteboardContext> {
  try {
    return fromBoundScreenshotWhiteboardContext(await RecordingFreedomService.LoadScreenshotAnnotationCapture())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot annotation load fallback:', error)
    const raw = window.localStorage?.getItem(browserScreenshotAnnotationKey)
    const record = raw ? safeJSON(raw) : null
    return fromBoundScreenshotWhiteboardContext(record as Partial<BoundScreenshotWhiteboardContext> | undefined)
  }
}

export async function saveAnnotationCapture(request: {sceneJson: string; snapshotDataUrl: string; eventsJsonl?: string}): Promise<AnnotationCapture> {
  try {
    return fromBoundAnnotationCapture(await RecordingFreedomService.SaveAnnotationCapture(request as BoundAnnotationCaptureRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation capture fallback:', error)
    window.localStorage?.setItem(browserAnnotationSceneKey, request.sceneJson)
    return {
      packageDir: 'browser-preview/data/video/recording-preview.rfrec',
      scenePath: 'browser-preview/data/video/recording-preview.rfrec/annotations/scene.excalidraw',
      eventsPath: 'browser-preview/data/video/recording-preview.rfrec/annotations/events.jsonl',
      snapshotPath: 'browser-preview/data/video/recording-preview.rfrec/annotations/exports/annotation.png',
      timelineSnapshotPath: 'browser-preview/data/video/recording-preview.rfrec/annotations/snapshots/annotation-000001.png',
      bytes: request.sceneJson.length + request.snapshotDataUrl.length }
  }
}

export async function saveScreenshotAnnotationCapture(request: {sceneJson: string; snapshotDataUrl: string; eventsJsonl?: string}): Promise<ScreenshotItem> {
  try {
    const result = await RecordingFreedomService.SaveScreenshotAnnotationCapture(request as BoundAnnotationCaptureRequest)
    return fromBoundScreenshotCaptureResult(result as BoundScreenshotCaptureResult).item
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot annotation save fallback:', error)
    const raw = window.localStorage?.getItem(browserScreenshotAnnotationKey)
    const context = fromBoundScreenshotWhiteboardContext((raw ? safeJSON(raw) : null) as Partial<BoundScreenshotWhiteboardContext> | undefined)
    const item = context.item ?? createBrowserScreenshotItem('region')
    const saved = {
      ...item,
      id: `browser-screenshot-${Date.now()}`,
      createdAt: new Date().toISOString() }
    saveBrowserScreenshotHistory([saved, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: saved}))
    return saved
  }
}

export async function claimAnnotationRenderJob(): Promise<AnnotationRenderJobClaim> {
  try {
    return fromBoundAnnotationRenderJobClaim(await RecordingFreedomService.ClaimAnnotationRenderJob())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation renderer idle fallback:', error)
    return {available: false}
  }
}

export async function completeAnnotationRenderJob(result: {id: string; dataUrl?: string; error?: string}): Promise<void> {
  try {
    await RecordingFreedomService.CompleteAnnotationRenderJob(result as BoundAnnotationRenderJobResult)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser annotation renderer completion fallback:', error)
  }
}

export async function hideWhiteboardWindow(): Promise<void> {
  try {
    await RecordingFreedomService.HideWhiteboardWindow()
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard hide fallback:', error)
    emitBrowserWhiteboardVisibility({visible: false, mode: 'whiteboard'})
    window.close()
  }
}

export async function loadWhiteboardScene(): Promise<WhiteboardScene> {
  if (!isWailsDesktopRuntime()) {
    const sceneJson = window.localStorage?.getItem(browserWhiteboardSceneKey) ?? ''
    return {
      available: sceneJson.trim() !== '',
      scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
      sceneJson,
      bytes: sceneJson.length,
      contentType: 'application/vnd.excalidraw+json' }
  }
  try {
    return fromBoundWhiteboardScene(await RecordingFreedomService.LoadWhiteboardScene())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard scene fallback:', error)
    const sceneJson = window.localStorage?.getItem(browserWhiteboardSceneKey) ?? ''
    return {
      available: sceneJson.trim() !== '',
      scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
      sceneJson,
      bytes: sceneJson.length,
      contentType: 'application/vnd.excalidraw+json' }
  }
}

export async function saveWhiteboardScene(sceneJson: string): Promise<WhiteboardScene> {
  if (!isWailsDesktopRuntime()) {
    window.localStorage?.setItem(browserWhiteboardSceneKey, sceneJson)
    return {
      available: true,
      scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
      sceneJson,
      bytes: sceneJson.length,
      updatedAt: new Date().toISOString(),
      contentType: 'application/vnd.excalidraw+json' }
  }
  try {
    return fromBoundWhiteboardScene(await RecordingFreedomService.SaveWhiteboardScene({sceneJson} as BoundWhiteboardSceneRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard scene save fallback:', error)
    window.localStorage?.setItem(browserWhiteboardSceneKey, sceneJson)
    return {
      available: true,
      scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
      sceneJson,
      bytes: sceneJson.length,
      updatedAt: new Date().toISOString(),
      contentType: 'application/vnd.excalidraw+json' }
  }
}

export async function saveWhiteboardSnapshot(request: {sceneJson: string; snapshotDataUrl: string}): Promise<{scene: WhiteboardScene; item: ScreenshotItem}> {
  if (!isWailsDesktopRuntime()) {
    window.localStorage?.setItem(browserWhiteboardSceneKey, request.sceneJson)
    const item = createBrowserScreenshotItem('whiteboard')
    saveBrowserScreenshotHistory([item, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: item}))
    return {
      scene: {
        available: true,
        scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
        sceneJson: request.sceneJson,
        bytes: request.sceneJson.length,
        updatedAt: new Date().toISOString(),
        contentType: 'application/vnd.excalidraw+json' },
      item }
  }
  try {
    return fromBoundWhiteboardSnapshot(await RecordingFreedomService.SaveWhiteboardSnapshot(request as BoundWhiteboardSnapshotRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard snapshot save fallback:', error)
    window.localStorage?.setItem(browserWhiteboardSceneKey, request.sceneJson)
    const item = createBrowserScreenshotItem('whiteboard')
    saveBrowserScreenshotHistory([item, ...loadBrowserScreenshotHistory()])
    window.dispatchEvent(new CustomEvent(browserScreenshotCapturedEvent, {detail: item}))
    return {
      scene: {
        available: true,
        scenePath: 'browser-preview/data/whiteboards/board-current.excalidraw',
        sceneJson: request.sceneJson,
        bytes: request.sceneJson.length,
        updatedAt: new Date().toISOString(),
        contentType: 'application/vnd.excalidraw+json' },
      item }
  }
}

export async function saveWhiteboardExport(request: {format: 'png' | 'svg' | 'excalidraw'; dataUrl?: string; payload?: string}): Promise<WhiteboardExport> {
  try {
    return fromBoundWhiteboardExport(await RecordingFreedomService.SaveWhiteboardExport(request as BoundWhiteboardExportRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser whiteboard export fallback:', error)
    return {
      format: request.format,
      outputPath: `browser-preview/data/whiteboards/exports/whiteboard.${request.format}`,
      bytes: (request.payload ?? request.dataUrl ?? '').length }
  }
}

export async function listScreenshots(): Promise<ScreenshotItem[]> {
  try {
    const result = await RecordingFreedomService.ListScreenshots()
    return fromBoundScreenshotHistory(result as BoundScreenshotHistoryResult)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot history fallback:', error)
    return loadBrowserScreenshotHistory()
  }
}

export async function readScreenshotImage(id: string, thumbnail = false): Promise<ScreenshotImage> {
  try {
    return fromBoundScreenshotImage(await RecordingFreedomService.ReadScreenshotImage({id, thumbnail} as BoundScreenshotImageRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot image fallback:', error)
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id)
    return {
      available: Boolean(item),
      dataUrl: browserScreenshotDataUrl(item),
      path: item?.path,
      bytes: browserScreenshotDataUrl(item)?.length ?? 0 }
  }
}

export async function openScreenshot(id: string): Promise<ScreenshotItem | null> {
  try {
    return fromBoundScreenshotItem(await RecordingFreedomService.OpenScreenshot({id} as BoundScreenshotImageRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot open fallback:', error)
    return loadBrowserScreenshotHistory().find((item) => item.id === id) ?? null
  }
}

export async function openScreenshotDirectory(id: string): Promise<ScreenshotItem | null> {
  try {
    return fromBoundScreenshotItem(await RecordingFreedomService.OpenScreenshotDirectory({id} as BoundScreenshotImageRequest))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot directory fallback:', error)
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id) ?? null
    ;(window as Window & {__RF_LAST_OPEN_SCREENSHOT_DIRECTORY__?: {id: string; path?: string; at: string}}).__RF_LAST_OPEN_SCREENSHOT_DIRECTORY__ = {
      id,
      path: item?.path,
      at: new Date().toISOString() }
    return item
  }
}

export async function patchScreenshotItem(id: string, patch: {pinned?: boolean; fixed?: boolean}): Promise<ScreenshotItem[]> {
  try {
    const result = await RecordingFreedomService.PatchScreenshotItem({id, ...patch} as BoundScreenshotItemPatchRequest)
    return fromBoundScreenshotHistory(result as BoundScreenshotHistoryResult)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot patch fallback:', error)
    const next = loadBrowserScreenshotHistory().map((item) => item.id === id
      ? {
        ...item,
        pinned: false,
        fixed: patch.pinned === false ? false : patch.fixed ?? item.fixed }
      : item)
    saveBrowserScreenshotHistory(next)
    const pinned = fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
    if (screenshotPinStateContains(pinned, id)) {
      const updated = next.find((item) => item.id === id)
      const state = updated ? updateBrowserScreenshotPinStateItem(pinned, updated) : removeBrowserScreenshotPinStateItem(pinned, id)
      window.localStorage?.setItem(browserScreenshotPinStateKey, JSON.stringify(state))
      window.dispatchEvent(new CustomEvent(browserScreenshotPinEvent, {detail: state}))
    }
    return next
  }
}

export async function deleteScreenshotItem(id: string): Promise<ScreenshotItem[]> {
  try {
    const result = await RecordingFreedomService.DeleteScreenshotItem(id)
    return fromBoundScreenshotHistory(result as BoundScreenshotHistoryResult)
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot delete fallback:', error)
    const next = loadBrowserScreenshotHistory().filter((item) => item.id !== id)
    saveBrowserScreenshotHistory(next)
    const pinned = fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
    if (screenshotPinStateContains(pinned, id)) {
      const state = removeBrowserScreenshotPinStateItem(pinned, id)
      window.localStorage?.setItem(browserScreenshotPinStateKey, JSON.stringify(state))
      window.dispatchEvent(new CustomEvent(browserScreenshotPinEvent, {detail: state}))
    }
    return next
  }
}

export async function showPinnedScreenshot(id: string): Promise<ScreenshotPinState> {
  try {
    return fromBoundScreenshotPinState(await RecordingFreedomService.ShowPinnedScreenshot(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot pin fallback:', error)
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id)
    const current = fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
    const state = item ? appendBrowserScreenshotPinStateItem(current, {
      item,
      dataUrl: browserScreenshotDataUrl(item),
      fixed: item.fixed === true }) : fromBrowserScreenshotPinState({visible: false, fixed: false})
    window.localStorage?.setItem(browserScreenshotPinStateKey, JSON.stringify(state))
    window.dispatchEvent(new CustomEvent(browserScreenshotPinEvent, {detail: state}))
    return state
  }
}

export async function hidePinnedScreenshot(): Promise<void> {
  try {
    await RecordingFreedomService.HidePinnedScreenshot()
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot pin hide fallback:', error)
    const state = {visible: false, fixed: false}
    window.localStorage?.setItem(browserScreenshotPinStateKey, JSON.stringify(state))
    window.dispatchEvent(new CustomEvent(browserScreenshotPinEvent, {detail: state}))
  }
}

export async function loadPinnedScreenshot(): Promise<ScreenshotPinState> {
  if (!isWailsDesktopRuntime()) {
    return fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
  }
  try {
    return fromBoundScreenshotPinState(await RecordingFreedomService.LoadPinnedScreenshot())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot pin state fallback:', error)
    return fromBrowserScreenshotPinState(safeJSON(window.localStorage?.getItem(browserScreenshotPinStateKey)))
  }
}

export async function openScreenshotInWhiteboard(id: string): Promise<ScreenshotWhiteboardContext> {
  if (!isWailsDesktopRuntime()) {
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id)
    const context: ScreenshotWhiteboardContext = {
      available: Boolean(item),
      item,
      dataUrl: browserScreenshotDataUrl(item) }
    window.localStorage?.setItem(browserScreenshotWhiteboardKey, JSON.stringify(context))
    window.dispatchEvent(new CustomEvent(browserScreenshotWhiteboardEvent, {detail: context}))
    const popup = window.open(themedPopupURL('/#/whiteboard'), 'recordingfreedom-whiteboard', 'width=1120,height=760')
    popup?.focus()
    emitBrowserWhiteboardVisibility({visible: true, mode: 'whiteboard'})
    return context
  }
  try {
    return fromBoundScreenshotWhiteboardContext(await RecordingFreedomService.OpenScreenshotInWhiteboard(id))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot whiteboard fallback:', error)
    const item = loadBrowserScreenshotHistory().find((entry) => entry.id === id)
    const context: ScreenshotWhiteboardContext = {
      available: Boolean(item),
      item,
      dataUrl: browserScreenshotDataUrl(item) }
    window.localStorage?.setItem(browserScreenshotWhiteboardKey, JSON.stringify(context))
    window.dispatchEvent(new CustomEvent(browserScreenshotWhiteboardEvent, {detail: context}))
    const popup = window.open(themedPopupURL('/#/whiteboard'), 'recordingfreedom-whiteboard', 'width=1120,height=760')
    popup?.focus()
    emitBrowserWhiteboardVisibility({visible: true, mode: 'whiteboard'})
    return context
  }
}

export async function consumeScreenshotWhiteboardContext(): Promise<ScreenshotWhiteboardContext> {
  if (!isWailsDesktopRuntime()) {
    const context = fromBrowserScreenshotWhiteboardContext(safeJSON(window.localStorage?.getItem(browserScreenshotWhiteboardKey)))
    window.localStorage?.removeItem(browserScreenshotWhiteboardKey)
    return context
  }
  try {
    return fromBoundScreenshotWhiteboardContext(await RecordingFreedomService.ConsumeScreenshotWhiteboardContext())
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser screenshot whiteboard context fallback:', error)
    const context = fromBrowserScreenshotWhiteboardContext(safeJSON(window.localStorage?.getItem(browserScreenshotWhiteboardKey)))
    window.localStorage?.removeItem(browserScreenshotWhiteboardKey)
    return context
  }
}

export async function startScrollingScreenshot(): Promise<RegionSelectionSession> {
  try {
    return finalizeRegionSelectionSession(
      fromBoundRegionSelectionSession(await RecordingFreedomService.StartScrollingScreenshot()),
      'scrolling-screenshot',
      `browser-scrolling-screenshot-${Date.now()}`,
    )
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser scrolling screenshot fallback:', error)
    const session = emitBrowserRegionSession(browserRegionSelectionSession('scrolling-screenshot', `browser-scrolling-screenshot-${Date.now()}`))
    const popup = window.open(themedPopupURL('/#/region-overlay'), 'recordingfreedom-scrolling-screenshot', 'width=1280,height=720')
    popup?.focus()
    return session
  }
}

export async function showRegionSelector(): Promise<RegionSelectionSession> {
  try {
    return finalizeRegionSelectionSession(
      fromBoundRegionSelectionSession(await RecordingFreedomService.ShowRegionSelector()),
      'capture',
      `browser-region-${Date.now()}`,
    )
  } catch (error) {
    console.info('Using browser region selector fallback:', error)
    return emitBrowserRegionSession(browserRegionSelectionSession('capture', `browser-region-${Date.now()}`))
  }
}

export async function completeRegionSelection(request: RegionSelectionSession['bounds']): Promise<RegionSelectionResult> {
  try {
    return fromBoundRegionSelectionResult(await RecordingFreedomService.CompleteRegionSelection(toBoundRegionSelectionRequest(request)))
  } catch (error) {
    console.info('Using browser region selection completion fallback:', error)
    const source: CaptureSource = {
      id: 'region:custom',
      type: 'region',
      label: 'Region',
      name: 'Custom Region',
      meta: `${request.width} x ${request.height} selected region`,
      x: request.x,
      y: request.y,
      width: request.width,
      height: request.height,
      nativeId: 'region:browser-preview',
      available: false,
      capability: 'native-backend-queued',
      unavailableReason: 'Desktop region overlay is only available in the Wails runtime.' }
    const result = {source, geometry: request, cancelled: false}
    ;(window as Window & {__RF_LAST_REGION_SELECTION__?: RegionSelectionResult}).__RF_LAST_REGION_SELECTION__ = result
    return result
  }
}

export async function assistRegionSelection(request: RegionAssistRequest): Promise<RegionAssistResult> {
  try {
    return fromBoundRegionAssistResult(await RecordingFreedomService.AssistRegionSelection(toBoundRegionAssistRequest(request)))
  } catch (error) {
    if (isWailsDesktopRuntime()) throw error
    console.info('Using browser region assist fallback:', error)
    return browserRegionAssist(request)
  }
}

export async function cancelRegionSelector(): Promise<RegionSelectionResult> {
  try {
    return fromBoundRegionSelectionResult(await RecordingFreedomService.CancelRegionSelection())
  } catch (error) {
    console.info('Using browser region selection cancel fallback:', error)
    ;(window as Window & {__RF_LAST_REGION_CANCEL__?: {cancelled: boolean; at: string}}).__RF_LAST_REGION_CANCEL__ = {
      cancelled: true,
      at: new Date().toISOString() }
    return {cancelled: true}
  }
}

export async function updateSelectedRegion(request: RegionSelectionSession['bounds']): Promise<RegionSelectionResult> {
  try {
    return fromBoundRegionSelectionResult(await RecordingFreedomService.UpdateSelectedRegion(toBoundRegionSelectionRequest(request)))
  } catch (error) {
    console.info('Using browser selected region update fallback:', error)
    return {geometry: request, cancelled: false}
  }
}

export async function cancelSelectedRegion(): Promise<RegionSelectionResult> {
  try {
    return fromBoundRegionSelectionResult(await RecordingFreedomService.CancelSelectedRegion())
  } catch (error) {
    console.info('Using browser selected region cancel fallback:', error)
    return {cancelled: true}
  }
}

export async function hideRegionFrame(): Promise<void> {
  try {
    await RecordingFreedomService.HideRegionFrame()
  } catch (error) {
    console.info('Using browser region frame hide fallback:', error)
  }
}

export async function readAnnotationPreviewImage(packagePath: string, snapshotPath: string): Promise<AnnotationPreviewImage> {
  try {
    const result: BoundAnnotationPreviewImageResult = await RecordingFreedomService.ReadAnnotationPreviewImage({
      packageDir: packagePath,
      snapshotPath } as BoundAnnotationPreviewImageRequest)
    return {
      available: result.available === true,
      dataUrl: result.dataUrl,
      relativePath: result.relativePath,
      bytes: result.bytes }
  } catch (error) {
    console.info('Desktop annotation preview image unavailable:', error)
    return {available: false}
  }
}

function browserRegionSelectionSession(purpose: NonNullable<RegionSelectionSession['purpose']>, id: string): RegionSelectionSession {
  const previous = (window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}).__RF_REGION_SESSION__
  const bounds = {x: 0, y: 0, width: window.innerWidth, height: window.innerHeight}
  return {
    id,
    bounds,
    captureBounds: bounds,
    minimumWidth: purpose === 'screenshot' ? 12 : 64,
    minimumHeight: purpose === 'screenshot' ? 12 : 64,
    displayCount: 1,
    purpose,
    candidates: previous?.candidates ? [...previous.candidates] : undefined }
}

function finalizeRegionSelectionSession(
  session: RegionSelectionSession,
  purpose: NonNullable<RegionSelectionSession['purpose']>,
  browserFallbackId: string,
): RegionSelectionSession {
  if (isWailsDesktopRuntime()) return session
  const previous = (window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}).__RF_REGION_SESSION__
  const hasUsableBounds = session.bounds.width > 0 && session.bounds.height > 0
  const next = hasUsableBounds ? {...session, purpose: session.purpose ?? purpose} : browserRegionSelectionSession(purpose, browserFallbackId)
  if ((!next.candidates || next.candidates.length === 0) && previous?.candidates?.length) {
    next.candidates = [...previous.candidates]
  }
  return emitBrowserRegionSession(next)
}

function emitBrowserRegionSession(session: RegionSelectionSession): RegionSelectionSession {
  ;(window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}).__RF_REGION_SESSION__ = session
  window.dispatchEvent(new CustomEvent('rf-region-session', {detail: session}))
  return session
}

function emitBrowserRegionFrame(bounds: RegionSelectionSession['bounds'], purpose: NonNullable<RegionSelectionSession['purpose']>) {
  const frame = {
    bounds,
    overlayBounds: {x: 0, y: 0, width: window.innerWidth, height: window.innerHeight},
    mode: 'edit',
    purpose }
  ;(window as Window & {__RF_REGION_FRAME__?: typeof frame}).__RF_REGION_FRAME__ = frame
  window.dispatchEvent(new CustomEvent('rf-region-frame', {detail: frame}))
}

function fromBoundRegionSelectionSession(session: BoundRegionSelectionSession): RegionSelectionSession {
  const initialPointer = (session as BoundRegionSelectionSession & {initialPointer?: {x: number; y: number} | null}).initialPointer
  return {
    id: session.id,
    bounds: {
      x: session.bounds.x,
      y: session.bounds.y,
      width: session.bounds.width,
      height: session.bounds.height },
    captureBounds: session.captureBounds ? {
      x: session.captureBounds.x,
      y: session.captureBounds.y,
      width: session.captureBounds.width,
      height: session.captureBounds.height } : undefined,
    displayBounds: (session.displayBounds ?? []).map((display) => ({
      id: display.id,
      bounds: fromBoundRegionRect(display.bounds),
      captureBounds: fromBoundRegionRect(display.captureBounds),
      scaleFactor: display.scaleFactor })),
    minimumWidth: session.minimumWidth,
    minimumHeight: session.minimumHeight,
    displayCount: session.displayCount,
    purpose: session.purpose === 'annotation' || session.purpose === 'screenshot' || session.purpose === 'scrolling-screenshot' ? session.purpose : 'capture',
    candidates: (session.candidates ?? []).map(fromBoundRegionSmartCandidate),
    initialPointer: initialPointer ? {x: initialPointer.x, y: initialPointer.y} : undefined }
}

function fromBoundRegionSmartCandidate(candidate: BoundRegionSmartCandidate): RegionSmartCandidate {
  return {
    id: candidate.id,
    kind: candidate.kind,
    label: candidate.label,
    bounds: fromBoundRegionRect(candidate.bounds),
    sourceId: candidate.sourceId,
    score: candidate.score }
}

function fromBoundRegionAssistResult(result: BoundRegionAssistResult): RegionAssistResult {
  return {
    candidates: (result.candidates ?? []).map(fromBoundRegionSmartCandidate),
    best: result.best ? fromBoundRegionSmartCandidate(result.best) : undefined,
    source: normalizeRegionAssistSource(result.source) }
}

function fromBoundRegionSelectionResult(result: BoundRegionSelectionResult): RegionSelectionResult {
  return {
    sessionId: result.sessionId,
    source: result.source?.id ? fromBoundSource(result.source as BoundCaptureSource) : undefined,
    geometry: result.geometry ? {
      x: result.geometry.x,
      y: result.geometry.y,
      width: result.geometry.width,
      height: result.geometry.height } : undefined,
    cancelled: result.cancelled,
    error: result.error }
}

function fromBoundWhiteboardScene(result: BoundWhiteboardSceneResult): WhiteboardScene {
  return {
    available: result.available,
    scenePath: result.scenePath,
    sceneJson: result.sceneJson,
    bytes: result.bytes,
    updatedAt: result.updatedAt,
    contentType: result.contentType }
}

function fromBoundWhiteboardExport(result: BoundWhiteboardExportResult): WhiteboardExport {
  return {
    format: result.format === 'svg' || result.format === 'excalidraw' ? result.format : 'png',
    outputPath: result.outputPath,
    bytes: result.bytes }
}

function fromBoundScreenshotCaptureResult(result: BoundScreenshotCaptureResult): {item: ScreenshotItem} {
  return {
    item: fromBoundScreenshotItem(result.item) }
}

function fromBoundWhiteboardSnapshot(result: BoundWhiteboardSnapshotResult): {scene: WhiteboardScene; item: ScreenshotItem} {
  return {
    scene: fromBoundWhiteboardScene(result.scene),
    item: fromBoundScreenshotItem(result.item) }
}

function fromBoundScreenshotHistory(result: BoundScreenshotHistoryResult): ScreenshotItem[] {
  return (result.items ?? []).map(fromBoundScreenshotItem)
}

function fromBoundScreenshotItem(item: BoundScreenshotItem): ScreenshotItem {
  return {
    id: item.id,
    path: item.path,
    thumbnailPath: item.thumbnailPath,
    createdAt: item.createdAt,
    width: item.width,
    height: item.height,
    mode: item.mode || 'region',
    region: item.region ? fromBoundRegionRect(item.region) : undefined,
    pinned: false,
    fixed: item.fixed === true,
    ocrStatus: normalizeOcrStatus(item.ocrStatus),
    ocrResultId: item.ocrResultId,
    ocrModelId: item.ocrModelId,
    ocrLanguage: item.ocrLanguage,
    ocrUpdatedAt: item.ocrUpdatedAt,
    ocrError: item.ocrError }
}

function fromBoundScreenshotPinState(state: Partial<BoundScreenshotPinState> | undefined): ScreenshotPinState {
  return normalizeScreenshotPinState({
    visible: state?.visible === true,
    item: state?.item ? fromBoundScreenshotItem(state.item) : undefined,
    dataUrl: state?.dataUrl,
    fixed: state?.fixed === true,
    pins: Array.isArray(state?.pins) ? state.pins.map(fromBoundScreenshotPinnedItem).filter((pin): pin is ScreenshotPinnedItem => Boolean(pin)) : undefined })
}

function fromBoundScreenshotPinnedItem(pin: unknown): ScreenshotPinnedItem | null {
  const record = pin && typeof pin === 'object' ? pin as {item?: BoundScreenshotItem; dataUrl?: string; fixed?: boolean} : {}
  const item = record.item ? fromBoundScreenshotItem(record.item) : null
  if (!item) return null
  return {
    item,
    dataUrl: typeof record.dataUrl === 'string' ? record.dataUrl : browserScreenshotDataUrl(item),
    fixed: record.fixed === true || item.fixed === true }
}

function fromBoundScreenshotWhiteboardContext(context: Partial<BoundScreenshotWhiteboardContext> | undefined): ScreenshotWhiteboardContext {
  return {
    available: context?.available === true,
    item: context?.item ? fromBoundScreenshotItem(context.item) : undefined,
    dataUrl: context?.dataUrl }
}

function fromBoundAnnotationOverlayState(state: BoundAnnotationOverlayState): AnnotationOverlayState {
  return {
    mode: state.mode === 'screenshot' ? 'screenshot' : 'annotation',
    packageDir: state.packageDir,
    manifestPath: state.manifestPath,
    windowBounds: fromBoundRegionRect(state.windowBounds),
    canvasBounds: fromBoundRegionRect(state.canvasBounds),
    toolbarBounds: state.toolbarBounds ? fromBoundRegionRect(state.toolbarBounds) : undefined,
    toolbarPlacement: state.toolbarPlacement === 'bottom' ? 'bottom' : 'top',
    target: {
      type: state.target.type,
      id: state.target.id,
      geometry: state.target.geometry ? {
        x: state.target.geometry.x,
        y: state.target.geometry.y,
        width: state.target.geometry.width,
        height: state.target.geometry.height,
        displayIndex: state.target.geometry.displayIndex,
        nativeId: state.target.geometry.nativeId } : undefined },
    captureExcluded: state.captureExcluded === true,
    sourceImageDataURL: state.sourceImageDataUrl,
    sourceImageCapturedAt: state.sourceImageCapturedAt }
}

function fromBoundAnnotationCapture(result: BoundAnnotationCaptureResult): AnnotationCapture {
  return {
    packageDir: result.packageDir,
    scenePath: result.scenePath,
    eventsPath: result.eventsPath,
    snapshotPath: result.snapshotPath,
    timelineSnapshotPath: result.timelineSnapshotPath,
    bytes: result.bytes }
}

function fromBoundAnnotationRenderJobClaim(result: BoundAnnotationRenderJobClaim): AnnotationRenderJobClaim {
  const job = result.job
  return {
    available: result.available === true && Boolean(job),
    job: job
      ? {
        id: job.id,
        packageDir: job.packageDir,
        scenePath: job.scenePath,
        relativeScenePath: job.relativeScenePath,
        outputPath: job.outputPath,
        relativeOutputPath: job.relativeOutputPath,
        sceneJson: job.sceneJson,
        canvasWidth: job.canvasWidth,
        canvasHeight: job.canvasHeight,
        index: job.index,
        startOffsetMs: job.startOffsetMs,
        endOffsetMs: job.endOffsetMs }
      : undefined }
}

function browserAnnotationOverlayState(
  mode: AnnotationOverlayState['mode'] = 'annotation',
  bounds?: RegionSelectionSession['bounds'],
): AnnotationOverlayState {
  const width = Math.max(320, bounds?.width ?? window.innerWidth ?? 1280)
  const height = Math.max(240, bounds?.height ?? window.innerHeight ?? 720)
  return {
    mode,
    packageDir: mode === 'screenshot' ? undefined : 'browser-preview/data/video/recording-preview.rfrec',
    manifestPath: mode === 'screenshot' ? undefined : 'browser-preview/data/video/recording-preview.rfrec/manifest.json',
    windowBounds: {x: 0, y: 0, width, height},
    canvasBounds: {x: 0, y: 0, width, height},
    target: {
      type: mode === 'screenshot' ? 'screenshot-region' : 'screen',
      id: mode === 'screenshot' ? 'browser-screenshot-region' : 'browser-preview',
      geometry: {x: bounds?.x ?? 0, y: bounds?.y ?? 0, width: bounds?.width ?? width, height: bounds?.height ?? height} },
    captureExcluded: false }
}

function saveBrowserScreenshotHistory(items: ScreenshotItem[]) {
  const unique = new Map<string, ScreenshotItem>()
  for (const item of items) {
    if (!item.id || unique.has(item.id)) continue
    unique.set(item.id, item)
  }
  window.localStorage?.setItem(browserScreenshotHistoryKey, JSON.stringify(Array.from(unique.values()).slice(0, 200)))
}

function createBrowserScreenshotItem(mode: string, region?: RegionSelectionSession['bounds']): ScreenshotItem {
  const createdAt = new Date().toISOString()
  const width = Math.max(64, Math.round(region?.width ?? window.innerWidth ?? 1280))
  const height = Math.max(64, Math.round(region?.height ?? window.innerHeight ?? 720))
  const id = `browser-screenshot-${Date.now()}`
  return {
    id,
    path: `browser-preview/data/screenshots/${id}.png`,
    thumbnailPath: `browser-preview/data/screenshots/thumbnails/${id}.png`,
    createdAt,
    width,
    height,
    mode,
    region: region ? {
      x: Math.round(region.x),
      y: Math.round(region.y),
      width,
      height } : undefined,
    pinned: false,
    fixed: false,
    ocrStatus: 'none' }
}

function browserRegionAssist(request: RegionAssistRequest): RegionAssistResult {
  const candidates = request.candidates ?? []
  let best: RegionSmartCandidate | undefined
  let bestScore = -1
  let source: RegionAssistResult['source'] = 'static'
  if (request.selection) {
    source = 'selection'
    for (const candidate of candidates) {
      const score = browserCandidateSelectionScore(candidate.bounds, request.selection) + browserRegionKindWeight(candidate.kind)
      if (score > bestScore) {
        best = {...candidate, score}
        bestScore = score
      }
    }
    if (bestScore < 0.5) best = undefined
  } else {
    const point = {x: request.pointerX ?? -1, y: request.pointerY ?? -1}
    const containing = candidates
      .filter((candidate) => browserRectContainsPoint(candidate.bounds, point))
      .sort((left, right) => {
        const kindRank = browserRegionKindWeight(right.kind) - browserRegionKindWeight(left.kind)
        if (kindRank !== 0) return kindRank
        return (left.bounds.width * left.bounds.height) - (right.bounds.width * right.bounds.height)
      })
    const level = Math.max(0, Math.min(containing.length - 1, Math.round(request.candidateLevel ?? 0)))
    const leveled = containing[level]
    if (leveled) {
      const area = Math.max(1, leveled.bounds.width * leveled.bounds.height)
      best = {...leveled, score: (leveled.score ?? 0) + browserRegionKindWeight(leveled.kind) + 1000000 / area}
      source = browserRegionAssistSourceForKind(leveled.kind)
    } else {
      for (const candidate of candidates) {
        if (!browserRectContainsPoint(candidate.bounds, point)) continue
        const area = Math.max(1, candidate.bounds.width * candidate.bounds.height)
        const score = (candidate.score ?? 0) + browserRegionKindWeight(candidate.kind) + 1000000 / area
        if (score > bestScore) {
          best = {...candidate, score}
          bestScore = score
        }
      }
      source = browserRegionAssistSourceForKind(best?.kind)
    }
  }
  return {candidates, best, source}
}

function browserRegionAssistSourceForKind(kind: RegionSmartCandidate['kind'] | undefined): RegionAssistResult['source'] {
  if (kind === 'element') return 'element'
  if (kind === 'edge') return 'image-hover'
  return 'static'
}

function normalizeRegionAssistSource(source: unknown): RegionAssistResult['source'] {
  if (source === 'element' || source === 'image-hover' || source === 'selection' || source === 'static') {
    return source
  }
  return undefined
}

function browserRegionKindWeight(kind: RegionSmartCandidate['kind']) {
  if (kind === 'element') return 0.36
  if (kind === 'edge') return 0.16
  if (kind === 'window') return 0.08
  return 0
}

function browserCandidateSelectionScore(candidate: RegionSelectionSession['bounds'], selection: RegionSelectionSession['bounds']) {
  const left = Math.max(candidate.x, selection.x)
  const top = Math.max(candidate.y, selection.y)
  const right = Math.min(candidate.x + candidate.width, selection.x + selection.width)
  const bottom = Math.min(candidate.y + candidate.height, selection.y + selection.height)
  if (right <= left || bottom <= top) return -1
  const intersection = (right - left) * (bottom - top)
  const candidateArea = Math.max(1, candidate.width * candidate.height)
  const selectionArea = Math.max(1, selection.width * selection.height)
  const overlap = intersection / Math.min(candidateArea, selectionArea)
  const edgeDistance = Math.abs(candidate.x - selection.x) +
    Math.abs(candidate.y - selection.y) +
    Math.abs(candidate.x + candidate.width - selection.x - selection.width) +
    Math.abs(candidate.y + candidate.height - selection.y - selection.height)
  const closeEdges = Math.max(0, 1 - edgeDistance / 280)
  const areaRatio = Math.min(candidateArea, selectionArea) / Math.max(candidateArea, selectionArea)
  return overlap * 0.54 + closeEdges * 0.34 + areaRatio * 0.12
}

function browserRectContainsPoint(rect: RegionSelectionSession['bounds'], point: {x: number; y: number}) {
  return rect.width > 0 &&
    rect.height > 0 &&
    point.x >= rect.x &&
    point.x < rect.x + rect.width &&
    point.y >= rect.y &&
    point.y < rect.y + rect.height
}

function fromBrowserScreenshotPinState(value: unknown): ScreenshotPinState {
  const record = value && typeof value === 'object' ? value as Partial<ScreenshotPinState> : {}
  const item = fromBrowserScreenshotItem(record.item)
  const pins = Array.isArray(record.pins)
    ? record.pins.map(fromBrowserScreenshotPinnedItem).filter((pin): pin is ScreenshotPinnedItem => Boolean(pin))
    : undefined
  return normalizeScreenshotPinState({
    visible: record.visible === true,
    item: item ?? undefined,
    dataUrl: typeof record.dataUrl === 'string' ? record.dataUrl : browserScreenshotDataUrl(item),
    fixed: record.fixed === true,
    pins })
}

function fromBrowserScreenshotPinnedItem(value: unknown): ScreenshotPinnedItem | null {
  const record = value && typeof value === 'object' ? value as Partial<ScreenshotPinnedItem> : {}
  const item = fromBrowserScreenshotItem(record.item)
  if (!item) return null
  return {
    item,
    dataUrl: typeof record.dataUrl === 'string' ? record.dataUrl : browserScreenshotDataUrl(item),
    fixed: record.fixed === true || item.fixed === true }
}

function normalizeScreenshotPinState(state: ScreenshotPinState): ScreenshotPinState {
  const pins = normalizeScreenshotPins(state.pins ?? (state.item ? [{
    item: state.item,
    dataUrl: state.dataUrl,
    fixed: state.fixed }] : []))
  if (pins.length === 0) {
    return {visible: false, fixed: false, pins: []}
  }
  const active = pins[pins.length - 1]
  return {
    visible: state.visible === true,
    item: active.item,
    dataUrl: active.dataUrl,
    fixed: active.fixed,
    pins }
}

function normalizeScreenshotPins(pins: ScreenshotPinnedItem[]) {
  const next: ScreenshotPinnedItem[] = []
  const seen = new Set<string>()
  for (const pin of pins) {
    const item = pin.item
    if (!item?.id || seen.has(item.id)) continue
    seen.add(item.id)
    next.push({
      item,
      dataUrl: pin.dataUrl || browserScreenshotDataUrl(item),
      fixed: pin.fixed === true || item.fixed === true })
  }
  return next
}

function appendBrowserScreenshotPinStateItem(state: ScreenshotPinState, pin: ScreenshotPinnedItem): ScreenshotPinState {
  const pins = normalizeScreenshotPins([...(state.pins ?? []), pin])
    .filter((entry) => entry.item.id !== pin.item.id)
  pins.push(pin)
  return normalizeScreenshotPinState({visible: true, fixed: pin.fixed, pins})
}

function updateBrowserScreenshotPinStateItem(state: ScreenshotPinState, item: ScreenshotItem): ScreenshotPinState {
  const pins = normalizeScreenshotPins(state.pins ?? []).map((pin) => pin.item.id === item.id
    ? {...pin, item, fixed: item.fixed === true}
    : pin)
  return normalizeScreenshotPinState({...state, visible: pins.length > 0, pins})
}

function removeBrowserScreenshotPinStateItem(state: ScreenshotPinState, id: string): ScreenshotPinState {
  const pins = normalizeScreenshotPins(state.pins ?? []).filter((pin) => pin.item.id !== id)
  return normalizeScreenshotPinState({...state, visible: pins.length > 0, pins})
}

function screenshotPinStateContains(state: ScreenshotPinState, id: string) {
  return state.item?.id === id || (state.pins ?? []).some((pin) => pin.item.id === id)
}

function fromBrowserScreenshotWhiteboardContext(value: unknown): ScreenshotWhiteboardContext {
  const record = value && typeof value === 'object' ? value as Partial<ScreenshotWhiteboardContext> : {}
  const item = fromBrowserScreenshotItem(record.item)
  return {
    available: record.available === true && Boolean(item),
    item: item ?? undefined,
    dataUrl: typeof record.dataUrl === 'string' ? record.dataUrl : browserScreenshotDataUrl(item) }
}

function toBoundRegionSelectionRequest(request: RegionSelectionSession['bounds']): BoundRegionSelectionRequest {
  return {
    x: Math.round(request.x),
    y: Math.round(request.y),
    width: Math.round(request.width),
    height: Math.round(request.height) }
}

function toBoundRegionAssistRequest(request: RegionAssistRequest): BoundRegionAssistRequest {
  return {
    sessionId: request.sessionId,
    purpose: request.purpose,
    pointerX: Math.round(request.pointerX ?? 0),
    pointerY: Math.round(request.pointerY ?? 0),
    selection: request.selection ? toBoundRegionSelectionRequest(request.selection) : undefined,
    candidateLevel: Math.max(0, Math.round(request.candidateLevel ?? 0)) }
}

function fromWhiteboardVisibilityEvent(value: unknown): WhiteboardVisibilityUpdate {
  const record = value && typeof value === 'object' ? value as Record<string, unknown> : {}
  return {
    visible: record.visible === true,
    mode: record.mode === 'annotation' ? 'annotation' : 'whiteboard' }
}

function emitBrowserWhiteboardVisibility(event: WhiteboardVisibilityUpdate) {
  const normalized = fromWhiteboardVisibilityEvent(event)
  ;(window as Window & {__RF_LAST_WHITEBOARD_VISIBILITY__?: WhiteboardVisibilityUpdate}).__RF_LAST_WHITEBOARD_VISIBILITY__ = normalized
  window.dispatchEvent(new CustomEvent(browserWhiteboardVisibilityEvent, {detail: normalized}))
}
