import {
  ArrowUpRight,
  Check,
  ClipboardCopy,
  Circle,
  Eraser,
  Eye,
  EyeOff,
  Minus,
  MousePointer2,
  PenLine,
  RectangleHorizontal,
  RefreshCcw,
  Save,
  ScanText,
  Settings2,
  Type,
  Undo2,
  X,
} from 'lucide-react'
import {useCallback, useEffect, useLayoutEffect, useRef, useState, type PointerEvent as ReactPointerEvent} from 'react'
import {Excalidraw, exportToBlob, exportToClipboard, serializeAsJSON} from '@excalidraw/excalidraw'
import type {ExcalidrawImperativeAPI} from '@excalidraw/excalidraw/types'
import '@excalidraw/excalidraw/index.css'
import {copyByLocale} from './i18n'
import {defaultSettings, normalizeLocale, normalizeTheme, type AppSettings, type LocaleCode, type ScreenshotItem, type ThemeCode, type WhiteboardTool} from './services/mockBackend'
import {hideAnnotationOverlay, hideScreenshotAnnotationOverlay, loadAnnotationCapture, loadScreenshotAnnotationCapture, loadSettings, openOcrResult, patchWhiteboardSettings, queueRecognizeScreenshot, queueRecognizeWhiteboard, reselectAnnotationRegion, reselectScreenshotAnnotationRegion, saveAnnotationCapture, saveScreenshotAnnotationCapture, setAnnotationOverlayHitRegions, subscribeOcrJobEvents, subscribeSettingsChanged, type AnnotationCapture, type AnnotationOverlayState, type CapsuleWindowHitRegion, type OcrBlock, type OcrJobUpdate, type OcrResult, type ScreenshotWhiteboardContext} from './services/recorderBackend'
import {OcrPositionTextLayer, countOcrPositionTextBlocks} from './components/ocr/OcrPositionTextLayer'
import {writeClipboardImage, writeClipboardText} from './utils/clipboard'
import {AnnotationStyleControls, type AnnotationStyle} from './components/AnnotationStyleControls'

const annotationTools: Array<{tool: WhiteboardTool; icon: typeof PenLine; label: 'select' | 'pen' | 'arrow' | 'line' | 'rectangle' | 'ellipse' | 'text' | 'eraser'}> = [
  {tool: 'selection', icon: MousePointer2, label: 'select'},
  {tool: 'freedraw', icon: PenLine, label: 'pen'},
  {tool: 'arrow', icon: ArrowUpRight, label: 'arrow'},
  {tool: 'line', icon: Minus, label: 'line'},
  {tool: 'rectangle', icon: RectangleHorizontal, label: 'rectangle'},
  {tool: 'ellipse', icon: Circle, label: 'ellipse'},
  {tool: 'text', icon: Type, label: 'text'},
  {tool: 'eraser', icon: Eraser, label: 'eraser'},
]

const annotationColors = ['#ef4444', '#f59e0b', '#22c55e', '#38bdf8', '#a78bfa', '#f8fafc', '#111827']
const annotationStrokeWidths = ['thin', 'medium', 'bold'] as const
const annotationStyleTools: WhiteboardTool[] = ['freedraw', 'arrow', 'line', 'rectangle', 'ellipse', 'text']

const maxPendingAnnotationElementEvents = 512
type AnnotationElementEvent = {
  type: 'element-created' | 'element-updated' | 'element-deleted'
  clientSequence: number
  clientEventId: string
  elementId: string
  elementType: string
  elementVersion?: number
  isDeleted?: boolean
  bounds?: {x: number; y: number; width: number; height: number}
  element?: unknown
}

type AnnotationSaveOutcome =
  | {kind: 'annotation'; capture: AnnotationCapture}
  | {kind: 'screenshot'; item: ScreenshotItem}

