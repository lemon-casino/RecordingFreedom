import {
  ArrowUpRight,
  Circle,
  Eraser,
  Minus,
  MousePointer2,
  PenLine,
  RectangleHorizontal,
  RefreshCcw,
  Save,
  Type,
  Undo2,
  X,
} from 'lucide-react'
import {useCallback, useEffect, useLayoutEffect, useRef, useState} from 'react'
import {Excalidraw, exportToBlob, serializeAsJSON} from '@excalidraw/excalidraw'
import type {ExcalidrawImperativeAPI} from '@excalidraw/excalidraw/types'
import '@excalidraw/excalidraw/index.css'
import {copyByLocale} from './i18n'
import {defaultSettings, normalizeLocale, normalizeTheme, type AppSettings, type LocaleCode, type ThemeCode, type WhiteboardTool} from './services/mockBackend'
import {hideAnnotationOverlay, loadAnnotationCapture, loadSettings, patchWhiteboardSettings, reselectAnnotationRegion, saveAnnotationCapture, setAnnotationOverlayHitRegions, subscribeSettingsChanged, type AnnotationOverlayState, type CapsuleWindowHitRegion} from './services/recorderBackend'

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

function AnnotationOverlayWindow() {
  const [overlayState, setOverlayState] = useState<AnnotationOverlayState | null>(() => (window as any).__RF_ANNOTATION_OVERLAY__ ?? null)
  const [locale, setLocale] = useState<LocaleCode>('zh-CN')
  const [theme, setTheme] = useState<ThemeCode>('night-teal')
  const [initialData, setInitialData] = useState<any | null>(null)
  const [activeTool, setActiveToolState] = useState<WhiteboardTool>('freedraw')
  const [strokeColor, setStrokeColor] = useState('#ef4444')
  const [strokeWidth, setStrokeWidth] = useState(defaultSettings.whiteboard.lastStrokeWidth)
  const [opacity, setOpacity] = useState(100)
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const apiRef = useRef<ExcalidrawImperativeAPI | null>(null)
  const lastSceneRef = useRef('')
  const saveTimerRef = useRef<number | null>(null)
  const elementSignatureRef = useRef<Map<string, string>>(new Map())
  const pendingElementEventsRef = useRef<Map<string, AnnotationElementEvent>>(new Map())
  const clientSequenceRef = useRef(0)
  const capsuleRef = useRef<HTMLElement | null>(null)
  const canvasRef = useRef<HTMLElement | null>(null)
  const copy = copyByLocale[locale]
  const canvasReceivesInput = annotationCanvasReceivesInput(activeTool)
  const overlayKey = annotationOverlayKey(overlayState)

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
    let cancelled = false
    setInitialData(null)
    void Promise.all([loadSettings(), loadAnnotationCapture()])
      .then(([settings, scene]) => {
        if (cancelled) return
        applySettings(settings)
        const parsed = scene.available && scene.sceneJson ? safeParseScene(scene.sceneJson) : null
        const nextInitialData = parsed ?? defaultAnnotationScene(settings)
        resetAnnotationElementTracking(nextInitialData)
        setInitialData(nextInitialData)
        lastSceneRef.current = scene.sceneJson ?? ''
      })
      .catch((error) => {
        console.info('Using annotation overlay load fallback:', error)
        if (!cancelled) {
          const nextInitialData = defaultAnnotationScene(defaultSettings)
          resetAnnotationElementTracking(nextInitialData)
          setInitialData(nextInitialData)
          lastSceneRef.current = ''
        }
      })
    return () => {
      cancelled = true
    }
  }, [overlayKey])

  useEffect(() => {
    document.documentElement.lang = locale
    document.documentElement.dataset.theme = theme
  }, [locale, theme])

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
    setActiveToolState(settings.whiteboard.lastTool)
    setStrokeColor(settings.whiteboard.lastStrokeColor || '#ef4444')
    setStrokeWidth(settings.whiteboard.lastStrokeWidth)
    setOpacity(normalizeOpacity(settings.whiteboard.lastOpacity))
  }

  const applyStyle = useCallback((color: string, width: typeof strokeWidth, nextOpacity: number) => {
    apiRef.current?.updateScene({
      appState: {
        currentItemStrokeColor: color,
        currentItemStrokeWidthKey: width,
        currentItemOpacity: nextOpacity,
        viewBackgroundColor: 'transparent',
      },
    } as any)
  }, [])

  useEffect(() => {
    applyStyle(strokeColor, strokeWidth, opacity)
  }, [applyStyle, opacity, strokeColor, strokeWidth])

  const setActiveTool = (tool: WhiteboardTool, persist = true) => {
    setActiveToolState(tool)
    apiRef.current?.setActiveTool({type: tool} as any)
    if (persist) {
      void patchWhiteboardSettings({lastTool: tool, lastMode: 'annotation'})
    }
  }

  const scheduleSave = useCallback((sceneJson: string, hasElements: boolean) => {
    if (!hasElements && !lastSceneRef.current) return
    lastSceneRef.current = sceneJson
    setDirty(true)
    if (saveTimerRef.current !== null) window.clearTimeout(saveTimerRef.current)
    saveTimerRef.current = window.setTimeout(() => {
      void saveCurrentAnnotation()
    }, 700)
  }, [])

  const saveCurrentAnnotation = async () => {
    const api = apiRef.current
    if (!api) return
    setSaving(true)
    try {
      const sceneElements = api.getSceneElements()
      const sceneJson = (serializeAsJSON as any)(sceneElements, api.getAppState(), api.getFiles(), 'local')
      const canvasSize = annotationSnapshotCanvasSize(overlayState)
      const blob = await exportToBlob({
        elements: annotationSnapshotExportElements(sceneElements, canvasSize.width, canvasSize.height),
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
      const snapshotDataUrl = await blobToDataURL(blob)
      const eventsJsonl = pendingAnnotationEventsJSONL()
      await saveAnnotationCapture({sceneJson, snapshotDataUrl, eventsJsonl})
      pendingElementEventsRef.current.clear()
      lastSceneRef.current = sceneJson
      setDirty(false)
    } catch (error) {
      console.error('Failed to save annotation capture:', error)
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
    lastSceneRef.current = ''
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
    window.setTimeout(() => applyStyle(strokeColor, strokeWidth, opacity), 0)
  }

  const reselectRegion = async () => {
    try {
      resetLocalAnnotationScene()
      await reselectAnnotationRegion()
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

  if (!initialData) {
    return (
      <main className="annotation-overlay-shell" data-theme={theme}>
        <section className="annotation-capsule" aria-label={copy.whiteboard.title}>
          <span className="annotation-capsule-title">{copy.whiteboard.loading}</span>
        </section>
      </main>
    )
  }

  return (
    <main className={`annotation-overlay-shell ${canvasReceivesInput ? 'is-drawing' : 'is-pass-through'}`} data-theme={theme}>
      <section ref={capsuleRef} className="annotation-capsule" aria-label={copy.whiteboard.title}>
        <span className="annotation-capsule-title">{copy.whiteboard.open}</span>
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
        <button type="button" aria-label={copy.whiteboard.undo} title={copy.whiteboard.undo} onClick={undoAnnotationStep}>
          <Undo2 size={16} />
        </button>
        <button type="button" aria-label={copy.whiteboard.reselectRegion} title={copy.whiteboard.reselectRegion} onClick={() => void reselectRegion()}>
          <RefreshCcw size={16} />
        </button>
        <button type="button" aria-label={copy.whiteboard.save} title={copy.whiteboard.save} onClick={() => void saveCurrentAnnotation()}>
          <Save size={16} />
          <span>{saving ? copy.whiteboard.saved : dirty ? copy.whiteboard.unsaved : copy.whiteboard.ready}</span>
        </button>
        <button type="button" aria-label={copy.whiteboard.close} title={copy.whiteboard.close} onClick={() => void hideAnnotationOverlay()}>
          <X size={17} />
        </button>
      </section>
      <section
        ref={canvasRef}
        className="annotation-overlay-canvas"
        aria-label={copy.whiteboard.title}
        style={{
          width: overlayState?.canvasBounds.width || window.innerWidth,
          height: overlayState?.canvasBounds.height || window.innerHeight,
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
                applyStyle(strokeColor, strokeWidth, opacity)
              }, 0)
            }
          }}
          onChange={((elements: readonly unknown[], appState: unknown, files: unknown) => {
            try {
              trackAnnotationElementEvents(elements)
              const sceneJson = (serializeAsJSON as any)(elements, appState, files, 'local')
              scheduleSave(sceneJson, elements.length > 0)
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
      </section>
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
    },
    elements: [],
    files: {},
  }
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
  ].join(':')
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

function blobToDataURL(blob: Blob) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error)
    reader.readAsDataURL(blob)
  })
}

export default AnnotationOverlayWindow
