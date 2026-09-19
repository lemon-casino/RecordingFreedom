import {applyTheme} from './components/themeOptions'
import {useEffect, useRef, useState, type CSSProperties, type PointerEvent as ReactPointerEvent} from 'react'
import {copyByLocale} from './i18n'
import {clampNumber, normalizeLocale, normalizePipConfig, normalizeTheme, pipMaximumScale, type LocaleCode, type PIPConfig, type PIPShape, type ThemeCode} from './services/mockBackend'
import {
  hidePipOverlay,
  logClientEvent,
  loadSettings,
  readPipPreviewImage,
  showPipOverlay,
  subscribeSettingsChanged,
  updatePipOverlay,
  type PIPOverlayCamera,
  type PIPOverlayState } from './services/recorderBackend'
import {readableError} from './services/recorderBackend'
import {Camera, CircleDot, FlipHorizontal, Move, Square, X} from 'lucide-react'

export type PIPEditAction = 'move' | 'n' | 'e' | 's' | 'w' | 'ne' | 'nw' | 'se' | 'sw'

const pipResizeActions: PIPEditAction[] = ['n', 'e', 's', 'w', 'ne', 'nw', 'se', 'sw']
const pipMinimumContentSize = 24

type PIPOverlayWindowGlobal = Window & {
  __RF_PIP_OVERLAY__?: PIPOverlayState
  __RF_STOP_PIP_CAMERA__?: () => void
  __RF_PIP_STOP_TOKEN__?: number
}