function AnnotationOverlayWindow() {
  const [overlayState, setOverlayState] = useState<AnnotationOverlayState | null>(() => (window as any).__RF_ANNOTATION_OVERLAY__ ?? null)
  const [locale, setLocale] = useState<LocaleCode>('zh-CN')
  const [theme, setTheme] = useState<ThemeCode>('night-teal')
  const [initialData, setInitialData] = useState<any | null>(null)
  const [activeTool, setActiveToolState] = useState<WhiteboardTool>('freedraw')
  const [strokeColor, setStrokeColor] = useState('#ef4444')
  const [strokeWidth, setStrokeWidth] = useState(defaultSettings.whiteboard.lastStrokeWidth)
  const [opacity, setOpacity] = useState(100)
  const [annotationStylePanelOpen, setAnnotationStylePanelOpen] = useState(false)
  const [annotationStyle, setAnnotationStyle] = useState<AnnotationStyle>({
    strokeStyle: 'solid',
    fillStyle: 'solid',
    fillColor: 'transparent',
    roundness: 'round',
    roughness: 1,
    arrowType: 'round',
    arrowhead: 'arrow',
    fontFamily: 1,
    fontSize: 20,
    textAlign: 'left',
  })
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [ocrBusy, setOcrBusy] = useState(false)
  const [ocrResultId, setOcrResultId] = useState('')
  const [ocrResult, setOcrResult] = useState<OcrResult | null>(null)
  const [ocrMessage, setOcrMessage] = useState('')
  const [ocrSucceeded, setOcrSucceeded] = useState(false)
  const [imageCopied, setImageCopied] = useState(false)
  const [ocrPositionTextVisible, setOcrPositionTextVisible] = useState(false)
  const [ocrHoveredBlockId, setOcrHoveredBlockId] = useState('')
  const [ocrCopiedBlockId, setOcrCopiedBlockId] = useState('')
  const [toolbarOffset, setToolbarOffset] = useState({x: 0, y: 0})
  const apiRef = useRef<ExcalidrawImperativeAPI | null>(null)
  const lastSavedSceneRef = useRef('')
  const lastSavedContentSignatureRef = useRef('')
  const saveTimerRef = useRef<number | null>(null)
  const elementSignatureRef = useRef<Map<string, string>>(new Map())
  const pendingElementEventsRef = useRef<Map<string, AnnotationElementEvent>>(new Map())
  const clientSequenceRef = useRef(0)
  const ocrSourceRef = useRef<{sourceKind: string; sourceId: string} | null>(null)
  const toolbarDragRef = useRef<{startX: number; startY: number; offsetX: number; offsetY: number} | null>(null)
  const capsuleRef = useRef<HTMLDivElement | null>(null)
  const canvasRef = useRef<HTMLElement | null>(null)
  const copy = copyByLocale[locale]
  const canvasReceivesInput = annotationCanvasReceivesInput(activeTool) || ocrPositionTextVisible
  const showAnnotationStyleControls = annotationStyleTools.includes(activeTool)
  const isScreenshotMode = overlayState?.mode === 'screenshot'
  const overlayKey = annotationOverlayKey(overlayState)
  const canvasBounds = annotationCanvasBounds(overlayState)
  const toolbarBounds = annotationToolbarBounds(overlayState, canvasBounds)

  const startToolbarDrag = (event: ReactPointerEvent<HTMLElement>) => {
    const target = event.target as HTMLElement
    if (target.closest('button, input, select, textarea, .annotation-style-capsule')) return
    event.preventDefault()
    toolbarDragRef.current = {
      startX: event.clientX,
      startY: event.clientY,
      offsetX: toolbarOffset.x,
      offsetY: toolbarOffset.y,
    }
  }

  useEffect(() => {
    const onPointerMove = (event: PointerEvent) => {
      const drag = toolbarDragRef.current
      if (!drag) return
      const nextX = drag.offsetX + event.clientX - drag.startX
      const nextY = drag.offsetY + event.clientY - drag.startY
      const minX = 8 - toolbarBounds.x
      const minY = 8 - toolbarBounds.y
      const maxX = Math.max(minX, window.innerWidth - toolbarBounds.x - toolbarBounds.width - 8)
      const maxY = Math.max(minY, window.innerHeight - toolbarBounds.y - toolbarBounds.height - 8)
      setToolbarOffset({
        x: Math.round(Math.min(maxX, Math.max(minX, nextX))),
        y: Math.round(Math.min(maxY, Math.max(minY, nextY))),
      })
    }
    const onPointerUp = () => {
      toolbarDragRef.current = null
    }
    window.addEventListener('pointermove', onPointerMove)
    window.addEventListener('pointerup', onPointerUp)
    window.addEventListener('pointercancel', onPointerUp)
    return () => {
      window.removeEventListener('pointermove', onPointerMove)
      window.removeEventListener('pointerup', onPointerUp)
      window.removeEventListener('pointercancel', onPointerUp)
    }
  }, [toolbarBounds.height, toolbarBounds.width, toolbarBounds.x, toolbarBounds.y])

  function resetAnnotationElementTracking(scene: any) {
    elementSignatureRef.current = elementSignatureMap(scene?.elements ?? [])
    pendingElementEventsRef.current.clear()
    clientSequenceRef.current = 0
  }

  function trackAnnotationElementEvents(elements: readonly unknown[]) {
    const previous = elementSignatureRef.current
    const next = new Map<string, string>()
    for (const element of elements) {
      const identity = annotationElementIdentity(element)
      if (!identity) continue
      const signature = annotationElementSignature(element)
      next.set(identity.id, signature)
      if (previous.get(identity.id) === signature) continue
      queueAnnotationElementEvent(previous.has(identity.id) ? 'element-updated' : 'element-created', element)
    }
    previous.forEach((_, id) => {
      if (!next.has(id)) {
        queueAnnotationRemovedElementEvent(id)
      }
    })
    elementSignatureRef.current = next
  }

  function queueAnnotationElementEvent(type: AnnotationElementEvent['type'], element: unknown) {
    const identity = annotationElementIdentity(element)
    if (!identity) return
    clientSequenceRef.current += 1
    const event: AnnotationElementEvent = {
      type,
      clientSequence: clientSequenceRef.current,
      clientEventId: `annotation-client-${Date.now()}-${clientSequenceRef.current}`,
      elementId: identity.id,
      elementType: identity.type,
      elementVersion: identity.version,
      isDeleted: identity.isDeleted,
      bounds: identity.bounds,
      element,
    }
    setPendingAnnotationElementEvent(identity.id, event)
  }

  function queueAnnotationRemovedElementEvent(elementId: string) {
    clientSequenceRef.current += 1
    setPendingAnnotationElementEvent(elementId, {
      type: 'element-deleted',
      clientSequence: clientSequenceRef.current,
      clientEventId: `annotation-client-${Date.now()}-${clientSequenceRef.current}`,
      elementId,
      elementType: 'unknown',
      isDeleted: true,
    })
  }

  function setPendingAnnotationElementEvent(elementId: string, event: AnnotationElementEvent) {
    if (!pendingElementEventsRef.current.has(elementId) && pendingElementEventsRef.current.size >= maxPendingAnnotationElementEvents) {
      const oldestKey = pendingElementEventsRef.current.keys().next().value
      if (oldestKey) pendingElementEventsRef.current.delete(oldestKey)
    }
    pendingElementEventsRef.current.set(elementId, event)
  }

  function pendingAnnotationEventsJSONL() {
    return Array.from(pendingElementEventsRef.current.values())
      .sort((left, right) => left.clientSequence - right.clientSequence)
      .map((event) => JSON.stringify(event))
      .join('\n')
  }

  useEffect(() => {
    document.body.classList.add('rf-annotation-overlay-window')
    return () => document.body.classList.remove('rf-annotation-overlay-window')
  }, [])

  useEffect(() => {
    const unsubscribe = subscribeSettingsChanged(applySettings)
    return () => unsubscribe()
  }, [])

  useEffect(() => {
    const unsubscribe = subscribeOcrJobEvents((event) => {
      const source = ocrSourceRef.current
      if (!source || !annotationOcrEventMatches(event, source)) return
      if (event.status === 'queued') {
        setOcrBusy(true)
        setOcrSucceeded(false)
        setOcrMessage(copy.whiteboard.ocrQueued)
      } else if (event.status === 'running') {
        setOcrBusy(true)
        setOcrMessage(copy.whiteboard.ocrStatusRunning)
      } else if (event.status === 'ready') {
        setOcrBusy(false)
        setOcrSucceeded(true)
        setOcrMessage(copy.whiteboard.ocrStatusReady)
        if (event.result) {
          setOcrResultId(event.result.id)
          setOcrResult(event.result)
          setOcrPositionTextVisible(false)
          setOcrHoveredBlockId('')
          setOcrCopiedBlockId('')
          removeAnnotationOcrPositionTextElements(apiRef.current, scheduleSave)
        }
      } else if (event.status === 'failed') {
        setOcrBusy(false)
        setOcrSucceeded(false)
        setOcrMessage(event.error || copy.whiteboard.ocrStatusFailed)
      } else if (event.status === 'cancelled') {
        setOcrBusy(false)
        setOcrMessage(copy.whiteboard.ready)
      }
    })
    return () => unsubscribe()
  }, [copy])

  useEffect(() => {
    ocrSourceRef.current = null
    setOcrBusy(false)
    setOcrResultId('')
    setOcrResult(null)
    setOcrSucceeded(false)
    setImageCopied(false)
    setOcrPositionTextVisible(false)
    setOcrHoveredBlockId('')
    setOcrCopiedBlockId('')
    setOcrMessage('')
  }, [overlayKey])

  useEffect(() => {
    let cancelled = false
    setInitialData(null)
    setDirty(false)
    setSaving(false)
    if (saveTimerRef.current !== null) {
      window.clearTimeout(saveTimerRef.current)
      saveTimerRef.current = null
    }
    const sceneTask = isScreenshotMode ? loadScreenshotAnnotationCapture() : loadAnnotationCapture()
    void Promise.all([loadSettings(), sceneTask])
      .then(([settings, scene]) => {
        if (cancelled) return
        applySettings(settings)
        const parsed = isScreenshotMode
          ? screenshotAnnotationScene(settings, scene as ScreenshotWhiteboardContext, overlayState)
          : scene.available && 'sceneJson' in scene && scene.sceneJson ? safeParseScene(scene.sceneJson) : null
        const baseScene = parsed ?? defaultAnnotationScene(settings)
        const nextInitialData = !isScreenshotMode && overlayState?.sourceImageDataURL
          ? annotationSceneWithFrozenSource(baseScene, overlayState.sourceImageDataURL, overlayState.canvasBounds)
          : baseScene
        resetAnnotationElementTracking(nextInitialData)
        setInitialData(nextInitialData)
        const loadedSceneJson = 'sceneJson' in scene ? scene.sceneJson ?? '' : ''
        lastSavedSceneRef.current = loadedSceneJson
        lastSavedContentSignatureRef.current = loadedSceneJson ? annotationContentSignatureFromScene(nextInitialData) : ''
      })
      .catch((error) => {
        console.info('Using annotation overlay load fallback:', error)
        if (!cancelled) {
          const nextInitialData = defaultAnnotationScene(defaultSettings)
          resetAnnotationElementTracking(nextInitialData)
          setInitialData(nextInitialData)
          lastSavedSceneRef.current = ''
          lastSavedContentSignatureRef.current = ''
        }
      })
    return () => {
      cancelled = true
    }
  }, [isScreenshotMode, overlayKey])

  useEffect(() => {
    document.documentElement.lang = locale
    document.documentElement.dataset.theme = theme
  }, [locale, theme])

  useEffect(() => {
    setAnnotationStylePanelOpen(false)
    setToolbarOffset({x: 0, y: 0})
  }, [activeTool])

  useEffect(() => {
    const onState = (event: Event) => {
      const next = (event as CustomEvent<AnnotationOverlayState>).detail
      if (next) setOverlayState(next)
    }
    window.addEventListener('rf-annotation-overlay', onState)
    return () => window.removeEventListener('rf-annotation-overlay', onState)
  }, [])

  useEffect(() => () => {
    if (saveTimerRef.current !== null) window.clearTimeout(saveTimerRef.current)
    void setAnnotationOverlayHitRegions({
      enabled: false,
      force: true,
      viewportWidth: window.innerWidth || 1,
      viewportHeight: window.innerHeight || 1,
      devicePixelRatio: window.devicePixelRatio || 1,
      regions: [],
    })
  }, [])

  useLayoutEffect(() => {
    let frame = 0
    const publish = (force = false) => {
      if (frame) window.cancelAnimationFrame(frame)
      frame = window.requestAnimationFrame(() => {
        frame = 0
        const viewportWidth = window.innerWidth || 1
        const viewportHeight = window.innerHeight || 1
        const regions = [
          elementHitRegion(capsuleRef.current, 'pill', 999),
          canvasReceivesInput ? elementHitRegion(canvasRef.current, 'rect', 0) : null,
        ].filter((region): region is CapsuleWindowHitRegion => region !== null)
        void setAnnotationOverlayHitRegions({
          enabled: regions.length > 0,
          force,
          viewportWidth,
          viewportHeight,
          devicePixelRatio: window.devicePixelRatio || 1,
          regions,
        })
      })
    }
    publish(true)
    const resizeObserver = new ResizeObserver(() => publish())
    if (capsuleRef.current) resizeObserver.observe(capsuleRef.current)
    if (canvasRef.current) resizeObserver.observe(canvasRef.current)
    const onResize = () => publish()
    window.addEventListener('resize', onResize, {passive: true})
    return () => {
      if (frame) window.cancelAnimationFrame(frame)
      resizeObserver.disconnect()
      window.removeEventListener('resize', onResize)
      void setAnnotationOverlayHitRegions({
        enabled: false,
        force: true,
        viewportWidth: window.innerWidth || 1,
        viewportHeight: window.innerHeight || 1,
        devicePixelRatio: window.devicePixelRatio || 1,
        regions: [],
      })
    }
  }, [canvasReceivesInput, initialData, overlayState?.canvasBounds.height, overlayState?.canvasBounds.width])

  const applySettings = (settings: AppSettings) => {
    setLocale(normalizeLocale(settings.locale))
    setTheme(normalizeTheme(settings.window.theme))
    setActiveToolState(annotationTools.some(({tool}) => tool === settings.whiteboard.lastTool) ? settings.whiteboard.lastTool : 'freedraw')
    setStrokeColor(settings.whiteboard.lastStrokeColor || '#ef4444')
    setStrokeWidth(settings.whiteboard.lastStrokeWidth)
    setOpacity(normalizeOpacity(settings.whiteboard.lastOpacity))
  }

  const applyStyle = useCallback((color: string, width: typeof strokeWidth, nextOpacity: number, style: AnnotationStyle) => {
    apiRef.current?.updateScene({
      appState: {
        currentItemStrokeColor: color,
        currentItemStrokeWidthKey: width,
        currentItemOpacity: nextOpacity,
        currentItemStrokeStyle: style.strokeStyle,
        currentItemFillStyle: style.fillStyle,
        currentItemBackgroundColor: style.fillColor,
        currentItemRoundness: style.roundness,
        currentItemRoughness: style.roughness,
        currentItemArrowType: style.arrowType,
        currentItemStartArrowhead: null,
        currentItemEndArrowhead: style.arrowhead === 'none' ? null : style.arrowhead,
        currentItemFontFamily: style.fontFamily,
        currentItemFontSize: style.fontSize,
        currentItemTextAlign: style.textAlign,
        viewBackgroundColor: 'transparent',
      },
    } as any)
  }, [])

  useEffect(() => {
    applyStyle(strokeColor, strokeWidth, opacity, annotationStyle)
  }, [annotationStyle, applyStyle, opacity, strokeColor, strokeWidth])

  const setActiveTool = (tool: WhiteboardTool, persist = true) => {
    setActiveToolState(tool)
    apiRef.current?.setActiveTool({type: tool} as any)
    if (persist) {
      void patchWhiteboardSettings({lastTool: tool, lastMode: 'annotation'})
    }
  }

  const scheduleSave = useCallback((sceneJson: string, hasElements: boolean, contentSignature: string) => {
    if (!hasElements && !lastSavedSceneRef.current) return
    if (contentSignature && contentSignature === lastSavedContentSignatureRef.current) {
      if (saveTimerRef.current !== null) {
        window.clearTimeout(saveTimerRef.current)
        saveTimerRef.current = null
      }
      setDirty(false)
      return
    }
    setDirty(true)
    if (isScreenshotMode) return
    if (saveTimerRef.current !== null) window.clearTimeout(saveTimerRef.current)
    saveTimerRef.current = window.setTimeout(() => {
      void saveCurrentAnnotation()
    }, 700)
  }, [isScreenshotMode])

  const saveCurrentAnnotation = async (options: {force?: boolean} = {}): Promise<AnnotationSaveOutcome | null> => {
    const api = apiRef.current
    if (!api) return null
    if (saveTimerRef.current !== null) {
      window.clearTimeout(saveTimerRef.current)
      saveTimerRef.current = null
    }
    try {
      const sceneElements = api.getSceneElements()
      const files = api.getFiles()
      const sceneJson = (serializeAsJSON as any)(sceneElements, api.getAppState(), files, 'local')
      const contentSignature = annotationContentSignature(sceneElements, files)
      if (!options.force && !dirty && contentSignature === lastSavedContentSignatureRef.current && pendingElementEventsRef.current.size === 0) return null
      setSaving(true)
      const canvasSize = annotationSnapshotCanvasSize(overlayState)
      const blob = await exportToBlob({
        elements: annotationSnapshotExportElements(sceneElements, canvasSize.width, canvasSize.height),
        appState: {
          ...api.getAppState(),
          exportBackground: false,
          exportScale: 1,
          viewBackgroundColor: 'transparent',
        },
        files,
        mimeType: 'image/png',
        exportPadding: 0,
        getDimensions: () => ({width: canvasSize.width, height: canvasSize.height, scale: 1}),
      } as any)
      const snapshotDataUrl = await blobToDataURL(blob)
      const eventsJsonl = pendingAnnotationEventsJSONL()
      let outcome: AnnotationSaveOutcome
      if (isScreenshotMode) {
        const item = await saveScreenshotAnnotationCapture({sceneJson, snapshotDataUrl, eventsJsonl})
        outcome = {kind: 'screenshot', item}
      } else {
        const capture = await saveAnnotationCapture({sceneJson, snapshotDataUrl, eventsJsonl})
        outcome = {kind: 'annotation', capture}
      }
      pendingElementEventsRef.current.clear()
      lastSavedSceneRef.current = sceneJson
      lastSavedContentSignatureRef.current = contentSignature
      setDirty(false)
      window.setTimeout(() => {
        const currentApi = apiRef.current
        if (!currentApi) return
        const currentSignature = annotationContentSignature(currentApi.getSceneElements(), currentApi.getFiles())
        if (currentSignature === lastSavedContentSignatureRef.current) setDirty(false)
      }, 0)
      return outcome
    } catch (error) {
      console.error('Failed to save annotation capture:', error)
      setDirty(true)
      return null
    } finally {
      setSaving(false)
    }
  }

  const resetLocalAnnotationScene = () => {
    if (saveTimerRef.current !== null) {
      window.clearTimeout(saveTimerRef.current)
      saveTimerRef.current = null
    }
    pendingElementEventsRef.current.clear()
    lastSavedSceneRef.current = ''
    lastSavedContentSignatureRef.current = ''
    setDirty(false)
    const scene = defaultAnnotationScene({
      ...defaultSettings,
      whiteboard: {
        ...defaultSettings.whiteboard,
        lastStrokeColor: strokeColor,
        lastStrokeWidth: strokeWidth,
        lastOpacity: opacity,
      },
    })
    resetAnnotationElementTracking(scene)
    apiRef.current?.resetScene()
    window.setTimeout(() => applyStyle(strokeColor, strokeWidth, opacity, annotationStyle), 0)
  }

  const updateAnnotationStyle = <K extends keyof AnnotationStyle>(key: K, value: AnnotationStyle[K]) => {
    setAnnotationStyle((current) => ({...current, [key]: value}))
  }

  const copyAnnotationImage = async () => {
    const api = apiRef.current
    setImageCopied(false)
    if (!api) {
      setOcrMessage(copy.whiteboard.copyImageFailed)
      return
    }
    try {
      const canvasSize = annotationSnapshotCanvasSize(overlayState)
      const exportElements = annotationSnapshotExportElements(api.getSceneElements(), canvasSize.width, canvasSize.height)
      const blob = await exportToBlob({
        elements: exportElements as any,
        appState: {
          ...api.getAppState(),
          exportBackground: false,
          exportScale: 1,
          viewBackgroundColor: 'transparent',
        },
        files: api.getFiles(),
        mimeType: 'image/png',
        exportPadding: 0,
        getDimensions: () => ({width: canvasSize.width, height: canvasSize.height, scale: 1}),
      } as any)
      try {
        await writeClipboardImage(blob)
      } catch {
        await exportToClipboard({
          elements: exportElements as any,
          appState: {
            ...api.getAppState(),
            exportBackground: false,
            viewBackgroundColor: 'transparent',
          },
          files: api.getFiles(),
          type: 'png',
        })
      }
      setImageCopied(true)
      setOcrMessage(copy.whiteboard.copiedImage)
    } catch (error) {
      console.error('Failed to copy annotation image:', error)
      setImageCopied(false)
      setOcrMessage(copy.whiteboard.copyImageFailed)
    }
  }

  const reselectRegion = async () => {
    try {
      resetLocalAnnotationScene()
      if (isScreenshotMode) {
        await reselectScreenshotAnnotationRegion()
      } else {
        await reselectAnnotationRegion()
      }
    } catch (error) {
      console.error('Failed to reselect annotation region:', error)
    }
  }

  const undoAnnotationStep = () => {
    const target = document.querySelector<HTMLElement>('.annotation-overlay-canvas .excalidraw') ?? document.querySelector<HTMLElement>('.annotation-overlay-canvas')
    if (!target) return
    target.focus({preventScroll: true})
    const isMac = /Mac|iPhone|iPad|iPod/i.test(window.navigator.platform)
    target.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'z',
      code: 'KeyZ',
      ctrlKey: !isMac,
      metaKey: isMac,
      bubbles: true,
      cancelable: true,
    }))
  }

  const copyAnnotationOcrPositionText = useCallback((block: OcrBlock, blockId: string) => {
    const text = block.text.trim()
    if (!text) {
      setOcrMessage(copy.screenshot.copyTextEmpty)
      return
    }
    void writeClipboardText(text)
      .then(() => {
        setOcrCopiedBlockId(blockId)
        setOcrMessage(copy.screenshot.copiedText)
        window.setTimeout(() => setOcrCopiedBlockId((current) => current === blockId ? '' : current), 1200)
      })
      .catch((error) => setOcrMessage(readableAnnotationError(error) || copy.screenshot.copyTextEmpty))
  }, [copy])

  const queueAnnotationOCR = async () => {
    if (ocrBusy) return
    setOcrBusy(true)
    setOcrResultId('')
    setOcrResult(null)
    setOcrSucceeded(false)
    setOcrPositionTextVisible(false)
    setOcrHoveredBlockId('')
    setOcrCopiedBlockId('')
    setOcrMessage(copy.whiteboard.ocrPreparing)
    removeAnnotationOcrPositionTextElements(apiRef.current, scheduleSave)
    try {
      const saved = await saveCurrentAnnotation({force: true})
      if (!saved) throw new Error(copy.whiteboard.saveFailed)
      if (saved.kind === 'screenshot') {
        const snapshot = await queueRecognizeScreenshot(saved.item.id)
        ocrSourceRef.current = {
          sourceKind: snapshot.request.sourceKind,
          sourceId: snapshot.request.sourceId,
        }
      } else {
        const sourceId = annotationOcrSourceId(saved.capture)
        const imagePath = saved.capture.timelineSnapshotPath || saved.capture.snapshotPath
        const snapshot = await queueRecognizeWhiteboard({
          imagePath,
          sceneId: sourceId,
          language: 'zh-en',
          priority: 'background',
        })
        ocrSourceRef.current = {
          sourceKind: snapshot.request.sourceKind,
          sourceId: snapshot.request.sourceId,
        }
      }
      setOcrMessage(copy.whiteboard.ocrQueued)
    } catch (error) {
      console.error('Failed to queue annotation OCR:', error)
      setOcrBusy(false)
      setOcrMessage(readableAnnotationError(error) || copy.whiteboard.ocrStatusFailed)
    }
  }

  const openAnnotationOcrResult = async () => {
    if (!ocrResultId) {
      await queueAnnotationOCR()
      return
    }
    try {
      const result = ocrResult?.id === ocrResultId ? ocrResult : await openOcrResult(ocrResultId)
      setOcrResult(result)
      if (ocrPositionTextVisible) {
        setOcrPositionTextVisible(false)
        setOcrHoveredBlockId('')
        setOcrCopiedBlockId('')
        setOcrMessage(copy.whiteboard.ocrBlocksHidden)
        return
      }
      const api = apiRef.current
      if (!api) return
      const blockCount = countOcrPositionTextBlocks(result)
      if (blockCount === 0) {
        setOcrMessage(copy.screenshot.ocrNoText)
        return
      }
      removeAnnotationOcrPositionTextElements(api, scheduleSave)
      setOcrPositionTextVisible(true)
      setOcrHoveredBlockId('')
      setOcrCopiedBlockId('')
      setOcrMessage(copy.screenshot.ocrBlocks(blockCount))
    } catch (error) {
      console.error('Failed to open annotation OCR result:', error)
      setOcrMessage(readableAnnotationError(error) || copy.whiteboard.ocrStatusFailed)
    }
  }

  if (!initialData) {
    return (
      <main className="annotation-overlay-shell" data-theme={theme}>
        <section className="annotation-capsule" aria-label={copy.whiteboard.title}>
          <span className="annotation-capsule-title">{copy.whiteboard.loading}</span>
        </section>
      </main>
    )
  }

  const saveStatusLabel = saving
    ? copy.whiteboard.saving
    : dirty
      ? copy.whiteboard.unsaved
      : lastSavedSceneRef.current
        ? copy.whiteboard.saved
        : copy.whiteboard.ready
  const canOpenOcrResult = Boolean(ocrResultId || ocrResult?.id)

  return (
    <main className={`annotation-overlay-shell ${canvasReceivesInput ? 'is-drawing' : 'is-pass-through'}`} data-theme={theme}>
      <div
        ref={capsuleRef}
        className={`annotation-toolbar-stack ${overlayState?.toolbarPlacement === 'bottom' ? 'is-bottom' : 'is-top'}`}
        style={{
          left: toolbarBounds.x,
          top: toolbarBounds.y,
          width: toolbarBounds.width,
          transform: `translate(${toolbarOffset.x}px, ${toolbarOffset.y}px)`,
        }}
      >
      <section className="annotation-capsule" aria-label={copy.whiteboard.title} onPointerDown={startToolbarDrag}>
        <span className="annotation-capsule-title">{isScreenshotMode ? copy.screenshot.region : copy.whiteboard.open}</span>
        <div className="annotation-tools" role="toolbar" aria-label={copy.whiteboard.title}>
          {annotationTools.map(({tool, icon: Icon, label}) => (
            <button
              key={tool}
              className={activeTool === tool ? 'selected' : ''}
              type="button"
              aria-label={copy.whiteboard[label]}
              title={copy.whiteboard[label]}
              onClick={() => setActiveTool(tool)}
            >
              <Icon size={16} />
            </button>
          ))}
        </div>
        {showAnnotationStyleControls && (
          <button
            className={annotationStylePanelOpen ? 'selected' : ''}
            type="button"
            aria-label={copy.whiteboard.styleSettings}
            title={copy.whiteboard.styleSettings}
            onClick={() => setAnnotationStylePanelOpen((open) => !open)}
          >
            <Settings2 size={16} />
          </button>
        )}
        <button type="button" aria-label={copy.whiteboard.undo} title={copy.whiteboard.undo} onClick={undoAnnotationStep}>
          <Undo2 size={16} />
        </button>
        <button type="button" aria-label={copy.whiteboard.reselectRegion} title={copy.whiteboard.reselectRegion} onClick={() => void reselectRegion()}>
          <RefreshCcw size={16} />
        </button>
        <button className={ocrSucceeded ? 'action-success' : ''} type="button" disabled={ocrBusy} aria-label={copy.whiteboard.recognizeText} title={copy.whiteboard.recognizeText} onClick={() => void queueAnnotationOCR()}>
          <ScanText size={16} />
          {ocrSucceeded && <span className="toolbar-action-check" aria-hidden="true"><Check size={9} /></span>}
        </button>
        <button type="button" disabled={!canOpenOcrResult} className={ocrPositionTextVisible ? 'selected' : ''} aria-label={copy.whiteboard.openOcrResult} title={copy.whiteboard.openOcrResult} onClick={() => void openAnnotationOcrResult()}>
          <Eye size={16} />
        </button>
        <button className={imageCopied ? 'action-success' : ''} type="button" aria-label={copy.whiteboard.copyImage} title={copy.whiteboard.copyImage} onClick={() => void copyAnnotationImage()}>
          <ClipboardCopy size={16} />
          {imageCopied && <span className="toolbar-action-check" aria-hidden="true"><Check size={9} /></span>}
        </button>
        {ocrMessage && <span className="annotation-ocr-status" role="status" aria-live="polite">{ocrMessage}</span>}
        <button className="annotation-save-status" type="button" aria-label={copy.whiteboard.save} title={copy.whiteboard.save} onClick={() => void saveCurrentAnnotation()}>
          <Save size={16} />
          <span>{saveStatusLabel}</span>
        </button>
        <button type="button" aria-label={copy.whiteboard.close} title={copy.whiteboard.close} onClick={() => void (isScreenshotMode ? hideScreenshotAnnotationOverlay() : hideAnnotationOverlay())}>
          <X size={17} />
        </button>
      </section>
      {showAnnotationStyleControls && annotationStylePanelOpen && (
        <section className="annotation-style-capsule" aria-label={copy.whiteboard.styleSettings}>
          <div className="annotation-style-group" aria-label={copy.whiteboard.strokeColor}>
            {annotationColors.map((color) => (
              <button
                key={color}
                className={strokeColor.toLowerCase() === color ? 'selected' : ''}
                type="button"
                aria-label={`${copy.whiteboard.strokeColor} ${color}`}
                title={color}
                style={{'--swatch': color} as any}
                onClick={() => {
                  setStrokeColor(color)
                  void patchWhiteboardSettings({lastStrokeColor: color})
                }}
              />
            ))}
          </div>
          <div className="annotation-width-group" aria-label={copy.whiteboard.strokeWidth}>
            {annotationStrokeWidths.map((width) => (
              <button
                key={width}
                className={strokeWidth === width ? 'selected' : ''}
                type="button"
                onClick={() => {
                  setStrokeWidth(width)
                  void patchWhiteboardSettings({lastStrokeWidth: width})
                }}
              >
                {copy.whiteboard[width]}
              </button>
            ))}
          </div>
          <label className="annotation-opacity-group" aria-label={copy.whiteboard.opacity} title={copy.whiteboard.opacity}>
            <span>{opacity}%</span>
            <input
              type="range"
              min={5}
              max={100}
              step={5}
              value={opacity}
              onChange={(event) => {
                const nextOpacity = normalizeOpacity(Number(event.currentTarget.value))
                setOpacity(nextOpacity)
                void patchWhiteboardSettings({lastOpacity: nextOpacity})
              }}
            />
          </label>
          <AnnotationStyleControls
            copy={copy}
            activeTool={activeTool}
            style={annotationStyle}
            onChange={updateAnnotationStyle}
          />
          <button
            className="annotation-style-hide"
            type="button"
            aria-label={copy.whiteboard.hideStyleSettings}
            title={copy.whiteboard.hideStyleSettings}
            onClick={() => setAnnotationStylePanelOpen(false)}
          >
            <EyeOff size={15} />
          </button>
        </section>
      )}
      </div>
      <section
        ref={canvasRef}
        className="annotation-overlay-canvas"
        aria-label={copy.whiteboard.title}
        style={{
          left: canvasBounds.x,
          top: canvasBounds.y,
          width: canvasBounds.width,
          height: canvasBounds.height,
        }}
      >
        <Excalidraw
          key={overlayKey}
          initialData={initialData}
          langCode={locale}
          theme="dark"
          excalidrawAPI={(api) => {
            apiRef.current = api
            if (api) {
              window.setTimeout(() => {
                setActiveTool(activeTool, false)
                applyStyle(strokeColor, strokeWidth, opacity, annotationStyle)
              }, 0)
            }
          }}
          onChange={((elements: readonly unknown[], appState: unknown, files: unknown) => {
            try {
              trackAnnotationElementEvents(elements)
              const sceneJson = (serializeAsJSON as any)(elements, appState, files, 'local')
              scheduleSave(sceneJson, elements.length > 0, annotationContentSignature(elements, files))
            } catch (error) {
              console.error('Failed to serialize annotation scene:', error)
            }
          }) as any}
          UIOptions={{
            canvasActions: {
              export: false,
              loadScene: false,
              saveToActiveFile: false,
              saveAsImage: false,
              clearCanvas: false,
              toggleTheme: false,
              changeViewBackgroundColor: false,
            },
            tools: {image: false},
          }}
          renderTopRightUI={() => null}
        />
        {ocrPositionTextVisible && ocrResult && (
          <OcrPositionTextLayer
            copy={copy}
            result={ocrResult}
            hoveredBlockId={ocrHoveredBlockId}
            copiedBlockId={ocrCopiedBlockId}
            onHover={setOcrHoveredBlockId}
            onCopy={copyAnnotationOcrPositionText}
            className="annotation"
            style={{
              position: 'absolute',
              inset: 0,
              minHeight: 0,
              maxHeight: 'none',
            }}
          />
        )}
      </section>
      <div
        className="annotation-overlay-frame"
        aria-hidden="true"
        style={{
          left: canvasBounds.x,
          top: canvasBounds.y,
          width: canvasBounds.width,
          height: canvasBounds.height,
        }}
      >
        <span className="corner top-left" />
        <span className="corner top-right" />
        <span className="corner bottom-left" />
        <span className="corner bottom-right" />
      </div>
    </main>
  )
}

