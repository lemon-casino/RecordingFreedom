import {logClientEvent, readOcrResultImage, translateOcr} from '../services/recorderBackend'
import {writeClipboardText} from '../utils/clipboard'
import {AppSettings, normalizeOcrTranslationSettings, ocrTranslationUnavailableMessage} from '../services/mockBackend'
import {OcrBlock, OcrTranslationResult} from '../services/recorderBackend'
import {ocrBlockPolygonPoints, ocrBlockStableId} from './ocr/OcrPositionTextLayer'
import {Eye, Maximize2, Minimize2} from 'lucide-react'
import {useCallback, useState} from 'react'
import {useEffect, useRef} from 'react'
import {Copy as CopyIcon, FileText, Languages} from 'lucide-react'
import type {RecorderCopy} from '../i18n'
import {OcrPositionTextLayer} from './ocr/OcrPositionTextLayer'
import {readableError, type OcrResult} from '../services/recorderBackend'

export function OcrResultPanel({
  copy,
  result,
  translation,
  loading,
  error,
  expanded,
  autoTranslate = false,
  onToggleExpanded,
  onClose,
  onCopy }: {
  copy: RecorderCopy
  result: OcrResult | null
  translation: AppSettings['ocr']['translation']
  loading: boolean
  error: string
  expanded: boolean
  autoTranslate?: boolean
  onToggleExpanded: () => void
  onClose: () => void
  onCopy: () => void
}) {
  const plainText = result?.plainText.trim() ?? ''
  const [previewDataUrl, setPreviewDataUrl] = useState('')
  const [previewError, setPreviewError] = useState('')
  const [hoveredBlockId, setHoveredBlockId] = useState('')
  const [copiedBlockId, setCopiedBlockId] = useState('')
  const [translationCopied, setTranslationCopied] = useState(false)
  const [translationMessage, setTranslationMessage] = useState('')
  const [translationBusy, setTranslationBusy] = useState(false)
  const [translationResult, setTranslationResult] = useState<OcrTranslationResult | null>(null)
  const previewLogKeyRef = useRef('')
  const renderedLogKeyRef = useRef('')
  const autoTranslateKeyRef = useRef('')
  const translatedText = translationResult?.blocks.map((block) => block.translated.trim()).filter(Boolean).join('\n').trim() ?? ''

  useEffect(() => {
    let cancelled = false
    previewLogKeyRef.current = ''
    renderedLogKeyRef.current = ''
    setPreviewDataUrl('')
    setPreviewError('')
    setHoveredBlockId('')
    setCopiedBlockId('')
    setTranslationCopied(false)
    setTranslationMessage('')
    setTranslationBusy(false)
    setTranslationResult(null)
    autoTranslateKeyRef.current = ''
    if (!result) return
    void readOcrResultImage(result.id)
      .then((image) => {
        if (cancelled) return
        const logKey = `${result.id}:${image.available ? 'available' : 'missing'}:${image.bytes || 0}`
        if (previewLogKeyRef.current !== logKey) {
          previewLogKeyRef.current = logKey
          void logClientEvent('ocr-result', 'preview-loaded', {
            resultId: result.id,
            sourceKind: result.sourceKind,
            sourceId: result.sourceId,
            width: result.width,
            height: result.height,
            blockCount: result.blocks.length,
            available: image.available,
            bytes: image.bytes || 0 })
        }
        if (image.available && image.dataUrl) {
          setPreviewDataUrl(image.dataUrl)
        }
      })
      .catch((error) => {
        if (cancelled) return
        setPreviewError(readableError(error))
        void logClientEvent('ocr-result', 'preview-error', {
          resultId: result.id,
          sourceKind: result.sourceKind,
          sourceId: result.sourceId })
      })
    return () => {
      cancelled = true
    }
  }, [result?.id])

  useEffect(() => {
    if (!result || !previewDataUrl) return
    const polygonCount = result.blocks.filter((block) => block.box.length >= 4 && ocrBlockPolygonPoints(block)).length
    const logKey = `${result.id}:${result.width}x${result.height}:${result.blocks.length}:${polygonCount}`
    if (renderedLogKeyRef.current === logKey) return
    let cancelled = false
    const frame = window.requestAnimationFrame(() => {
      if (cancelled || renderedLogKeyRef.current === logKey) return
      renderedLogKeyRef.current = logKey
      void logClientEvent('ocr-result', 'rendered', {
        resultId: result.id,
        sourceKind: result.sourceKind,
        sourceId: result.sourceId,
        width: result.width,
        height: result.height,
        blockCount: result.blocks.length,
        polygonCount,
        hasPreview: true })
    })
    return () => {
      cancelled = true
      window.cancelAnimationFrame(frame)
    }
  }, [previewDataUrl, result])

  const copyBlockText = (block: OcrBlock, blockId = block.id) => {
    const text = block.text.trim()
    if (!text) return
    void writeClipboardText(text)
      .then(() => {
        void logClientEvent('ocr-result', 'copy-block', {
          resultId: result?.id ?? '',
          sourceKind: result?.sourceKind ?? '',
          sourceId: result?.sourceId ?? '',
          blockId })
        setCopiedBlockId(blockId)
        window.setTimeout(() => setCopiedBlockId((current) => current === blockId ? '' : current), 1200)
      })
      .catch(() => undefined)
  }

  const runTranslation = useCallback((force = false) => {
    if (!result || !plainText || translationBusy) return
    const normalized = normalizeOcrTranslationSettings(translation)
    const unavailable = ocrTranslationUnavailableMessage(normalized, copy)
    if (unavailable) {
      setTranslationResult(null)
      setTranslationMessage(unavailable)
      return
    }
    setTranslationBusy(true)
    setTranslationMessage(copy.screenshot.translationWorking)
    void translateOcr({
      ocrResultId: result.id,
      provider: normalized.provider,
      sourceLanguage: normalized.sourceLanguage,
      targetLanguage: normalized.targetLanguage,
      baseUrl: normalized.baseUrl,
      apiKey: normalized.apiKey,
      model: normalized.model,
      force })
      .then((next) => {
        setTranslationResult(next)
        setTranslationMessage(copy.screenshot.translationReady)
        void logClientEvent('ocr-result', 'translation-rendered', {
          resultId: result.id,
          sourceKind: result.sourceKind,
          sourceId: result.sourceId,
          provider: next.provider,
          targetLanguage: next.targetLanguage,
          blockCount: next.blocks.length })
      })
      .catch((error) => {
        setTranslationResult(null)
        setTranslationMessage(`${copy.screenshot.translationFailed}: ${readableError(error)}`)
      })
      .finally(() => setTranslationBusy(false))
  }, [copy, plainText, result, translation, translationBusy])

  useEffect(() => {
    if (!autoTranslate || !result || !plainText || translationBusy || translationResult) return
    const key = `${result.id}:${translation.provider}:${translation.targetLanguage}:${translation.model ?? ''}:${translation.baseUrl ?? ''}`
    if (autoTranslateKeyRef.current === key) return
    autoTranslateKeyRef.current = key
    runTranslation(false)
  }, [autoTranslate, plainText, result, runTranslation, translation.baseUrl, translation.model, translation.provider, translation.targetLanguage, translationBusy, translationResult])

  const copyTranslatedText = () => {
    if (!translatedText) return
    void writeClipboardText(translatedText)
      .then(() => {
        setTranslationCopied(true)
        window.setTimeout(() => setTranslationCopied(false), 1200)
      })
      .catch(() => undefined)
  }

  return (
    <section className={`ocr-result-panel ${expanded ? 'expanded' : ''}`}>
      <div className="ocr-result-header">
        <span>
          <Eye size={16} />
          <strong>{copy.screenshot.ocrResult}</strong>
        </span>
        <div className="ocr-result-header-actions">
          <button
            type="button"
            className="icon-button"
            aria-label={expanded ? copy.screenshot.collapseOcrResult : copy.screenshot.expandOcrResult}
            title={expanded ? copy.screenshot.collapseOcrResult : copy.screenshot.expandOcrResult}
            onClick={onToggleExpanded}
          >
            {expanded ? <Minimize2 size={15} /> : <Maximize2 size={15} />}
          </button>
          <button type="button" className="sheet-close" onClick={onClose}>{copy.common.close}</button>
        </div>
      </div>
      {loading && <div className="source-empty"><FileText size={18} /><span>{copy.screenshot.ocrStatusRunning}</span></div>}
      {!loading && error && <div className="source-empty error"><FileText size={18} /><span>{error}</span></div>}
      {!loading && !error && result && (
        <>
          <div className="ocr-result-summary">
            <span>{result.modelId || copy.screenshot.ocr}</span>
            <b>{copy.screenshot.ocrBlocks(result.blocks.length)}</b>
          </div>
          {previewDataUrl && (
            <div className="ocr-preview-frame">
              <img src={previewDataUrl} alt={copy.screenshot.ocrResult} draggable={false} />
              <svg viewBox={`0 0 ${Math.max(1, result.width)} ${Math.max(1, result.height)}`} preserveAspectRatio="none" aria-hidden="true">
                {result.blocks.map((block, index) => {
                  const blockId = ocrBlockStableId(block, index)
                  return (
                    <polygon
                      key={blockId}
                      points={ocrBlockPolygonPoints(block)}
                      className={blockId === hoveredBlockId ? 'active' : ''}
                    />
                  )
                })}
              </svg>
              <OcrPositionTextLayer
                copy={copy}
                result={result}
                translationResult={translationResult}
                hoveredBlockId={hoveredBlockId}
                copiedBlockId={copiedBlockId}
                onHover={setHoveredBlockId}
                onCopy={copyBlockText}
              />
            </div>
          )}
          {!previewDataUrl && previewError && <div className="ocr-preview-note">{previewError}</div>}
          <div className="ocr-result-text">
            {plainText || copy.screenshot.ocrNoText}
          </div>
          <div className="ocr-result-actions">
            <button type="button" className="menu-row" disabled={!plainText} onClick={onCopy}>
              <CopyIcon size={16} />
              <span><strong>{copy.screenshot.copyText}</strong></span>
            </button>
            <button type="button" className="menu-row" disabled={!plainText || translationBusy} onClick={() => runTranslation(false)}>
              <Languages size={16} />
              <span><strong>{translationBusy ? copy.screenshot.translationWorking : copy.screenshot.translateText}</strong></span>
            </button>
            <button type="button" className="menu-row" disabled={!translatedText} onClick={copyTranslatedText}>
              <CopyIcon size={16} />
              <span><strong>{translationCopied ? copy.screenshot.copiedTranslation : copy.screenshot.copyTranslation}</strong></span>
            </button>
          </div>
          {translationMessage && <div className="ocr-translation-note">{translationMessage}</div>}
          {translationResult && translationResult.blocks.length > 0 && (
            <div className="ocr-translation-list" aria-label={copy.screenshot.translationReady}>
              {translationResult.blocks.map((block) => (
                <div className="ocr-translation-row" key={block.blockId}>
                  <span>{block.source}</span>
                  <strong>{block.translated}</strong>
                </div>
              ))}
            </div>
          )}
          <div className="ocr-block-list" aria-label={copy.screenshot.ocrBlocks(result.blocks.length)}>
            {result.blocks.map((block, index) => {
              const blockId = ocrBlockStableId(block, index)
              return (
                <button
                  type="button"
                  className={`ocr-block-row ${blockId === hoveredBlockId ? 'active' : ''}`}
                  key={blockId}
                  onPointerEnter={() => setHoveredBlockId(blockId)}
                  onPointerLeave={() => setHoveredBlockId('')}
                  onFocus={() => setHoveredBlockId(blockId)}
                  onBlur={() => setHoveredBlockId('')}
                  onClick={() => copyBlockText(block, blockId)}
                  title={copiedBlockId === blockId ? copy.screenshot.copiedText : copy.screenshot.copyText}
                >
                  <span>{block.text || copy.screenshot.ocrNoText}</span>
                  <b>{copiedBlockId === blockId ? copy.screenshot.copiedText : `${Math.round((block.confidence || 0) * 100)}%`}</b>
                </button>
              )
            })}
          </div>
        </>
      )}
      {!loading && !error && !result && <div className="source-empty"><FileText size={18} /><span>{copy.screenshot.ocrNoText}</span></div>}
    </section>
  )
}

