import {
  ArrowUpRight,
  Circle,
  Download,
  Eraser,
  Eye,
  FileText,
  Hand,
  Image as ImageIcon,
  Languages,
  Minus,
  MousePointer2,
  PenLine,
  RectangleHorizontal,
  Redo2,
  Save,
  Sparkles,
  Trash2,
  Type,
  Undo2,
  X,
} from 'lucide-react'
import {useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState, type CSSProperties} from 'react'
import {Excalidraw, exportToBlob, exportToSvg, sceneCoordsToViewportCoords, serializeAsJSON} from '@excalidraw/excalidraw'
import type {ExcalidrawImperativeAPI} from '@excalidraw/excalidraw/types'
import '@excalidraw/excalidraw/index.css'
import {copyByLocale, type RecorderCopy} from './i18n'
import {defaultSettings, normalizeLocale, normalizeTheme, type AppSettings, type LocaleCode, type ThemeCode, type WhiteboardStrokeWidth, type WhiteboardTool} from './services/mockBackend'
import {consumeScreenshotWhiteboardContext, hideWhiteboardWindow, loadSettings, loadWhiteboardScene, openOcrResult, patchWhiteboardSettings, queueRecognizeWhiteboard, saveWhiteboardExport, saveWhiteboardScene, saveWhiteboardSnapshot, subscribeOcrJobEvents, subscribeScreenshotWhiteboardContext, subscribeSettingsChanged, translateOcr, type OcrBlock, type OcrResult, type OcrTranslationResult, type ScreenshotWhiteboardContext} from './services/recorderBackend'
import {OcrPositionTextLayer, countOcrPositionTextBlocks} from './components/ocr/OcrPositionTextLayer'
import {writeClipboardText} from './utils/clipboard'

type SaveState = 'ready' | 'dirty' | 'saving' | 'saved' | 'failed'
type WhiteboardWindowGlobal = Window & {__RF_SCREENSHOT_WHITEBOARD__?: ScreenshotWhiteboardContext}
type SelectedWhiteboardImage = {elementId: string; fileId: string; dataURL: string; x: number; y: number; width: number; height: number}
type RecordingFreedomOcrElementKind = 'block' | 'text' | 'position-text' | 'translation'

const whiteboardColors = ['#ef4444', '#f59e0b', '#22c55e', '#38bdf8', '#a78bfa', '#f8fafc', '#111827']
const whiteboardTools: Array<{tool: WhiteboardTool; icon: typeof PenLine; labelKey: keyof AppSettings['whiteboard'] | string}> = [
  {tool: 'selection', icon: MousePointer2, labelKey: 'select'},
  {tool: 'hand', icon: Hand, labelKey: 'hand'},
  {tool: 'freedraw', icon: PenLine, labelKey: 'pen'},
  {tool: 'laser', icon: Sparkles, labelKey: 'laser'},
  {tool: 'arrow', icon: ArrowUpRight, labelKey: 'arrow'},
  {tool: 'line', icon: Minus, labelKey: 'line'},
  {tool: 'rectangle', icon: RectangleHorizontal, labelKey: 'rectangle'},
  {tool: 'ellipse', icon: Circle, labelKey: 'ellipse'},
  {tool: 'text', icon: Type, labelKey: 'text'},
  {tool: 'eraser', icon: Eraser, labelKey: 'eraser'},
]

const strokeWidths: WhiteboardStrokeWidth[] = ['thin', 'medium', 'bold']
const whiteboardPNGContentPrefix = 'data:image/png;base64,'