function normalizeOpacity(value: unknown) {
  const numeric = typeof value === 'number' && Number.isFinite(value) ? value : 100
  if (numeric < 5) return 5
  if (numeric > 100) return 100
  return Math.round(numeric / 5) * 5
}

function annotationCanvasReceivesInput(tool: WhiteboardTool) {
  return tool !== 'selection' && tool !== 'hand'
}

function annotationContentSignature(elements: readonly unknown[], files: unknown) {
  const fileEntries = Object.entries((files && typeof files === 'object' ? files : {}) as Record<string, any>)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([id, file]) => ({
      id,
      mimeType: file?.mimeType,
      dataLength: typeof file?.dataURL === 'string' ? file.dataURL.length : 0,
    }))
  return JSON.stringify({
    elements: elements.map(annotationContentElement),
    files: fileEntries,
  })
}

function annotationContentSignatureFromScene(scene: any) {
  return annotationContentSignature(scene?.elements ?? [], scene?.files ?? {})
}

function annotationContentElement(element: unknown) {
  if (!element || typeof element !== 'object') return element
  const record = element as Record<string, any>
  return {
    id: record.id,
    type: record.type,
    version: record.version,
    isDeleted: record.isDeleted === true,
    x: record.x,
    y: record.y,
    width: record.width,
    height: record.height,
    angle: record.angle,
    strokeColor: record.strokeColor,
    backgroundColor: record.backgroundColor,
    strokeWidth: record.strokeWidth,
    opacity: record.opacity,
    text: record.text,
    points: record.points,
    fileId: record.fileId,
  }
}