function PIPOverlayWindow() {
  const pipWindow = window as PIPOverlayWindowGlobal
  const [overlayState, setOverlayState] = useState<PIPOverlayState | undefined>(pipWindow.__RF_PIP_OVERLAY__)
  const overlayStateRef = useRef<PIPOverlayState | undefined>(overlayState)
  const editRef = useRef<{
    action: PIPEditAction
    startX: number
    startY: number
    content: {x: number; y: number; width: number; height: number}
    state: PIPOverlayState
    latest: PIPConfig
  } | null>(null)
  const pendingPreviewRef = useRef<PIPConfig | null>(null)
  const previewFrameRef = useRef<number | null>(null)
  const videoRef = useRef<HTMLVideoElement | null>(null)
  const activeCameraStreamRef = useRef<MediaStream | null>(null)
  const cameraStreamsRef = useRef<Set<MediaStream>>(new Set())
  const cameraRequestTokenRef = useRef(0)
  const overlayOperationIdRef = useRef(overlayState?.clientOperationId ?? 0)
  const overlayClosedRef = useRef(!overlayState || overlayState.config.preset === 'off')
  const previewImageModifiedRef = useRef(0)
  const previewImageDataUrlRef = useRef<string | null>(null)
  const previewImageReadyLoggedRef = useRef(false)
  const previewImageWaitingLoggedRef = useRef(false)
  const [cameraReady, setCameraReady] = useState(false)
  const [cameraError, setCameraError] = useState<string | null>(null)
  const [previewImageDataUrl, setPreviewImageDataUrl] = useState<string | null>(null)
  const [overlayLocale, setOverlayLocale] = useState<LocaleCode>(navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en')
  const [overlayTheme, setOverlayTheme] = useState<ThemeCode>('night-teal')
  const copy = copyByLocale[overlayLocale]
  const logPipCameraEvent = (event: string, fields: Record<string, unknown> = {}) => {
    void logClientEvent('pip-camera', event, fields)
  }

  const pipStopTokenChanged = (stopToken: number) => (pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0) !== stopToken

  const nextPipOverlayOperationId = () => {
    overlayOperationIdRef.current += 1
    return overlayOperationIdRef.current
  }

  const cancelPendingPipOverlayEdits = () => {
    if (previewFrameRef.current !== null) {
      window.cancelAnimationFrame(previewFrameRef.current)
      previewFrameRef.current = null
    }
    pendingPreviewRef.current = null
    editRef.current = null
  }

  const pipOverlayStateIsStale = (state: PIPOverlayState | undefined) => {
    const operationId = state?.clientOperationId ?? 0
    return operationId > 0 && operationId < overlayOperationIdRef.current
  }

  const setPipPreviewImage = (dataUrl: string | null) => {
    previewImageDataUrlRef.current = dataUrl
    setPreviewImageDataUrl(dataUrl)
  }

  const clearPipPreviewImage = () => {
    previewImageModifiedRef.current = 0
    previewImageReadyLoggedRef.current = false
    previewImageWaitingLoggedRef.current = false
    setPipPreviewImage(null)
  }

  const trackPipCameraStream = (stream: MediaStream, requestToken: number, stopToken: number, isCancelled: () => boolean) => {
    cameraStreamsRef.current.add(stream)
    logPipCameraEvent('stream-opened', {
      requestToken,
      stopToken,
      tracks: describeMediaStream(stream) })
    if (isCancelled() || cameraRequestTokenRef.current !== requestToken || pipStopTokenChanged(stopToken)) {
      logPipCameraEvent('stream-opened-cancelled', {requestToken, stopToken})
      stopAndForgetPipCameraStream(stream)
    }
  }

  const stopAndForgetPipCameraStream = (stream: MediaStream) => {
    logPipCameraEvent('stream-stop', {tracks: describeMediaStream(stream)})
    cameraStreamsRef.current.delete(stream)
    stopMediaStream(stream)
  }

  const stopActivePipCameraStream = () => {
    if (videoRef.current) {
      videoRef.current.pause()
      videoRef.current.srcObject = null
      videoRef.current.removeAttribute('src')
      videoRef.current.load()
    }
    activeCameraStreamRef.current = null
    const streams = Array.from(cameraStreamsRef.current)
    cameraStreamsRef.current.clear()
    if (streams.length > 0) {
      logPipCameraEvent('stop-active-streams', {count: streams.length})
    }
    streams.forEach(stopMediaStream)
  }

  const cancelPipCameraStream = () => {
    logPipCameraEvent('cancel', {
      nextStopToken: (pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0) + 1,
      nextRequestToken: cameraRequestTokenRef.current + 1 })
    pipWindow.__RF_PIP_STOP_TOKEN__ = (pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0) + 1
    cameraRequestTokenRef.current += 1
    stopActivePipCameraStream()
    clearPipPreviewImage()
    setCameraReady(false)
  }

  useEffect(() => {
    overlayStateRef.current = overlayState
  }, [overlayState])

  useEffect(() => {
    document.body.classList.add('rf-pip-overlay-window')
    pipWindow.__RF_PIP_STOP_TOKEN__ = pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0
    pipWindow.__RF_STOP_PIP_CAMERA__ = cancelPipCameraStream
    const stopOnWindowLifecycle = () => cancelPipCameraStream()
    const stopOnHidden = () => {
      logPipCameraEvent('visibility-change', {visibilityState: document.visibilityState})
    }
    window.addEventListener('pagehide', stopOnWindowLifecycle)
    window.addEventListener('beforeunload', stopOnWindowLifecycle)
    document.addEventListener('visibilitychange', stopOnHidden)
    return () => {
      cancelPipCameraStream()
      window.removeEventListener('pagehide', stopOnWindowLifecycle)
      window.removeEventListener('beforeunload', stopOnWindowLifecycle)
      document.removeEventListener('visibilitychange', stopOnHidden)
      delete pipWindow.__RF_STOP_PIP_CAMERA__
      document.body.classList.remove('rf-pip-overlay-window')
    }
  }, [])

  useEffect(() => {
    void loadSettings()
      .then((settings) => {
        setOverlayLocale(normalizeLocale(settings.locale))
        setOverlayTheme(normalizeTheme(settings.window.theme))
      })
      .catch((error) => console.info('Using PIP overlay language fallback:', error))
  }, [])

  useEffect(() => subscribeSettingsChanged((settings) => {
    setOverlayLocale(normalizeLocale(settings.locale))
    setOverlayTheme(normalizeTheme(settings.window.theme))
  }), [])

  useEffect(() => {
    applyTheme(overlayTheme)
  }, [overlayTheme])

  useEffect(() => {
    const onState = (event: Event) => {
      const next = (event as CustomEvent<PIPOverlayState>).detail
      if (next) {
        if (pipOverlayStateIsStale(next)) {
          logPipCameraEvent('overlay-state-stale', {
            incomingOperationId: next.clientOperationId ?? 0,
            currentOperationId: overlayOperationIdRef.current,
            preset: next.config.preset })
          if (overlayClosedRef.current && next.config.preset !== 'off') {
            cancelPipCameraStream()
            void hidePipOverlay()
          }
          return
        }
        if (next.clientOperationId && next.clientOperationId > overlayOperationIdRef.current) {
          overlayOperationIdRef.current = next.clientOperationId
        }
        if (next.config.preset === 'off') {
          logPipCameraEvent('overlay-state-off')
          overlayClosedRef.current = true
          cancelPipCameraStream()
          overlayStateRef.current = undefined
          setOverlayState(undefined)
          setCameraError(null)
          return
        }
        logPipCameraEvent('overlay-state', {
          mode: next.mode,
          preset: next.config.preset,
          cameraId: next.camera?.deviceId ?? '',
          cameraName: next.camera?.name ?? next.cameraName ?? '',
          clientOperationId: next.clientOperationId ?? 0 })
        overlayClosedRef.current = false
        overlayStateRef.current = next
        setOverlayState(next)
      }
    }
    window.addEventListener('rf-pip-overlay', onState)
    return () => window.removeEventListener('rf-pip-overlay', onState)
  }, [])

  useEffect(() => {
    if (!overlayState || overlayState.config.preset === 'off') {
      cancelPipCameraStream()
      setCameraError(null)
      return
    }
    const recordingPreviewImagePath = overlayState.mode === 'recording' ? overlayState.previewImagePath?.trim() ?? '' : ''
    if (recordingPreviewImagePath) {
      const requestToken = cameraRequestTokenRef.current + 1
      cameraRequestTokenRef.current = requestToken
      const stopToken = pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0
      let cancelled = false
      let timer: number | null = null
      const startedAt = Date.now()
      stopActivePipCameraStream()
      clearPipPreviewImage()
      setCameraReady(false)
      setCameraError(null)
      const isCancelled = () => cancelled || cameraRequestTokenRef.current !== requestToken || pipStopTokenChanged(stopToken)
      logPipCameraEvent('preview-image-poll-start', {
        requestToken,
        stopToken,
        mode: overlayState.mode })
      const scheduleNextPoll = () => {
        if (!isCancelled()) {
          timer = window.setTimeout(pollPreviewImage, 125)
        }
      }
      async function pollPreviewImage() {
        if (isCancelled()) return
        try {
          const result = await readPipPreviewImage(recordingPreviewImagePath, previewImageModifiedRef.current)
          if (isCancelled()) return
          if (result.modifiedUnixNano && result.modifiedUnixNano > previewImageModifiedRef.current) {
            previewImageModifiedRef.current = result.modifiedUnixNano
          }
          if (result.available && result.dataUrl) {
            setPipPreviewImage(result.dataUrl)
            setCameraReady(true)
            setCameraError(null)
            if (!previewImageReadyLoggedRef.current) {
              previewImageReadyLoggedRef.current = true
              logPipCameraEvent('preview-image-ready', {
                requestToken,
                modifiedUnixNano: result.modifiedUnixNano ?? 0 })
            }
          } else if (!previewImageWaitingLoggedRef.current && Date.now() - startedAt > 1800) {
            previewImageWaitingLoggedRef.current = true
            logPipCameraEvent('preview-image-waiting', {
              requestToken,
              path: recordingPreviewImagePath })
          }
        } catch (error) {
          if (!isCancelled()) {
            logPipCameraEvent('preview-image-error', {requestToken, error: readableError(error)})
            setCameraError(readableError(error))
            setCameraReady(Boolean(previewImageDataUrlRef.current))
          }
        } finally {
          scheduleNextPoll()
        }
      }
      void pollPreviewImage()
      return () => {
        cancelled = true
        if (timer !== null) {
          window.clearTimeout(timer)
        }
        logPipCameraEvent('preview-image-cleanup', {requestToken})
        if (cameraRequestTokenRef.current === requestToken) {
          cameraRequestTokenRef.current += 1
        }
      }
    }
    if (overlayState.mode === 'recording') {
      cancelPipCameraStream()
      setCameraError(null)
      logPipCameraEvent('preview-image-path-missing', {mode: overlayState.mode})
      return
    }
    clearPipPreviewImage()
    if (!navigator.mediaDevices?.getUserMedia) {
      cancelPipCameraStream()
      setCameraReady(false)
      setCameraError('media-devices-unavailable')
      return
    }
    const cameraTarget = overlayState.camera ?? (overlayState.cameraName ? {name: overlayState.cameraName} : undefined)
    const requestToken = cameraRequestTokenRef.current + 1
    cameraRequestTokenRef.current = requestToken
    const stopToken = pipWindow.__RF_PIP_STOP_TOKEN__ ?? 0
    let cancelled = false
    stopActivePipCameraStream()
    setCameraReady(false)
    setCameraError(null)
    const isCancelled = () => cancelled || cameraRequestTokenRef.current !== requestToken || pipStopTokenChanged(stopToken)
    logPipCameraEvent('stream-request-start', {
      requestToken,
      stopToken,
      targetDeviceId: cameraTarget?.deviceId ?? '',
      targetNativeId: cameraTarget?.nativeId ?? '',
      targetName: cameraTarget?.name ?? '',
      mode: overlayState.mode })
    void openPipCameraStream(
      cameraTarget,
      (stream) => trackPipCameraStream(stream, requestToken, stopToken, isCancelled),
      isCancelled,
    ).then(async (nextStream) => {
      if (isCancelled()) {
        logPipCameraEvent('stream-request-cancelled-before-attach', {requestToken})
        stopAndForgetPipCameraStream(nextStream)
        return
      }
      activeCameraStreamRef.current = nextStream
      const video = videoRef.current
      if (video) {
        video.srcObject = nextStream
        try {
          await video.play()
          logPipCameraEvent('video-play-ready', {requestToken, mode: overlayState.mode})
        } catch (error) {
          logPipCameraEvent('video-play-deferred', {requestToken, mode: overlayState.mode, error: readableError(error)})
          console.info('PIP preview video play was deferred:', error)
        }
      }
      if (isCancelled()) {
        logPipCameraEvent('stream-request-cancelled-after-attach', {requestToken})
        if (activeCameraStreamRef.current === nextStream) activeCameraStreamRef.current = null
        stopAndForgetPipCameraStream(nextStream)
        return
      }
      logPipCameraEvent('stream-ready', {requestToken, tracks: describeMediaStream(nextStream)})
      setCameraReady(true)
      setCameraError(null)
    }).catch((error) => {
      if (isCancelled()) return
      logPipCameraEvent('stream-error', {requestToken, error: readableError(error)})
      setCameraReady(false)
      setCameraError(readableError(error))
    })
    return () => {
      cancelled = true
      logPipCameraEvent('stream-effect-cleanup', {requestToken})
      if (cameraRequestTokenRef.current === requestToken) {
        cameraRequestTokenRef.current += 1
      }
      stopActivePipCameraStream()
      setCameraReady(false)
    }
  }, [overlayState?.camera?.deviceId, overlayState?.camera?.name, overlayState?.camera?.nativeId, overlayState?.cameraName, overlayState?.config.preset, overlayState?.mode, overlayState?.previewImagePath])

  useEffect(() => () => {
    if (previewFrameRef.current !== null) {
      window.cancelAnimationFrame(previewFrameRef.current)
      previewFrameRef.current = null
    }
  }, [])

  const overlayConfig = async (config: PIPConfig, commit: boolean, operationId = nextPipOverlayOperationId()) => {
    const state = overlayStateRef.current
    const mode = state?.mode ?? 'edit'
    const cameraTarget = state?.camera ?? (state?.cameraName ? {name: state.cameraName} : '')
    const previewImagePath = state?.previewImagePath ?? ''
    const nextConfig = normalizePipConfig(config, config.preset)
    if (nextConfig.preset !== 'off') {
      overlayClosedRef.current = false
    }
    const nextState = commit
      ? await updatePipOverlay(nextConfig, mode, cameraTarget, previewImagePath, operationId)
      : await showPipOverlay(nextConfig, mode, cameraTarget, previewImagePath, operationId)
    if (operationId !== overlayOperationIdRef.current || overlayClosedRef.current || pipOverlayStateIsStale(nextState)) {
      logPipCameraEvent('overlay-config-stale', {
        operationId,
        currentOperationId: overlayOperationIdRef.current,
        closed: overlayClosedRef.current,
        preset: nextState.config.preset })
      if (overlayClosedRef.current && nextState.config.preset !== 'off') {
        cancelPipCameraStream()
        void hidePipOverlay()
      }
      return
    }
    overlayClosedRef.current = nextState.config.preset === 'off'
    overlayStateRef.current = nextState
    setOverlayState(nextState)
  }

  const previewConfig = (config: PIPConfig) => {
    if (overlayClosedRef.current) return
    pendingPreviewRef.current = config
    if (previewFrameRef.current !== null) return
    previewFrameRef.current = window.requestAnimationFrame(() => {
      previewFrameRef.current = null
      const pending = pendingPreviewRef.current
      pendingPreviewRef.current = null
      if (pending && !overlayClosedRef.current) {
        const operationId = nextPipOverlayOperationId()
        void overlayConfig(pending, false, operationId).catch((error) => console.info('PIP preview update failed:', error))
      }
    })
  }

  const commitConfig = async (config: PIPConfig) => {
    cancelPendingPipOverlayEdits()
    const operationId = nextPipOverlayOperationId()
    await overlayConfig(config, true, operationId)
  }

  const beginEdit = (event: ReactPointerEvent<HTMLElement>, action: PIPEditAction) => {
    const state = overlayStateRef.current
    if (!state || state.config.preset === 'off' || event.button !== 0) return
    event.preventDefault()
    event.stopPropagation()
    event.currentTarget.setPointerCapture(event.pointerId)
    editRef.current = {
      action,
      startX: event.screenX,
      startY: event.screenY,
      content: pipAbsoluteContentRect(state),
      state,
      latest: state.config }
  }

  const updateEdit = (event: ReactPointerEvent<HTMLElement>) => {
    const edit = editRef.current
    if (!edit) return
    event.preventDefault()
    const rect = resizePipRect(edit.state, edit.content, edit.action, event.screenX - edit.startX, event.screenY - edit.startY)
    const next = pipConfigFromAbsoluteRect(edit.state, rect)
    edit.latest = next
    previewConfig(next)
  }

  const completeEdit = (event: ReactPointerEvent<HTMLElement>) => {
    const edit = editRef.current
    if (!edit) return
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    editRef.current = null
    void commitConfig(edit.latest).catch((error) => console.info('PIP commit failed:', error))
  }

  const updateShape = (shape: PIPShape) => {
    const state = overlayStateRef.current
    if (!state) return
    void commitConfig({...state.config, shape})
  }

  const toggleMirror = () => {
    const state = overlayStateRef.current
    if (!state) return
    void commitConfig({...state.config, mirror: !state.config.mirror})
  }

  const closePip = () => {
    const state = overlayStateRef.current ?? overlayState
    const operationId = nextPipOverlayOperationId()
    overlayClosedRef.current = true
    cancelPendingPipOverlayEdits()
    cancelPipCameraStream()
    overlayStateRef.current = undefined
    setOverlayState(undefined)
    setCameraError(null)
    if (!state) {
      void hidePipOverlay()
      return
    }
    const cameraTarget = state.camera ?? (state.cameraName ? {name: state.cameraName} : '')
    void updatePipOverlay({...state.config, preset: 'off'}, state.mode, cameraTarget, state.previewImagePath ?? '', operationId)
      .catch((error) => console.info('PIP close failed:', error))
      .finally(() => {
        if (operationId !== overlayOperationIdRef.current && !overlayClosedRef.current) return
        cancelPipCameraStream()
        void hidePipOverlay()
        for (const delay of [120, 500]) {
          window.setTimeout(() => {
            if (operationId !== overlayOperationIdRef.current || !overlayClosedRef.current) return
            cancelPipCameraStream()
            overlayStateRef.current = undefined
            setOverlayState(undefined)
            void hidePipOverlay()
          }, delay)
        }
      })
  }

  const content = overlayState?.config.preset !== 'off' ? overlayState?.contentBounds : undefined
  const isRecordingPipOverlay = overlayState?.mode === 'recording'
  const usesBackendPreviewImage = isRecordingPipOverlay && Boolean(overlayState.previewImagePath?.trim())
  const cameraName = overlayState?.camera?.name || overlayState?.cameraName || copy.panels.cameraSidecar
  const showCameraPlaceholder = !isRecordingPipOverlay && !cameraReady
  const cameraPlaceholderTitle = cameraError
      ? copy.pipOverlay.cameraUnavailable
      : copy.pipOverlay.cameraPreparing
  const cameraPlaceholderDetail = cameraError || cameraName
  const featherPx = content && overlayState ? Math.max(2, Math.round(content.width * overlayState.config.edgeFeather)) : 12
  const frameStyle = content ? {
    left: content.x,
    top: content.y,
    width: content.width,
    height: content.height,
    '--pip-feather': `${featherPx}px` } as CSSProperties : undefined

  return (
    <main
      className={`pip-overlay-shell ${overlayState?.mode ?? 'edit'}`}
      aria-label={copy.pipOverlay.label}
      onPointerMove={updateEdit}
      onPointerUp={completeEdit}
      onPointerCancel={completeEdit}
    >
      {overlayState && content && frameStyle && (
        <div className="pip-live-frame" style={frameStyle}>
          <div className={`pip-live-media ${overlayState.config.shape} ${overlayState.config.mirror ? 'mirrored' : ''}`}>
            {usesBackendPreviewImage ? (
              previewImageDataUrl ? (
                <img
                  className={`pip-camera-preview-image ${cameraReady ? 'ready' : ''}`}
                  src={previewImageDataUrl}
                  alt=""
                  draggable={false}
                />
              ) : null
            ) : (
              <video ref={videoRef} autoPlay muted playsInline className={cameraReady ? 'ready' : ''} />
            )}
            {showCameraPlaceholder && (
              <div className={`pip-camera-placeholder ${overlayState.mode} ${cameraError ? 'error' : 'pending'}`}>
                <Camera size={24} />
                <strong>{cameraPlaceholderTitle}</strong>
                <span>{cameraPlaceholderDetail}</span>
              </div>
            )}
          </div>
          <button className="pip-drag-anchor" type="button" aria-label={copy.pipOverlay.move} title={copy.pipOverlay.move} onPointerDown={(event) => beginEdit(event, 'move')}>
            <Move size={18} />
          </button>
          {pipResizeActions.map((action) => (
            <button
              key={action}
              className={`pip-resize-handle ${action}`}
              type="button"
              aria-label={copy.pipOverlay.resize}
              title={copy.pipOverlay.resize}
              onPointerDown={(event) => beginEdit(event, action)}
            />
          ))}
          <div className="pip-overlay-tools">
            <button type="button" className={overlayState.config.shape === 'circle' ? 'selected' : ''} aria-label={copy.pipShapeLabels.circle} title={copy.pipShapeLabels.circle} onClick={() => updateShape('circle')}>
              <CircleDot size={15} />
            </button>
            <button type="button" className={overlayState.config.shape === 'square' ? 'selected' : ''} aria-label={copy.pipShapeLabels.square} title={copy.pipShapeLabels.square} onClick={() => updateShape('square')}>
              <Square size={15} />
            </button>
            <button type="button" className={overlayState.config.mirror ? 'selected' : ''} aria-label={copy.pipOverlay.mirror} title={copy.pipOverlay.mirror} onClick={toggleMirror}>
              <FlipHorizontal size={15} />
            </button>
            <button type="button" aria-label={copy.pipOverlay.close} title={copy.pipOverlay.close} onClick={closePip}>
              <X size={15} />
            </button>
          </div>
        </div>
      )}
    </main>
  )
}

async function openPipCameraStream(
  target?: PIPOverlayCamera,
  onStreamOpened: (stream: MediaStream) => void = () => undefined,
  isCancelled: () => boolean = () => false,
): Promise<MediaStream> {
  const videoConstraints = {
    width: {ideal: 1280},
    height: {ideal: 720} }
  const device = await selectPipPreviewDevice(target)
  if (device?.deviceId) {
    if (isCancelled()) throw new Error('pip camera request cancelled')
    try {
      void logClientEvent('pip-camera', 'get-user-media-exact-start', {
        deviceLabel: device.label,
        deviceId: device.deviceId })
      const exactStream = await navigator.mediaDevices.getUserMedia({
        video: {
          ...videoConstraints,
          deviceId: {exact: device.deviceId} },
        audio: false })
      onStreamOpened(exactStream)
      if (isCancelled()) {
        stopMediaStream(exactStream)
        throw new Error('pip camera request cancelled')
      }
      return exactStream
    } catch (error) {
      if (isCancelled()) throw error
      void logClientEvent('pip-camera', 'get-user-media-exact-failed', {
        deviceLabel: device.label,
        error: readableError(error) })
      console.info('PIP preview selected camera unavailable, using default camera:', error)
    }
  }
  if (isCancelled()) throw new Error('pip camera request cancelled')
  void logClientEvent('pip-camera', 'get-user-media-default-start', {
    targetName: target?.name ?? '',
    targetNativeId: target?.nativeId ?? '' })
  const defaultStream = await navigator.mediaDevices.getUserMedia({
    video: videoConstraints,
    audio: false })
  onStreamOpened(defaultStream)
  if (isCancelled()) {
    stopMediaStream(defaultStream)
    throw new Error('pip camera request cancelled')
  }
  return defaultStream
}

async function selectPipPreviewDevice(target: PIPOverlayCamera | undefined): Promise<MediaDeviceInfo | undefined> {
  const devices = await navigator.mediaDevices.enumerateDevices?.().catch((error) => {
    void logClientEvent('pip-camera', 'enumerate-devices-failed', {error: readableError(error)})
    return []
  })
  const videoDevices = devices?.filter((device) => device.kind === 'videoinput') ?? []
  if (videoDevices.length === 0) return undefined
  const matched = findMatchingPipPreviewDevice(target, videoDevices)
  if (matched) return matched
  return undefined
}

function describeMediaStream(stream: MediaStream) {
  return stream.getTracks().map((track) => {
    const settings = typeof track.getSettings === 'function' ? track.getSettings() : {}
    return [
      track.kind,
      track.readyState,
      track.label,
      settings.deviceId,
      settings.width && settings.height ? `${settings.width}x${settings.height}` : '',
      settings.frameRate ? `${settings.frameRate}fps` : '',
    ].filter(Boolean).join('|')
  }).join(';')
}

function stopMediaStream(stream: MediaStream) {
  stream.getTracks().forEach((track) => track.stop())
}

function findMatchingPipPreviewDevice(target: PIPOverlayCamera | undefined, devices: MediaDeviceInfo[]) {
  const tokens = pipPreviewMatchTokens(target)
  if (tokens.length === 0) return undefined
  return devices.find((device) => {
    const label = normalizePipPreviewText(device.label)
    const id = normalizePipPreviewText(device.deviceId)
    if (!label && !id) return false
    return tokens.some((token) => {
      const normalized = normalizePipPreviewText(token)
      if (!normalized || normalized.length < 3) return false
      const labelMatches = label !== '' && (label === normalized || label.includes(normalized) || normalized.includes(label))
      const idMatches = id !== '' && id === normalized
      return labelMatches || idMatches
    })
  })
}

function pipPreviewMatchTokens(target: PIPOverlayCamera | undefined) {
  const rawTokens = [
    target?.name,
    stripDefaultDevicePrefix(target?.name),
    target?.nativeId,
    target?.deviceId,
  ]
  return Array.from(new Set(rawTokens.map((token) => token?.trim()).filter((token): token is string => Boolean(token && token.length >= 3))))
}

function stripDefaultDevicePrefix(value?: string) {
  return value?.replace(/^\s*default\s+/i, '').replace(/^\s*默认\s*/, '')
}

function normalizePipPreviewText(value?: string) {
  return (value ?? '')
    .toLowerCase()
    .replace(/^\s*default\s+/i, '')
    .replace(/^\s*默认\s*/, '')
    .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function pipAbsoluteContentRect(state: PIPOverlayState) {
  return {
    x: state.windowBounds.x + state.contentBounds.x,
    y: state.windowBounds.y + state.contentBounds.y,
    width: state.contentBounds.width,
    height: state.contentBounds.height }
}

function pipCanvasMargin(bounds: PIPOverlayState['overlayBounds']) {
  return Math.max(16, Math.trunc(bounds.width / 40))
}

function pipMaxContentSize(state: PIPOverlayState) {
  const margin = pipCanvasMargin(state.overlayBounds)
  const canvasMax = Math.min(state.overlayBounds.width, state.overlayBounds.height) - margin * 2
  const scaleMax = Math.round(state.overlayBounds.width * pipMaximumScale)
  return Math.max(pipMinimumContentSize, Math.min(canvasMax, scaleMax))
}

function resizePipRect(state: PIPOverlayState, content: {x: number; y: number; width: number; height: number}, action: PIPEditAction, dx: number, dy: number) {
  if (action === 'move') {
    return {...content, x: content.x + dx, y: content.y + dy}
  }
  let delta = 0
  if (action === 'e') delta = dx
  if (action === 's') delta = dy
  if (action === 'w') delta = -dx
  if (action === 'n') delta = -dy
  if (action === 'se') delta = Math.max(dx, dy)
  if (action === 'sw') delta = Math.max(-dx, dy)
  if (action === 'ne') delta = Math.max(dx, -dy)
  if (action === 'nw') delta = Math.max(-dx, -dy)
  const size = clampNumber(Math.round(content.width + delta), pipMinimumContentSize, pipMaxContentSize(state))
  const next = {...content, width: size, height: size}
  if (action.includes('w')) next.x = content.x + content.width - size
  if (action.includes('n')) next.y = content.y + content.height - size
  return next
}

function pipConfigFromAbsoluteRect(state: PIPOverlayState, rect: {x: number; y: number; width: number; height: number}): PIPConfig {
  const overlay = state.overlayBounds
  const margin = pipCanvasMargin(overlay)
  const size = clampNumber(Math.round(Math.max(rect.width, rect.height)), pipMinimumContentSize, pipMaxContentSize(state))
  const minX = overlay.x + margin
  const minY = overlay.y + margin
  const maxX = overlay.x + overlay.width - margin - size
  const maxY = overlay.y + overlay.height - margin - size
  const x = clampNumber(rect.x, minX, Math.max(minX, maxX))
  const y = clampNumber(rect.y, minY, Math.max(minY, maxY))
  const availableWidth = Math.max(1, maxX - minX)
  const availableHeight = Math.max(1, maxY - minY)
  return normalizePipConfig({
    ...state.config,
    preset: 'free',
    position: {
      x: (x - minX) / availableWidth,
      y: (y - minY) / availableHeight },
    scale: size / Math.max(1, overlay.width) }, 'free')
}

export default PIPOverlayWindow