function WhiteboardWindow() {
  const [locale, setLocale] = useState<LocaleCode>('zh-CN')
  const [theme, setTheme] = useState<ThemeCode>('night-teal')
  const [initialData, setInitialData] = useState<any | null>(null)
  const [activeTool, setActiveToolState] = useState<WhiteboardTool>('freedraw')
  const [strokeColor, setStrokeColor] = useState('#ef4444')
  const [strokeWidth, setStrokeWidth] = useState<WhiteboardStrokeWidth>('medium')
  const [opacity, setOpacity] = useState(100)
  const [saveState, setSaveState] = useState<SaveState>('ready')
  const [statusText, setStatusText] = useState('')
  const [clearArmed, setClearArmed] = useState(false)
  const [ocrBusy, setOcrBusy] = useState(false)
  const [ocrResultId, setOcrResultId] = useState('')
  const [ocrResult, setOcrResult] = useState<OcrResult | null>(null)
  const [ocrTranslationResult, setOcrTranslationResult] = useState<OcrTranslationResult | null>(null)
  const [ocrTranslationBusy, setOcrTranslationBusy] = useState(false)
  const [ocrBlocksVisible, setOcrBlocksVisible] = useState(false)
  const [ocrPositionTextVisible, setOcrPositionTextVisible] = useState(false)
  const [ocrHoveredBlockId, setOcrHoveredBlockId] = useState('')
  const [ocrCopiedBlockId, setOcrCopiedBlockId] = useState('')
  const [ocrOverlayStyle, setOcrOverlayStyle] = useState<CSSProperties | undefined>(undefined)
  const [ocrSourceId, setOcrSourceId] = useState('')
  const [selectedImageElementId, setSelectedImageElementId] = useState('')
  const apiRef = useRef<ExcalidrawImperativeAPI | null>(null)
  const whiteboardCanvasRef = useRef<HTMLElement | null>(null)
  const settingsRef = useRef<AppSettings>(defaultSettings)
  const pendingImportedSceneRef = useRef<ReturnType<typeof screenshotScene> | null>(null)
  const lastSceneRef = useRef('')
  const lastScreenshotImportKeyRef = useRef('')
  const screenshotImportGuardUntilRef = useRef(0)
  const ocrSourceIdRef = useRef('')
  const ocrBusyRef = useRef(false)
  const statusHoldUntilRef = useRef(0)
  const selectedImageRef = useRef<SelectedWhiteboardImage | null>(null)
  const ocrImageAnchorRef = useRef<SelectedWhiteboardImage | null>(null)
  const saveTimerRef = useRef<number | null>(null)
  const clearTimerRef = useRef<number | null>(null)
  const copy = copyByLocale[locale]

  const excalidrawTheme = 'dark'

  const setOcrBusyState = useCallback((busy: boolean) => {
    ocrBusyRef.current = busy
    setOcrBusy(busy)
  }, [])

  const holdStatusText = useCallback((message: string, holdMs = 1600) => {
    statusHoldUntilRef.current = Date.now() + holdMs
    setStatusText(message)
  }, [])

  const refreshOcrOverlayPlacement = useCallback(() => {
    const api = apiRef.current
    const anchor = ocrImageAnchorRef.current ?? selectedImageRef.current
    const canvas = whiteboardCanvasRef.current
    if (!api || !anchor || !canvas) {
      setOcrOverlayStyle(undefined)
      return
    }
    setOcrOverlayStyle(whiteboardOcrOverlayStyle(api, canvas, anchor))
  }, [])

  const copyOcrPositionText = useCallback((block: OcrBlock, blockId: string) => {
    const text = block.text.trim()
    if (!text) {
      holdStatusText(copy.screenshot.copyTextEmpty)
      return
    }
    void writeClipboardText(text)
      .then(() => {
        setOcrCopiedBlockId(blockId)
        holdStatusText(copy.screenshot.copiedText)
        window.setTimeout(() => setOcrCopiedBlockId((current) => current === blockId ? '' : current), 1200)
      })
      .catch((error) => holdStatusText(readableError(error) || copy.screenshot.copyTextEmpty))
  }, [copy, holdStatusText])

  useEffect(() => {
    document.body.classList.add('rf-whiteboard-window')
    return () => document.body.classList.remove('rf-whiteboard-window')
  }, [])

  useEffect(() => {
    document.documentElement.lang = locale
  }, [locale])

  useEffect(() => {
    document.documentElement.dataset.theme = theme
  }, [theme])

  useEffect(() => {
    let cancelled = false
    void Promise.all([loadSettings(), loadWhiteboardScene(), consumeScreenshotWhiteboardContext()])
      .then(([settings, scene, screenshot]) => {
        if (cancelled) return
        applySettings(settings)
        const screenshotImport = screenshot.available && screenshot.dataUrl
          ? screenshotScene(settings, screenshot)
          : null
        if (screenshotImport) {
          lastScreenshotImportKeyRef.current = screenshotContextKey(screenshot)
          pendingImportedSceneRef.current = screenshotImport
          screenshotImportGuardUntilRef.current = Date.now() + 2500
        }
        const parsed = screenshotImport
          ? screenshotImport
          : scene.available && scene.sceneJson ? safeParseScene(scene.sceneJson) : null
        setInitialData(parsed ?? defaultScene(settings))
        if (!screenshot.available && scene.sceneJson) lastSceneRef.current = scene.sceneJson
        if (screenshotImport) {
          const sceneJson = importedSceneJSON(screenshotImport)
          lastSceneRef.current = sceneJson
          void saveWhiteboardScene(sceneJson)
            .then((saved) => {
              if (!cancelled && canShowSaveStatus(ocrBusyRef, statusHoldUntilRef)) setStatusText(saved.updatedAt ? `${copy.whiteboard.saved} · ${saved.scenePath}` : copy.whiteboard.saved)
            })
            .catch((error) => console.error('Failed to save imported screenshot scene:', error))
        }
        setStatusText(screenshot.available ? copy.whiteboard.unsaved : scene.available ? copy.whiteboard.ready : copy.whiteboard.unsaved)
      })
      .catch((error) => {
        console.error('Failed to load whiteboard:', error)
        if (!cancelled) setInitialData(defaultScene(defaultSettings))
      })
    const unsubscribe = subscribeSettingsChanged((settings) => applySettings(settings))
    return () => {
      cancelled = true
      unsubscribe()
    }
  }, [])

  useEffect(() => () => {
    if (saveTimerRef.current !== null) window.clearTimeout(saveTimerRef.current)
    if (clearTimerRef.current !== null) window.clearTimeout(clearTimerRef.current)
  }, [])

  useEffect(() => {
    ocrSourceIdRef.current = ocrSourceId
  }, [ocrSourceId])

  useLayoutEffect(() => {
    if (!ocrPositionTextVisible) {
      setOcrOverlayStyle(undefined)
      return
    }
    refreshOcrOverlayPlacement()
    const frame = window.requestAnimationFrame(refreshOcrOverlayPlacement)
    return () => window.cancelAnimationFrame(frame)
  }, [ocrPositionTextVisible, ocrResult?.id, selectedImageElementId, refreshOcrOverlayPlacement])

  useEffect(() => {
    if (!ocrPositionTextVisible) return
    const update = () => refreshOcrOverlayPlacement()
    const canvas = whiteboardCanvasRef.current
    const observer = typeof ResizeObserver !== 'undefined' && canvas ? new ResizeObserver(update) : null
    observer?.observe(canvas as Element)
    window.addEventListener('resize', update)
    return () => {
      observer?.disconnect()
      window.removeEventListener('resize', update)
    }
  }, [ocrPositionTextVisible, refreshOcrOverlayPlacement])

  useEffect(() => subscribeOcrJobEvents((event) => {
    if (event.sourceKind !== 'whiteboard' && event.sourceKind !== 'whiteboard-selection') return
    const currentSourceId = ocrSourceIdRef.current
    if (currentSourceId && event.sourceId !== currentSourceId) return
    if (event.status === 'queued') {
      setOcrBusyState(true)
      holdStatusText(copy.whiteboard.ocrQueued)
      return
    }
    if (event.status === 'running') {
      setOcrBusyState(true)
      holdStatusText(copy.whiteboard.ocrStatusRunning)
      return
    }
    if (event.status === 'ready') {
      setOcrBusyState(false)
      if (event.result?.id) {
        setOcrResultId(event.result.id)
        setOcrResult(event.result)
        setOcrTranslationResult(null)
        setOcrTranslationBusy(false)
        setOcrBlocksVisible(false)
        setOcrPositionTextVisible(false)
        setOcrHoveredBlockId('')
        setOcrCopiedBlockId('')
        removeOcrBlockElements(apiRef.current, scheduleSave)
        removeOcrPositionTextElements(apiRef.current, scheduleSave)
      }
      holdStatusText(event.result ? copy.screenshot.ocrBlocks(event.result.blocks.length) : copy.whiteboard.ocrStatusReady)
      return
    }
    if (event.status === 'failed') {
      setOcrBusyState(false)
      holdStatusText(event.error || copy.whiteboard.ocrStatusFailed)
    }
  }), [copy, holdStatusText, setOcrBusyState])

  const applySettings = (settings: AppSettings) => {
    settingsRef.current = settings
    setLocale(normalizeLocale(settings.locale))
    setTheme(normalizeTheme(settings.window.theme))
    setActiveToolState(settings.whiteboard.lastTool)
    setStrokeColor(settings.whiteboard.lastStrokeColor || '#ef4444')
    setStrokeWidth(settings.whiteboard.lastStrokeWidth)
    setOpacity(normalizeOpacity(settings.whiteboard.lastOpacity))
  }

  const setActiveTool = (tool: WhiteboardTool, persist = true) => {
    setActiveToolState(tool)
    apiRef.current?.setActiveTool({type: tool} as any)
    if (persist) {
      void patchWhiteboardSettings({lastTool: tool})
    }
  }

  const chooseStrokeColor = (color: string) => {
    setStrokeColor(color)
    void patchWhiteboardSettings({lastStrokeColor: color})
  }

  const chooseStrokeWidth = (width: WhiteboardStrokeWidth) => {
    setStrokeWidth(width)
    void patchWhiteboardSettings({lastStrokeWidth: width})
  }

  const chooseOpacity = (value: number) => {
    const nextOpacity = normalizeOpacity(value)
    setOpacity(nextOpacity)
    void patchWhiteboardSettings({lastOpacity: nextOpacity})
  }

  const applyStyle = useCallback((color: string, width: WhiteboardStrokeWidth, nextOpacity: number) => {
    apiRef.current?.updateScene({
      appState: {
        currentItemStrokeColor: color,
        currentItemStrokeWidthKey: width,
        currentItemOpacity: nextOpacity,
      },
    } as any)
  }, [])

  useEffect(() => {
    applyStyle(strokeColor, strokeWidth, opacity)
  }, [applyStyle, strokeColor, strokeWidth, opacity])

  const scheduleSave = useCallback((sceneJson: string) => {
    lastSceneRef.current = sceneJson
    setSaveState('dirty')
    if (canShowSaveStatus(ocrBusyRef, statusHoldUntilRef)) setStatusText(copy.whiteboard.unsaved)
    if (saveTimerRef.current !== null) window.clearTimeout(saveTimerRef.current)
    saveTimerRef.current = window.setTimeout(() => {
      void saveScene(sceneJson)
    }, 700)
  }, [copy.whiteboard.unsaved])

  const saveScene = async (sceneJson = lastSceneRef.current, options: {snapshot?: boolean} = {}) => {
    if (!sceneJson.trim()) return
    setSaveState('saving')
    try {
      const saved = options.snapshot
        ? (await saveWhiteboardSnapshot({sceneJson, snapshotDataUrl: await currentSnapshotDataURL()})).scene
        : await saveWhiteboardScene(sceneJson)
      lastSceneRef.current = sceneJson
      setSaveState('saved')
      if (canShowSaveStatus(ocrBusyRef, statusHoldUntilRef)) setStatusText(saved.updatedAt ? `${copy.whiteboard.saved} · ${saved.scenePath}` : copy.whiteboard.saved)
    } catch (error) {
      console.error('Failed to save whiteboard scene:', error)
      setSaveState('failed')
      if (canShowSaveStatus(ocrBusyRef, statusHoldUntilRef)) setStatusText(copy.whiteboard.saveFailed)
    }
  }

  const importScreenshotContext = useCallback((screenshot: ScreenshotWhiteboardContext) => {
    if (!screenshot.available || !screenshot.dataUrl) return
    const importKey = screenshotContextKey(screenshot)
    if (lastScreenshotImportKeyRef.current === importKey) return
    lastScreenshotImportKeyRef.current = importKey
    const scene = screenshotScene(settingsRef.current, screenshot)
    screenshotImportGuardUntilRef.current = Date.now() + 2500
    setInitialData(scene)
    setSaveState('dirty')
    if (canShowSaveStatus(ocrBusyRef, statusHoldUntilRef)) setStatusText(copy.whiteboard.unsaved)
    const api = apiRef.current
    if (!api) {
      pendingImportedSceneRef.current = scene
      scheduleSave(importedSceneJSON(scene))
      return
    }
    try {
      applyImportedSceneToApi(api, scene)
      persistApiSceneSoon(api, scheduleSave)
    } catch (error) {
      console.error('Failed to import screenshot into whiteboard:', error)
    }
  }, [copy.whiteboard.unsaved, scheduleSave])

  useEffect(() => {
    const whiteboardWindow = window as WhiteboardWindowGlobal
    if (whiteboardWindow.__RF_SCREENSHOT_WHITEBOARD__) {
      importScreenshotContext(whiteboardWindow.__RF_SCREENSHOT_WHITEBOARD__)
    }
    return subscribeScreenshotWhiteboardContext(importScreenshotContext)
  }, [importScreenshotContext])

  const onSceneChange = useCallback((elements: readonly unknown[], appState: unknown, files: unknown) => {
    try {
      const selectedImage = selectedWhiteboardImageFromScene(elements, appState, files)
      selectedImageRef.current = selectedImage
      setSelectedImageElementId(selectedImage?.elementId ?? '')
      if (selectedImage && ocrImageAnchorRef.current?.elementId === selectedImage.elementId) {
        ocrImageAnchorRef.current = selectedImage
      }
      if (ocrPositionTextVisible) window.requestAnimationFrame(refreshOcrOverlayPlacement)
      if (Date.now() < screenshotImportGuardUntilRef.current && !sceneHasImageElement(elements)) return
      const sceneJson = (serializeAsJSON as any)(elements, appState, files, 'local')
      scheduleSave(sceneJson)
    } catch (error) {
      console.error('Failed to serialize whiteboard scene:', error)
    }
  }, [ocrPositionTextVisible, refreshOcrOverlayPlacement, scheduleSave])

  const saveNow = () => {
    void saveScene(currentSceneJSON(), {snapshot: true})
  }

  const queueCurrentWhiteboardOCR = async () => {
    const sceneJson = currentSceneJSON()
    if (!sceneJson.trim() || ocrBusy) return
    setOcrBusyState(true)
    setOcrResultId('')
    setOcrResult(null)
    setOcrTranslationResult(null)
    setOcrTranslationBusy(false)
    setOcrBlocksVisible(false)
    setOcrPositionTextVisible(false)
    setOcrHoveredBlockId('')
    setOcrCopiedBlockId('')
    ocrImageAnchorRef.current = null
    removeOcrBlockElements(apiRef.current, scheduleSave)
    removeOcrPositionTextElements(apiRef.current, scheduleSave)
    removeOcrTranslationElements(apiRef.current, scheduleSave)
    holdStatusText(copy.whiteboard.ocrPreparing)
    try {
      const snapshotDataUrl = await currentSnapshotDataURL()
      const saved = await saveWhiteboardSnapshot({sceneJson, snapshotDataUrl})
      lastSceneRef.current = sceneJson
      setSaveState('saved')
      setOcrSourceId(saved.item.id)
      ocrSourceIdRef.current = saved.item.id
      await queueRecognizeWhiteboard({
        imagePath: saved.item.path,
        sceneId: saved.item.id,
        language: 'zh-en',
      })
      holdStatusText(copy.whiteboard.ocrQueued)
    } catch (error) {
      console.error('Failed to queue whiteboard OCR:', error)
      setOcrBusyState(false)
      holdStatusText(readableError(error) || copy.whiteboard.ocrStatusFailed)
    }
  }

  const queueSelectedImageOCR = async () => {
    const api = apiRef.current
    if (!api || ocrBusy) return
    const selected = selectedWhiteboardImageElement(api) ?? selectedImageRef.current
    if (!selected) {
      holdStatusText(copy.whiteboard.noImageSelection)
      return
    }
    setOcrBusyState(true)
    setOcrResultId('')
    setOcrResult(null)
    setOcrTranslationResult(null)
    setOcrTranslationBusy(false)
    setOcrBlocksVisible(false)
    setOcrPositionTextVisible(false)
    setOcrHoveredBlockId('')
    setOcrCopiedBlockId('')
    ocrImageAnchorRef.current = selected
    removeOcrBlockElements(apiRef.current, scheduleSave)
    removeOcrPositionTextElements(apiRef.current, scheduleSave)
    removeOcrTranslationElements(apiRef.current, scheduleSave)
    holdStatusText(copy.whiteboard.ocrPreparing)
    try {
      const sceneJson = currentSceneJSON()
      const savedScene = await saveWhiteboardScene(sceneJson)
      const snapshotDataUrl = await imageDataURLToPNG(selected.dataURL)
      const exported = await saveWhiteboardExport({format: 'png', dataUrl: snapshotDataUrl})
      const sourceId = savedScene.scenePath || selected.elementId
      setOcrSourceId(sourceId)
      ocrSourceIdRef.current = sourceId
      await queueRecognizeWhiteboard({
        imagePath: exported.outputPath,
        sceneId: sourceId,
        elementId: selected.elementId,
        language: 'zh-en',
      })
      holdStatusText(copy.whiteboard.ocrQueued)
    } catch (error) {
      console.error('Failed to queue selected image OCR:', error)
      setOcrBusyState(false)
      holdStatusText(readableError(error) || copy.whiteboard.ocrStatusFailed)
    }
  }

  const openWhiteboardOcrResult = async () => {
    if (!ocrResultId) {
      await queueCurrentWhiteboardOCR()
      return
    }
    try {
      const result = ocrResult?.id === ocrResultId ? ocrResult : await openOcrResult(ocrResultId)
      setOcrResult(result)
      if (ocrPositionTextVisible) {
        setOcrPositionTextVisible(false)
        setOcrHoveredBlockId('')
        setOcrCopiedBlockId('')
        holdStatusText(copy.whiteboard.ocrBlocksHidden)
        return
      }
      const api = apiRef.current
      if (!api) return
      const anchor = ocrImageAnchorRef.current ??
        selectedImageRef.current ??
        firstWhiteboardImageFromScene(api.getSceneElements(), api.getFiles())
      if (!anchor) {
        insertOcrText()
        return
      }
      const blockCount = countOcrPositionTextBlocks(result)
      if (blockCount === 0) {
        holdStatusText(copy.screenshot.ocrNoText)
        return
      }
      ocrImageAnchorRef.current = anchor
      removeOcrPositionTextElements(api, scheduleSave)
      setOcrPositionTextVisible(true)
      setOcrHoveredBlockId('')
      setOcrCopiedBlockId('')
      window.requestAnimationFrame(refreshOcrOverlayPlacement)
      holdStatusText(copy.screenshot.ocrBlocks(blockCount))
    } catch (error) {
      console.error('Failed to open whiteboard OCR result:', error)
      holdStatusText(readableError(error) || copy.whiteboard.ocrStatusFailed)
    }
  }

  const toggleOcrBlockOverlay = () => {
    if (ocrBlocksVisible) {
      removeOcrBlockElements(apiRef.current, scheduleSave)
      setOcrBlocksVisible(false)
      holdStatusText(copy.whiteboard.ocrBlocksHidden)
      return
    }
    const api = apiRef.current
    const anchor = ocrImageAnchorRef.current ?? selectedImageRef.current
    if (!api || !ocrResult || !anchor) {
      holdStatusText(copy.whiteboard.noImageSelection)
      return
    }
    const elements = buildOcrBlockElements(ocrResult, anchor)
    if (elements.length === 0) {
      holdStatusText(copy.screenshot.ocrNoText)
      return
    }
    const current = removeOcrBlockElementsFromList(api.getSceneElements())
    api.updateScene({elements: [...current, ...elements] as any})
    persistApiSceneSoon(api, scheduleSave)
    setOcrBlocksVisible(true)
    holdStatusText(copy.screenshot.ocrBlocks(elements.length))
  }

  const insertOcrText = () => {
    const api = apiRef.current
    const result = ocrResult
    const text = result?.plainText.trim()
    if (!api || !result || !text) {
      holdStatusText(copy.screenshot.copyTextEmpty)
      return
    }
    const anchor = ocrImageAnchorRef.current ?? selectedImageRef.current
    const textElement = buildOcrTextElement(result, text, anchor)
    holdStatusText(copy.whiteboard.ocrTextInserted)
    api.updateScene({elements: [...api.getSceneElements(), textElement] as any})
    persistApiSceneSoon(api, scheduleSave)
    window.setTimeout(() => holdStatusText(copy.whiteboard.ocrTextInserted), 0)
  }

  const applyOcrTranslationOverlay = (translated: OcrTranslationResult | null) => {
    const api = apiRef.current
    const result = ocrResult
    const text = ocrTranslationPlainText(translated)
    if (!api || !result || !text) return false
    const anchor = ocrImageAnchorRef.current ?? selectedImageRef.current
    if (anchor) {
      const elements = buildOcrTranslationTextElements(result, translated, anchor)
      if (elements.length > 0) {
        const current = removeOcrTranslationElementsFromList(api.getSceneElements())
        api.updateScene({elements: [...current, ...elements] as any})
        persistApiSceneSoon(api, scheduleSave)
        holdStatusText(copy.whiteboard.ocrTranslationInserted)
        window.setTimeout(() => holdStatusText(copy.whiteboard.ocrTranslationInserted), 0)
        return true
      }
    }
    const textElement = buildOcrTextElement(result, text, anchor)
    api.updateScene({elements: [...api.getSceneElements(), textElement] as any})
    persistApiSceneSoon(api, scheduleSave)
    holdStatusText(copy.whiteboard.ocrTranslationInserted)
    window.setTimeout(() => holdStatusText(copy.whiteboard.ocrTranslationInserted), 0)
    return true
  }

  const translateWhiteboardOcrText = async () => {
    const result = ocrResult
    const text = result?.plainText.trim()
    if (!result || !text || ocrTranslationBusy) {
      holdStatusText(copy.screenshot.copyTextEmpty)
      return
    }
    const normalized = normalizeOcrTranslationSettings(settingsRef.current.ocr.translation)
    const unavailable = ocrTranslationUnavailableMessage(normalized, copy)
    if (unavailable) {
      setOcrTranslationResult(null)
      holdStatusText(unavailable, 2600)
      return
    }
    setOcrTranslationBusy(true)
    holdStatusText(copy.screenshot.translationWorking)
    try {
      const translated = await translateOcr({
        ocrResultId: result.id,
        provider: normalized.provider,
        sourceLanguage: normalized.sourceLanguage,
        targetLanguage: normalized.targetLanguage,
        baseUrl: normalized.baseUrl,
        apiKey: normalized.apiKey,
        model: normalized.model,
        force: false,
      })
      setOcrTranslationResult(translated)
      applyOcrTranslationOverlay(translated)
      holdStatusText(copy.screenshot.translationReady)
    } catch (error) {
      setOcrTranslationResult(null)
      holdStatusText(`${copy.screenshot.translationFailed}: ${readableError(error)}`, 3200)
    } finally {
      setOcrTranslationBusy(false)
    }
  }

  const insertOcrTranslationText = () => {
    if (!applyOcrTranslationOverlay(ocrTranslationResult)) {
      holdStatusText(copy.screenshot.copyTextEmpty)
    }
  }

  const clearScene = () => {
    if (!clearArmed) {
      setClearArmed(true)
      setStatusText(copy.whiteboard.clearConfirm)
      if (clearTimerRef.current !== null) window.clearTimeout(clearTimerRef.current)
      clearTimerRef.current = window.setTimeout(() => setClearArmed(false), 1800)
      return
    }
    setClearArmed(false)
    apiRef.current?.resetScene()
    window.setTimeout(() => {
      const api = apiRef.current
      if (!api) return
      const sceneJson = (serializeAsJSON as any)(api.getSceneElements(), api.getAppState(), api.getFiles(), 'local')
      scheduleSave(sceneJson)
    }, 0)
  }

  const invokeHistoryShortcut = (direction: 'undo' | 'redo') => {
    const target = document.querySelector<HTMLElement>('.whiteboard-canvas .excalidraw') ?? document.querySelector<HTMLElement>('.whiteboard-canvas')
    if (!target) return
    target.focus({preventScroll: true})
    const isMac = /Mac|iPhone|iPad|iPod/i.test(window.navigator.platform)
    target.dispatchEvent(new KeyboardEvent('keydown', {
      key: 'z',
      code: 'KeyZ',
      ctrlKey: !isMac,
      metaKey: isMac,
      shiftKey: direction === 'redo',
      bubbles: true,
      cancelable: true,
    }))
  }

  const exportPNG = async () => {
    const api = apiRef.current
    if (!api) return
    try {
      const blob = await exportToBlob({
        elements: api.getSceneElements(),
        appState: {
          ...api.getAppState(),
          exportBackground: true,
          viewBackgroundColor: api.getAppState().viewBackgroundColor,
        },
        files: api.getFiles(),
        mimeType: 'image/png',
      } as any)
      const dataUrl = await blobToDataURL(blob)
      const result = await saveWhiteboardExport({format: 'png', dataUrl})
      setStatusText(copy.whiteboard.exported(result.outputPath))
    } catch (error) {
      console.error('Failed to export whiteboard PNG:', error)
      setStatusText(copy.whiteboard.exportFailed)
    }
  }

  const exportExcalidraw = async () => {
    const sceneJson = currentSceneJSON()
    if (!sceneJson) return
    try {
      const result = await saveWhiteboardExport({format: 'excalidraw', payload: sceneJson})
      setStatusText(copy.whiteboard.exported(result.outputPath))
    } catch (error) {
      console.error('Failed to export whiteboard scene:', error)
      setStatusText(copy.whiteboard.exportFailed)
    }
  }

  const currentSceneJSON = () => {
    const api = apiRef.current
    if (!api) return lastSceneRef.current
    try {
      return (serializeAsJSON as any)(api.getSceneElements(), api.getAppState(), api.getFiles(), 'local')
    } catch (error) {
      console.error('Failed to serialize current whiteboard scene:', error)
      return lastSceneRef.current
    }
  }

  const currentSnapshotDataURL = async () => {
    const api = apiRef.current
    if (!api) throw new Error('whiteboard canvas is not ready')
    const canvasRect = document.querySelector<HTMLElement>('.whiteboard-canvas')?.getBoundingClientRect()
    const width = Math.round(Math.max(640, Math.min(2400, canvasRect?.width ?? 1120)))
    const height = Math.round(Math.max(360, Math.min(1600, canvasRect?.height ?? 720)))
    const blob = await exportToBlob({
      elements: api.getSceneElements(),
      appState: {
        ...api.getAppState(),
        exportBackground: true,
        exportScale: 1,
        viewBackgroundColor: api.getAppState().viewBackgroundColor,
      },
      files: api.getFiles(),
      mimeType: 'image/png',
      exportPadding: 0,
      getDimensions: () => ({width, height, scale: 1}),
    } as any)
    return blobToDataURL(blob)
  }

  const exportSVG = async () => {
    const api = apiRef.current
    if (!api) return
    try {
      const svg = await exportToSvg({
        elements: api.getSceneElements(),
        appState: api.getAppState(),
        files: api.getFiles(),
      } as any)
      const payload = new XMLSerializer().serializeToString(svg)
      const result = await saveWhiteboardExport({format: 'svg', payload})
      setStatusText(copy.whiteboard.exported(result.outputPath))
    } catch (error) {
      console.error('Failed to export whiteboard SVG:', error)
      setStatusText(copy.whiteboard.exportFailed)
    }
  }

  const toolButtons = useMemo(() => whiteboardTools.map(({tool, icon: Icon, labelKey}) => {
    const label = copy.whiteboard[labelKey as keyof typeof copy.whiteboard] as string
    return (
      <button
        key={tool}
        className={activeTool === tool ? 'selected' : ''}
        type="button"
        aria-label={label}
        title={label}
        onClick={() => setActiveTool(tool)}
      >
        <Icon size={17} />
      </button>
    )
  }), [activeTool, copy])

  if (!initialData) {
    return (
      <main className="whiteboard-loading">
        <span>{copy.whiteboard.loading}</span>
      </main>
    )
  }

  return (
    <main className="whiteboard-shell" data-theme={theme}>
      <section className="whiteboard-capsule" aria-label={copy.whiteboard.title}>
        <div className="whiteboard-title">
          <strong>{copy.whiteboard.title}</strong>
          <span>{copy.whiteboard.subtitle}</span>
        </div>
        <div className="whiteboard-tools" role="toolbar" aria-label={copy.whiteboard.title}>
          {toolButtons}
        </div>
        <div className="whiteboard-style-group" aria-label={copy.whiteboard.strokeColor}>
          {whiteboardColors.map((color) => (
            <button
              key={color}
              className={strokeColor.toLowerCase() === color ? 'selected' : ''}
              type="button"
              aria-label={`${copy.whiteboard.strokeColor} ${color}`}
              title={color}
              style={{'--swatch': color} as any}
              onClick={() => chooseStrokeColor(color)}
            />
          ))}
        </div>
        <div className="whiteboard-width-group" aria-label={copy.whiteboard.strokeWidth}>
          {strokeWidths.map((width) => (
            <button
              key={width}
              className={strokeWidth === width ? 'selected' : ''}
              type="button"
              onClick={() => chooseStrokeWidth(width)}
            >
              {copy.whiteboard[width]}
            </button>
          ))}
        </div>
        <label className="whiteboard-opacity-group" aria-label={copy.whiteboard.opacity} title={copy.whiteboard.opacity}>
          <span>{opacity}%</span>
          <input
            type="range"
            min={5}
            max={100}
            step={5}
            value={opacity}
            onChange={(event) => chooseOpacity(Number(event.currentTarget.value))}
          />
        </label>
        <div className="whiteboard-actions">
          <button type="button" aria-label={copy.whiteboard.undo} title={copy.whiteboard.undo} onClick={() => invokeHistoryShortcut('undo')}>
            <Undo2 size={16} />
          </button>
          <button type="button" aria-label={copy.whiteboard.redo} title={copy.whiteboard.redo} onClick={() => invokeHistoryShortcut('redo')}>
            <Redo2 size={16} />
          </button>
          <button type="button" aria-label={copy.whiteboard.save} title={copy.whiteboard.save} onClick={saveNow}>
            <Save size={16} />
          </button>
          <button type="button" disabled={ocrBusy} aria-label={copy.whiteboard.recognizeText} title={copy.whiteboard.recognizeText} onClick={() => void queueCurrentWhiteboardOCR()}>
            <FileText size={16} />
            <span>OCR</span>
          </button>
          <button type="button" disabled={ocrBusy || !selectedImageElementId} aria-label={copy.whiteboard.recognizeSelectedImage} title={selectedImageElementId ? copy.whiteboard.recognizeSelectedImage : copy.whiteboard.noImageSelection} onClick={() => void queueSelectedImageOCR()}>
            <ImageIcon size={16} />
            <span>OCR</span>
          </button>
          <button type="button" disabled={!ocrResultId} className={ocrPositionTextVisible ? 'selected' : ''} aria-label={copy.whiteboard.openOcrResult} title={copy.whiteboard.openOcrResult} onClick={() => void openWhiteboardOcrResult()}>
            <Eye size={16} />
          </button>
          <button type="button" disabled={!ocrResult || !ocrImageAnchorRef.current} className={ocrBlocksVisible ? 'selected' : ''} aria-label={copy.whiteboard.toggleOcrBlocks} title={copy.whiteboard.toggleOcrBlocks} onClick={toggleOcrBlockOverlay}>
            <RectangleHorizontal size={16} />
          </button>
          <button type="button" disabled={!ocrResult?.plainText.trim()} aria-label={copy.whiteboard.insertOcrText} title={copy.whiteboard.insertOcrText} onClick={insertOcrText}>
            <Type size={16} />
          </button>
          <button type="button" disabled={!ocrResult?.plainText.trim() || ocrTranslationBusy} aria-label={copy.whiteboard.translateOcrText} title={copy.whiteboard.translateOcrText} onClick={() => void translateWhiteboardOcrText()}>
            <Languages size={16} />
          </button>
          <button type="button" disabled={!ocrTranslationPlainText(ocrTranslationResult)} aria-label={copy.whiteboard.insertOcrTranslation} title={copy.whiteboard.insertOcrTranslation} onClick={insertOcrTranslationText}>
            <Languages size={16} />
            <Type size={12} />
          </button>
          <button type="button" aria-label={copy.whiteboard.exportPng} title={copy.whiteboard.exportPng} onClick={() => void exportPNG()}>
            <Download size={16} />
            <span>PNG</span>
          </button>
          <button type="button" aria-label={copy.whiteboard.exportSvg} title={copy.whiteboard.exportSvg} onClick={() => void exportSVG()}>
            <Download size={16} />
            <span>SVG</span>
          </button>
          <button type="button" aria-label={copy.whiteboard.exportExc} title={copy.whiteboard.exportExc} onClick={() => void exportExcalidraw()}>
            <Download size={16} />
            <span>EXC</span>
          </button>
          <button className={clearArmed ? 'danger armed' : 'danger'} type="button" aria-label={copy.whiteboard.clear} title={copy.whiteboard.clear} onClick={clearScene}>
            <Trash2 size={16} />
          </button>
          <button type="button" aria-label={copy.whiteboard.close} title={copy.whiteboard.close} onClick={() => void hideWhiteboardWindow()}>
            <X size={17} />
          </button>
        </div>
      </section>
      <section ref={whiteboardCanvasRef} className="whiteboard-canvas" aria-label={copy.whiteboard.title}>
        <Excalidraw
          initialData={initialData}
          langCode={locale}
          theme={excalidrawTheme}
          excalidrawAPI={(api) => {
            apiRef.current = api
            if (api) {
              window.setTimeout(() => {
                setActiveTool(activeTool, false)
                applyStyle(strokeColor, strokeWidth, opacity)
                if (pendingImportedSceneRef.current) {
                  const scene = pendingImportedSceneRef.current
                  pendingImportedSceneRef.current = null
                  applyImportedSceneToApi(api, scene)
                  persistApiSceneSoon(api, scheduleSave)
                }
              }, 0)
            }
          }}
          onChange={onSceneChange as any}
          onScrollChange={() => refreshOcrOverlayPlacement()}
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
            tools: {image: true},
          }}
          renderTopRightUI={() => null}
        />
        {ocrPositionTextVisible && ocrResult && ocrOverlayStyle && (
          <OcrPositionTextLayer
            copy={copy}
            result={ocrResult}
            translationResult={ocrTranslationResult}
            hoveredBlockId={ocrHoveredBlockId}
            copiedBlockId={ocrCopiedBlockId}
            onHover={setOcrHoveredBlockId}
            onCopy={copyOcrPositionText}
            className="whiteboard"
            style={ocrOverlayStyle}
          />
        )}
      </section>
      <footer className={`whiteboard-status ${saveState}`}>
        <span>{statusText || copy.whiteboard.ready}</span>
      </footer>
    </main>
  )
}