function elementHitRegion(element: HTMLElement | null, kind: CapsuleWindowHitRegion['kind'], radius: number): CapsuleWindowHitRegion | null {
  if (!element) return null
  const rect = element.getBoundingClientRect()
  if (rect.width <= 0 || rect.height <= 0) return null
  return {
    x: rect.left,
    y: rect.top,
    width: rect.width,
    height: rect.height,
    kind,
    radius,
  }
}

function defaultAnnotationScene(settings: AppSettings) {
  return {
    appState: {
      viewBackgroundColor: 'transparent',
      currentItemStrokeColor: settings.whiteboard.lastStrokeColor || '#ef4444',
      currentItemStrokeWidthKey: settings.whiteboard.lastStrokeWidth || 'medium',
      currentItemOpacity: normalizeOpacity(settings.whiteboard.lastOpacity),
      currentItemBackgroundColor: 'transparent',
      currentItemRoughness: 1,
      currentItemFontFamily: 1,
      currentItemFontSize: 20,
      currentItemTextAlign: 'left',
    },
    elements: [],
    files: {},
  }
}

function annotationSceneWithFrozenSource(scene: any, dataURL: string, canvasBounds: {width: number; height: number}) {
  const source = dataURL.trim()
  if (!source) return scene
  const elements = Array.isArray(scene?.elements) ? scene.elements : []
  const files = scene?.files && typeof scene.files === 'object' ? scene.files : {}
  const sourceElementIds = new Set<string>()
  const sourceFileIds = new Set<string>()
  for (const element of elements) {
    if (!element || typeof element !== 'object') continue
    const customData = (element as {customData?: {recordingFreedomFrozenSource?: boolean}}).customData
    if (customData?.recordingFreedomFrozenSource !== true) continue
    const id = typeof (element as {id?: unknown}).id === 'string' ? (element as {id: string}).id : ''
    const fileId = typeof (element as {fileId?: unknown}).fileId === 'string' ? (element as {fileId: string}).fileId : ''
    if (id) sourceElementIds.add(id)
    if (fileId) sourceFileIds.add(fileId)
  }
  const retainedElements = elements.filter((element: {id?: string}) => !sourceElementIds.has(element.id ?? ''))
  const retainedFiles = Object.fromEntries(Object.entries(files).filter(([id]) => !sourceFileIds.has(id)))
  const fileId = `rf-frozen-annotation-source-${Date.now()}`
  const width = Math.max(1, Math.round(canvasBounds.width))
  const height = Math.max(1, Math.round(canvasBounds.height))
  return {
    ...scene,
    elements: [{
      id: `${fileId}-image`,
      type: 'image',
      x: 0,
      y: 0,
      width,
      height,
      angle: 0,
      strokeColor: 'transparent',
      backgroundColor: 'transparent',
      fillStyle: 'solid',
      strokeWidth: 1,
      strokeStyle: 'solid',
      roughness: 0,
      opacity: 100,
      groupIds: [],
      frameId: null,
      roundness: null,
      seed: 1,
      version: 1,
      versionNonce: 1,
      isDeleted: false,
      boundElements: null,
      updated: Date.now(),
      link: null,
      locked: true,
      status: 'saved',
      fileId,
      scale: [1, 1],
      customData: {recordingFreedomFrozenSource: true},
    }, ...retainedElements],
    files: {
      ...retainedFiles,
      [fileId]: {
        id: fileId,
        dataURL: source,
        mimeType: source.startsWith('data:image/svg+xml')
          ? 'image/svg+xml'
          : source.startsWith('data:image/jpeg')
            ? 'image/jpeg'
            : source.startsWith('data:image/webp')
              ? 'image/webp'
              : 'image/png',
        created: Date.now(),
      },
    },
  }
}

