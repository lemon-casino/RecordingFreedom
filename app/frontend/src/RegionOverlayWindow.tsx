import {applyTheme} from './components/themeOptions'
import {useCallback, useEffect, useRef, useState, type PointerEvent as ReactPointerEvent} from 'react'
import {Check, MousePointer2, X} from 'lucide-react'
import {copyByLocale} from './i18n'
import {clampNumber, normalizeLocale, normalizeTheme, type LocaleCode, type ThemeCode} from './services/mockBackend'
import {
  assistRegionSelection,
  beginScreenshotAnnotationOverlay,
  cancelRegionSelector,
  cancelSelectedRegion,
  completeAnnotationRegionSelection,
  completeRegionSelection,
  completeScreenshotRegionSelection,
  completeScrollingScreenshotSelection,
  loadSettings,
  showAnnotationRegionSelector,
  showRegionSelector,
  startScrollingScreenshot,
  showScreenshotRegionSelector,
  subscribeSettingsChanged,
  updateScreenshotRegionSelection,
  updateSelectedRegion,
  type RegionSelectionSession,
  type RegionSmartCandidate } from './services/recorderBackend'
import {readableError} from './services/recorderBackend'

type RegionFrameState = {
  bounds: {x: number; y: number; width: number; height: number}
  overlayBounds?: {x: number; y: number; width: number; height: number}
  mode?: 'edit' | 'recording'
  purpose?: 'capture' | 'annotation' | 'screenshot' | 'scrolling-screenshot'
}

type RegionEditAction = 'move' | 'n' | 'e' | 's' | 'w' | 'ne' | 'nw' | 'se' | 'sw'

const regionResizeActions: RegionEditAction[] = ['n', 'e', 's', 'w', 'ne', 'nw', 'se', 'sw']

function useRegionFrameState() {
  const frameWindow = window as Window & {__RF_REGION_FRAME__?: RegionFrameState}
  const [frame, setFrame] = useState<RegionFrameState | undefined>(frameWindow.__RF_REGION_FRAME__)
  const clearFrame = useCallback(() => {
    delete (window as Window & {__RF_REGION_FRAME__?: RegionFrameState}).__RF_REGION_FRAME__
    setFrame(undefined)
  }, [])

  useEffect(() => {
    const onFrame = (event: Event) => {
      const next = (event as CustomEvent<RegionFrameState | undefined>).detail
      if (next?.bounds) {
        ;(window as Window & {__RF_REGION_FRAME__?: RegionFrameState}).__RF_REGION_FRAME__ = next
        setFrame(next)
      } else {
        clearFrame()
      }
    }
    window.addEventListener('rf-region-frame', onFrame)
    return () => window.removeEventListener('rf-region-frame', onFrame)
  }, [clearFrame])

  useEffect(() => {
    window.addEventListener('rf-region-session', clearFrame)
    return () => window.removeEventListener('rf-region-session', clearFrame)
  }, [clearFrame])

  return [frame, clearFrame] as const
}

function useRegionEditorDrag(bounds: RegionFrameState['bounds'] | undefined, updateRegion: (bounds: RegionFrameState['bounds']) => Promise<unknown>) {
  const editRef = useRef<{
    action: RegionEditAction
    startX: number
    startY: number
    bounds: RegionFrameState['bounds']
    latest: RegionFrameState['bounds']
  } | null>(null)

  const beginEdit = (event: ReactPointerEvent<HTMLElement>, action: RegionEditAction) => {
    if (!bounds || event.button !== 0) return
    event.preventDefault()
    event.stopPropagation()
    event.currentTarget.setPointerCapture(event.pointerId)
    editRef.current = {
      action,
      startX: event.screenX,
      startY: event.screenY,
      bounds,
      latest: bounds }
  }

  const updateEdit = (event: ReactPointerEvent<HTMLElement>) => {
    const edit = editRef.current
    if (!edit) return
    event.preventDefault()
    const next = resizeRegionBounds(edit.bounds, edit.action, event.screenX - edit.startX, event.screenY - edit.startY)
    edit.latest = next
    void updateRegion(next)
  }

  const completeEdit = (event: ReactPointerEvent<HTMLElement>) => {
    const edit = editRef.current
    if (!edit) return
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    editRef.current = null
    void updateRegion(edit.latest)
  }

  return {beginEdit, updateEdit, completeEdit}
}

