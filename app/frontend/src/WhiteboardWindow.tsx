import {
  ArrowUpRight,
  Circle,
  Download,
  Eraser,
  Hand,
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
import {useCallback, useEffect, useMemo, useRef, useState} from 'react'
import {Excalidraw, exportToBlob, exportToSvg, serializeAsJSON} from '@excalidraw/excalidraw'
import type {ExcalidrawImperativeAPI} from '@excalidraw/excalidraw/types'
import '@excalidraw/excalidraw/index.css'
import {copyByLocale} from './i18n'
import {defaultSettings, normalizeLocale, normalizeTheme, type AppSettings, type LocaleCode, type ThemeCode, type WhiteboardStrokeWidth, type WhiteboardTool} from './services/mockBackend'
import {consumeScreenshotWhiteboardContext, hideWhiteboardWindow, loadSettings, loadWhiteboardScene, patchWhiteboardSettings, saveWhiteboardExport, saveWhiteboardScene, subscribeScreenshotWhiteboardContext, subscribeSettingsChanged, type ScreenshotWhiteboardContext} from './services/recorderBackend'

type SaveState = 'ready' | 'dirty' | 'saving' | 'saved' | 'failed'
type WhiteboardWindowGlobal = Window & {__RF_SCREENSHOT_WHITEBOARD__?: ScreenshotWhiteboardContext}

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
  const apiRef = useRef<ExcalidrawImperativeAPI | null>(null)
  const settingsRef = useRef<AppSettings>(defaultSettings)
  const pendingImportedSceneRef = useRef<ReturnType<typeof screenshotScene> | null>(null)
  const lastSceneRef = useRef('')
  const lastScreenshotImportKeyRef = useRef('')
  const screenshotImportGuardUntilRef = useRef(0)
  const saveTimerRef = useRef<number | null>(null)
  const clearTimerRef = useRef<number | null>(null)
  const copy = copyByLocale[locale]

  const excalidrawTheme = 'dark'

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
              if (!cancelled) setStatusText(saved.updatedAt ? `${copy.whiteboard.saved} · ${saved.scenePath}` : copy.whiteboard.saved)
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
    setStatusText(copy.whiteboard.unsaved)
    if (saveTimerRef.current !== null) window.clearTimeout(saveTimerRef.current)
    saveTimerRef.current = window.setTimeout(() => {
      void saveScene(sceneJson)
    }, 700)
  }, [copy.whiteboard.unsaved])

  const saveScene = async (sceneJson = lastSceneRef.current) => {
    if (!sceneJson.trim()) return
    setSaveState('saving')
    try {
      const saved = await saveWhiteboardScene(sceneJson)
      setSaveState('saved')
      setStatusText(saved.updatedAt ? `${copy.whiteboard.saved} · ${saved.scenePath}` : copy.whiteboard.saved)
    } catch (error) {
      console.error('Failed to save whiteboard scene:', error)
      setSaveState('failed')
      setStatusText(copy.whiteboard.saveFailed)
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
    setStatusText(copy.whiteboard.unsaved)
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
      if (Date.now() < screenshotImportGuardUntilRef.current && !sceneHasImageElement(elements)) return
      const sceneJson = (serializeAsJSON as any)(elements, appState, files, 'local')
      scheduleSave(sceneJson)
    } catch (error) {
      console.error('Failed to serialize whiteboard scene:', error)
    }
  }, [scheduleSave])

  const saveNow = () => {
    void saveScene()
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
      <section className="whiteboard-canvas" aria-label={copy.whiteboard.title}>
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
      </section>
      <footer className={`whiteboard-status ${saveState}`}>
        <span>{statusText || copy.whiteboard.ready}</span>
      </footer>
    </main>
  )
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

export default WhiteboardWindow