function screenshotAnnotationScene(settings: AppSettings, context: ScreenshotWhiteboardContext, overlayState: AnnotationOverlayState | null) {
  if (!context.available || !context.dataUrl) return null
  const canvas = annotationSnapshotCanvasSize(overlayState)
  const fileId = `rf-screenshot-annotation-${context.item?.id ?? Date.now()}`
  return {
    appState: {
      viewBackgroundColor: 'transparent',
      currentItemStrokeColor: settings.whiteboard.lastStrokeColor || '#ef4444',
      currentItemStrokeWidthKey: settings.whiteboard.lastStrokeWidth || 'medium',
      currentItemOpacity: normalizeOpacity(settings.whiteboard.lastOpacity),
      currentItemBackgroundColor: 'transparent',
      currentItemRoughness: 1,
      currentItemFontFamily: 1,
      currentItemFontSize: 20,
      currentItemTextAlign: 'left',
    },
    elements: [{
      id: `${fileId}-image`,
      type: 'image',
      x: 0,
      y: 0,
      width: canvas.width,
      height: canvas.height,
      angle: 0,
      strokeColor: 'transparent',
      backgroundColor: 'transparent',
      fillStyle: 'solid',
      strokeWidth: 1,
      strokeStyle: 'solid',
      roughness: 0,
      opacity: 100,
      groupIds: [],
      frameId: null,
      roundness: null,
      seed: 1,
      version: 1,
      versionNonce: 1,
      isDeleted: false,
      boundElements: null,
      updated: Date.now(),
      link: null,
      locked: true,
      status: 'saved',
      fileId,
      scale: [1, 1],
    }],
    files: {
      [fileId]: {
        id: fileId,
        dataURL: context.dataUrl,
        mimeType: screenshotAnnotationMimeType(context.dataUrl),
        created: Date.now(),
      },
    },
  }
}