function resizeRegionBounds(bounds: RegionFrameState['bounds'], action: RegionEditAction, dx: number, dy: number) {
  const next = {...bounds}
  if (action === 'move') {
    next.x = Math.round(bounds.x + dx)
    next.y = Math.round(bounds.y + dy)
    return next
  }
  if (action.includes('e')) {
    next.width = Math.round(Math.max(minRegionEditorSize, bounds.width + dx))
  }
  if (action.includes('s')) {
    next.height = Math.round(Math.max(minRegionEditorSize, bounds.height + dy))
  }
  if (action.includes('w')) {
    const width = Math.round(Math.max(minRegionEditorSize, bounds.width - dx))
    next.x = Math.round(bounds.x + bounds.width - width)
    next.width = width
  }
  if (action.includes('n')) {
    const height = Math.round(Math.max(minRegionEditorSize, bounds.height - dy))
    next.y = Math.round(bounds.y + bounds.height - height)
    next.height = height
  }
  return next
}

const minRegionEditorSize = 64
const regionManualDragThreshold = 6

function regionPurposeClassName(purpose: RegionFrameState['purpose'] | undefined) {
  switch (purpose) {
    case 'annotation':
      return 'region-purpose-annotation'
    case 'screenshot':
      return 'region-purpose-screenshot'
    case 'scrolling-screenshot':
      return 'region-purpose-scrolling-screenshot'
    default:
      return 'region-purpose-capture'
  }
}

function normalizedClientRect(startX: number, startY: number, currentX: number, currentY: number) {
  const x = Math.round(Math.min(startX, currentX))
  const y = Math.round(Math.min(startY, currentY))
  const width = Math.round(Math.abs(currentX - startX))
  const height = Math.round(Math.abs(currentY - startY))
  return {x, y, width, height}
}

function clampedClientPoint(clientX: number, clientY: number) {
  const maxX = Math.max(0, (window.innerWidth || 1) - 1)
  const maxY = Math.max(0, (window.innerHeight || 1) - 1)
  return {
    x: clampNumber(clientX, 0, maxX),
    y: clampNumber(clientY, 0, maxY) }
}