function readableError(error: unknown) {
  return error instanceof Error ? error.message : String(error || '')
}

function normalizeOcrTranslationSettings(value: Partial<AppSettings['ocr']['translation']> | undefined): AppSettings['ocr']['translation'] {
  const provider = value?.provider === 'deepl' || value?.provider === 'openai-compatible' ? value.provider : 'disabled'
  const privacyConfirmed = provider !== 'disabled' && value?.privacyConfirmed === true
  return {
    provider,
    baseUrl: cleanOptionalString(value?.baseUrl),
    apiKey: cleanOptionalString(value?.apiKey),
    apiKeySet: value?.apiKeySet === true || Boolean(cleanOptionalString(value?.apiKey)),
    model: cleanOptionalString(value?.model),
    sourceLanguage: cleanOptionalString(value?.sourceLanguage) || 'auto',
    targetLanguage: cleanOptionalString(value?.targetLanguage) || 'zh-CN',
    privacyConfirmed,
    privacyConfirmedAt: privacyConfirmed ? cleanOptionalString(value?.privacyConfirmedAt) : undefined,
  }
}

function ocrTranslationUnavailableMessage(translation: AppSettings['ocr']['translation'], copy: RecorderCopy): string {
  if (translation.provider === 'disabled') return copy.screenshot.translationUnavailable
  if (!translation.privacyConfirmed) return copy.screenshot.translationPrivacyRequired
  if (!translation.baseUrl) return copy.screenshot.translationMissingBaseUrl
  if (!translation.apiKey && !translation.apiKeySet) return copy.screenshot.translationMissingApiKey
  if (translation.provider === 'openai-compatible' && !translation.model) return copy.screenshot.translationMissingModel
  return ''
}