function screenshotAnnotationMimeType(dataUrl: string) {
  if (dataUrl.startsWith('data:image/svg+xml')) return 'image/svg+xml'
  if (dataUrl.startsWith('data:image/jpeg')) return 'image/jpeg'
  if (dataUrl.startsWith('data:image/webp')) return 'image/webp'
  return 'image/png'
}

function safeParseScene(sceneJson: string) {
  try {
    return JSON.parse(sceneJson)
  } catch {
    return null
  }
}

function annotationOverlayKey(state: AnnotationOverlayState | null) {
  if (!state) return 'annotation-overlay'
  const geometry = state.target.geometry ?? state.windowBounds
  return [
    state.packageDir || 'annotation-overlay',
    state.target.type || 'target',
    state.target.id || 'unknown',
    geometry.x,
    geometry.y,
    geometry.width,
    geometry.height,
    state.sourceImageCapturedAt || state.sourceImageDataURL?.length || 'no-source-image',
  ].join(':')
}

function annotationCanvasBounds(state: AnnotationOverlayState | null) {
  return {
    x: Math.round(state?.canvasBounds.x ?? 0),
    y: Math.round(state?.canvasBounds.y ?? 0),
    width: Math.max(1, Math.round(state?.canvasBounds.width ?? window.innerWidth ?? 1)),
    height: Math.max(1, Math.round(state?.canvasBounds.height ?? window.innerHeight ?? 1)),
  }
}

