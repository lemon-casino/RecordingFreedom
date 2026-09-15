import {readScreenshotImage} from '../services/recorderBackend'
import {ClipboardPaste, Trash2} from 'lucide-react'
import {screenshotMeta} from './panelShared'
import {CopyIcon, FolderOpen, PenLine, Pin, Unlock, X} from 'lucide-react'
import {useRef, useState, type PointerEvent as ReactPointerEvent} from 'react'
import {Image as ImageIcon, Languages, Lock} from 'lucide-react'
import type {RecorderCopy} from '../i18n'
import {readableError} from '../services/recorderBackend'
import {type ScreenshotItem} from '../services/mockBackend'
import {screenshotDisplayName, screenshotOcrStatusText} from './screenshotShared'

export const screenshotDeleteRevealWidth = 116

export function ScreenshotHistoryRow({
  item,
  copy,
  onOpen,
  onCopyOcr,
  onTranslateOcr,
  onOpenFolder,
  onAnnotate,
  onPaste,
  onPin,
  onToggleFixed,
  onDelete }: {
  item: ScreenshotItem
  copy: RecorderCopy
  onOpen: (item: ScreenshotItem) => void
  onCopyOcr: (item: ScreenshotItem) => void
  onTranslateOcr: (item: ScreenshotItem, anchorElement?: Element) => void
  onOpenFolder: (item: ScreenshotItem) => void
  onAnnotate: (item: ScreenshotItem) => void
  onPaste?: (item: ScreenshotItem) => void
  onPin: (item: ScreenshotItem) => void
  onToggleFixed: (item: ScreenshotItem) => void
  onDelete: (item: ScreenshotItem) => void
}) {
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [dragOffset, setDragOffset] = useState<number | null>(null)
  const [previewDataUrl, setPreviewDataUrl] = useState('')
  const [previewError, setPreviewError] = useState('')
  const dragRef = useRef<{pointerId: number; startX: number; startY: number; base: number; captured: boolean} | null>(null)
  const suppressClickUntilRef = useRef(0)
  const previewRequestRef = useRef('')
  const offset = dragOffset ?? (deleteOpen ? screenshotDeleteRevealWidth : 0)

  const loadPreview = () => {
    if (previewDataUrl || previewRequestRef.current === item.id) return
    previewRequestRef.current = item.id
    setPreviewError('')
    void readScreenshotImage(item.id, true)
      .then((image) => {
        if (previewRequestRef.current !== item.id) return
        if (image.available && image.dataUrl) {
          setPreviewDataUrl(image.dataUrl)
          return
        }
        setPreviewError(screenshotDisplayName(item))
      })
      .catch((error) => {
        if (previewRequestRef.current !== item.id) return
        setPreviewError(readableError(error) || screenshotDisplayName(item))
      })
  }

  const suppressNextClick = () => {
    suppressClickUntilRef.current = Date.now() + 260
  }

  const beginSwipe = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (event.pointerType === 'mouse' && event.button !== 0) return
    const target = event.target as HTMLElement | null
    if (target?.closest('.screenshot-history-actions, .screenshot-history-delete-actions')) return
    dragRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      base: deleteOpen ? screenshotDeleteRevealWidth : 0,
      captured: false }
  }

  const moveSwipe = (event: ReactPointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current
    if (!drag || drag.pointerId !== event.pointerId) return
    const deltaX = event.clientX - drag.startX
    const deltaY = event.clientY - drag.startY
    if (Math.abs(deltaX) < 4 && Math.abs(deltaY) < 4) return
    if (Math.abs(deltaX) >= Math.abs(deltaY)) {
      event.preventDefault()
      if (!drag.captured) {
        event.currentTarget.setPointerCapture?.(event.pointerId)
        drag.captured = true
      }
      setDragOffset(Math.max(0, Math.min(screenshotDeleteRevealWidth, drag.base + deltaX)))
    }
  }

  const finishSwipe = (event: ReactPointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current
    if (!drag || drag.pointerId !== event.pointerId) return
    const deltaX = event.clientX - drag.startX
    const finalOffset = Math.max(0, Math.min(screenshotDeleteRevealWidth, drag.base + deltaX))
    const moved = Math.abs(deltaX) > 7 || Math.abs(event.clientY - drag.startY) > 7
    if (moved) suppressNextClick()
    setDeleteOpen(deltaX < -32 ? false : finalOffset > screenshotDeleteRevealWidth * 0.44)
    setDragOffset(null)
    dragRef.current = null
    if (drag.captured) event.currentTarget.releasePointerCapture?.(event.pointerId)
  }

  const cancelSwipe = (event: ReactPointerEvent<HTMLDivElement>) => {
    const drag = dragRef.current
    if (drag?.pointerId === event.pointerId) {
      setDragOffset(null)
      dragRef.current = null
      if (drag?.captured) event.currentTarget.releasePointerCapture?.(event.pointerId)
      suppressNextClick()
    }
  }

  const handleOpen = () => {
    if (Date.now() < suppressClickUntilRef.current) return
    if (deleteOpen) {
      setDeleteOpen(false)
      return
    }
    onOpen(item)
  }

  const handleAction = (action: (item: ScreenshotItem) => void) => {
    if (Date.now() < suppressClickUntilRef.current) return
    action(item)
  }

  return (
    <div
      className={`screenshot-history-row ${deleteOpen ? 'delete-open' : ''} ${deleteOpen || dragOffset !== null ? 'delete-revealed' : ''}`.trim()}
      onPointerDown={beginSwipe}
      onPointerMove={moveSwipe}
      onPointerUp={finishSwipe}
      onPointerCancel={cancelSwipe}
    >
      <div className="screenshot-history-delete-actions" aria-hidden={!deleteOpen && dragOffset === null}>
        <button
          type="button"
          className="delete-back"
          tabIndex={deleteOpen ? 0 : -1}
          onPointerDown={(event) => event.stopPropagation()}
          onClick={(event) => {
            event.stopPropagation()
            setDeleteOpen(false)
          }}
        >
          <X size={14} />
          <span>{copy.screenshot.deleteReturn}</span>
        </button>
        <button
          type="button"
          className="delete-confirm"
          tabIndex={deleteOpen ? 0 : -1}
          onPointerDown={(event) => event.stopPropagation()}
          onClick={(event) => {
            event.stopPropagation()
            setDeleteOpen(false)
            onDelete(item)
          }}
        >
          <Trash2 size={14} />
          <span>{copy.screenshot.delete}</span>
        </button>
      </div>
      <div className="screenshot-history-slide" style={{transform: `translateX(${offset}px)`}}>
        <button
          type="button"
          className="screenshot-history-main"
          onPointerEnter={loadPreview}
          onFocus={loadPreview}
          onClick={handleOpen}
        >
          <span className="screenshot-history-icon">
            <ImageIcon size={17} />
            <span className={`screenshot-history-thumbnail ${previewDataUrl ? 'loaded' : ''}`} aria-hidden="true">
              {previewDataUrl ? (
                <img src={previewDataUrl} alt="" draggable={false} />
              ) : (
                <span>{previewError || screenshotDisplayName(item)}</span>
              )}
            </span>
          </span>
          <span>
            <strong>{screenshotDisplayName(item)}</strong>
            <small>{screenshotMeta(item, copy)}</small>
            <small className={`screenshot-ocr-status ${item.ocrStatus}`}>{screenshotOcrStatusText(item, copy)}</small>
          </span>
        </button>
        <div
          className="screenshot-history-actions"
          onPointerDown={(event) => event.stopPropagation()}
          onClick={(event) => event.stopPropagation()}
        >
          <button type="button" disabled={item.ocrStatus !== 'ready'} aria-label={copy.screenshot.copyText} title={copy.screenshot.copyText} onClick={() => handleAction(onCopyOcr)}>
            <CopyIcon size={15} />
          </button>
          <button type="button" disabled={item.ocrStatus !== 'ready'} aria-label={copy.screenshot.translateText} title={copy.screenshot.translateText} onClick={(event) => handleAction(() => onTranslateOcr(item, event.currentTarget))}>
            <Languages size={15} />
          </button>
          <button type="button" aria-label={copy.screenshot.openFolder} title={copy.screenshot.openFolder} onClick={() => handleAction(onOpenFolder)}>
            <FolderOpen size={15} />
          </button>
          <button type="button" aria-label={copy.screenshot.annotate} title={copy.screenshot.annotate} onClick={() => handleAction(onAnnotate)}>
            <PenLine size={15} />
          </button>
          {onPaste && (
            <button type="button" aria-label={copy.screenshot.paste} title={copy.screenshot.paste} onClick={() => handleAction(onPaste)}>
              <ClipboardPaste size={15} />
            </button>
          )}
          <button type="button" aria-label={copy.screenshot.pin} title={copy.screenshot.pin} onClick={() => handleAction(onPin)}>
            <Pin size={15} />
          </button>
          <button
            type="button"
            className={item.fixed ? 'selected' : ''}
            aria-label={item.fixed ? copy.screenshot.unfix : copy.screenshot.fix}
            title={item.fixed ? copy.screenshot.unfix : copy.screenshot.fix}
            onClick={() => handleAction(onToggleFixed)}
          >
            {item.fixed ? <Lock size={15} /> : <Unlock size={15} />}
          </button>
        </div>
      </div>
    </div>
  )
}