function ocrTranslationPlainText(result: OcrTranslationResult | null) {
  return result?.blocks.map((block) => block.translated.trim()).filter(Boolean).join('\n').trim() ?? ''
}

function cleanOptionalString(value: unknown): string | undefined {
  return typeof value === 'string' && value.trim() ? value.trim() : undefined
}

function canShowSaveStatus(ocrBusyRef: {current: boolean}, statusHoldUntilRef: {current: number}) {
  return !ocrBusyRef.current && Date.now() >= statusHoldUntilRef.current
}

function defaultScene(settings: AppSettings) {
  return {
    appState: {
      viewBackgroundColor: '#111827',
      currentItemStrokeColor: settings.whiteboard.lastStrokeColor || '#ef4444',
      currentItemStrokeWidthKey: settings.whiteboard.lastStrokeWidth || 'medium',
      currentItemOpacity: normalizeOpacity(settings.whiteboard.lastOpacity),
    },
    elements: [],
    files: {},
  }
}

function screenshotScene(settings: AppSettings, context: ScreenshotWhiteboardContext) {
  const item = context.item
  const width = Math.max(1, item?.width ?? 1280)
  const height = Math.max(1, item?.height ?? 720)
  const maxWidth = 1400
  const scale = width > maxWidth ? maxWidth / width : 1
  const sceneWidth = Math.round(width * scale)
  const sceneHeight = Math.round(height * scale)
  const fileId = `rf-screenshot-${item?.id ?? Date.now()}`
  return {
    appState: {
      viewBackgroundColor: '#111827',
      currentItemStrokeColor: settings.whiteboard.lastStrokeColor || '#ef4444',
      currentItemStrokeWidthKey: settings.whiteboard.lastStrokeWidth || 'medium',
      currentItemOpacity: normalizeOpacity(settings.whiteboard.lastOpacity),
      scrollX: 80,
      scrollY: 80,
    },
    elements: [{
      id: `${fileId}-image`,
      type: 'image',
      x: 0,
      y: 0,
      width: sceneWidth,
      height: sceneHeight,
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
        mimeType: 'image/png',
        created: Date.now(),
      },
    },
  }
}