function annotationToolbarBounds(state: AnnotationOverlayState | null, canvasBounds: {x: number; y: number; width: number; height: number}) {
  const viewportWidth = Math.max(1, Math.round(window.innerWidth || canvasBounds.width || 1))
  const edgePadding = 8
  const maxWidth = Math.max(1, viewportWidth - edgePadding * 2)
  const configured = state?.toolbarBounds
  if (configured && configured.width > 0 && configured.height > 0) {
    const width = Math.min(maxWidth, Math.max(1, Math.round(configured.width)))
    const maxLeft = Math.max(edgePadding, viewportWidth - width - edgePadding)
    return {
      x: Math.min(maxLeft, Math.max(edgePadding, Math.round(configured.x))),
      y: Math.round(configured.y),
      width,
      height: Math.max(1, Math.round(configured.height)),
    }
  }
  const width = Math.min(720, maxWidth)
  const height = canvasBounds.width < 720 ? 96 : 48
  const left = Math.max(edgePadding, Math.round((viewportWidth - width) / 2))
  const top = state?.toolbarPlacement === 'bottom'
    ? canvasBounds.y + canvasBounds.height + 8
    : 8
  return {x: left, y: top, width, height}
}

function elementSignatureMap(elements: readonly unknown[]) {
  const signatures = new Map<string, string>()
  for (const element of elements) {
    const identity = annotationElementIdentity(element)
    if (!identity) continue
    signatures.set(identity.id, annotationElementSignature(element))
  }
  return signatures
}

