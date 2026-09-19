import {applyTheme} from './components/themeOptions'
import {useEffect, useLayoutEffect, useMemo, useRef, useState, type CSSProperties, type RefObject} from 'react'
import {Copy as CopyIcon, Eye, FileText, Image as ImageIcon, Languages, Lock, Pin, X} from 'lucide-react'
import {OcrPositionTextLayer, ocrBlockPolygonPoints, ocrBlockStableId} from './components/ocr/OcrPositionTextLayer'
import {copyByLocale} from './i18n'
import {defaultSettings, normalizeLocale, normalizeTheme, normalizeOcrTranslationSettings, type AppSettings, type LocaleCode, type ScreenshotItem, type ThemeCode} from './services/mockBackend'
import {
  hideFloatingPanel,
  hidePinnedScreenshot,
  loadPinnedScreenshot,
  patchScreenshotItem,
  loadSettings,
  openOcrResult,
  queueRecognizePinnedScreenshot,
  subscribeOcrJobEvents,
  subscribeScreenshotHistoryChanged,
  subscribeScreenshotPin,
  subscribeSettingsChanged,
  type OcrBlock,
  type OcrResult,
  type ScreenshotPinState } from './services/recorderBackend'
import {copyOcrResultText, screenshotDisplayName, screenshotOcrBusy, screenshotOcrPrimaryTitle, screenshotOcrStatusText, screenshotPinItems, screenshotPinStateWithPins, translateAndCopyOcrResultText} from './components/screenshotShared'
import {writeClipboardText} from './utils/clipboard'
import {readableError} from './services/recorderBackend'

function useContainedImageLayerStyle(containerRef: RefObject<HTMLElement | null>, imageWidth: number, imageHeight: number) {
  const [style, setStyle] = useState<CSSProperties>({
    left: 0,
    top: 0,
    width: '100%',
    height: '100%' })

  useLayoutEffect(() => {
    const element = containerRef.current
    if (!element) return
    const safeWidth = Number.isFinite(imageWidth) && imageWidth > 0 ? imageWidth : 1
    const safeHeight = Number.isFinite(imageHeight) && imageHeight > 0 ? imageHeight : 1
    const update = () => {
      const rect = element.getBoundingClientRect()
      if (rect.width <= 0 || rect.height <= 0) return
      const imageRatio = safeWidth / safeHeight
      const containerRatio = rect.width / rect.height
      let width = rect.width
      let height = rect.height
      let left = 0
      let top = 0
      if (containerRatio > imageRatio) {
        width = rect.height * imageRatio
        left = (rect.width - width) / 2
      } else {
        height = rect.width / imageRatio
        top = (rect.height - height) / 2
      }
      const next: CSSProperties = {
        left: `${left}px`,
        top: `${top}px`,
        width: `${width}px`,
        height: `${height}px` }
      setStyle((current) => {
        if (current.left === next.left && current.top === next.top && current.width === next.width && current.height === next.height) {
          return current
        }
        return next
      })
    }
    update()
    const observer = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(update) : null
    observer?.observe(element)
    window.addEventListener('resize', update)
    return () => {
      observer?.disconnect()
      window.removeEventListener('resize', update)
    }
  }, [containerRef, imageHeight, imageWidth])

  return style
}