function screenshotContextKey(context: ScreenshotWhiteboardContext) {
  const item = context.item
  return [
    item?.id ?? '',
    item?.path ?? '',
    item?.width ?? 0,
    item?.height ?? 0,
    context.dataUrl?.length ?? 0,
  ].join(':')
}

function importedSceneJSON(scene: ReturnType<typeof screenshotScene>) {
  return JSON.stringify({
    type: 'excalidraw',
    version: 2,
    source: 'recordingfreedom',
    elements: scene.elements,
    appState: scene.appState,
    files: scene.files,
  })
}

function sceneHasImageElement(elements: readonly unknown[]) {
  return elements.some((element) => {
    if (!element || typeof element !== 'object') return false
    return (element as {type?: string; isDeleted?: boolean}).type === 'image' &&
      (element as {isDeleted?: boolean}).isDeleted !== true
  })
}

function selectedWhiteboardImageElement(api: ExcalidrawImperativeAPI) {
  return selectedWhiteboardImageFromScene(api.getSceneElements(), api.getAppState(), api.getFiles())
}

function whiteboardOcrOverlayStyle(api: ExcalidrawImperativeAPI, canvas: HTMLElement, anchor: SelectedWhiteboardImage): CSSProperties | undefined {
  const appState = api.getAppState() as {
    scrollX?: number
    scrollY?: number
    zoom?: {value?: number}
  }
  const rect = canvas.getBoundingClientRect()
  const zoomValue = Number.isFinite(appState.zoom?.value) ? appState.zoom?.value as number : 1
  const viewportState = {
    zoom: {value: zoomValue},
    offsetLeft: rect.left,
    offsetTop: rect.top,
    scrollX: Number.isFinite(appState.scrollX) ? appState.scrollX as number : 0,
    scrollY: Number.isFinite(appState.scrollY) ? appState.scrollY as number : 0,
  }
  const topLeft = sceneCoordsToViewportCoords({sceneX: anchor.x, sceneY: anchor.y}, viewportState as any)
  const bottomRight = sceneCoordsToViewportCoords({sceneX: anchor.x + anchor.width, sceneY: anchor.y + anchor.height}, viewportState as any)
  const width = Math.max(1, bottomRight.x - topLeft.x)
  const height = Math.max(1, bottomRight.y - topLeft.y)
  return {
    position: 'absolute',
    left: topLeft.x - rect.left,
    top: topLeft.y - rect.top,
    width,
    height,
    minHeight: 0,
    maxHeight: 'none',
  }
}

