import type {CSSProperties} from 'react'
import type {RecorderCopy} from '../../i18n'
import type {OcrBlock, OcrResult, OcrTranslationResult} from '../../services/recorderBackend'

export function OcrPositionTextLayer({
  copy,
  result,
  translationResult = null,
  hoveredBlockId,
  copiedBlockId,
  onHover,
  onCopy,
  className = '',
  style,
}: {
  copy: RecorderCopy
  result: OcrResult
  translationResult?: OcrTranslationResult | null
  hoveredBlockId: string
  copiedBlockId: string
  onHover: (blockId: string) => void
  onCopy: (block: OcrBlock, blockId: string) => void
  className?: string
  style?: CSSProperties
}) {
  const translatedByBlockId = new Map((translationResult?.blocks ?? [])
    .map((block) => [block.blockId, block.translated.trim()] as const)
    .filter(([, translated]) => Boolean(translated)))
  const blockButtons = result.blocks.map((block, index) => {
    const blockId = ocrBlockStableId(block, index)
    const bounds = ocrBlockBoundsPercent(block, result.width, result.height)
    if (!bounds) return null
    const copied = copiedBlockId === blockId
    const translated = translatedByBlockId.get(block.id) || translatedByBlockId.get(blockId) || ''
    const label = translated || block.text.trim() || copy.screenshot.ocrNoText
    const displayBlock = translated ? {...block, text: translated} : block
    const buttonStyle = {
      left: `${bounds.left}%`,
      top: `${bounds.top}%`,
      width: `${bounds.width}%`,
    } satisfies CSSProperties
    return (
      <button
        type="button"
        key={blockId}
        className={`ocr-position-text-button ${translated ? 'translated' : ''} ${blockId === hoveredBlockId ? 'active' : ''} ${copied ? 'copied' : ''}`.trim()}
        style={buttonStyle}
        aria-label={`${copy.screenshot.copyText}: ${label}`}
        title={copied ? copy.screenshot.copiedText : label}
        data-ocr-block-id={blockId}
        onPointerEnter={() => onHover(blockId)}
        onPointerLeave={() => onHover('')}
        onFocus={() => onHover(blockId)}
        onBlur={() => onHover('')}
        onClick={() => onCopy(displayBlock, blockId)}
      >
        <span>{copied ? copy.screenshot.copiedText : label}</span>
      </button>
    )
  }).filter(Boolean)

  if (blockButtons.length === 0) return null
  return (
    <div className={`ocr-position-text-layer ${className}`.trim()} style={style} aria-label={copy.screenshot.ocrBlocks(result.blocks.length)}>
      {blockButtons}
    </div>
  )
}

export function countOcrPositionTextBlocks(result: OcrResult) {
  return result.blocks.filter((block) => ocrBlockBoundsPercent(block, result.width, result.height)).length
}

export function ocrBlockPolygonPoints(block: OcrBlock) {
  if (block.box.length === 0) return ''
  return block.box.map((point) => `${point.x},${point.y}`).join(' ')
}

export function ocrBlockStableId(block: OcrBlock, index: number) {
  if (block.id) return block.id
  const boxKey = block.box.map((point) => `${Math.round(point.x)}:${Math.round(point.y)}`).join('|')
  return `block-${index}-${block.lineIndex}-${boxKey}-${block.text}`
}

export function ocrBlockBoundsPercent(block: OcrBlock, resultWidth: number, resultHeight: number) {
  if (!Number.isFinite(resultWidth) || !Number.isFinite(resultHeight) || resultWidth <= 0 || resultHeight <= 0 || block.box.length === 0) {
    return null
  }
  const xs = block.box.map((point) => point.x).filter(Number.isFinite)
  const ys = block.box.map((point) => point.y).filter(Number.isFinite)
  if (xs.length === 0 || ys.length === 0) return null
  const minX = clampNumber(Math.min(...xs), 0, resultWidth)
  const maxX = clampNumber(Math.max(...xs), 0, resultWidth)
  const minY = clampNumber(Math.min(...ys), 0, resultHeight)
  const maxY = clampNumber(Math.max(...ys), 0, resultHeight)
  if (maxX <= minX || maxY <= minY) return null
  return {
    left: (minX / resultWidth) * 100,
    top: (minY / resultHeight) * 100,
    width: Math.max(4, ((maxX - minX) / resultWidth) * 100),
    height: Math.max(8, ((maxY - minY) / resultHeight) * 100),
  }
}

function clampNumber(value: number, min: number, max: number) {
  if (!Number.isFinite(value)) return min
  return Math.min(max, Math.max(min, value))
}