function RegionOverlayWindow() {
  const overlayWindow = window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}
  const initialSession = overlayWindow.__RF_REGION_SESSION__
  const [editFrame, clearEditFrame] = useRegionFrameState()
  const isScreenshotRegionEdit = editFrame?.mode === 'edit' && editFrame.purpose === 'screenshot'
  const editDrag = useRegionEditorDrag(
    editFrame?.mode === 'edit' ? editFrame.bounds : undefined,
    isScreenshotRegionEdit ? updateScreenshotRegionSelection : updateSelectedRegion,
  )
  const [session, setSession] = useState<RegionSelectionSession | undefined>(initialSession)
  const [drag, setDrag] = useState<{startX: number; startY: number; currentX: number; currentY: number} | null>(null)
  const [cursor, setCursor] = useState({x: -1, y: -1})
  const [invalid, setInvalid] = useState(false)
  const [assistCandidate, setAssistCandidate] = useState<RegionSmartCandidate | null>(null)
  const [assistRequestPending, setAssistRequestPending] = useState(false)
  const shellRef = useRef<HTMLElement | null>(null)
  const pointerDownCandidateRef = useRef<RegionSmartCandidate | null>(null)
  const assistRequestSeqRef = useRef(0)
  const assistTimerRef = useRef<number | null>(null)
  const assistCandidateLevelRef = useRef(0)
  const lastAssistPointRef = useRef<{x: number; y: number} | null>(null)
  const handledRightPointerRef = useRef(false)
  const [overlayLocale, setOverlayLocale] = useState<LocaleCode>(navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en')
  const [overlayTheme, setOverlayTheme] = useState<ThemeCode>('night-teal')
  const copy = copyByLocale[overlayLocale]
  const minimumWidth = session?.minimumWidth ?? 64
  const minimumHeight = session?.minimumHeight ?? 64
  const dragDistance = drag ? pointerDistance(drag.startX, drag.startY, drag.currentX, drag.currentY) : 0
  const isManualDrag = Boolean(drag && dragDistance > regionManualDragThreshold)
  const selectedRect = drag && isManualDrag ? normalizedClientRect(drag.startX, drag.startY, drag.currentX, drag.currentY) : null
  const isEditingRegion = editFrame?.mode === 'edit'
  const isRecordingRegion = editFrame?.mode === 'recording'
  const isAnnotationRegionSelection = session?.purpose === 'annotation'
  const isScreenshotRegionSelection = session?.purpose === 'screenshot'
  const isScrollingScreenshotSelection = session?.purpose === 'scrolling-screenshot'
  const activePurposeClass = regionPurposeClassName(editFrame?.purpose ?? session?.purpose)
  const selectionPurposeClass = regionPurposeClassName(session?.purpose)
  const editPurposeClass = regionPurposeClassName(editFrame?.purpose)
  const sessionCandidates = session?.candidates ?? []
  const visibleAssistCandidate = !isEditingRegion && !isRecordingRegion
    ? selectedRect
      ? bestLocalRegionCandidate(sessionCandidates, selectedRect) ?? assistCandidate
      : assistCandidate
    : null
  const shouldShowCrosshair = !isEditingRegion &&
    !isRecordingRegion &&
    cursor.x >= 0 &&
    !selectedRect &&
    !visibleAssistCandidate &&
    !assistRequestPending
  const overlayOrigin = editFrame?.overlayBounds ?? session?.bounds ?? {x: 0, y: 0, width: 0, height: 0}
  const editableRect = isEditingRegion ? {
    x: editFrame.bounds.x - overlayOrigin.x,
    y: editFrame.bounds.y - overlayOrigin.y,
    width: editFrame.bounds.width,
    height: editFrame.bounds.height } : null
  const recordingRect = isRecordingRegion ? {
    x: editFrame.bounds.x - overlayOrigin.x,
    y: editFrame.bounds.y - overlayOrigin.y,
    width: editFrame.bounds.width,
    height: editFrame.bounds.height } : null

  useEffect(() => {
    document.body.classList.add('rf-region-overlay-window')
    return () => {
      document.body.classList.remove('rf-region-overlay-window')
      if (assistTimerRef.current !== null) {
        window.clearTimeout(assistTimerRef.current)
        assistTimerRef.current = null
      }
    }
  }, [])

  useEffect(() => {
    void loadSettings()
      .then((settings) => {
        setOverlayLocale(normalizeLocale(settings.locale))
        setOverlayTheme(normalizeTheme(settings.window.theme))
      })
      .catch((error) => console.info('Using region overlay language fallback:', error))
  }, [])

  useEffect(() => subscribeSettingsChanged((settings) => {
    setOverlayLocale(normalizeLocale(settings.locale))
    setOverlayTheme(normalizeTheme(settings.window.theme))
  }), [])

  useEffect(() => {
    applyTheme(overlayTheme)
  }, [overlayTheme])

  useEffect(() => {
    const onSession = (event: Event) => {
      const next = (event as CustomEvent<RegionSelectionSession>).detail
      if (next) {
        assistRequestSeqRef.current += 1
        if (assistTimerRef.current !== null) {
          window.clearTimeout(assistTimerRef.current)
          assistTimerRef.current = null
        }
        delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
        setSession(next)
        setDrag(null)
        setCursor({x: -1, y: -1})
        setInvalid(false)
        setAssistCandidate(null)
        setAssistRequestPending(false)
        pointerDownCandidateRef.current = null
        assistCandidateLevelRef.current = 0
        lastAssistPointRef.current = null
      }
    }
    window.addEventListener('rf-region-session', onSession)
    return () => window.removeEventListener('rf-region-session', onSession)
  }, [])

  const clearRegionSelectionRuntimeState = () => {
    assistRequestSeqRef.current += 1
    if (assistTimerRef.current !== null) {
      window.clearTimeout(assistTimerRef.current)
      assistTimerRef.current = null
    }
    delete (window as Window & {__RF_REGION_SESSION__?: RegionSelectionSession}).__RF_REGION_SESSION__
    delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
    clearEditFrame()
    setSession(undefined)
    setDrag(null)
    setCursor({x: -1, y: -1})
    setInvalid(false)
    setAssistCandidate(null)
    setAssistRequestPending(false)
    pointerDownCandidateRef.current = null
    assistCandidateLevelRef.current = 0
    lastAssistPointRef.current = null
  }

  const cancelSelection = async () => {
    clearRegionSelectionRuntimeState()
    await cancelRegionSelector()
    if (!window.navigator.userAgent.includes('Wails')) {
      window.close()
    }
  }

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        if (isEditingRegion) {
          clearEditFrame()
          void (isScreenshotRegionEdit ? cancelRegionSelector() : cancelSelectedRegion())
          return
        }
        void cancelSelection()
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [isEditingRegion, isScreenshotRegionEdit])

  const cancelPendingHoverAssist = () => {
    assistRequestSeqRef.current += 1
    setAssistRequestPending(false)
    if (assistTimerRef.current !== null) {
      window.clearTimeout(assistTimerRef.current)
      assistTimerRef.current = null
    }
  }

  const completeSelection = async (rect: RegionSelectionSession['bounds']) => {
    if (rect.width < minimumWidth || rect.height < minimumHeight) {
      setInvalid(true)
      window.setTimeout(() => setInvalid(false), 360)
      return
    }
    try {
      if (isAnnotationRegionSelection) {
        await completeAnnotationRegionSelection(rect)
        return
      }
      if (isScreenshotRegionSelection) {
        await beginScreenshotAnnotationOverlay(rect)
        return
      }
      if (isScrollingScreenshotSelection) {
        await completeScrollingScreenshotSelection(rect)
        return
      }
      await completeRegionSelection(rect)
    } catch (error) {
      console.error('Failed to complete region selection:', error)
      setInvalid(true)
      window.setTimeout(() => setInvalid(false), 360)
      window.alert(readableError(error) || copy.screenshot.captureFailed)
    }
  }

  const requestHoverAssistCandidateForSession = (
    activeSession: RegionSelectionSession | undefined,
    point: {x: number; y: number},
    immediate = false,
    allowEditing = false,
  ) => {
    if (!activeSession || isRecordingRegion || (isEditingRegion && !allowEditing)) {
      setAssistRequestPending(false)
      return
    }
    const activeCandidates = activeSession.candidates ?? []
    const requestSeq = assistRequestSeqRef.current + 1
    assistRequestSeqRef.current = requestSeq
    const lastPoint = lastAssistPointRef.current
    if (!lastPoint || pointerDistance(lastPoint.x, lastPoint.y, point.x, point.y) > 10) {
      assistCandidateLevelRef.current = 0
    }
    lastAssistPointRef.current = point
    setAssistRequestPending(true)
    const run = () => {
      assistTimerRef.current = null
      const level = assistCandidateLevelRef.current
      void assistRegionSelection({
        sessionId: activeSession.id,
        purpose: activeSession.purpose,
        pointerX: point.x,
        pointerY: point.y,
        candidateLevel: level,
        candidates: activeCandidates }).then((result) => {
        if (assistRequestSeqRef.current !== requestSeq) return
        ;(window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__ = result
        setAssistCandidate(result.best ?? bestLocalRegionCandidateForPoint(activeCandidates, point))
        setAssistRequestPending(false)
      }).catch((error) => {
        if (assistRequestSeqRef.current !== requestSeq) return
        console.info('Region hover assist failed; using local candidates:', error)
        setAssistCandidate(bestLocalRegionCandidateForPoint(activeCandidates, point))
        setAssistRequestPending(false)
      })
    }
    if (assistTimerRef.current !== null) {
      window.clearTimeout(assistTimerRef.current)
      assistTimerRef.current = null
    }
    if (immediate) {
      run()
      return
    }
    assistTimerRef.current = window.setTimeout(run, 70)
  }

  const requestHoverAssistCandidate = (point: {x: number; y: number}, immediate = false) => {
    requestHoverAssistCandidateForSession(session, point, immediate)
  }

  useEffect(() => {
    if (!session?.initialPointer || isEditingRegion || isRecordingRegion || lastAssistPointRef.current) return
    const point = clampedClientPoint(session.initialPointer.x, session.initialPointer.y)
    setCursor(point)
    setAssistCandidate(bestLocalRegionCandidateForPoint(session.candidates ?? [], point))
    requestHoverAssistCandidateForSession(session, point, true)
  }, [session?.id, isEditingRegion, isRecordingRegion])

  const returnToAutoSelection = (point: {x: number; y: number}, activeSession = session, allowEditing = false) => {
    cancelPendingHoverAssist()
    const activeCandidates = activeSession?.candidates ?? []
    pointerDownCandidateRef.current = null
    setDrag(null)
    setInvalid(false)
    assistCandidateLevelRef.current = 0
    setCursor(point)
    setAssistCandidate(bestLocalRegionCandidateForPoint(activeCandidates, point))
    requestHoverAssistCandidateForSession(activeSession, point, true, allowEditing)
  }

  const showSelectionSessionForPurpose = (purpose: RegionFrameState['purpose'] | undefined) => {
    if (purpose === 'screenshot') return showScreenshotRegionSelector()
    if (purpose === 'scrolling-screenshot') return startScrollingScreenshot()
    if (purpose === 'annotation') return showAnnotationRegionSelector()
    return showRegionSelector()
  }

  const returnEditingRegionToAutoSelection = (point: {x: number; y: number}) => {
    const purpose = editFrame?.purpose
    clearEditFrame()
    cancelPendingHoverAssist()
    pointerDownCandidateRef.current = null
    setDrag(null)
    setInvalid(false)
    assistCandidateLevelRef.current = 0
    lastAssistPointRef.current = null
    setAssistCandidate(null)
    delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
    void showSelectionSessionForPurpose(purpose).then((nextSession) => {
      setSession(nextSession)
      returnToAutoSelection(point, nextSession, true)
    }).catch((error) => {
      console.error('Failed to return region editor to auto selection:', error)
      setInvalid(true)
      window.setTimeout(() => setInvalid(false), 360)
    })
  }

  const cancelRegionFromPointer = (point: {x: number; y: number}) => {
    if (isRecordingRegion) return
    if (isEditingRegion) {
      returnEditingRegionToAutoSelection(point)
      return
    }
    if (drag) {
      returnToAutoSelection(point)
      return
    }
    cancelPendingHoverAssist()
    pointerDownCandidateRef.current = null
    setAssistCandidate(null)
    delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
    void cancelSelection()
  }

  const shellMode = isRecordingRegion ? 'recording' : isEditingRegion ? 'editing' : 'selecting'

  return (
    <main
      ref={shellRef}
      className={`region-overlay-shell ${shellMode} ${activePurposeClass}`}
      aria-label={copy.aria.regionOverlay}
      onPointerMove={isEditingRegion ? editDrag.updateEdit : undefined}
      onPointerCancel={isEditingRegion ? editDrag.completeEdit : undefined}
      onPointerMoveCapture={(event) => {
        if (isEditingRegion || isRecordingRegion) return
        const point = clampedClientPoint(event.clientX, event.clientY)
        setCursor(point)
        if (drag) {
          const nextDrag = {...drag, currentX: point.x, currentY: point.y}
          setDrag(nextDrag)
          const nextDistance = pointerDistance(nextDrag.startX, nextDrag.startY, nextDrag.currentX, nextDrag.currentY)
          if (nextDistance > regionManualDragThreshold) {
            setAssistCandidate(bestLocalRegionCandidate(sessionCandidates, normalizedClientRect(nextDrag.startX, nextDrag.startY, nextDrag.currentX, nextDrag.currentY)))
          }
        } else {
          setAssistCandidate(bestLocalRegionCandidateForPoint(sessionCandidates, point))
          requestHoverAssistCandidate(point)
        }
      }}
      onWheel={(event) => {
        if (isEditingRegion || isRecordingRegion || drag) return
        event.preventDefault()
        const point = cursor.x >= 0 ? cursor : clampedClientPoint(event.clientX, event.clientY)
        const direction = event.deltaY > 0 ? -1 : 1
        assistCandidateLevelRef.current = Math.max(0, assistCandidateLevelRef.current + direction)
        requestHoverAssistCandidate(point, true)
      }}
      onMouseDownCapture={(event) => {
        if (isRecordingRegion || event.button !== 2) return
        event.preventDefault()
        event.stopPropagation()
        if (handledRightPointerRef.current) return
        handledRightPointerRef.current = true
        window.setTimeout(() => {
          handledRightPointerRef.current = false
        }, 240)
        cancelRegionFromPointer(clampedClientPoint(event.clientX, event.clientY))
      }}
      onPointerDown={(event) => {
        if (isRecordingRegion) return
        if (event.button === 2) {
          event.preventDefault()
          event.stopPropagation()
          handledRightPointerRef.current = true
          window.setTimeout(() => {
            handledRightPointerRef.current = false
          }, 240)
          cancelRegionFromPointer(clampedClientPoint(event.clientX, event.clientY))
          return
        }
        if (isEditingRegion) return
        if (event.button !== 0) return
        cancelPendingHoverAssist()
        event.currentTarget.setPointerCapture(event.pointerId)
        const point = clampedClientPoint(event.clientX, event.clientY)
        const candidate = assistCandidate && rectContainsPoint(assistCandidate.bounds, point)
          ? assistCandidate
          : bestLocalRegionCandidateForPoint(sessionCandidates, point)
        pointerDownCandidateRef.current = candidate
        setAssistCandidate(candidate)
        setDrag({startX: point.x, startY: point.y, currentX: point.x, currentY: point.y})
        setInvalid(false)
      }}
      onPointerUp={(event) => {
        if (isRecordingRegion) return
        if (isEditingRegion) {
          editDrag.completeEdit(event)
          return
        }
        if (!drag) return
        if (event.currentTarget.hasPointerCapture(event.pointerId)) {
          event.currentTarget.releasePointerCapture(event.pointerId)
        }
        const point = clampedClientPoint(event.clientX, event.clientY)
        const rect = normalizedClientRect(drag.startX, drag.startY, point.x, point.y)
        const distance = pointerDistance(drag.startX, drag.startY, point.x, point.y)
        const clickCandidate = pointerDownCandidateRef.current && distance <= regionManualDragThreshold
          ? pointerDownCandidateRef.current
          : null
        pointerDownCandidateRef.current = null
        setDrag(null)
        setAssistCandidate(null)
        setAssistRequestPending(false)
        assistCandidateLevelRef.current = 0
        if (clickCandidate) {
          void completeSelection(clickCandidate.bounds)
          return
        }
        if (distance <= regionManualDragThreshold) {
          returnToAutoSelection(point)
          return
        }
        // Once the user has dragged past the manual threshold, the drag
        // rectangle is authoritative. Smart assist is only for a click
        // without a manual selection.
        void completeSelection(rect)
      }}
      onContextMenu={(event) => {
        event.preventDefault()
        event.stopPropagation()
        if (handledRightPointerRef.current) {
          handledRightPointerRef.current = false
          return
        }
        cancelRegionFromPointer(clampedClientPoint(event.clientX, event.clientY))
      }}
      onPointerLeave={(event) => {
        if (isEditingRegion || isRecordingRegion) return
        setCursor({x: -1, y: -1})
        cancelPendingHoverAssist()
        pointerDownCandidateRef.current = null
        assistCandidateLevelRef.current = 0
        lastAssistPointRef.current = null
        setAssistCandidate(null)
        delete (window as Window & {__RF_LAST_REGION_ASSIST__?: unknown}).__RF_LAST_REGION_ASSIST__
      }}
    >
      <div className="region-overlay-scrim" />
      {shouldShowCrosshair && (
        <>
          <div className="region-crosshair horizontal" style={{top: cursor.y}} />
          <div className="region-crosshair vertical" style={{left: cursor.x}} />
        </>
      )}
      {visibleAssistCandidate && (
        <div
          className={`region-smart-candidate ${visibleAssistCandidate.kind}`}
          style={{
            left: visibleAssistCandidate.bounds.x,
            top: visibleAssistCandidate.bounds.y,
            width: visibleAssistCandidate.bounds.width,
            height: visibleAssistCandidate.bounds.height }}
        >
          <span>{regionCandidateLabel(visibleAssistCandidate)}</span>
        </div>
      )}
      {!isEditingRegion && !isRecordingRegion && selectedRect && (
        <div
          className={`region-selection-rect ${selectionPurposeClass} ${invalid ? 'invalid' : ''}`}
          style={{
            left: selectedRect.x,
            top: selectedRect.y,
            width: selectedRect.width,
            height: selectedRect.height }}
        >
          <b className="region-size-badge">
            {selectedRect.width} x {selectedRect.height}
          </b>
          <span className="corner top-left" />
          <span className="corner top-right" />
          <span className="corner bottom-left" />
          <span className="corner bottom-right" />
        </div>
      )}
      {recordingRect && (
        <div
          className="region-recording-frame region-purpose-capture"
          style={{
            left: recordingRect.x,
            top: recordingRect.y,
            width: recordingRect.width,
            height: recordingRect.height }}
        />
      )}
      {editableRect && (
        <div
          className={`region-edit-rect ${editPurposeClass}`}
          style={{
            left: editableRect.x,
            top: editableRect.y,
            width: editableRect.width,
            height: editableRect.height }}
        >
          <button className="region-edit-move" type="button" aria-label="Move selected region" onPointerDown={(event) => editDrag.beginEdit(event, 'move')}>
            <span />
          </button>
          {regionResizeActions.map((action) => (
            <button
              key={action}
              className={`region-edit-resize ${action}`}
              type="button"
              aria-label={`Resize ${action}`}
              onPointerDown={(event) => editDrag.beginEdit(event, action)}
            />
          ))}
          <div className="region-edit-controls">
            <b>{editableRect.width} x {editableRect.height}</b>
            {isScreenshotRegionEdit && (
              <button
                className="region-edit-confirm"
                type="button"
                aria-label={copy.regionOverlay.saveScreenshot}
                title={copy.regionOverlay.saveScreenshot}
                onPointerDown={(event) => event.stopPropagation()}
                onClick={() => void completeScreenshotRegionSelection(editFrame.bounds)}
              >
                <Check size={16} />
                <span>{copy.regionOverlay.saveScreenshot}</span>
              </button>
            )}
            <button
              type="button"
              aria-label={copy.regionOverlay.cancel}
              title={copy.regionOverlay.cancel}
              onPointerDown={(event) => event.stopPropagation()}
              onClick={() => void (isScreenshotRegionEdit ? cancelRegionSelector() : cancelSelectedRegion())}
            >
              <X size={17} />
            </button>
          </div>
        </div>
      )}
      {!isRecordingRegion && <button
        className="region-cancel-button"
        type="button"
        aria-label={copy.regionOverlay.cancel}
        title={copy.regionOverlay.cancel}
        onPointerDown={(event) => event.stopPropagation()}
        onClick={() => void (isEditingRegion ? (isScreenshotRegionEdit ? cancelRegionSelector() : cancelSelectedRegion()) : cancelSelection())}
      >
        <X size={22} />
      </button>}
      {!isEditingRegion && !isRecordingRegion && <div className="region-overlay-badge" aria-hidden="true">
        <MousePointer2 size={16} />
        <span>{copy.regionOverlay.esc}</span>
      </div>}
    </main>
  )
}