function selectedWhiteboardImageFromScene(elements: readonly unknown[], appState: unknown, files: unknown): SelectedWhiteboardImage | null {
  const selectedIds = selectedElementIDSet(appState)
  if (selectedIds.size === 0) return null
  const fileMap = files && typeof files === 'object' ? files as Record<string, any> : {}
  for (const element of elements) {
    if (!element || typeof element !== 'object') continue
    const image = element as {id?: string; type?: string; fileId?: string | null; isDeleted?: boolean; x?: number; y?: number; width?: number; height?: number}
    if (image.type !== 'image' || image.isDeleted === true || !image.id || !selectedIds.has(image.id) || !image.fileId) continue
    const file = fileMap[image.fileId]
    const dataURL = typeof file?.dataURL === 'string' ? file.dataURL : ''
    if (!dataURL.startsWith('data:image/')) continue
    return {
      elementId: image.id,
      fileId: image.fileId,
      dataURL,
      x: finiteNumber(image.x, 0),
      y: finiteNumber(image.y, 0),
      width: Math.max(1, finiteNumber(image.width, 1)),
      height: Math.max(1, finiteNumber(image.height, 1)),
    }
  }
  return null
}

function firstWhiteboardImageFromScene(elements: readonly unknown[], files: unknown): SelectedWhiteboardImage | null {
  const fileMap = files && typeof files === 'object' ? files as Record<string, any> : {}
  for (const element of elements) {
    if (!element || typeof element !== 'object') continue
    const image = element as {id?: string; type?: string; fileId?: string | null; isDeleted?: boolean; x?: number; y?: number; width?: number; height?: number}
    if (image.type !== 'image' || image.isDeleted === true || !image.id || !image.fileId) continue
    const file = fileMap[image.fileId]
    const dataURL = typeof file?.dataURL === 'string' ? file.dataURL : ''
    if (!dataURL.startsWith('data:image/')) continue
    return {
      elementId: image.id,
      fileId: image.fileId,
      dataURL,
      x: finiteNumber(image.x, 0),
      y: finiteNumber(image.y, 0),
      width: Math.max(1, finiteNumber(image.width, 1)),
      height: Math.max(1, finiteNumber(image.height, 1)),
    }
  }
  return null
}