function ScreenshotPinWindow() {
  const pinWindow = window as Window & {__RF_SCREENSHOT_PIN__?: ScreenshotPinState}
  const [pinState, setPinState] = useState<ScreenshotPinState | undefined>(pinWindow.__RF_SCREENSHOT_PIN__)
  const [selectedPinId, setSelectedPinId] = useState('')
  const [locale, setLocale] = useState<LocaleCode>(navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en')
  const [theme, setTheme] = useState<ThemeCode>('night-teal')
  const [translationSettings, setTranslationSettings] = useState<AppSettings['ocr']['translation']>(defaultSettings.ocr.translation)
  const [ocrResult, setOcrResult] = useState<OcrResult | null>(null)
  const [ocrMessage, setOcrMessage] = useState('')
  const [highlightOcr, setHighlightOcr] = useState(false)
  const [hoveredBlockId, setHoveredBlockId] = useState('')
  const [copiedBlockId, setCopiedBlockId] = useState('')
  const pins = useMemo(() => screenshotPinItems(pinState), [pinState])
  const activePin = pins.find((pin) => pin.item.id === selectedPinId) ?? pins[pins.length - 1]
  const itemRef = useRef<ScreenshotItem | undefined>(activePin?.item)
  const copy = copyByLocale[locale]
  const item = activePin?.item
  const activeDataUrl = activePin?.dataUrl || pinState?.dataUrl || ''
  const activeFixed = activePin?.fixed === true
  const ocrBusy = item ? screenshotOcrBusy(item) : false
  const ocrReady = item?.ocrStatus === 'ready' && Boolean(item.ocrResultId) && Boolean(ocrResult)
  const ocrStatusText = item ? screenshotOcrStatusText(item, copy) : ''
  const pinCanvasRef = useRef<HTMLDivElement | null>(null)
  const pinImageLayerStyle = useContainedImageLayerStyle(
    pinCanvasRef,
    ocrResult?.width || item?.width || 1,
    ocrResult?.height || item?.height || 1,
  )

  useEffect(() => {
    document.body.classList.add('rf-screenshot-pin-window')
    return () => document.body.classList.remove('rf-screenshot-pin-window')
  }, [])

  useEffect(() => {
    if (pins.length === 0) {
      if (selectedPinId) setSelectedPinId('')
      return
    }
    if (!selectedPinId || !pins.some((pin) => pin.item.id === selectedPinId)) {
      setSelectedPinId(pins[pins.length - 1].item.id)
    }
  }, [pins, selectedPinId])

  useEffect(() => {
    void Promise.all([loadSettings(), loadPinnedScreenshot()])
      .then(([settings, state]) => {
        setLocale(normalizeLocale(settings.locale))
        setTheme(normalizeTheme(settings.window.theme))
        setTranslationSettings(normalizeOcrTranslationSettings(settings.ocr.translation))
        setPinState(state)
      })
      .catch((error) => console.info('Using screenshot pin fallback:', error))
    const unsubscribeSettings = subscribeSettingsChanged((settings) => {
      setLocale(normalizeLocale(settings.locale))
      setTheme(normalizeTheme(settings.window.theme))
      setTranslationSettings(normalizeOcrTranslationSettings(settings.ocr.translation))
    })
    const unsubscribePin = subscribeScreenshotPin((state) => {
      setPinState(state)
    })
    const unsubscribeHistory = subscribeScreenshotHistoryChanged((items) => {
      setPinState((current) => {
        const currentPins = screenshotPinItems(current)
        if (currentPins.length === 0) return current
        let changed = false
        const nextPins = currentPins.map((pin) => {
          const updated = items.find((entry) => entry.id === pin.item.id)
          if (!updated) return pin
          changed = true
          return {...pin, item: updated, fixed: updated.fixed}
        })
        return changed ? screenshotPinStateWithPins(current, nextPins) : current
      })
    })
    const unsubscribeOcr = subscribeOcrJobEvents((event) => {
      const currentItem = itemRef.current
      if (!currentItem || event.sourceId !== currentItem.id) return
      if (event.status === 'queued') setOcrMessage(copy.screenshot.ocrQueued)
      if (event.status === 'running') setOcrMessage(copy.screenshot.ocrStatusRunning)
      if (event.status === 'ready') setOcrMessage(copy.screenshot.ocrStatusReady)
      if (event.status === 'failed') setOcrMessage(event.error || copy.screenshot.ocrStatusFailed)
    })
    return () => {
      unsubscribeSettings()
      unsubscribePin()
      unsubscribeHistory()
      unsubscribeOcr()
    }
  }, [copy])

  useEffect(() => {
    document.documentElement.lang = locale
    applyTheme(theme)
  }, [locale, theme])

  useEffect(() => {
    itemRef.current = item
  }, [item])

  useEffect(() => {
    setHoveredBlockId('')
    setCopiedBlockId('')
    if (!item?.ocrResultId || item.ocrStatus !== 'ready') {
      setOcrResult(null)
      setHighlightOcr(false)
      return
    }
    let cancelled = false
    void openOcrResult(item.ocrResultId)
      .then((result) => {
        if (cancelled) return
        setOcrResult(result)
        setOcrMessage(result.plainText ? copy.screenshot.ocrBlocks(result.blocks.length) : copy.screenshot.ocrNoText)
      })
      .catch((error) => {
        if (cancelled) return
        setOcrResult(null)
        setOcrMessage(readableError(error) || copy.screenshot.ocrStatusFailed)
      })
    return () => {
      cancelled = true
    }
  }, [copy, item?.id, item?.ocrResultId, item?.ocrStatus])

  const queuePinnedOcr = () => {
    if (!item || ocrBusy) return
    setOcrMessage(copy.screenshot.ocrQueued)
    void queueRecognizePinnedScreenshot(item.id)
      .catch((error) => {
        console.error('Failed to queue pinned screenshot OCR:', error)
        setOcrMessage(readableError(error) || copy.screenshot.ocrStatusFailed)
      })
  }

  const openPinnedOcrResult = async () => {
    if (!item?.ocrResultId || item.ocrStatus !== 'ready') {
      queuePinnedOcr()
      return
    }
    await hideFloatingPanel(0)
    setHighlightOcr(true)
    if (ocrResult?.blocks.length) {
      setOcrMessage(copy.screenshot.ocrBlocks(ocrResult.blocks.length))
    }
  }

  const togglePinnedFixed = () => {
    if (!item) return
    const nextFixed = !activeFixed
    void patchScreenshotItem(item.id, {fixed: nextFixed})
      .then((items) => {
        const updated = items.find((entry) => entry.id === item.id) ?? {...item, fixed: nextFixed}
        setPinState((current) => screenshotPinStateWithPins(current, screenshotPinItems(current).map((pin) => pin.item.id === item.id
          ? {...pin, item: updated, fixed: updated.fixed}
          : pin)))
      })
      .catch((error) => setOcrMessage(readableError(error) || copy.screenshot.ocrStatusFailed))
  }

  const copyPinnedOcrText = () => {
    if (!ocrResult) return
    void copyOcrResultText(ocrResult, copy).then(setOcrMessage).catch(() => setOcrMessage(copy.screenshot.copyTextEmpty))
  }

  const translatePinnedOcrText = () => {
    if (!item?.ocrResultId || item.ocrStatus !== 'ready') {
      queuePinnedOcr()
      return
    }
    setOcrMessage(copy.screenshot.translationWorking)
    void translateAndCopyOcrResultText(item.ocrResultId, translationSettings, copy)
      .then(setOcrMessage)
      .catch((error) => {
        console.error('Failed to translate pinned screenshot OCR text:', error)
        setOcrMessage(`${copy.screenshot.translationFailed}: ${readableError(error)}`)
      })
  }

  const copyPinnedBlockText = (block: OcrBlock, blockId: string) => {
    const text = block.text.trim()
    if (!text) return
    void writeClipboardText(text)
      .then(() => {
        setCopiedBlockId(blockId)
        setOcrMessage(copy.screenshot.copiedText)
        window.setTimeout(() => setCopiedBlockId((current) => current === blockId ? '' : current), 1200)
      })
      .catch(() => setOcrMessage(copy.screenshot.copyTextEmpty))
  }

  if (!pinState?.visible || !activeDataUrl || !item) {
    return (
      <main className="screenshot-pin-shell empty" data-theme={theme}>
        <span>{copy.screenshot.empty}</span>
      </main>
    )
  }

  return (
    <main className={`screenshot-pin-shell ${activeFixed ? 'fixed' : ''} ${pins.length > 1 ? 'multi' : ''}`} data-theme={theme}>
      <section className="screenshot-pin-toolbar">
        <span className="screenshot-pin-title">
          <ImageIcon size={15} />
          <span>
            <strong>{screenshotDisplayName(item)}</strong>
            <small>{ocrMessage || ocrStatusText}</small>
          </span>
        </span>
        <div>
          <button type="button" disabled={ocrBusy} aria-label={screenshotOcrPrimaryTitle(item, copy)} title={screenshotOcrPrimaryTitle(item, copy)} onClick={() => void openPinnedOcrResult()}>
            <FileText size={15} />
          </button>
          <button type="button" disabled={!ocrReady} className={highlightOcr ? 'selected' : ''} aria-label={copy.screenshot.openOcrResult} title={copy.screenshot.openOcrResult} onClick={() => setHighlightOcr((visible) => !visible)}>
            <Eye size={15} />
          </button>
          <button type="button" disabled={!ocrReady || !ocrResult?.plainText.trim()} aria-label={copy.screenshot.copyText} title={copy.screenshot.copyText} onClick={copyPinnedOcrText}>
            <CopyIcon size={15} />
          </button>
          <button type="button" disabled={!ocrReady || !ocrResult?.plainText.trim()} aria-label={copy.screenshot.translateText} title={copy.screenshot.translateText} onClick={translatePinnedOcrText}>
            <Languages size={15} />
          </button>
          <button type="button" aria-label={activeFixed ? copy.screenshot.unfix : copy.screenshot.fix} title={activeFixed ? copy.screenshot.unfix : copy.screenshot.fix} onClick={togglePinnedFixed}>
            {activeFixed ? <Lock size={15} /> : <Pin size={15} />}
          </button>
          <button type="button" aria-label={copy.common.close} title={copy.common.close} onClick={() => void hidePinnedScreenshot()}>
            <X size={16} />
          </button>
        </div>
      </section>
      <div className="screenshot-pin-content">
        {pins.length > 1 && (
          <aside className="screenshot-pin-stack" aria-label={copy.screenshot.pinned}>
            {pins.map((pin) => (
              <button
                key={pin.item.id}
                type="button"
                className={pin.item.id === item.id ? 'selected' : ''}
                title={screenshotDisplayName(pin.item)}
                onClick={() => setSelectedPinId(pin.item.id)}
              >
                {pin.dataUrl ? <img src={pin.dataUrl} alt="" draggable={false} /> : <ImageIcon size={18} />}
                <span>{screenshotDisplayName(pin.item)}</span>
              </button>
            ))}
          </aside>
        )}
        <div className="screenshot-pin-canvas" ref={pinCanvasRef}>
          <img src={activeDataUrl} alt={copy.screenshot.pinned} draggable={false} />
          {highlightOcr && ocrResult && ocrReady && ocrResult.blocks.length > 0 && (
            <svg viewBox={`0 0 ${Math.max(1, ocrResult.width)} ${Math.max(1, ocrResult.height)}`} preserveAspectRatio="xMidYMid meet" aria-label={copy.screenshot.ocrBlocks(ocrResult.blocks.length)}>
              {ocrResult.blocks.map((block, index) => {
                const blockId = ocrBlockStableId(block, index)
                return (
                  <polygon
                    key={blockId}
                    points={ocrBlockPolygonPoints(block)}
                    className={`${blockId === hoveredBlockId ? 'active' : ''} ${blockId === copiedBlockId ? 'copied' : ''}`.trim()}
                    tabIndex={0}
                    role="button"
                    aria-label={block.text || copy.screenshot.ocrNoText}
                    onPointerEnter={() => setHoveredBlockId(blockId)}
                    onPointerLeave={() => setHoveredBlockId('')}
                    onFocus={() => setHoveredBlockId(blockId)}
                    onBlur={() => setHoveredBlockId('')}
                    onClick={() => copyPinnedBlockText(block, blockId)}
                    onKeyDown={(event) => {
                      if (event.key !== 'Enter' && event.key !== ' ') return
                      event.preventDefault()
                      copyPinnedBlockText(block, blockId)
                    }}
                  />
                )
              })}
            </svg>
          )}
          {highlightOcr && ocrResult && ocrReady && ocrResult.blocks.length > 0 && (
            <OcrPositionTextLayer
              copy={copy}
              result={ocrResult}
              hoveredBlockId={hoveredBlockId}
              copiedBlockId={copiedBlockId}
              onHover={setHoveredBlockId}
              onCopy={copyPinnedBlockText}
              className="pin"
              style={pinImageLayerStyle}
            />
          )}
        </div>
      </div>
    </main>
  )
}

export default ScreenshotPinWindow