function bestLocalRegionCandidateForPoint(candidates: RegionSmartCandidate[], point: {x: number; y: number}) {
  let best: RegionSmartCandidate | null = null
  let bestScore = -1
  for (const candidate of candidates) {
    if (!rectContainsPoint(candidate.bounds, point)) continue
    const area = Math.max(1, candidate.bounds.width * candidate.bounds.height)
    const score = (candidate.score ?? 0) + localRegionKindWeight(candidate.kind) + 1000000 / area
    if (score > bestScore) {
      best = candidate
      bestScore = score
    }
  }
  return best
}

function bestLocalRegionCandidate(candidates: RegionSmartCandidate[], selection: RegionSelectionSession['bounds']) {
  let best: RegionSmartCandidate | null = null
  let bestScore = -1
  for (const candidate of candidates) {
    const score = localRegionCandidateScore(candidate.bounds, selection) + localRegionKindWeight(candidate.kind)
    if (score > bestScore) {
      best = candidate
      bestScore = score
    }
  }
  return bestScore >= 0.5 ? best : null
}

function localRegionCandidateScore(candidate: RegionSelectionSession['bounds'], selection: RegionSelectionSession['bounds']) {
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

function localRegionKindWeight(kind: RegionSmartCandidate['kind']) {
  if (kind === 'element') return 0.36
  if (kind === 'edge') return 0.16
  if (kind === 'window') return 0.08
  return 0
}

function rectContainsPoint(rect: RegionSelectionSession['bounds'], point: {x: number; y: number}) {
  return rect.width > 0 &&
    rect.height > 0 &&
    point.x >= rect.x &&
    point.x < rect.x + rect.width &&
    point.y >= rect.y &&
    point.y < rect.y + rect.height
}

function pointerDistance(startX: number, startY: number, currentX: number, currentY: number) {
  return Math.hypot(currentX - startX, currentY - startY)
}

function regionCandidateLabel(candidate: RegionSmartCandidate) {
  const label = candidate.label?.trim()
  if (label) return label
  if (candidate.kind === 'window') return 'Window'
  if (candidate.kind === 'screen') return 'Screen'
  if (candidate.kind === 'element') return 'Element'
  if (candidate.kind === 'edge') return 'Edge'
  return 'Target'
}

export default RegionOverlayWindow