function removeOcrBlockElements(api: ExcalidrawImperativeAPI | null, persist: (sceneJson: string) => void) {
  if (!api) return
  const current = api.getSceneElements()
  const next = removeOcrBlockElementsFromList(current)
  if (next.length === current.length) return
  api.updateScene({elements: next as any})
  persistApiSceneSoon(api, persist)
}

function removeOcrPositionTextElements(api: ExcalidrawImperativeAPI | null, persist: (sceneJson: string) => void) {
  if (!api) return
  const current = api.getSceneElements()
  const next = removeOcrPositionTextElementsFromList(current)
  if (next.length === current.length) return
  api.updateScene({elements: next as any})
  persistApiSceneSoon(api, persist)
}

function removeOcrTranslationElements(api: ExcalidrawImperativeAPI | null, persist: (sceneJson: string) => void) {
  if (!api) return
  const current = api.getSceneElements()
  const next = removeOcrTranslationElementsFromList(current)
  if (next.length === current.length) return
  api.updateScene({elements: next as any})
  persistApiSceneSoon(api, persist)
}

function removeOcrBlockElementsFromList(elements: readonly unknown[]) {
  return elements.filter((element) => !isRecordingFreedomOcrElement(element, 'block'))
}

function removeOcrPositionTextElementsFromList(elements: readonly unknown[]) {
  return elements.filter((element) => !isRecordingFreedomOcrElement(element, 'position-text'))
}

function removeOcrTranslationElementsFromList(elements: readonly unknown[]) {
  return elements.filter((element) => !isRecordingFreedomOcrElement(element, 'translation'))
}

function isRecordingFreedomOcrElement(element: unknown, kind: RecordingFreedomOcrElementKind) {
  if (!element || typeof element !== 'object') return false
  const customData = (element as {customData?: {recordingFreedomOcr?: {kind?: string}}}).customData
  return customData?.recordingFreedomOcr?.kind === kind
}

function buildOcrBlockElements(result: OcrResult, anchor: SelectedWhiteboardImage) {
  const now = Date.now()
  return result.blocks.flatMap((block, index) => {
    const bounds = ocrBlockBounds(block)
    if (!bounds) return []
    const x = anchor.x + (bounds.x / Math.max(1, result.width)) * anchor.width
    const y = anchor.y + (bounds.y / Math.max(1, result.height)) * anchor.height
    const width = Math.max(4, (bounds.width / Math.max(1, result.width)) * anchor.width)
    const height = Math.max(4, (bounds.height / Math.max(1, result.height)) * anchor.height)
    return [{
      id: `rf-ocr-block-${safeElementId(result.id)}-${safeElementId(block.id || String(index))}-${now}-${index}`,
      type: 'rectangle',
      x,
      y,
      width,
      height,
      angle: 0,
      strokeColor: '#ef4444',
      backgroundColor: 'transparent',
      fillStyle: 'solid',
      strokeWidth: 2,
      strokeStyle: 'solid',
      roughness: 0,
      opacity: 100,
      groupIds: [],
      frameId: null,
      roundness: {type: 3},
      seed: 1000 + index,
      version: 1,
      versionNonce: 2000 + index,
      isDeleted: false,
      boundElements: null,
      updated: now,
      link: null,
      locked: false,
      customData: {
        recordingFreedomOcr: {
          kind: 'block',
          resultId: result.id,
          blockId: block.id,
          text: block.text,
        },
      },
    }]
  })
}

