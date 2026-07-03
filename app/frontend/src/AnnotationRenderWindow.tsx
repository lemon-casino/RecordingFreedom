import {useEffect, useRef, useState} from 'react'
import {exportToBlob} from '@excalidraw/excalidraw'
import '@excalidraw/excalidraw/index.css'
import {claimAnnotationRenderJob, completeAnnotationRenderJob, logClientEvent, type AnnotationRenderJob} from './services/recorderBackend'

type RenderStatus = 'idle' | 'rendering' | 'failed'

function AnnotationRenderWindow() {
  const [status, setStatus] = useState<RenderStatus>('idle')
  const runningRef = useRef(false)
  const disposedRef = useRef(false)
  const timerRef = useRef<number | null>(null)

  useEffect(() => {
    document.body.classList.add('rf-annotation-render-window')
    return () => document.body.classList.remove('rf-annotation-render-window')
  }, [])

  useEffect(() => {
    disposedRef.current = false
    const pump = () => {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current)
        timerRef.current = null
      }
      void runRenderPump()
    }
    const schedule = (delay: number) => {
      if (disposedRef.current) return
      if (timerRef.current !== null) window.clearTimeout(timerRef.current)
      timerRef.current = window.setTimeout(pump, delay)
    }
    const runRenderPump = async () => {
      if (runningRef.current || disposedRef.current) return
      runningRef.current = true
      try {
        let renderedAny = false
        for (;;) {
          const claim = await claimAnnotationRenderJob()
          if (!claim.available || !claim.job) break
          renderedAny = true
          setStatus('rendering')
          await renderAndCompleteJob(claim.job)
        }
        setStatus('idle')
        schedule(renderedAny ? 120 : 1000)
      } catch (error) {
        setStatus('failed')
        void logClientEvent('annotation-renderer', 'pump-error', {}, readableError(error))
        schedule(1500)
      } finally {
        runningRef.current = false
      }
    }
    window.addEventListener('rf-annotation-renderer-wake', pump)
    pump()
    return () => {
      disposedRef.current = true
      window.removeEventListener('rf-annotation-renderer-wake', pump)
      if (timerRef.current !== null) window.clearTimeout(timerRef.current)
    }
  }, [])

  return (
    <main className="annotation-renderer-shell" aria-label="Annotation renderer">
      <span>{status}</span>
    </main>
  )
}

async function renderAndCompleteJob(job: AnnotationRenderJob) {
  try {
    const dataUrl = await renderAnnotationJob(job)
    await completeAnnotationRenderJob({id: job.id, dataUrl})
  } catch (error) {
    const message = readableError(error)
    await completeAnnotationRenderJob({id: job.id, error: message})
    void logClientEvent('annotation-renderer', 'job-error', {
      id: job.id,
      scene: job.relativeScenePath,
    }, message)
  }
}

async function renderAnnotationJob(job: AnnotationRenderJob) {
  const width = Math.round(job.canvasWidth)
  const height = Math.round(job.canvasHeight)
  if (width <= 0 || height <= 0) {
    throw new Error(`Invalid annotation canvas size ${job.canvasWidth}x${job.canvasHeight}`)
  }
  const scene = parseAnnotationScene(job.sceneJson)
  const elements = [...scene.elements, transparentBoundsElement(width, height, job.index)]
  const blob = await exportToBlob({
    elements,
    appState: {
      ...(scene.appState ?? {}),
      exportBackground: false,
      exportScale: 1,
      viewBackgroundColor: 'transparent',
    },
    files: scene.files,
    mimeType: 'image/png',
    exportPadding: 0,
    getDimensions: () => ({width, height, scale: 1}),
  } as any)
  return blobToDataURL(blob)
}

function parseAnnotationScene(sceneJson: string): {elements: any[]; appState: Record<string, unknown>; files: Record<string, unknown>} {
  const parsed = JSON.parse(sceneJson)
  return {
    elements: Array.isArray(parsed?.elements) ? parsed.elements : [],
    appState: parsed?.appState && typeof parsed.appState === 'object' ? parsed.appState : {},
    files: parsed?.files && typeof parsed.files === 'object' ? parsed.files : {},
  }
}

function transparentBoundsElement(width: number, height: number, index: number) {
  return {
    id: `rf-annotation-export-bounds-${index}`,
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
    reader.onerror = () => reject(reader.error ?? new Error('Failed to read rendered annotation blob'))
    reader.readAsDataURL(blob)
  })
}

function readableError(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}

export default AnnotationRenderWindow