function annotationElementIdentity(element: unknown) {
  if (!element || typeof element !== 'object') return null
  const record = element as Record<string, unknown>
  const id = typeof record.id === 'string' ? record.id : ''
  if (!id) return null
  const type = typeof record.type === 'string' ? record.type : 'unknown'
  const version = typeof record.version === 'number' ? record.version : undefined
  const isDeleted = record.isDeleted === true
  return {
    id,
    type,
    version,
    isDeleted,
    bounds: annotationElementBounds(record),
  }
}

function annotationElementSignature(element: unknown) {
  if (!element || typeof element !== 'object') return ''
  const record = element as Record<string, unknown>
  return JSON.stringify({
    id: record.id,
    type: record.type,
    version: record.version,
    versionNonce: record.versionNonce,
    isDeleted: record.isDeleted === true,
    x: roundedNumber(record.x),
    y: roundedNumber(record.y),
    width: roundedNumber(record.width),
    height: roundedNumber(record.height),
    angle: roundedNumber(record.angle),
    strokeColor: record.strokeColor,
    backgroundColor: record.backgroundColor,
    fillStyle: record.fillStyle,
    strokeWidth: record.strokeWidth,
    strokeStyle: record.strokeStyle,
    opacity: record.opacity,
    text: record.text,
    pointsLength: Array.isArray(record.points) ? record.points.length : undefined,
  })
}

function annotationElementBounds(record: Record<string, unknown>) {
  return {
    x: roundedNumber(record.x),
    y: roundedNumber(record.y),
    width: roundedNumber(record.width),
    height: roundedNumber(record.height),
  }
}

function roundedNumber(value: unknown) {
  return typeof value === 'number' && Number.isFinite(value) ? Math.round(value * 100) / 100 : 0
}

function annotationSnapshotCanvasSize(overlayState: AnnotationOverlayState | null) {
  const width = Math.round(overlayState?.canvasBounds.width ?? window.innerWidth ?? 0)
  const height = Math.round(overlayState?.canvasBounds.height ?? window.innerHeight ?? 0)
  return {
    width: Math.max(1, width),
    height: Math.max(1, height),
  }
}

function annotationSnapshotExportElements(elements: readonly unknown[], width: number, height: number) {
  return [
    ...elements,
    transparentAnnotationBoundsElement(width, height),
  ]
}

function transparentAnnotationBoundsElement(width: number, height: number) {
  return {
    id: 'rf-annotation-snapshot-bounds',
    type: 'rectangle',
    x: 0,
    y: 0,
    width,
    height,
    angle: 0,
    strokeColor: 'transparent',
    backgroundColor: 'transparent',
    fillStyle: 'solid',
    strokeWidth: 1,
    strokeStyle: 'solid',
    roughness: 0,
    opacity: 0,
    groupIds: [],
    frameId: null,
    roundness: null,
    seed: 1,
    version: 1,
    versionNonce: 1,
    isDeleted: false,
    boundElements: null,
    updated: 1,
    link: null,
    locked: true,
  }
}

function removeAnnotationOcrPositionTextElements(api: ExcalidrawImperativeAPI | null, persist: (sceneJson: string, hasElements: boolean, contentSignature: string) => void) {
  if (!api) return
  const current = api.getSceneElements()
  const next = removeAnnotationOcrPositionTextElementsFromList(current)
  if (next.length === current.length) return
  api.updateScene({elements: next as any})
  persistAnnotationApiSceneSoon(api, persist)
}

function removeAnnotationOcrPositionTextElementsFromList(elements: readonly unknown[]) {
  return elements.filter((element) => {
    if (!element || typeof element !== 'object') return true
    const customData = (element as {customData?: {recordingFreedomOcr?: {kind?: string}}}).customData
    return customData?.recordingFreedomOcr?.kind !== 'position-text'
  })
}

function persistAnnotationApiSceneSoon(api: ExcalidrawImperativeAPI, persist: (sceneJson: string, hasElements: boolean, contentSignature: string) => void) {
  window.setTimeout(() => {
    try {
      const elements = api.getSceneElements()
      const files = api.getFiles()
      const sceneJson = (serializeAsJSON as any)(elements, api.getAppState(), files, 'local')
      persist(sceneJson, elements.length > 0, annotationContentSignature(elements, files))
    } catch (error) {
      console.error('Failed to persist annotation OCR text scene:', error)
    }
  }, 0)
}

function blobToDataURL(blob: Blob) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(blob)
  })
}

function annotationOcrSourceId(capture: AnnotationCapture) {
  return capture.packageDir || capture.scenePath || capture.snapshotPath || `annotation-${Date.now()}`
}

function annotationOcrEventMatches(event: OcrJobUpdate, source: {sourceKind: string; sourceId: string}) {
  return event.sourceKind === source.sourceKind && event.sourceId === source.sourceId
}

function readableAnnotationError(error: unknown) {
  if (!error) return ''
  if (error instanceof Error) return error.message
  if (typeof error === 'string') return error
  try {
    return JSON.stringify(error)
  } catch {
    return String(error)
  }
}

export default AnnotationOverlayWindow