function buildOcrTranslationTextElements(result: OcrResult, translation: OcrTranslationResult | null, anchor: SelectedWhiteboardImage) {
  const translatedByBlockId = new Map((translation?.blocks ?? [])
    .map((block) => [block.blockId, block.translated.trim()] as const)
    .filter(([, translated]) => Boolean(translated)))
  const now = Date.now()
  return result.blocks.flatMap((block, index) => {
    const translated = translatedByBlockId.get(block.id)
    if (!translated) return []
    const bounds = ocrBlockBounds(block)
    if (!bounds) return []
    const x = anchor.x + (bounds.x / Math.max(1, result.width)) * anchor.width
    const y = anchor.y + (bounds.y / Math.max(1, result.height)) * anchor.height
    const width = Math.max(28, (bounds.width / Math.max(1, result.width)) * anchor.width)
    const height = Math.max(18, (bounds.height / Math.max(1, result.height)) * anchor.height)
    const fontSize = Math.max(12, Math.min(28, Math.round(height * 0.72)))
    return [{
      id: `rf-ocr-translation-${safeElementId(result.id)}-${safeElementId(block.id || String(index))}-${now}-${index}`,
      type: 'text',
      x,
      y,
      width,
      height,
      angle: 0,
      strokeColor: '#e0f2fe',
      backgroundColor: 'transparent',
      fillStyle: 'solid',
      strokeWidth: 1,
      strokeStyle: 'solid',
      roughness: 0,
      opacity: 100,
      groupIds: [],
      frameId: null,
      roundness: null,
      seed: 4000 + index,
      version: 1,
      versionNonce: 5000 + index,
      isDeleted: false,
      boundElements: null,
      updated: now,
      link: null,
      locked: false,
      text: translated,
      originalText: translated,
      fontSize,
      fontFamily: 1,
      textAlign: 'center',
      verticalAlign: 'middle',
      baseline: Math.round(fontSize * 1.18),
      containerId: null,
      lineHeight: 1.18,
      customData: {
        recordingFreedomOcr: {
          kind: 'translation',
          resultId: result.id,
          blockId: block.id,
          source: block.text,
        },
      },
    }]
  })
}

function buildOcrTextElement(result: OcrResult, text: string, anchor: SelectedWhiteboardImage | null) {
  const now = Date.now()
  const lines = text.split(/\r?\n/).filter((line) => line.trim() !== '')
  const fontSize = 20
  const lineHeight = 1.25
  const width = Math.max(180, Math.min(520, anchor?.width ?? 360))
  const height = Math.max(40, Math.ceil(lines.length * fontSize * lineHeight) + 12)
  return {
    id: `rf-ocr-text-${safeElementId(result.id)}-${now}`,
    type: 'text',
    x: anchor ? anchor.x : 0,
    y: anchor ? anchor.y + anchor.height + 18 : 0,
    width,
    height,
    angle: 0,
    strokeColor: '#f8fafc',
    backgroundColor: 'transparent',
    fillStyle: 'solid',
    strokeWidth: 1,
    strokeStyle: 'solid',
    roughness: 0,
    opacity: 100,
    groupIds: [],
    frameId: null,
    roundness: null,
    seed: 3001,
    version: 1,
    versionNonce: 3002,
    isDeleted: false,
    boundElements: null,
    updated: now,
    link: null,
    locked: false,
    text,
    originalText: text,
    fontSize,
    fontFamily: 1,
    textAlign: 'left',
    verticalAlign: 'top',
    baseline: Math.round(fontSize * lineHeight),
    containerId: null,
    lineHeight,
    customData: {
      recordingFreedomOcr: {
        kind: 'text',
        resultId: result.id,
      },
    },
  }
}

function ocrBlockBounds(block: OcrBlock) {
  if (block.box.length === 0) return null
  const xs = block.box.map((point) => point.x).filter(Number.isFinite)
  const ys = block.box.map((point) => point.y).filter(Number.isFinite)
  if (xs.length === 0 || ys.length === 0) return null
  const left = Math.min(...xs)
  const top = Math.min(...ys)
  const right = Math.max(...xs)
  const bottom = Math.max(...ys)
  if (right <= left || bottom <= top) return null
  return {x: left, y: top, width: right - left, height: bottom - top}
}

function safeElementId(value: string) {
  return value.replace(/[^a-zA-Z0-9_-]+/g, '-').slice(0, 80) || 'ocr'
}

function finiteNumber(value: unknown, fallback: number) {
  return typeof value === 'number' && Number.isFinite(value) ? value : fallback
}

function selectedElementIDSet(appState: unknown) {
  const state = appState && typeof appState === 'object' ? appState as {selectedElementIds?: unknown} : {}
  const selected = state.selectedElementIds
  if (!selected || typeof selected !== 'object') return new Set<string>()
  return new Set(Object.entries(selected as Record<string, unknown>)
    .filter(([, selected]) => selected === true)
    .map(([id]) => id))
}

function applyImportedSceneToApi(api: ExcalidrawImperativeAPI, scene: ReturnType<typeof screenshotScene>) {
  const files = Object.values(scene.files)
  if (files.length > 0) api.addFiles(files as any)
  api.updateScene({
    elements: scene.elements as any,
    appState: scene.appState as any,
  } as any)
}

function persistApiSceneSoon(api: ExcalidrawImperativeAPI, persist: (sceneJson: string) => void) {
  window.setTimeout(() => {
    try {
      const sceneJson = (serializeAsJSON as any)(api.getSceneElements(), api.getAppState(), api.getFiles(), 'local')
      persist(sceneJson)
    } catch (error) {
      console.error('Failed to serialize imported screenshot:', error)
    }
  }, 0)
}

function normalizeOpacity(value: unknown) {
  const numeric = typeof value === 'number' && Number.isFinite(value) ? value : 100
  if (numeric < 5) return 5
  if (numeric > 100) return 100
  return Math.round(numeric / 5) * 5
}

function safeParseScene(sceneJson: string) {
  try {
    return JSON.parse(sceneJson)
  } catch {
    return null
  }
}

function blobToDataURL(blob: Blob) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(blob)
  })
}

function imageDataURLToPNG(dataURL: string) {
  return new Promise<string>((resolve, reject) => {
    if (dataURL.startsWith(whiteboardPNGContentPrefix)) {
      resolve(dataURL)
      return
    }
    const image = new Image()
    image.onload = () => {
      const canvas = document.createElement('canvas')
      canvas.width = Math.max(1, image.naturalWidth || image.width)
      canvas.height = Math.max(1, image.naturalHeight || image.height)
      const context = canvas.getContext('2d')
      if (!context) {
        reject(new Error('failed to create image export canvas'))
        return
      }
      context.drawImage(image, 0, 0, canvas.width, canvas.height)
      resolve(canvas.toDataURL('image/png'))
    }
    image.onerror = () => reject(new Error('failed to load selected image'))
    image.src = dataURL
  })
}

export default WhiteboardWindow
